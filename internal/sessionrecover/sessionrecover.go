// Package sessionrecover decide e executa a recuperação de uma execução cujo executor sumiu (AJ03-04):
// tenta retomar a sessão nativa quando o cliente suporta; se isso não for possível, inicia outra sessão a
// partir do checkpoint com a reconciliação do que já foi feito; nunca encerra um comando longo saudável só
// por falta de saída; retira a autoridade do escritor antigo (geração da reserva) antes de qualquer retomada;
// e preserva patch, arquivos não rastreados autorizados, commits e operações externas já aplicadas.
package sessionrecover

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/executor"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/worktrees"
)

// Strategy é o que a recuperação faz.
type Strategy string

// Estratégias.
const (
	Wait         Strategy = "wait"          // há comando longo saudável: não interromper
	NativeResume Strategy = "native_resume" // retoma a sessão original
	NewSession   Strategy = "new_session"   // outra sessão com checkpoint e reconciliação
)

// ErrSessionLost: a sessão nativa não existe mais (o cliente recusou a retomada).
var ErrSessionLost = errors.New("sessão nativa perdida")

// Observation é o que se sabe do executor.
type Observation struct {
	Caps           executor.Capabilities
	SessionID      string
	ProcessAlive   bool // o processo do executor (ou do comando) segue vivo
	CommandHealthy bool // há um comando em andamento que ainda consome recursos/progride
	LeaseValid     bool // o titular ainda detém a reserva com a geração dele
}

// Decide escolhe a estratégia. Lease perdido nunca vira espera: o escritor antigo é desautorizado.
func Decide(o Observation) Strategy {
	if o.LeaseValid && o.ProcessAlive && o.CommandHealthy {
		return Wait // silêncio de stdout NÃO é falha: comando saudável não é encerrado
	}
	if o.Caps.NativeResume && o.SessionID != "" {
		return NativeResume
	}
	return NewSession
}

// Resumer retoma ou inicia sessões no executor.
type Resumer interface {
	Resume(ctx context.Context, sessionID, brief string) error
	Start(ctx context.Context, brief string) (sessionID string, err error)
}

// Preserved é o que a recuperação garante não perder.
type Preserved struct {
	Checkpoint     string   `json:"checkpoint"`      // ref com patch e arquivos não rastreados autorizados
	Commits        []string `json:"commits"`         // commits da tentativa
	OutOfScope     []string `json:"out_of_scope"`    // não rastreados fora do escopo: sinalizados, não incluídos como autorizados
	AppliedEffects []string `json:"applied_effects"` // operações externas confirmadas: não repetir
	Uncertain      []string `json:"uncertain"`       // operações externas incertas: verificar antes de repetir
}

// Request pede a recuperação de uma tentativa.
type Request struct {
	ProjectID, Slot string // reserva da vaga (FRONTEND/BACKEND)
	OldHolder       string
	NewHolder       string
	AttemptID       string
	EffectPrefix    string // prefixo das chaves de negócio dos efeitos externos da execução
	Obs             Observation
}

// Result é o desfecho.
type Result struct {
	Strategy    Strategy             `json:"strategy"`
	Session     string               `json:"session"`
	Reservation sentinel.Reservation `json:"reservation"`
	Preserved   Preserved            `json:"preserved"`
	Brief       string               `json:"brief"`
}

// Recoverer recupera execuções.
type Recoverer struct {
	DB       interface{ persistence.DBTX }
	Wt       *worktrees.Manager
	Reserver *sentinel.Reserver
	Now      func() time.Time
}

func (r *Recoverer) preserve(ctx context.Context, req Request) (Preserved, error) {
	p := Preserved{Commits: []string{}, OutOfScope: []string{}, AppliedEffects: []string{}, Uncertain: []string{}}
	a, err := r.Wt.Get(ctx, req.AttemptID)
	if err != nil {
		return p, err
	}
	cp, err := r.Wt.Checkpoint(ctx, req.AttemptID)
	if err != nil {
		return p, fmt.Errorf("preservar antes de retomar: %w", err)
	}
	p.Checkpoint = cp.Ref
	if bad, err := r.Wt.Violations(ctx, req.AttemptID); err == nil {
		p.OutOfScope = append(p.OutOfScope, bad...)
	}
	if cs, err := r.Wt.Commits(ctx, a); err == nil {
		p.Commits = append(p.Commits, cs...)
	}
	if req.EffectPrefix != "" {
		rows, err := r.DB.Query(ctx, `SELECT business_key, state FROM outbox WHERE project_id=$1 AND business_key LIKE $2 ORDER BY created_at, business_key`, req.ProjectID, req.EffectPrefix+"%")
		if err != nil {
			return p, persistence.MapError(err)
		}
		defer rows.Close()
		for rows.Next() {
			var k, st string
			if err := rows.Scan(&k, &st); err != nil {
				return p, err
			}
			switch st {
			case "confirmed", "done":
				p.AppliedEffects = append(p.AppliedEffects, k)
			default:
				p.Uncertain = append(p.Uncertain, k+" ("+st+")")
			}
		}
	}
	return p, nil
}

func brief(p Preserved, reason string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Retomada (%s). Estado preservado no checkpoint %s.\n", reason, p.Checkpoint)
	fmt.Fprintf(&b, "Commits já feitos: %d. Operações externas já aplicadas (NÃO repetir): %d. Incertas (verificar antes): %d.\n", len(p.Commits), len(p.AppliedEffects), len(p.Uncertain))
	for _, e := range p.AppliedEffects {
		b.WriteString("- aplicada: " + e + "\n")
	}
	for _, e := range p.Uncertain {
		b.WriteString("- incerta: " + e + "\n")
	}
	if len(p.OutOfScope) > 0 {
		b.WriteString("Arquivos fora do escopo autorizado foram sinalizados e não entram como trabalho aprovado: " + strings.Join(p.OutOfScope, ", ") + "\n")
	}
	return b.String()
}

// Recover aplica a estratégia. Com `Wait` nada é tocado. Nos demais casos o escritor antigo perde a
// autoridade (nova geração da reserva) ANTES de a retomada começar e o trabalho é preservado primeiro.
func (r *Recoverer) Recover(ctx context.Context, rs Resumer, req Request) (Result, error) {
	var res Result
	res.Strategy = Decide(req.Obs)
	if res.Strategy == Wait {
		return res, nil
	}
	if err := r.Wt.SetWriter(ctx, req.AttemptID, false); err != nil {
		return res, err
	}
	res.Reservation, _ = r.Reserver.Takeover(ctx, req.ProjectID, req.Slot, req.NewHolder)
	p, err := r.preserve(ctx, req)
	if err != nil {
		return res, err
	}
	res.Preserved = p
	if res.Strategy == NativeResume {
		res.Brief = brief(p, "sessão nativa")
		if err := rs.Resume(ctx, req.Obs.SessionID, res.Brief); err == nil {
			res.Session = req.Obs.SessionID
			_ = r.Wt.SetWriter(ctx, req.AttemptID, true)
			return res, nil
		} else if !errors.Is(err, ErrSessionLost) {
			return res, err
		}
		res.Strategy = NewSession // a sessão original não existe mais: fallback
	}
	res.Brief = brief(p, "nova sessão a partir do checkpoint")
	id, err := rs.Start(ctx, res.Brief)
	if err != nil {
		return res, err
	}
	res.Session = id
	_ = r.Wt.SetWriter(ctx, req.AttemptID, true)
	return res, nil
}
