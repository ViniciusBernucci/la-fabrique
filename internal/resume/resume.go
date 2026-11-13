package resume

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// Estados do resultado.
const (
	StatusResumed = "resumed"
	StatusBlocked = "blocked"
)

// EffectPrefix é a convenção das chaves de negócio dos efeitos de um nó:
// "exec:<execução>:<nó>:". A retomada usa o prefixo para reconciliar os efeitos
// dos nós afetados antes de refazê-los.
func EffectPrefix(executionID, node string) string { return "exec:" + executionID + ":" + node + ":" }

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Resumer retoma execuções.
type Resumer struct {
	DB          DB
	Checkpoints *checkpoints.Store
	Redactor    *security.Redactor
	Exec        workflows.Executor
	Now         func() time.Time
}

func (r *Resumer) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}

// Request pede a retomada de um nó.
type Request struct {
	ProjectID   string
	ExecutionID string
	Node        string
	Current     Inputs  // insumos de agora
	Options     Options // como julgar refs
}

// Outcome é o resultado auditável.
type Outcome struct {
	Status       string              `json:"status"`
	Reasons      []string            `json:"reasons"`
	State        *workflows.RunState `json:"state,omitempty"`
	Selection    *Selection          `json:"selection,omitempty"`
	Rerun        *RerunResult        `json:"rerun,omitempty"`
	Diffs        []Diff              `json:"diffs"`
	Incorporated []Diff              `json:"incorporated"`
	Decision     *decisions.Request  `json:"decision,omitempty"`
	CheckpointID string              `json:"checkpoint_id"`
	VersionID    string              `json:"version_id"`
	KeptEffects  []string            `json:"kept_effects"`
}

// RecordBaseline grava os insumos como referência da execução (substitui o anterior).
func (r *Resumer) RecordBaseline(ctx context.Context, projectID, executionID, checkpointID string, in Inputs) error {
	b, _ := json.Marshal(in)
	var cp any
	if checkpointID != "" {
		cp = checkpointID
	}
	_, err := r.DB.Exec(ctx, `
		INSERT INTO resume_baselines (execution_id, project_id, checkpoint_id, inputs, inputs_hash, recorded_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
		ON CONFLICT (execution_id) DO UPDATE SET checkpoint_id = $3, inputs = $4::jsonb, inputs_hash = $5, recorded_at = $6`,
		executionID, projectID, cp, string(b), hashJSON(in), r.now())
	return persistence.MapError(err)
}

func (r *Resumer) baseline(ctx context.Context, executionID string) (*Inputs, error) {
	var raw []byte
	err := r.DB.QueryRow(ctx, `SELECT inputs FROM resume_baselines WHERE execution_id = $1`, executionID).Scan(&raw)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, persistence.MapError(err)
	}
	var in Inputs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("baseline ilegível: %w", err)
	}
	return &in, nil
}

// park coloca a execução em espera (aguardando decisão humana), pré-condição do
// pedido de decisão. Execução cancelada não é reaberta.
func (r *Resumer) park(ctx context.Context, projectID, executionID string) error {
	var state string
	var version int
	if err := r.DB.QueryRow(ctx, `SELECT state, version FROM executions WHERE project_id = $1 AND id = $2`, projectID, executionID).Scan(&state, &version); err != nil {
		return persistence.MapError(err)
	}
	switch state {
	case "waiting":
		return nil
	case "cancelled":
		return fmt.Errorf("%w: a execução foi cancelada", persistence.ErrInvalidInput)
	}
	_, err := persistence.UpdateExecutionState(ctx, r.DB, projectID, executionID, version, "waiting")
	return err
}

func (r *Resumer) event(ctx context.Context, req Request, action string, detail any) {
	b, _ := json.Marshal(detail)
	_, _ = r.DB.Exec(ctx, `INSERT INTO resume_events (project_id, execution_id, node_key, action, detail) VALUES ($1, $2, $3, $4, $5::jsonb)`,
		req.ProjectID, req.ExecutionID, req.Node, action, string(b))
}

// Resume valida, reconcilia e refaz só o conjunto afetado do nó pedido.
func (r *Resumer) Resume(ctx context.Context, req Request) (Outcome, error) {
	out := Outcome{Reasons: []string{}, Diffs: []Diff{}, Incorporated: []Diff{}, KeptEffects: []string{}}
	// 1. Versão publicada à qual a execução está presa (o workflow original segue fixado).
	ver, err := releases.ExecutionVersion(ctx, r.DB, req.ExecutionID)
	if err != nil {
		return out, err
	}
	out.VersionID = ver.ID
	g := ver.FlatGraph

	// 2. Último checkpoint íntegro da execução.
	cp, err := r.Checkpoints.Latest(ctx, req.ProjectID, req.ExecutionID)
	if err != nil {
		return out, err
	}
	out.CheckpointID = cp.ID
	var st workflows.RunState
	if err := json.Unmarshal(cp.State, &st); err != nil {
		return out, fmt.Errorf("%w: estado dos nós ilegível", checkpoints.ErrCorrupt)
	}

	// 3. Seleção do nó e do conjunto afetado.
	sel, err := Select(g, st, req.Node)
	if err != nil {
		return out, err
	}
	out.Selection = &sel

	block := func(reason string, dec *decisions.Request) (Outcome, error) {
		out.Status = StatusBlocked
		out.Reasons = append(out.Reasons, reason)
		out.Decision = dec
		r.event(ctx, req, "blocked", map[string]any{"reasons": out.Reasons, "diffs": out.Diffs})
		return out, nil
	}

	// 4. Permissões e configuração: a versão publicada ainda precisa ser executável.
	if problems := releases.Revalidate(ctx, r.DB, ver); len(problems) > 0 {
		for _, p := range problems {
			out.Reasons = append(out.Reasons, p.Message)
		}
		return block("a versão publicada deixou de ser executável (permissões, contas ou modelos)", nil)
	}

	// 5. Efeitos externos dos nós afetados: incerto bloqueia; confirmado é reaproveitado.
	for _, n := range append([]string{req.Node}, sel.Affected...) {
		rows, err := r.DB.Query(ctx, `SELECT business_key, state FROM outbox WHERE project_id = $1 AND starts_with(business_key, $2) ORDER BY business_key`,
			req.ProjectID, EffectPrefix(req.ExecutionID, n))
		if err != nil {
			return out, persistence.MapError(err)
		}
		var uncertain []string
		for rows.Next() {
			var k, state string
			if err := rows.Scan(&k, &state); err != nil {
				rows.Close()
				return out, persistence.MapError(err)
			}
			if state == "confirmed" {
				out.KeptEffects = append(out.KeptEffects, k)
			} else if state != "failed" {
				uncertain = append(uncertain, k)
			}
		}
		rows.Close()
		if len(uncertain) > 0 {
			if err := r.park(ctx, req.ProjectID, req.ExecutionID); err != nil {
				return out, err
			}
			dec, _, err := decisions.Create(ctx, r.DB, r.Redactor, decisions.NewRequest{
				ProjectID: req.ProjectID, ExecutionID: req.ExecutionID, NodeID: n, Kind: decisions.KindUncertainResult,
				Title:    "Efeito externo incerto impede a retomada",
				Problem:  fmt.Sprintf("O nó %q tem efeito(s) sem confirmação (%s). Retomar poderia repeti-los.", n, strings.Join(uncertain, ", ")),
				Impact:   "A retomada fica bloqueada até o efeito ser reconciliado.",
				Evidence: []decisions.Evidence{{Label: "Chaves de negócio", Text: strings.Join(uncertain, "\n")}},
				Options: []decisions.Option{
					{ID: "reconcile", Label: "Reconciliar primeiro", Description: "Confirmar o estado no destino antes de retomar."},
					{ID: "abort", Label: "Não retomar", Description: "Manter a execução parada."},
				},
			})
			if err != nil {
				return out, err
			}
			return block(fmt.Sprintf("há efeito externo incerto no nó %q", n), &dec)
		}
	}

	// 6. Insumos: baseline × atual.
	base, err := r.baseline(ctx, req.ExecutionID)
	if err != nil {
		return out, err
	}
	if base != nil {
		out.Diffs = Reconcile(*base, req.Current, req.Options)
		if HasConflict(out.Diffs) {
			ev := make([]decisions.Evidence, 0, len(out.Diffs))
			for _, d := range out.Diffs {
				ev = append(ev, decisions.Evidence{Label: d.Kind + "." + d.Field, Text: fmt.Sprintf("antes: %s\nagora: %s\n%s (%s)", d.Before, d.After, d.Reason, d.Class)})
			}
			if err := r.park(ctx, req.ProjectID, req.ExecutionID); err != nil {
				return out, err
			}
			dec, _, err := decisions.Create(ctx, r.DB, r.Redactor, decisions.NewRequest{
				ProjectID: req.ProjectID, ExecutionID: req.ExecutionID, NodeID: req.Node, Kind: decisions.KindConflict,
				Title:    "Os insumos mudaram de forma conflitante",
				Problem:  "Desde o último ponto reconciliado, refs, requisito ou configuração mudaram de um jeito que pode invalidar o trabalho.",
				Impact:   "Nada foi alterado: o estado, o checkpoint e as edições humanas foram preservados.",
				Evidence: ev,
				Options: []decisions.Option{
					{ID: "keep_work", Label: "Manter o trabalho atual", Description: "Aceitar as mudanças e seguir com o plano atual."},
					{ID: "replan", Label: "Replanejar", Description: "Refazer o plano com o novo requisito antes de retomar."},
				},
			})
			if err != nil {
				return out, err
			}
			return block("os insumos mudaram de forma conflitante", &dec)
		}
		for _, d := range out.Diffs {
			out.Incorporated = append(out.Incorporated, d) // todas compatíveis
		}
	}

	// 7. Reexecução só do conjunto afetado.
	next, rr, err := Rerun(ctx, r.Exec, g, st, req.Node)
	if err != nil {
		return out, err
	}
	if base == nil || len(out.Incorporated) > 0 {
		if err := r.RecordBaseline(ctx, req.ProjectID, req.ExecutionID, cp.ID, req.Current); err != nil {
			return out, err
		}
		if len(out.Incorporated) > 0 {
			r.event(ctx, req, "incorporated", out.Incorporated)
		}
	}
	out.Status, out.State, out.Rerun = StatusResumed, &next, &rr
	r.event(ctx, req, "resumed", map[string]any{"selection": sel, "rerun": rr, "kept_effects": out.KeptEffects, "version": ver.ID, "checkpoint": cp.ID})
	return out, nil
}
