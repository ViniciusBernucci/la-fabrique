package sentinel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Tipos de evento que provocam reavaliação imediata.
const (
	EventCompleted    = "execution.completed"
	EventFailed       = "execution.failed"
	EventStateChanged = "state.changed"
	EventSlotReleased = "slot.released"
	EventObservation  = "observation.received"
)

// Event é um evento interno durável.
type Event struct {
	ID        int64           `json:"id"`
	ProjectID string          `json:"project_id"`
	Kind      string          `json:"kind"`
	Key       string          `json:"dedupe_key"`
	Payload   json.RawMessage `json:"payload"`
	Attempts  int             `json:"attempts"`
}

// Publish grava o evento (em db, que pode ser a transação do fato que o gerou). Repetir a
// mesma chave não duplica. created=false se já existia.
func Publish(ctx context.Context, db persistence.DBTX, projectID, kind, key string, payload any) (created bool, err error) {
	if kind == "" || key == "" {
		return false, fmt.Errorf("%w: kind e chave obrigatórios", persistence.ErrInvalidInput)
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	var pid any
	if projectID != "" {
		pid = projectID
	}
	tag, err := db.Exec(ctx, `INSERT INTO sentinel_events (project_id, kind, dedupe_key, payload) VALUES ($1,$2,$3,$4)
		ON CONFLICT DO NOTHING`, pid, kind, key, b)
	if err != nil {
		return false, persistence.MapError(err)
	}
	return tag.RowsAffected() == 1, nil
}

// Drain entrega os eventos pendentes, em ordem, ao coordenador. Evento tratado com sucesso é
// marcado; com falha continua pendente (com a tentativa registrada) e não impede os seguintes.
// Após reinício, o que ficou pendente é recuperado: o estado vive no banco.
func Drain(ctx context.Context, db persistence.DBTX, limit int, handle func(context.Context, Event) error) (handled, failed int, err error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Query(ctx, `SELECT id, coalesce(project_id::text,''), kind, dedupe_key, payload, attempts FROM sentinel_events
		WHERE handled_at IS NULL ORDER BY id LIMIT $1`, limit)
	if err != nil {
		return 0, 0, err
	}
	var evs []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.Kind, &e.Key, &e.Payload, &e.Attempts); err != nil {
			rows.Close()
			return 0, 0, err
		}
		evs = append(evs, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	for _, e := range evs {
		if herr := handle(ctx, e); herr != nil {
			failed++
			if _, err := db.Exec(ctx, `UPDATE sentinel_events SET attempts=attempts+1, last_error=$2 WHERE id=$1`, e.ID, truncate(herr.Error(), 300)); err != nil {
				return handled, failed, err
			}
			continue
		}
		if _, err := db.Exec(ctx, `UPDATE sentinel_events SET handled_at=now() WHERE id=$1 AND handled_at IS NULL`, e.ID); err != nil {
			return handled, failed, err
		}
		handled++
	}
	return handled, failed, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
