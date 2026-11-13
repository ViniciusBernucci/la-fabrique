package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Event é um evento monotônico de execução.
type Event struct {
	ExecutionID string
	Seq         int64
	ProjectID   string
	Type        string
	Payload     json.RawMessage
}

// AppendEvent grava o próximo evento da execução. A sequência nunca reaproveita um seq de
// evento removido pela retenção (o último seq fica no resumo da execução). Trava a linha da execução
// (escopada ao projeto) para que a sequência seja monotônica sem lacunas, mesmo
// com escritores concorrentes. Deve ser chamado dentro de uma transação.
func AppendEvent(ctx context.Context, tx pgx.Tx, projectID, executionID, eventType string, payload any) (Event, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("%w: payload não serializável", ErrInvalidInput)
	}
	if payload == nil {
		raw = []byte("{}")
	}
	var locked string
	err = tx.QueryRow(ctx,
		`SELECT id FROM executions WHERE id = $1 AND project_id = $2 FOR UPDATE`,
		executionID, projectID).Scan(&locked)
	if err != nil {
		return Event{}, MapError(err)
	}
	ev := Event{ExecutionID: executionID, ProjectID: projectID, Type: eventType, Payload: raw}
	err = tx.QueryRow(ctx, `
		INSERT INTO execution_events (execution_id, seq, project_id, type, payload)
		SELECT $1, GREATEST(COALESCE(MAX(seq), 0), COALESCE((SELECT last_seq FROM execution_summaries WHERE execution_id = $1), 0)) + 1, $2, $3, $4::jsonb
		FROM execution_events WHERE execution_id = $1
		RETURNING seq`,
		executionID, projectID, eventType, string(raw)).Scan(&ev.Seq)
	if err != nil {
		return Event{}, MapError(err)
	}
	return ev, nil
}

// EventsAfter devolve eventos com seq > after (cursor de reconexão SSE).
func EventsAfter(ctx context.Context, db DBTX, projectID, executionID string, after int64, limit int) ([]Event, error) {
	if limit < 1 || limit > 1000 {
		limit = 200
	}
	rows, err := db.Query(ctx, `
		SELECT execution_id, seq, project_id, type, payload
		FROM execution_events
		WHERE project_id = $1 AND execution_id = $2 AND seq > $3
		ORDER BY seq LIMIT $4`, projectID, executionID, after, limit)
	if err != nil {
		return nil, MapError(err)
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var payload []byte
		if err := rows.Scan(&e.ExecutionID, &e.Seq, &e.ProjectID, &e.Type, &payload); err != nil {
			return nil, MapError(err)
		}
		e.Payload = payload
		out = append(out, e)
	}
	return out, MapError(rows.Err())
}
