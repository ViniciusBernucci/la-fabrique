package central

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/authorizations"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Situação de uma autorização, calculada pelo servidor.
const (
	StatusActive    = "active"
	StatusExpired   = "expired"
	StatusExhausted = "exhausted"
	StatusRevoked   = "revoked"
)

// AuthView é uma autorização com a situação, o uso em andamento e o que resta.
type AuthView struct {
	authorizations.Authorization
	Status           string `json:"status"`
	Remaining        int    `json:"remaining"`
	InFlight         int    `json:"in_flight"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"` // negativo = já venceu
}

// StatusOf calcula a situação no instante dado.
func StatusOf(a authorizations.Authorization, now time.Time) string {
	switch {
	case a.State == "revoked":
		return StatusRevoked
	case !a.ExpiresAt.After(now):
		return StatusExpired
	case a.Uses >= a.MaxUses:
		return StatusExhausted
	}
	return StatusActive
}

// ListAuthorizations lista as autorizações (as mais recentes primeiro, até 200); `status` vazio ou
// "all" devolve todas.
func ListAuthorizations(ctx context.Context, db persistence.DBTX, now time.Time, status string) ([]AuthView, error) {
	rows, err := db.Query(ctx, `SELECT a.id::text, a.project_id::text, a.execution_id::text, a.node_id, a.decision_id::text, a.tool, a.operation, a.resource, a.target, a.params_hash,
		a.max_uses, a.uses, a.expires_at, a.state, a.revoked_at, a.revoke_reason, a.version, a.created_at,
		(SELECT count(*) FROM authorization_uses u WHERE u.authorization_id = a.id AND u.state = 'started')
		FROM authorizations a ORDER BY a.created_at DESC, a.id LIMIT 200`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []AuthView{}
	for rows.Next() {
		var v AuthView
		a := &v.Authorization
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.ExecutionID, &a.NodeID, &a.DecisionID, &a.Tool, &a.Operation, &a.Resource, &a.Target, &a.ParamsHash,
			&a.MaxUses, &a.Uses, &a.ExpiresAt, &a.State, &a.RevokedAt, &a.RevokeReason, &a.Version, &a.CreatedAt, &v.InFlight); err != nil {
			return nil, persistence.MapError(err)
		}
		v.Status = StatusOf(*a, now)
		v.Remaining = a.Remaining(now)
		v.ExpiresInSeconds = int64(a.ExpiresAt.Sub(now).Seconds())
		if status == "" || status == "all" || status == v.Status {
			out = append(out, v)
		}
	}
	return out, persistence.MapError(rows.Err())
}

// AuthorizationProject devolve o projeto da autorização (404 se não existir).
func AuthorizationProject(ctx context.Context, db persistence.DBTX, id string) (string, error) {
	var p string
	err := db.QueryRow(ctx, `SELECT project_id::text FROM authorizations WHERE id::text = $1`, id).Scan(&p)
	return p, persistence.MapError(err)
}

// ExecutionProject devolve o projeto da execução (404 se não existir).
func ExecutionProject(ctx context.Context, db persistence.DBTX, id string) (string, error) {
	var p string
	err := db.QueryRow(ctx, `SELECT project_id::text FROM executions WHERE id::text = $1`, id).Scan(&p)
	return p, persistence.MapError(err)
}

// Entry é uma linha da sequência do incidente e da retomada.
type Entry struct {
	At     time.Time       `json:"at"`
	Source string          `json:"source"` // execution | decision | authorization | resume | recovery
	Type   string          `json:"type"`
	Detail json.RawMessage `json:"detail"`
}

var sourceOrder = map[string]int{"execution": 0, "decision": 1, "authorization": 2, "recovery": 3, "resume": 4}

// Sequence reúne, em ordem de tempo, o que aconteceu com a execução: eventos, pedidos e decisões,
// autorizações (concessão, uso, revogação, parada), recuperação e retomada. Os registros já foram
// redigidos na origem. Até 500 linhas.
func Sequence(ctx context.Context, db persistence.DBTX, projectID, executionID string) ([]Entry, error) {
	queries := []struct{ source, sql string }{
		{"execution", `SELECT created_at, type, payload FROM execution_events WHERE project_id::text = $1 AND execution_id::text = $2`},
		{"decision", `SELECT created_at, 'decision_requested', jsonb_build_object('request_id', id, 'kind', kind, 'title', title, 'node_id', node_id) FROM decision_requests WHERE project_id::text = $1 AND execution_id::text = $2
			UNION ALL SELECT resolved_at, 'decision_resolution', jsonb_build_object('request_id', id, 'option', resolution_option, 'note', resolution_note) FROM decision_requests WHERE project_id::text = $1 AND execution_id::text = $2 AND state = 'resolved'`},
		{"authorization", `SELECT e.created_at, e.type, e.detail || jsonb_build_object('authorization_id', a.id) FROM authorization_events e JOIN authorizations a ON a.id = e.authorization_id WHERE a.project_id::text = $1 AND a.execution_id::text = $2`},
		{"recovery", `SELECT r.created_at, r.action, r.detail || jsonb_build_object('class', r.class, 'slot_released', r.slot_released) FROM recovery_events r JOIN queue_items q ON q.id = r.item_id WHERE q.project_id::text = $1 AND q.execution_id::text = $2`},
		{"resume", `SELECT created_at, action, detail || jsonb_build_object('node', node_key) FROM resume_events WHERE project_id::text = $1 AND execution_id::text = $2`},
	}
	out := []Entry{}
	for _, q := range queries {
		rows, err := db.Query(ctx, q.sql+` ORDER BY 1 LIMIT 500`, projectID, executionID)
		if err != nil {
			return nil, persistence.MapError(err)
		}
		for rows.Next() {
			e := Entry{Source: q.source}
			var d []byte
			if err := rows.Scan(&e.At, &e.Type, &d); err != nil {
				rows.Close()
				return nil, persistence.MapError(err)
			}
			e.Detail = d
			out = append(out, e)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, persistence.MapError(err)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.Before(out[j].At)
		}
		return sourceOrder[out[i].Source] < sourceOrder[out[j].Source]
	})
	if len(out) > 500 {
		out = out[:500]
	}
	return out, nil
}
