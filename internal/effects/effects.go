package effects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Resultado de uma verificação.
const (
	VerdictConfirmed    = "confirmed"    // o efeito existe no destino
	VerdictAbsent       = "absent"       // comprovadamente não existe: repetir é seguro
	VerdictInconclusive = "inconclusive" // não dá para saber
)

// Resultados de Run.
const (
	OutcomeApplied          = "applied"           // enviado e confirmado
	OutcomeAlreadyConfirmed = "already_confirmed" // já tinha sido feito: nada foi reenviado
	OutcomeReconciled       = "reconciled"        // a resposta se perdeu, mas a prova mostrou que o efeito existe
	OutcomeFailed           = "failed"            // recusado ou ausente depois das tentativas
)

// Erros com resultado definido.
var (
	// ErrNotApplied é devolvido pelo adaptador quando o destino RECUSOU a operação
	// (ex.: 400/403): sabidamente não houve efeito, sem precisar verificar.
	ErrNotApplied = errors.New("operação recusada pelo destino")
	// ErrBlocked: o estado do efeito é inconclusivo; a repetição automática está
	// bloqueada até uma decisão humana.
	ErrBlocked = errors.New("efeito inconclusivo: repetição automática bloqueada")
	// ErrInFlight: outro envio da mesma operação está em andamento.
	ErrInFlight = errors.New("o efeito já está sendo enviado por outro processo")
)

// Op identifica uma operação para o adaptador.
type Op struct {
	Key       string          // chave de negócio (use-a como marcador no destino quando possível)
	Operation string          // ex.: create_subtask, push, create_pr
	Request   json.RawMessage // os dados da operação
}

// Verdict é o resultado de uma verificação.
type Verdict struct {
	Status   string
	Proof    any    // o que foi encontrado (ref, ID remoto, SHA)
	Evidence string // texto para o humano quando inconclusivo
}

// Adapter fala com o destino. Apply envia; Verify procura prova verificável de que
// a operação aconteceu, mesmo quando a API não oferece chave de idempotência.
type Adapter interface {
	Apply(ctx context.Context, op Op) (proof any, err error)
	Verify(ctx context.Context, op Op) (Verdict, error)
}

// DB é o que o ledger precisa do banco.
type DB interface {
	persistence.DBTX
}

// Spec descreve uma operação.
type Spec struct {
	ProjectID   string
	BusinessKey string // única por efeito lógico, estável entre tentativas
	Operation   string
	Request     any
}

// Result é o desfecho de uma execução do ledger.
type Result struct {
	Entry   persistence.OutboxEntry
	Outcome string
}

// Ledger aplica efeitos externos de forma idempotente.
type Ledger struct {
	DB          DB
	Redactor    *security.Redactor
	Lease       time.Duration // quanto tempo um envio em andamento é considerado vivo (padrão 2 min)
	MaxAttempts int           // envios por Run quando a ausência é comprovada (padrão 3)
	// Escalate abre o bloqueio com evidência quando o estado é inconclusivo
	// (normalmente cria um pedido na Central de Decisões).
	Escalate func(ctx context.Context, e persistence.OutboxEntry, evidence string) error
}

func (l *Ledger) lease() time.Duration {
	if l.Lease > 0 {
		return l.Lease
	}
	return 2 * time.Minute
}

func (l *Ledger) maxAttempts() int {
	if l.MaxAttempts > 0 {
		return l.MaxAttempts
	}
	return 3
}

func (l *Ledger) redactRequest(req any) any {
	raw, err := json.Marshal(req)
	if err != nil || req == nil {
		return map[string]any{}
	}
	rd := l.Redactor
	if rd == nil {
		rd = security.NewRedactor()
	}
	clean := rd.Redact(string(raw))
	var out json.RawMessage
	if json.Valid([]byte(clean)) {
		out = json.RawMessage(clean)
		return out
	}
	return map[string]any{"redacted": true}
}

// Run garante que a operação aconteça no máximo uma vez no destino:
//  1. a intenção é gravada ANTES de qualquer envio;
//  2. já confirmada → nada é reenviado;
//  3. inconclusiva → ErrBlocked (humano decide);
//  4. envio em andamento dentro da concessão → ErrInFlight; vencida → reconcilia primeiro;
//  5. se a resposta se perder, a prova decide: confirmado não duplica, ausente permite repetir,
//     inconclusivo bloqueia e abre decisão com evidência.
func (l *Ledger) Run(ctx context.Context, spec Spec, ad Adapter) (Result, error) {
	if strings.TrimSpace(spec.BusinessKey) == "" || strings.TrimSpace(spec.Operation) == "" || spec.ProjectID == "" {
		return Result{}, fmt.Errorf("%w: projeto, operação e chave de negócio são obrigatórios", persistence.ErrInvalidInput)
	}
	entry, _, err := persistence.EnqueueOutbox(ctx, l.DB, spec.ProjectID, spec.BusinessKey, spec.Operation, l.redactRequest(spec.Request))
	if err != nil {
		return Result{}, err
	}
	op := func(e persistence.OutboxEntry) Op {
		return Op{Key: e.BusinessKey, Operation: e.Destination, Request: spec.requestJSON()}
	}
	switch entry.State {
	case persistence.OutboxConfirmed:
		return Result{Entry: entry, Outcome: OutcomeAlreadyConfirmed}, nil
	case persistence.OutboxUncertain:
		return Result{Entry: entry}, ErrBlocked
	case persistence.OutboxSending:
		var age time.Duration
		if err := l.DB.QueryRow(ctx, `SELECT now() - updated_at FROM outbox WHERE id = $1`, entry.ID).Scan(&age); err != nil {
			return Result{}, persistence.MapError(err)
		}
		if age < l.lease() {
			return Result{Entry: entry}, ErrInFlight
		}
		// Envio abandonado (queda do processo): reconcilia antes de qualquer repetição.
		return l.settle(ctx, entry, op(entry), ad, true)
	}
	// intent ou failed: reivindica o envio (só um processo vence).
	entry, err = persistence.TransitionOutbox(ctx, l.DB, spec.ProjectID, entry.ID, entry.Version, persistence.OutboxSending, nil)
	if errors.Is(err, persistence.ErrVersionConflict) {
		return Result{}, ErrInFlight
	}
	if err != nil {
		return Result{}, err
	}
	return l.settle(ctx, entry, op(entry), ad, false)
}

func (s Spec) requestJSON() json.RawMessage {
	raw, err := json.Marshal(s.Request)
	if err != nil || s.Request == nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

// settle conduz a entrada (em `sending`) até um estado final. reconcileFirst
// verifica antes de enviar (envio abandonado).
func (l *Ledger) settle(ctx context.Context, entry persistence.OutboxEntry, op Op, ad Adapter, reconcileFirst bool) (Result, error) {
	bg := context.WithoutCancel(ctx) // as gravações do ledger não podem se perder com o cancelamento do chamador
	transition := func(to string, proof any) error {
		var err error
		entry, err = persistence.TransitionOutbox(bg, l.DB, entry.ProjectID, entry.ID, entry.Version, to, proof)
		return err
	}
	verify := func() Verdict {
		vctx, cancel := context.WithTimeout(bg, 30*time.Second)
		defer cancel()
		v, err := ad.Verify(vctx, op)
		if err != nil {
			return Verdict{Status: VerdictInconclusive, Evidence: "a verificação no destino falhou: " + clip(err.Error())}
		}
		switch v.Status {
		case VerdictConfirmed, VerdictAbsent, VerdictInconclusive:
		default:
			return Verdict{Status: VerdictInconclusive, Evidence: "veredito desconhecido do adaptador"}
		}
		return v
	}
	block := func(evidence string) (Result, error) {
		if err := transition(persistence.OutboxUncertain, map[string]any{"evidence": evidence}); err != nil {
			return Result{}, err
		}
		if l.Escalate != nil {
			if err := l.Escalate(bg, entry, evidence); err != nil {
				return Result{Entry: entry}, fmt.Errorf("%w (a decisão não pôde ser aberta: %v)", ErrBlocked, err)
			}
		}
		return Result{Entry: entry}, ErrBlocked
	}

	pending := reconcileFirst
	for {
		if pending {
			pending = false
			switch v := verify(); v.Status {
			case VerdictConfirmed:
				if err := transition(persistence.OutboxConfirmed, v.Proof); err != nil {
					return Result{}, err
				}
				return Result{Entry: entry, Outcome: OutcomeReconciled}, nil
			case VerdictInconclusive:
				return block(v.Evidence)
			}
			// Ausente: repetir é seguro. Conta a nova tentativa.
			if err := l.retry(&entry, bg, "absent"); err != nil {
				return Result{}, err
			}
		}
		proof, err := ad.Apply(ctx, op)
		if err == nil {
			if terr := transition(persistence.OutboxConfirmed, proof); terr != nil {
				return Result{}, terr
			}
			return Result{Entry: entry, Outcome: OutcomeApplied}, nil
		}
		if errors.Is(err, ErrNotApplied) { // recusa definitiva: sem efeito, sem necessidade de provar
			if terr := transition(persistence.OutboxFailed, map[string]any{"refused": clip(err.Error())}); terr != nil {
				return Result{}, terr
			}
			return Result{Entry: entry, Outcome: OutcomeFailed}, err
		}
		// Resposta perdida: não presume nada; a prova decide.
		v := verify()
		switch v.Status {
		case VerdictConfirmed:
			if terr := transition(persistence.OutboxConfirmed, v.Proof); terr != nil {
				return Result{}, terr
			}
			return Result{Entry: entry, Outcome: OutcomeReconciled}, nil
		case VerdictInconclusive:
			return block(v.Evidence)
		}
		if entry.Attempts >= l.maxAttempts() {
			if terr := transition(persistence.OutboxFailed, map[string]any{"absent": true, "last_error": clip(err.Error())}); terr != nil {
				return Result{}, terr
			}
			return Result{Entry: entry, Outcome: OutcomeFailed}, fmt.Errorf("o efeito não aconteceu após %d tentativas: %w", entry.Attempts, err)
		}
		if rerr := l.retry(&entry, bg, clip(err.Error())); rerr != nil {
			return Result{}, rerr
		}
	}
}

// retry registra a ausência comprovada (failed) e reivindica uma nova tentativa (sending).
func (l *Ledger) retry(entry *persistence.OutboxEntry, ctx context.Context, why string) error {
	e, err := persistence.TransitionOutbox(ctx, l.DB, entry.ProjectID, entry.ID, entry.Version, persistence.OutboxFailed, map[string]any{"absent": true, "reason": why})
	if err != nil {
		return err
	}
	e, err = persistence.TransitionOutbox(ctx, l.DB, e.ProjectID, e.ID, e.Version, persistence.OutboxSending, nil)
	if err != nil {
		return err
	}
	*entry = e
	return nil
}

func clip(s string) string {
	if len(s) > 300 {
		return s[:300]
	}
	return s
}

// Resoluções humanas de um efeito inconclusivo.
const (
	ResolveApplied    = "applied"     // o efeito existe no destino
	ResolveNotApplied = "not_applied" // não existe: pode repetir
)

// Resolve registra a decisão humana sobre um efeito `uncertain`. "applied" fecha
// como confirmado (prova manual, sem reenviar); "not_applied" libera a repetição.
func (l *Ledger) Resolve(ctx context.Context, projectID, id, resolution, note string) (persistence.OutboxEntry, error) {
	e, err := persistence.GetOutbox(ctx, l.DB, projectID, id)
	if err != nil {
		return persistence.OutboxEntry{}, err
	}
	if e.State != persistence.OutboxUncertain {
		return persistence.OutboxEntry{}, fmt.Errorf("%w: o efeito não está inconclusivo (estado %s)", persistence.ErrInvalidInput, e.State)
	}
	proof := map[string]any{"manual": true, "resolution": resolution, "note": clip(note)}
	switch resolution {
	case ResolveApplied:
		return persistence.TransitionOutbox(ctx, l.DB, projectID, id, e.Version, persistence.OutboxConfirmed, proof)
	case ResolveNotApplied:
		return persistence.TransitionOutbox(ctx, l.DB, projectID, id, e.Version, persistence.OutboxFailed, proof)
	}
	return persistence.OutboxEntry{}, fmt.Errorf("%w: resolução desconhecida", persistence.ErrInvalidInput)
}
