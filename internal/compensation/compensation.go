package compensation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/authorizations"
	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
)

// Riscos.
const (
	RiskLow          = "low"
	RiskSensitive    = "sensitive"
	RiskIrreversible = "irreversible"
)

// Erros.
var (
	ErrNotConfirmed = errors.New("só se compensa um efeito confirmado")
	ErrIrreversible = errors.New("efeito irreversível: preservado, nada foi desfeito")
	ErrNeedsAuth    = errors.New("compensação sensível sem autorização válida")
	ErrFailed       = errors.New("a compensação falhou: o trabalho foi conservado e novas operações agravantes ficam bloqueadas")
	ErrBlocked      = errors.New("operação agravante bloqueada por compensação com falha")
)

// Spec classifica a compensação de uma operação.
type Spec struct {
	Risk    string
	Inverse string
}

// DefaultSpecs: operações conhecidas. O que não está aqui é irreversível (padrão seguro).
var DefaultSpecs = map[string]Spec{
	"create_subtask": {RiskLow, "delete_subtask"},
	"add_link":       {RiskLow, "remove_link"},
	"add_dependency": {RiskLow, "remove_dependency"},
	"push":           {RiskSensitive, "delete_remote_branch"},
	"create_pr":      {RiskSensitive, "close_pr"},
	"merge":          {RiskIrreversible, ""},
	"send_message":   {RiskIrreversible, ""},
	"comment":        {RiskIrreversible, ""},
}

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Compensator coordena as compensações.
type Compensator struct {
	DB     DB
	Ledger *effects.Ledger
	Auth   *authorizations.Store
	Specs  map[string]Spec
	// Adapter devolve o adaptador que desfaz o efeito: Apply desfaz, Verify prova
	// (Confirmed = o efeito original deixou de existir; Absent = ainda existe).
	Adapter func(orig persistence.OutboxEntry, s Spec) effects.Adapter
}

func (c *Compensator) spec(op string) Spec {
	specs := c.Specs
	if specs == nil {
		specs = DefaultSpecs
	}
	if s, ok := specs[op]; ok {
		return s
	}
	return Spec{Risk: RiskIrreversible}
}

// ActionFor é a ação que uma autorização precisa cobrir para a compensação sensível.
func ActionFor(orig persistence.OutboxEntry, s Spec) policy.Action {
	return policy.Action{Tool: "compensation", Operation: s.Inverse, Resource: orig.BusinessKey, Params: map[string]any{"outbox": orig.ID}}
}

// Request pede a compensação de um efeito.
type Request struct {
	ProjectID       string
	ExecutionID     string
	OutboxID        string
	AuthorizationID string // exigida para risco sensível
}

// Record é a compensação registrada.
type Record struct {
	ID       string `json:"id"`
	OutboxID string `json:"outbox_id"`
	Risk     string `json:"risk"`
	State    string `json:"state"`
	Reason   string `json:"reason"`
	Attempts int    `json:"attempts"`
}

func (c *Compensator) event(ctx context.Context, id, typ string, detail any) {
	b, _ := json.Marshal(detail)
	_, _ = c.DB.Exec(context.WithoutCancel(ctx), `INSERT INTO compensation_events (compensation_id, type, detail) VALUES ($1, $2, $3::jsonb)`, id, typ, string(b))
}

// upsert grava (ou atualiza) o registro da compensação do efeito.
func (c *Compensator) upsert(ctx context.Context, req Request, op, inverse, risk, state, reason, authID string) (Record, error) {
	var auth any
	if authID != "" {
		auth = authID
	}
	var r Record
	err := c.DB.QueryRow(ctx, `INSERT INTO compensations (project_id, execution_id, outbox_id, operation, inverse, risk, state, reason, authorization_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::uuid)
		ON CONFLICT (outbox_id) DO UPDATE SET state = EXCLUDED.state, reason = EXCLUDED.reason, inverse = EXCLUDED.inverse,
			authorization_id = COALESCE(EXCLUDED.authorization_id, compensations.authorization_id),
			attempts = compensations.attempts + 1, updated_at = now()
		RETURNING id, outbox_id, risk, state, reason, attempts`,
		req.ProjectID, req.ExecutionID, req.OutboxID, op, inverse, risk, state, trim(reason), auth).Scan(&r.ID, &r.OutboxID, &r.Risk, &r.State, &r.Reason, &r.Attempts)
	return r, persistence.MapError(err)
}

func trim(s string) string {
	if len(s) > 1000 {
		return s[:1000]
	}
	return s
}

// Compensate tenta compensar o efeito conforme o risco. É idempotente: um efeito
// já compensado (ou já recusado como irreversível) não é refeito.
func (c *Compensator) Compensate(ctx context.Context, req Request) (Record, error) {
	orig, err := persistence.GetOutbox(ctx, c.DB, req.ProjectID, req.OutboxID)
	if err != nil {
		return Record{}, err
	}
	var prev Record
	perr := c.DB.QueryRow(ctx, `SELECT id, outbox_id, risk, state, reason, attempts FROM compensations WHERE outbox_id = $1`, req.OutboxID).Scan(&prev.ID, &prev.OutboxID, &prev.Risk, &prev.State, &prev.Reason, &prev.Attempts)
	if perr == nil && (prev.State == "applied" || (prev.State == "refused" && prev.Risk == RiskIrreversible)) {
		if prev.State == "refused" {
			return prev, ErrIrreversible
		}
		return prev, nil
	}
	if orig.State != persistence.OutboxConfirmed {
		return Record{}, fmt.Errorf("%w (estado %s)", ErrNotConfirmed, orig.State)
	}
	op := orig.Destination
	spec := c.spec(op)
	switch spec.Risk {
	case RiskIrreversible:
		rec, err := c.upsert(ctx, req, op, "", RiskIrreversible, "refused", "efeito irreversível preservado: "+op, "")
		if err != nil {
			return Record{}, err
		}
		c.event(ctx, rec.ID, "refused", map[string]any{"reason": rec.Reason})
		return rec, ErrIrreversible
	case RiskSensitive:
		if req.AuthorizationID == "" || c.Auth == nil {
			return c.refuse(ctx, req, op, spec, "compensação sensível sem autorização", "")
		}
		if _, err := c.Auth.Consume(ctx, req.ProjectID, req.AuthorizationID, ActionFor(orig, spec)); err != nil {
			return c.refuse(ctx, req, op, spec, "autorização recusada: "+err.Error(), "")
		}
	}
	if c.Adapter == nil || c.Ledger == nil {
		return Record{}, errors.New("compensação sem adaptador ou ledger configurado")
	}
	res, runErr := c.Ledger.Run(ctx, effects.Spec{ProjectID: req.ProjectID, BusinessKey: orig.BusinessKey + ":compensate", Operation: "compensate:" + spec.Inverse,
		Request: map[string]any{"outbox": orig.ID, "operation": op, "inverse": spec.Inverse}}, c.Adapter(orig, spec))
	if runErr == nil {
		rec, err := c.upsert(ctx, req, op, spec.Inverse, spec.Risk, "applied", "compensação "+res.Outcome, req.AuthorizationID)
		if err != nil {
			return Record{}, err
		}
		c.event(ctx, rec.ID, "applied", map[string]any{"outcome": res.Outcome})
		_ = c.resolveBlocks(ctx, rec.ID, "", "compensação aplicada")
		return rec, nil
	}
	rec, err := c.upsert(ctx, req, op, spec.Inverse, spec.Risk, "failed", runErr.Error(), req.AuthorizationID)
	if err != nil {
		return Record{}, err
	}
	c.event(ctx, rec.ID, "failed", map[string]any{"error": runErr.Error()})
	if _, err := c.DB.Exec(ctx, `INSERT INTO compensation_blocks (project_id, execution_id, compensation_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, req.ProjectID, req.ExecutionID, rec.ID); err != nil {
		return rec, persistence.MapError(err)
	}
	c.event(ctx, rec.ID, "blocked", map[string]any{"execution": req.ExecutionID})
	return rec, fmt.Errorf("%w: %v", ErrFailed, runErr)
}

func (c *Compensator) refuse(ctx context.Context, req Request, op string, s Spec, reason, authID string) (Record, error) {
	rec, err := c.upsert(ctx, req, op, s.Inverse, s.Risk, "refused", reason, authID)
	if err != nil {
		return Record{}, err
	}
	c.event(ctx, rec.ID, "refused", map[string]any{"reason": reason})
	return rec, fmt.Errorf("%w: %s", ErrNeedsAuth, reason)
}

func (c *Compensator) resolveBlocks(ctx context.Context, compID, adminID, note string) error {
	var admin any
	if adminID != "" {
		admin = adminID
	}
	tag, err := c.DB.Exec(ctx, `UPDATE compensation_blocks SET resolved_at = now(), resolved_by = $2::uuid, note = $3 WHERE compensation_id = $1 AND resolved_at IS NULL`, compID, admin, note)
	if err == nil && tag.RowsAffected() > 0 {
		c.event(ctx, compID, "block_resolved", map[string]any{"note": note})
	}
	return err
}

// ResolveBlock encerra o bloqueio de uma compensação que falhou (decisão humana).
func (c *Compensator) ResolveBlock(ctx context.Context, compensationID, adminID, note string) error {
	if len(note) > 500 {
		return errors.New("observação longa demais (até 500)")
	}
	return persistence.WithTx(ctx, c.DB, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE compensation_blocks SET resolved_at = now(), resolved_by = $2::uuid, note = $3 WHERE compensation_id = $1 AND resolved_at IS NULL`,
			compensationID, nullIf(adminID), note)
		if err != nil {
			return persistence.MapError(err)
		}
		if tag.RowsAffected() == 0 {
			return persistence.ErrNotFound
		}
		_, err = tx.Exec(ctx, `INSERT INTO compensation_events (compensation_id, type, detail) VALUES ($1, 'block_resolved', $2::jsonb)`, compensationID, fmt.Sprintf(`{"note":%q}`, note))
		return persistence.MapError(err)
	})
}

func nullIf(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// readOnly diz se a operação só lê (nunca agrava).
func readOnly(op string) bool {
	op = strings.ToLower(op)
	for _, p := range []string{"read", "list", "get", "search", "status", "diff", "compensate", "resolve"} {
		if op == p || strings.HasPrefix(op, p+"_") || strings.HasPrefix(op, p+":") {
			return true
		}
	}
	return false
}

// CheckAggravating recusa uma nova operação que altera estado enquanto a execução
// tem bloqueio de compensação aberto. Leitura e compensação continuam permitidas.
func (c *Compensator) CheckAggravating(ctx context.Context, projectID, executionID, operation string) error {
	if readOnly(operation) {
		return nil
	}
	var n int
	if err := c.DB.QueryRow(ctx, `SELECT count(*) FROM compensation_blocks WHERE project_id = $1 AND execution_id = $2 AND resolved_at IS NULL`, projectID, executionID).Scan(&n); err != nil {
		return persistence.MapError(err)
	}
	if n > 0 {
		return fmt.Errorf("%w (%d bloqueio(s) aberto(s))", ErrBlocked, n)
	}
	return nil
}

// Event é uma linha da trilha.
type Event struct {
	Type   string          `json:"type"`
	Detail json.RawMessage `json:"detail"`
}

// Events devolve a trilha da compensação.
func (c *Compensator) Events(ctx context.Context, compensationID string) ([]Event, error) {
	rows, err := c.DB.Query(ctx, `SELECT type, detail FROM compensation_events WHERE compensation_id = $1 ORDER BY id`, compensationID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		var d []byte
		if err := rows.Scan(&e.Type, &d); err != nil {
			return nil, persistence.MapError(err)
		}
		e.Detail = d
		out = append(out, e)
	}
	return out, persistence.MapError(rows.Err())
}
