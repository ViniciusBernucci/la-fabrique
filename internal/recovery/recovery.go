package recovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Classes de falha, cada uma com o seu limite de tentativas.
type Class string

const (
	Infra  Class = "infra"  // worker, VPS, runtime de containers, rede
	AI     Class = "ai"     // provedor, cota, resposta inutilizável
	Review Class = "review" // rodadas de revisão esgotadas
)

// Limits são os máximos de recuperações automáticas por classe.
type Limits map[Class]int

// DefaultLimits: infraestrutura tolera mais tentativas que IA e revisão.
func DefaultLimits() Limits { return Limits{Infra: 3, AI: 2, Review: 2} }

// Ações do resultado.
const (
	ActionRecovered = "recovered" // voltou à fila sozinho
	ActionDecision  = "decision"  // incerteza: decisão humana aberta
	ActionLimit     = "limit"     // limite da classe esgotado: decisão aberta
	ActionDeferred  = "deferred"  // não foi possível inspecionar: nada mudou
)

// Inspector responde sobre o ambiente da tentativa interrompida.
type Inspector interface {
	// Quiescent diz se não há escritor vivo (sandbox sem processos, ou inexistente).
	Quiescent(ctx context.Context, it queue.Item) (bool, error)
	// Verify confere Git, artefatos e políticas contra o checkpoint (nil = sem checkpoint) e
	// devolve os problemas encontrados (vazio = íntegro).
	Verify(ctx context.Context, it queue.Item, cp *checkpoints.Checkpoint) ([]string, error)
}

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Recoverer recupera tentativas interrompidas.
type Recoverer struct {
	DB          DB
	Queue       *queue.Queue
	Checkpoints *checkpoints.Store
	Inspector   Inspector
	Redactor    *security.Redactor
	Limits      Limits
}

// Outcome é o resultado auditável de uma recuperação.
type Outcome struct {
	ItemID       string             `json:"item_id"`
	Action       string             `json:"action"`
	Class        Class              `json:"class"`
	Reasons      []string           `json:"reasons"`
	CheckpointID string             `json:"checkpoint_id,omitempty"`
	SlotReleased bool               `json:"slot_released"`
	Attempts     int                `json:"attempts"`
	Decision     *decisions.Request `json:"decision,omitempty"`
}

func (r *Recoverer) limit(c Class) int {
	if r.Limits != nil {
		if n, ok := r.Limits[c]; ok {
			return n
		}
	}
	return DefaultLimits()[c]
}

func (r *Recoverer) attempts(ctx context.Context, itemID string, c Class) (int, error) {
	var n int
	err := r.DB.QueryRow(ctx, `SELECT attempts FROM recovery_attempts WHERE item_id = $1 AND class = $2`, itemID, string(c)).Scan(&n)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return 0, nil
	}
	return n, persistence.MapError(err)
}

func (r *Recoverer) bump(ctx context.Context, itemID string, c Class) (int, error) {
	var n int
	err := r.DB.QueryRow(ctx, `INSERT INTO recovery_attempts (item_id, class, attempts) VALUES ($1, $2, 1)
		ON CONFLICT (item_id, class) DO UPDATE SET attempts = recovery_attempts.attempts + 1, updated_at = now() RETURNING attempts`, itemID, string(c)).Scan(&n)
	return n, persistence.MapError(err)
}

func (r *Recoverer) event(ctx context.Context, it queue.Item, o Outcome) {
	d, _ := json.Marshal(map[string]any{"reasons": o.Reasons, "attempts": o.Attempts})
	var cp any
	if o.CheckpointID != "" {
		cp = o.CheckpointID
	}
	_, _ = r.DB.Exec(ctx, `INSERT INTO recovery_events (item_id, project_id, action, class, checkpoint_id, slot_released, detail) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)`,
		it.ID, it.ProjectID, o.Action, string(o.Class), cp, o.SlotReleased, string(d))
}

// effects lê os efeitos da execução (convenção de chave exec:<execução>:): incertos e confirmados.
func (r *Recoverer) effects(ctx context.Context, it queue.Item) (uncertain, confirmed []string, err error) {
	if it.ExecutionID == "" {
		return nil, nil, nil
	}
	rows, err := r.DB.Query(ctx, `SELECT business_key, state FROM outbox WHERE project_id = $1 AND starts_with(business_key, $2) ORDER BY business_key`,
		it.ProjectID, "exec:"+it.ExecutionID+":")
	if err != nil {
		return nil, nil, persistence.MapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var k, st string
		if err := rows.Scan(&k, &st); err != nil {
			return nil, nil, persistence.MapError(err)
		}
		switch st {
		case "confirmed":
			confirmed = append(confirmed, k)
		case "failed":
		default:
			uncertain = append(uncertain, k)
		}
	}
	return uncertain, confirmed, rows.Err()
}

func (r *Recoverer) park(ctx context.Context, it queue.Item) error {
	var state string
	var version int
	if err := r.DB.QueryRow(ctx, `SELECT state, version FROM executions WHERE project_id = $1 AND id = $2`, it.ProjectID, it.ExecutionID).Scan(&state, &version); err != nil {
		return persistence.MapError(err)
	}
	if state == "waiting" {
		return nil
	}
	_, err := persistence.UpdateExecutionState(ctx, r.DB, it.ProjectID, it.ExecutionID, version, "waiting")
	return err
}

// Stale lista as tentativas interrompidas (lease vencida), sem alterar nada.
func (r *Recoverer) Stale(ctx context.Context) ([]queue.Item, error) { return r.Queue.ListStale(ctx) }

// Recover diagnostica UMA tentativa interrompida e decide: voltar à fila (seguro) ou pedir
// decisão (incerteza ou limite esgotado). Não toca em nenhum outro item.
func (r *Recoverer) Recover(ctx context.Context, it queue.Item, class Class) (Outcome, error) {
	out := Outcome{ItemID: it.ID, Class: class, Reasons: []string{}}
	if class != Infra && class != AI && class != Review {
		return out, &FieldError{"class", "use infra, ai ou review"}
	}
	// 1. Inspeção do ambiente: se não consegue responder, adia sem consumir tentativa.
	quiet, err := r.Inspector.Quiescent(ctx, it)
	if err != nil {
		out.Action = ActionDeferred
		out.Reasons = append(out.Reasons, "não foi possível verificar a quietude do sandbox: "+r.Redactor.Redact(err.Error()))
		r.event(ctx, it, out)
		return out, nil
	}
	var cp *checkpoints.Checkpoint
	cpProblem := ""
	if it.ExecutionID != "" {
		c, err := r.Checkpoints.Latest(ctx, it.ProjectID, it.ExecutionID)
		switch {
		case err == nil:
			cp = &c
			out.CheckpointID = c.ID
		case errors.Is(err, persistence.ErrNotFound):
		case errors.Is(err, checkpoints.ErrCorrupt) || errors.Is(err, checkpoints.ErrBadPath):
			cpProblem = "o último checkpoint está corrompido: " + err.Error()
		default:
			out.Action = ActionDeferred
			out.Reasons = append(out.Reasons, "não foi possível ler o checkpoint: "+err.Error())
			r.event(ctx, it, out)
			return out, nil
		}
	}
	problems, err := r.Inspector.Verify(ctx, it, cp)
	if err != nil {
		out.Action = ActionDeferred
		out.Reasons = append(out.Reasons, "não foi possível verificar Git, artefatos e políticas: "+r.Redactor.Redact(err.Error()))
		r.event(ctx, it, out)
		return out, nil
	}
	uncertain, confirmed, err := r.effects(ctx, it)
	if err != nil {
		return out, err
	}

	// 2. Incertezas.
	var doubts []string
	if !quiet {
		doubts = append(doubts, "ainda há processo vivo no sandbox: um escritor antigo pode estar ativo")
	}
	if cpProblem != "" {
		doubts = append(doubts, cpProblem)
	}
	for _, p := range problems {
		doubts = append(doubts, "estado verificável: "+p)
	}
	if len(uncertain) > 0 {
		doubts = append(doubts, "efeito(s) externo(s) sem confirmação: "+strings.Join(uncertain, ", "))
	}
	if cp == nil && cpProblem == "" && len(confirmed) > 0 {
		doubts = append(doubts, "há efeitos já confirmados mas nenhum checkpoint para retomar de onde parou")
	}

	// 3. Limite da classe (só conta quando a recuperação seria automática).
	n, err := r.attempts(ctx, it.ID, class)
	if err != nil {
		return out, err
	}
	out.Attempts = n
	limitHit := len(doubts) == 0 && n >= r.limit(class)
	if limitHit {
		doubts = append(doubts, fmt.Sprintf("limite de %d recuperação(ões) automática(s) da classe %q esgotado", r.limit(class), class))
	}

	if len(doubts) == 0 {
		cpID := ""
		if cp != nil {
			cpID = cp.ID
		}
		if _, err := r.bump(ctx, it.ID, class); err != nil {
			return out, err
		}
		out.Attempts = n + 1
		if _, err := r.Queue.RecoverRequeue(ctx, it.ID, cpID); err != nil {
			return out, err
		}
		out.Action, out.SlotReleased = ActionRecovered, true
		if cp == nil {
			out.Reasons = append(out.Reasons, "sem checkpoint e sem efeitos: recomeça do início")
		} else {
			out.Reasons = append(out.Reasons, "ambiente quieto, checkpoint íntegro, sem efeito incerto: retoma do checkpoint")
		}
		r.event(ctx, it, out)
		return out, nil
	}

	// 4. Incerteza ou limite: decisão humana, bloqueando só este item.
	out.Action = ActionDecision
	if limitHit {
		out.Action = ActionLimit
	}
	out.Reasons = append(out.Reasons, doubts...)
	release := quiet && cp != nil && cpProblem == "" // vaga só com quietude provada e checkpoint íntegro
	if it.ExecutionID == "" {
		out.Action = ActionDeferred
		out.Reasons = append(out.Reasons, "o item não tem execução para receber a decisão")
		r.event(ctx, it, out)
		return out, nil
	}
	if err := r.park(ctx, it); err != nil {
		return out, err
	}
	dec, _, err := decisions.Create(ctx, r.DB, r.Redactor, decisions.NewRequest{
		ProjectID: it.ProjectID, ExecutionID: it.ExecutionID, NodeID: it.TaskKey, Kind: decisions.KindUncertainResult,
		Title:    "Tentativa interrompida precisa de decisão",
		Problem:  "A tentativa da tarefa " + it.TaskKey + " foi interrompida (crash ou reboot) e retomá-la agora não é seguro: " + strings.Join(doubts, "; "),
		Impact:   "Só esta tarefa fica bloqueada; as demais tarefas elegíveis continuam.",
		Evidence: []decisions.Evidence{{Label: "Diagnóstico", Text: strings.Join(doubts, "\n")}},
		Options: []decisions.Option{
			{ID: "retry", Label: "Tentar de novo", Description: "Devolver a tarefa à fila e retomar do último checkpoint íntegro."},
			{ID: "stop", Label: "Manter parada", Description: "Não retomar até uma investigação."},
		},
	})
	if err != nil {
		return out, err
	}
	out.Decision = &dec
	cpID := ""
	if cp != nil {
		cpID = cp.ID
	}
	if _, err := r.Queue.RecoverPark(ctx, it.ID, queue.WaitDecision, dec.ID, cpID, release); err != nil {
		return out, err
	}
	out.SlotReleased = release
	r.event(ctx, it, out)
	return out, nil
}

// RecoverAll processa cada tentativa interrompida de forma independente: a incerteza de uma
// não impede as outras. `classOf` diz a classe da falha de cada item (nil = infraestrutura,
// o caso de worker/VPS que sumiu).
func (r *Recoverer) RecoverAll(ctx context.Context, classOf func(queue.Item) Class) ([]Outcome, error) {
	stale, err := r.Stale(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Outcome, 0, len(stale))
	for _, it := range stale {
		c := Infra
		if classOf != nil {
			c = classOf(it)
		}
		o, err := r.Recover(ctx, it, c)
		if err != nil {
			o.Action = ActionDeferred
			o.Reasons = append(o.Reasons, "erro ao recuperar: "+err.Error())
		}
		out = append(out, o)
	}
	return out, nil
}

// FieldError é um erro de entrada.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }
