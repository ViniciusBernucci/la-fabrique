package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/contract"
	"github.com/ViniciusBernucci/le-fabrique/internal/executor"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/providers"
)

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Dispatcher monta e registra despachos.
type Dispatcher struct {
	DB        DB
	Contracts *contract.Store
	Now       func() time.Time
}

// Request descreve o despacho de uma execução.
type Request struct {
	ProjectID    string
	Scope        string // escopo do contrato (contract.ScopeKey)
	ExecutionID  string
	Kind         executor.Kind
	Instructions string
	Skill        *executor.Skill
	Mode         providers.Mode
	Model        string
	Dir          string
	AllowedTools []string
}

// Dispatch é um despacho registrado.
type Dispatch struct {
	ID       string            `json:"id"`
	Task     executor.Task     `json:"task"`
	Snapshot contract.Snapshot `json:"-"`
	Resumed  bool              `json:"resumed"` // a execução já tinha despacho: mesma versão do contrato
}

// ErrNoContract indica escopo sem contrato publicado: não há despacho.
var ErrNoContract = errors.New("não há contrato publicado para o escopo: nada é despachado")

func (d *Dispatcher) task(r Request, sn contract.Snapshot) executor.Task {
	return executor.Task{Contract: executor.RefOf(sn), Instructions: r.Instructions, Skill: r.Skill, Mode: r.Mode, Model: r.Model, Dir: r.Dir,
		AllowedTools: r.AllowedTools, Limits: sn.Contract.Deterministic.Limits}
}

// Begin registra o despacho. A primeira vez fixa a versão mais recente (integridade conferida); a
// mesma execução de novo (retomada, reentrega) recebe a MESMA versão fixada.
func (d *Dispatcher) Begin(ctx context.Context, r Request) (Dispatch, error) {
	if r.ProjectID == "" || r.ExecutionID == "" || r.Scope == "" || (r.Kind != executor.CLI && r.Kind != executor.API) {
		return Dispatch{}, fmt.Errorf("%w: projeto, escopo, execução e tipo são obrigatórios", persistence.ErrInvalidInput)
	}
	var out Dispatch
	err := persistence.WithTx(ctx, d.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, r.ProjectID+"|"+r.ExecutionID); err != nil {
			return err
		}
		var id string
		var version int
		err := tx.QueryRow(ctx, `SELECT id::text, contract_version FROM dispatches WHERE project_id=$1 AND execution_id=$2`, r.ProjectID, r.ExecutionID).Scan(&id, &version)
		switch {
		case err == nil:
			sn, gerr := d.Contracts.Get(ctx, r.ProjectID, r.Scope, version)
			if gerr != nil {
				return gerr
			}
			out = Dispatch{ID: id, Task: d.task(r, sn), Snapshot: sn, Resumed: true}
			return nil
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		sn, gerr := d.Contracts.Get(ctx, r.ProjectID, r.Scope, 0)
		if gerr != nil {
			if errors.Is(gerr, persistence.ErrNotFound) {
				return ErrNoContract
			}
			return gerr
		}
		if err := tx.QueryRow(ctx, `INSERT INTO dispatches (project_id, execution_id, scope_key, contract_id, contract_version, contract_hash, executor_kind)
			VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id::text`, r.ProjectID, r.ExecutionID, r.Scope, sn.ID, sn.Version, sn.Hash, string(r.Kind)).Scan(&id); err != nil {
			return err
		}
		out = Dispatch{ID: id, Task: d.task(r, sn), Snapshot: sn}
		return nil
	})
	if err != nil {
		return Dispatch{}, persistence.MapError(err)
	}
	return out, nil
}

// Complete grava o resultado (uma vez): estado final e resumo.
func (d *Dispatcher) Complete(ctx context.Context, dispatchID string, o executor.Outcome) error {
	state := "done"
	switch {
	case o.Cancelled:
		state = "cancelled"
	case o.Failed:
		state = "failed"
	}
	res, _ := json.Marshal(map[string]any{"final": clip(o.Final), "failure": clip(o.Failure), "input_tokens": o.InputTokens, "output_tokens": o.OutputTokens,
		"events": o.Events, "session": o.Session.ID})
	now := time.Now().UTC()
	if d.Now != nil {
		now = d.Now().UTC()
	}
	tag, err := d.DB.Exec(ctx, `UPDATE dispatches SET state=$2, result=$3, finished_at=$4 WHERE id=$1 AND state='running'`, dispatchID, state, res, now)
	if err != nil {
		return persistence.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return nil // já finalizado: o primeiro resultado vale
	}
	return nil
}

func clip(s string) string {
	if len(s) > 500 {
		return s[:500]
	}
	return s
}

// Run despacha e executa: Begin → adaptador → Complete. resume pede a sessão nativa (quando houver).
func (d *Dispatcher) Run(ctx context.Context, ad executor.Adapter, r Request, resume *executor.Session, emit func(executor.Event)) (Dispatch, executor.Outcome, error) {
	r.Kind = ad.Capabilities().Kind
	dp, err := d.Begin(ctx, r)
	if err != nil {
		return Dispatch{}, executor.Outcome{}, err
	}
	out, err := ad.Run(ctx, dp.Task, resume, emit)
	if err != nil {
		return dp, out, err
	}
	return dp, out, d.Complete(context.WithoutCancel(ctx), dp.ID, out)
}

// Evidence liga versão do contrato, execução e resultado.
type Evidence struct {
	DispatchID      string          `json:"dispatch_id"`
	ExecutionID     string          `json:"execution_id"`
	Scope           string          `json:"scope"`
	ContractVersion int             `json:"contract_version"`
	ContractHash    string          `json:"contract_hash"`
	Executor        string          `json:"executor"`
	State           string          `json:"state"`
	Result          json.RawMessage `json:"result"`
}

// EvidenceOf devolve a evidência da execução.
func (d *Dispatcher) EvidenceOf(ctx context.Context, projectID, executionID string) (Evidence, error) {
	var e Evidence
	err := d.DB.QueryRow(ctx, `SELECT id::text, execution_id, scope_key, contract_version, contract_hash, executor_kind, state, result
		FROM dispatches WHERE project_id=$1 AND execution_id=$2`, projectID, executionID).
		Scan(&e.DispatchID, &e.ExecutionID, &e.Scope, &e.ContractVersion, &e.ContractHash, &e.Executor, &e.State, &e.Result)
	return e, persistence.MapError(err)
}

// Stale informa se o despacho usa uma versão mais antiga que a vigente (insumos revisados). Quem
// decide é o ponto seguro: a execução em andamento segue com a versão que recebeu; o despacho seguinte
// (outra execução) usa a nova.
func (d *Dispatcher) Stale(ctx context.Context, projectID, executionID string) (stale bool, latest int, err error) {
	var scope string
	var pinned int
	if err := d.DB.QueryRow(ctx, `SELECT scope_key, contract_version FROM dispatches WHERE project_id=$1 AND execution_id=$2`, projectID, executionID).Scan(&scope, &pinned); err != nil {
		return false, 0, persistence.MapError(err)
	}
	if err := d.DB.QueryRow(ctx, `SELECT max(version) FROM execution_contracts WHERE project_id=$1 AND scope_key=$2`, projectID, scope).Scan(&latest); err != nil {
		return false, 0, err
	}
	return latest > pinned, latest, nil
}
