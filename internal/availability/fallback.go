package availability

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/budget"
	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Ações da decisão.
const (
	ActionUse    = "use"    // seguir com o passo escolhido (sem troca)
	ActionSwitch = "switch" // trocar para a alternativa
	ActionWait   = "wait"   // sem alternativa: espera sem ocupar a vaga (há checkpoint seguro)
	ActionHold   = "hold"   // sem alternativa e sem checkpoint seguro: mantém a vaga
)

// Erros.
var (
	ErrNotSafe = errors.New("troca de conta exige checkpoint íntegro e sandbox em quiescência")
	ErrNoChain = errors.New("a cadeia explícita está vazia")
)

// Step é uma conta+modelo da cadeia (a posição 0 é o principal).
type Step struct {
	ConnectionID string
	Model        string
}

// Budgeter reserva custo (budget.Store implementa).
type Budgeter interface {
	Reserve(ctx context.Context, req budget.ReserveRequest) (budget.Reservation, error)
	Release(ctx context.Context, reservationID string) error
}

// Resolver escolhe a alternativa.
type Resolver struct {
	DB          DB
	Health      *Tracker
	Checkpoints *checkpoints.Store
	Budget      Budgeter // nil = sem checagem de orçamento
	Now         func() time.Time
}

func (r *Resolver) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}

// Request descreve a escolha.
type Request struct {
	ProjectID   string
	ExecutionID string
	NodeKey     string
	AgentID     string
	Provider    string
	NeedsTools  bool
	Chain       []Step // principal primeiro; só estas contas podem ser usadas
	// Estimativa de custo da próxima chamada (micro-USD), para a checagem de orçamento.
	EstimateMicros int64

	// Falha que originou o pedido (nil = escolha inicial, sem falha).
	Current *Step
	Failure *Failure

	// Prova de segurança para trocar: checkpoint íntegro da execução e sandbox quieto.
	CheckpointID string
	Quiescent    bool
}

// Skipped explica por que um passo da cadeia foi descartado.
type Skipped struct {
	Step   Step   `json:"step"`
	Reason string `json:"reason"`
}

// Decision é o resultado auditável.
type Decision struct {
	Action  string     `json:"action"`
	Step    *Step      `json:"step,omitempty"`
	RetryAt *time.Time `json:"retry_at,omitempty"`
	Reason  string     `json:"reason"`
	Skipped []Skipped  `json:"skipped"`
}

// Choose escolhe a primeira alternativa utilizável da cadeia, em ordem, sem
// rodízio. Sem falha de origem, nunca troca nada em andamento: só devolve o
// passo a usar.
func (r *Resolver) Choose(ctx context.Context, req Request) (Decision, error) {
	if len(req.Chain) == 0 {
		return Decision{}, ErrNoChain
	}
	if req.Failure != nil && req.Current == nil {
		return Decision{}, &FieldError{"current", "informe o passo que falhou"}
	}
	if req.Failure != nil {
		if _, err := r.Health.Record(ctx, req.Current.ConnectionID, *req.Failure); err != nil {
			return Decision{}, err
		}
	}
	d := Decision{Skipped: []Skipped{}}
	var earliest *time.Time
	sticky := true // todos os passos descartados dependem de intervenção?
	for _, st := range req.Chain {
		h, err := r.Health.Get(ctx, st.ConnectionID)
		if err != nil {
			return Decision{}, err
		}
		if !h.Usable() {
			d.Skipped = append(d.Skipped, Skipped{st, fmt.Sprintf("%s: %s", h.State, strings.TrimSpace(h.Reason))})
			if h.Until != nil {
				sticky = false
				if earliest == nil || h.Until.Before(*earliest) {
					u := *h.Until
					earliest = &u
				}
			}
			continue
		}
		if why := r.compatible(ctx, req, st); why != "" {
			d.Skipped = append(d.Skipped, Skipped{st, why})
			continue
		}
		sticky = false
		if why, err := r.affordable(ctx, req, st); err != nil {
			return Decision{}, err
		} else if why != "" {
			d.Skipped = append(d.Skipped, Skipped{st, why})
			continue
		}
		s := st
		d.Step = &s
		break
	}

	// Escolha inicial (sem falha): só devolve o passo a usar.
	if req.Failure == nil {
		if d.Step != nil {
			d.Action, d.Reason = ActionUse, "primeira alternativa utilizável da cadeia"
			return d, r.clearWaiting(ctx, req.ExecutionID)
		}
		return r.noAlternative(ctx, req, d, earliest, sticky)
	}
	if d.Step != nil {
		d.Action = ActionUse // mesma conta (falha isolada): repetir não troca nada
		if *d.Step != *req.Current {
			if err := r.requireSafe(ctx, req); err != nil {
				return Decision{}, err
			}
			d.Action = ActionSwitch
		}
		d.Reason = fmt.Sprintf("%s na conta %s; seguiu a cadeia autorizada", req.Failure.Class, req.Current.ConnectionID)
		if err := r.audit(ctx, req, d); err != nil {
			return Decision{}, err
		}
		return d, r.clearWaiting(ctx, req.ExecutionID)
	}
	return r.noAlternative(ctx, req, d, earliest, sticky)
}

// compatible confere provedor, conta, modelo e ferramentas.
func (r *Resolver) compatible(ctx context.Context, req Request, st Step) string {
	v, err := models.Check(ctx, r.DB, models.Requirement{Provider: req.Provider, ConnectionID: st.ConnectionID, Model: st.Model, NeedsTools: req.NeedsTools})
	if err != nil {
		return "não foi possível verificar a compatibilidade: " + err.Error()
	}
	if !v.OK {
		return "incompatível: " + strings.Join(v.Problems, " ")
	}
	return ""
}

// affordable testa o orçamento com uma reserva liberada em seguida.
func (r *Resolver) affordable(ctx context.Context, req Request, st Step) (string, error) {
	if r.Budget == nil || req.EstimateMicros <= 0 {
		return "", nil
	}
	res, err := r.Budget.Reserve(ctx, budget.ReserveRequest{AgentID: req.AgentID, ConnectionID: st.ConnectionID, AmountMicros: req.EstimateMicros, TTL: time.Minute})
	if err != nil {
		var ex *budget.ExceededError
		if errors.As(err, &ex) || errors.Is(err, budget.ErrNoBudget) {
			return "fora do orçamento: " + err.Error(), nil
		}
		return "", err
	}
	return "", r.Budget.Release(ctx, res.ID)
}

// requireSafe exige checkpoint íntegro da execução e quiescência antes de trocar.
func (r *Resolver) requireSafe(ctx context.Context, req Request) error {
	if !req.Quiescent || req.CheckpointID == "" {
		return fmt.Errorf("%w: %s", ErrNotSafe, safeMissing(req))
	}
	cp, err := r.Checkpoints.Get(ctx, req.ProjectID, req.CheckpointID) // reconfere hashes
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNotSafe, err)
	}
	if cp.ExecutionID != req.ExecutionID {
		return fmt.Errorf("%w: checkpoint de outra execução", ErrNotSafe)
	}
	return nil
}

func safeMissing(req Request) string {
	switch {
	case !req.Quiescent && req.CheckpointID == "":
		return "sem quiescência nem checkpoint"
	case !req.Quiescent:
		return "sandbox não está quieto"
	}
	return "sem checkpoint"
}

// noAlternative preserva a tarefa e agenda nova tentativa com backoff.
func (r *Resolver) noAlternative(ctx context.Context, req Request, d Decision, earliest *time.Time, sticky bool) (Decision, error) {
	now := r.now()
	safe := r.requireSafe(ctx, req) == nil
	var attempts int
	if err := r.DB.QueryRow(ctx, `SELECT attempts FROM waiting_tasks WHERE execution_id = $1`, req.ExecutionID).Scan(&attempts); err != nil &&
		!errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return Decision{}, persistence.MapError(err)
	}
	attempts++
	var retry *time.Time
	switch {
	case earliest != nil:
		retry = earliest
	case !sticky:
		u := now.Add(Backoff(attempts))
		retry = &u
	} // sticky (auth/billing em todos): sem horário; só intervenção libera.
	d.RetryAt, d.Action = retry, ActionHold
	if safe {
		d.Action = ActionWait
	}
	d.Reason = "nenhuma alternativa autorizada, compatível e dentro do orçamento está utilizável"
	if sticky {
		d.Reason += "; depende de intervenção (autenticação ou financeiro)"
	}
	cpID := any(nil)
	if safe {
		cpID = req.CheckpointID
	}
	err := persistence.WithTx(ctx, r.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO waiting_tasks (execution_id, project_id, node_key, reason, retry_at, attempts, slot_released, checkpoint_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
			ON CONFLICT (execution_id) DO UPDATE SET node_key = $3, reason = $4, retry_at = $5, attempts = $6,
				slot_released = $7, checkpoint_id = $8, updated_at = $9`,
			req.ExecutionID, req.ProjectID, req.NodeKey, d.Reason, retry, attempts, safe, cpID, now); err != nil {
			return persistence.MapError(err)
		}
		return persistence.MapError(r.insertEvent(ctx, tx, req, d))
	})
	return d, err
}

func (r *Resolver) audit(ctx context.Context, req Request, d Decision) error {
	return persistence.MapError(r.insertEvent(ctx, r.DB, req, d))
}

func (r *Resolver) insertEvent(ctx context.Context, q persistence.DBTX, req Request, d Decision) error {
	var fromConn, toConn, cp any
	var fromModel, toModel string
	if req.Current != nil {
		fromConn, fromModel = req.Current.ConnectionID, req.Current.Model
	}
	if d.Step != nil {
		toConn, toModel = d.Step.ConnectionID, d.Step.Model
	}
	if req.CheckpointID != "" {
		cp = req.CheckpointID
	}
	_, err := q.Exec(ctx, `INSERT INTO fallback_events (project_id, execution_id, node_key, action, from_conn, from_model, to_conn, to_model, reason, checkpoint_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		req.ProjectID, req.ExecutionID, req.NodeKey, d.Action, fromConn, fromModel, toConn, toModel, d.Reason, cp)
	return err
}

func (r *Resolver) clearWaiting(ctx context.Context, executionID string) error {
	_, err := r.DB.Exec(ctx, `DELETE FROM waiting_tasks WHERE execution_id = $1`, executionID)
	return persistence.MapError(err)
}

// Waiting é uma tarefa esperando disponibilidade.
type Waiting struct {
	ExecutionID  string     `json:"execution_id"`
	ProjectID    string     `json:"project_id"`
	NodeKey      string     `json:"node_key"`
	Reason       string     `json:"reason"`
	RetryAt      *time.Time `json:"retry_at"`
	Attempts     int        `json:"attempts"`
	SlotReleased bool       `json:"slot_released"`
}

// Due lista as tarefas em espera cujo horário de nova tentativa chegou.
func (r *Resolver) Due(ctx context.Context) ([]Waiting, error) {
	rows, err := r.DB.Query(ctx, `SELECT execution_id, project_id, node_key, reason, retry_at, attempts, slot_released FROM waiting_tasks
		WHERE retry_at IS NOT NULL AND retry_at <= $1 ORDER BY retry_at, execution_id`, r.now())
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Waiting{}
	for rows.Next() {
		var w Waiting
		if err := rows.Scan(&w.ExecutionID, &w.ProjectID, &w.NodeKey, &w.Reason, &w.RetryAt, &w.Attempts, &w.SlotReleased); err != nil {
			return nil, persistence.MapError(err)
		}
		out = append(out, w)
	}
	return out, persistence.MapError(rows.Err())
}

// Event é uma linha da trilha de fallback.
type Event struct {
	Action    string    `json:"action"`
	FromConn  string    `json:"from_connection"`
	FromModel string    `json:"from_model"`
	ToConn    string    `json:"to_connection"`
	ToModel   string    `json:"to_model"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// Events devolve a trilha de uma execução.
func (r *Resolver) Events(ctx context.Context, projectID, executionID string) ([]Event, error) {
	rows, err := r.DB.Query(ctx, `SELECT action, COALESCE(from_conn::text,''), from_model, COALESCE(to_conn::text,''), to_model, reason, created_at
		FROM fallback_events WHERE project_id = $1 AND execution_id = $2 ORDER BY id`, projectID, executionID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.Action, &e.FromConn, &e.FromModel, &e.ToConn, &e.ToModel, &e.Reason, &e.CreatedAt); err != nil {
			return nil, persistence.MapError(err)
		}
		out = append(out, e)
	}
	return out, persistence.MapError(rows.Err())
}
