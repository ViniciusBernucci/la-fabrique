package sentinel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Atores. Só a Sentinela efetiva transição; os demais entregam observações.
const (
	ActorSentinel = "sentinel"
	ActorAgent    = "agent"
	ActorAdapter  = "adapter"
	ActorHuman    = "human"
)

// Decisões gravadas na auditoria.
const (
	DecisionApplied  = "applied"
	DecisionRejected = "rejected"
	DecisionRefused  = "refused"
	DecisionStale    = "stale"
)

// DestProjection é o destino da outbox para projetar status na origem (Jira/ClickUp).
const DestProjection = "status-projection"

// Erros.
var (
	ErrNotAuthority = errors.New("somente a Sentinela efetiva transições de status")
	ErrStale        = errors.New("observação ou versão obsoleta: nenhum avanço")
)

// DB é o que o coordenador precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Coordinator é o coordenador único de transição.
type Coordinator struct {
	DB  DB
	Now func() time.Time
}

func (c *Coordinator) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

// Issue é a Issue com status canônico.
type Issue struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"project_id"`
	Key        string     `json:"external_key"`
	Kind       string     `json:"kind"`
	ParentID   string     `json:"parent_issue_id,omitempty"`
	FeatureID  string     `json:"feature_id,omitempty"`
	Status     string     `json:"status"`
	RawStatus  string     `json:"raw_status"`
	Version    int        `json:"version"`
	LastSeenAt *time.Time `json:"last_observed_at,omitempty"`
}

// NewIssue descreve uma Issue a acompanhar.
type NewIssue struct {
	ProjectID, Key, Kind, ParentID, FeatureID string
	Status                                    string // código ou alias; bruto é preservado
}

const issueCols = `i.id, i.project_id, i.external_key, i.kind, coalesce(i.parent_issue_id::text,''), coalesce(i.feature_id::text,''), c.code, i.raw_status, i.version, i.last_observed_at`

func scanIssue(row interface{ Scan(...any) error }) (Issue, error) {
	var i Issue
	if err := row.Scan(&i.ID, &i.ProjectID, &i.Key, &i.Kind, &i.ParentID, &i.FeatureID, &i.Status, &i.RawStatus, &i.Version, &i.LastSeenAt); err != nil {
		return Issue{}, persistence.MapError(err)
	}
	return i, nil
}

// Track registra a Issue (idempotente por projeto + chave externa). O status inicial é normalizado.
func (c *Coordinator) Track(ctx context.Context, n NewIssue) (Issue, error) {
	r, ok := flow.Normalize(n.Status)
	if !ok {
		return Issue{}, fmt.Errorf("%w: status desconhecido %q", persistence.ErrInvalidInput, n.Status)
	}
	if n.Kind != "feature" && n.Kind != "correction" {
		return Issue{}, fmt.Errorf("%w: tipo %q", persistence.ErrInvalidInput, n.Kind)
	}
	var parent, feature any
	if n.ParentID != "" {
		parent = n.ParentID
	}
	if n.FeatureID != "" {
		feature = n.FeatureID
	}
	_, err := c.DB.Exec(ctx, `INSERT INTO flow_issues (project_id, external_key, kind, parent_issue_id, feature_id, status_id, raw_status)
		VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (project_id, external_key) DO NOTHING`,
		n.ProjectID, n.Key, n.Kind, parent, feature, r.ID, r.Raw)
	if err != nil {
		return Issue{}, persistence.MapError(err)
	}
	return c.Get(ctx, n.ProjectID, n.Key)
}

// Get lê a Issue pela chave externa.
func (c *Coordinator) Get(ctx context.Context, projectID, key string) (Issue, error) {
	return scanIssue(c.DB.QueryRow(ctx, `SELECT `+issueCols+` FROM flow_issues i JOIN status_catalog c ON c.id=i.status_id
		WHERE i.project_id=$1 AND i.external_key=$2`, projectID, key))
}

// Request é um pedido de transição.
type Request struct {
	ProjectID, IssueKey string
	To                  string
	Actor               string // quem pede: só ActorSentinel efetiva
	Observer            string // quem observou/gerou o fato (agente, adaptador, humano)
	ObservationKey      string // identidade estável do fato (idempotência)
	ObservedAt          time.Time
	RawStatus           string // valor bruto observado (preservado)
	ExpectedVersion     int    // versão da Issue conhecida por quem decidiu
	Evidence            flow.Evidence
}

// Outcome é o resultado do pedido.
type Outcome struct {
	Decision  string         `json:"decision"`
	Issue     Issue          `json:"issue"`
	Reasons   []flow.Problem `json:"reasons,omitempty"`
	Duplicate bool           `json:"duplicate"` // a mesma observação já tinha sido decidida
}

func (c *Coordinator) audit(ctx context.Context, tx pgx.Tx, iss Issue, to string, dec string, req Request, reasons []flow.Problem, after int) error {
	rj, _ := json.Marshal(reasons)
	ej, _ := json.Marshal(req.Evidence)
	_, err := tx.Exec(ctx, `INSERT INTO flow_transitions (issue_id, from_code, to_code, decision, reasons, actor, observer, observation_key, raw_status, evidence, version_after)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (issue_id, observation_key, decision) DO NOTHING`,
		iss.ID, iss.Status, to, dec, rj, req.Actor, req.Observer, req.ObservationKey, req.RawStatus, ej, after)
	return err
}

// Transition valida e, se válida, efetiva a transição. Estado, auditoria, intenção de projeção e
// evento interno saem da mesma transação. Nunca escreve se o ator não for a Sentinela, se a
// versão ou a observação estiverem obsoletas ou se a guarda recusar.
func (c *Coordinator) Transition(ctx context.Context, req Request) (Outcome, error) {
	if req.ObservationKey == "" {
		return Outcome{}, fmt.Errorf("%w: chave da observação obrigatória", persistence.ErrInvalidInput)
	}
	target, ok := flow.Normalize(req.To)
	if !ok {
		return Outcome{}, fmt.Errorf("%w: status %q", persistence.ErrInvalidInput, req.To)
	}
	var out Outcome
	err := persistence.WithTx(ctx, c.DB, func(tx pgx.Tx) error {
		// Trava a linha antes de ler com o join (o join sob FOR UPDATE reavalia contra a linha antiga).
		var lockID string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM flow_issues WHERE project_id=$1 AND external_key=$2 FOR UPDATE`, req.ProjectID, req.IssueKey).Scan(&lockID); err != nil {
			return persistence.MapError(err)
		}
		iss, err := scanIssue(tx.QueryRow(ctx, `SELECT `+issueCols+` FROM flow_issues i JOIN status_catalog c ON c.id=i.status_id WHERE i.id=$1`, lockID))
		if err != nil {
			return err
		}
		out.Issue = iss
		// Reentrega: a mesma observação já decidida não escreve de novo.
		var prev string
		perr := tx.QueryRow(ctx, `SELECT decision FROM flow_transitions WHERE issue_id=$1 AND observation_key=$2 ORDER BY id LIMIT 1`, iss.ID, req.ObservationKey).Scan(&prev)
		if perr == nil {
			out.Decision, out.Duplicate = prev, true
			return nil
		}
		if !errors.Is(perr, pgx.ErrNoRows) {
			return perr
		}
		if req.Actor != ActorSentinel {
			out.Decision = DecisionRefused
			out.Reasons = []flow.Problem{{Code: "not_authority", Message: ErrNotAuthority.Error()}}
			return c.audit(ctx, tx, iss, target.Code, DecisionRefused, req, out.Reasons, iss.Version)
		}
		if req.ExpectedVersion != 0 && req.ExpectedVersion != iss.Version ||
			(!req.ObservedAt.IsZero() && iss.LastSeenAt != nil && req.ObservedAt.Before(*iss.LastSeenAt)) {
			out.Decision = DecisionStale
			out.Reasons = []flow.Problem{{Code: "stale", Message: ErrStale.Error()}}
			return c.audit(ctx, tx, iss, target.Code, DecisionStale, req, out.Reasons, iss.Version)
		}
		if gerr := flow.Check(iss.Status, target.Code, req.Evidence); gerr != nil {
			var rej *flow.Rejection
			if errors.As(gerr, &rej) {
				out.Decision, out.Reasons = DecisionRejected, rej.Problems
				return c.audit(ctx, tx, iss, target.Code, DecisionRejected, req, rej.Problems, iss.Version)
			}
			return gerr
		}
		seen := req.ObservedAt
		if seen.IsZero() {
			seen = c.now()
		}
		after := iss.Version + 1
		if _, err := tx.Exec(ctx, `UPDATE flow_issues SET status_id=(SELECT id FROM status_catalog WHERE code=$2), raw_status=$3, version=$4,
			last_observed_at=$5, updated_at=now() WHERE id=$1`, iss.ID, target.Code, req.RawStatus, after, seen); err != nil {
			return err
		}
		if err := c.audit(ctx, tx, iss, target.Code, DecisionApplied, req, nil, after); err != nil {
			return err
		}
		// Intenção de projeção e reavaliação: uma por revisão da Issue.
		if _, _, err := persistence.EnqueueOutbox(ctx, tx, req.ProjectID, fmt.Sprintf("status:%s:%d", iss.ID, after), DestProjection,
			map[string]any{"issue": iss.Key, "from": iss.Status, "to": target.Code, "version": after}); err != nil {
			return err
		}
		if _, err := Publish(ctx, tx, req.ProjectID, EventStateChanged, fmt.Sprintf("%s:%d", iss.ID, after), map[string]any{"issue": iss.Key, "to": target.Code}); err != nil {
			return err
		}
		out.Decision = DecisionApplied
		out.Issue.Status, out.Issue.Version, out.Issue.RawStatus = target.Code, after, req.RawStatus
		return nil
	})
	if err != nil {
		return Outcome{}, err
	}
	switch out.Decision {
	case DecisionRefused:
		return out, ErrNotAuthority
	case DecisionStale:
		return out, ErrStale
	}
	return out, nil
}

// History devolve a auditoria da Issue, em ordem.
type AuditRow struct {
	From, To, Decision, Actor, Observer, ObservationKey, RawStatus string
	Reasons                                                        json.RawMessage
	VersionAfter                                                   int
}

func (c *Coordinator) History(ctx context.Context, issueID string) ([]AuditRow, error) {
	rows, err := c.DB.Query(ctx, `SELECT from_code, to_code, decision, actor, observer, observation_key, raw_status, reasons, version_after
		FROM flow_transitions WHERE issue_id=$1 ORDER BY id`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditRow
	for rows.Next() {
		var a AuditRow
		if err := rows.Scan(&a.From, &a.To, &a.Decision, &a.Actor, &a.Observer, &a.ObservationKey, &a.RawStatus, &a.Reasons, &a.VersionAfter); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
