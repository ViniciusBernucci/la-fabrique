package queue

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Erros do scheduler.
var (
	ErrStale = errors.New("escritor obsoleto: a lease venceu ou a vaga foi dada a outro")
)

// Settings são as configurações persistentes do scheduler.
type Settings struct {
	GlobalLimit  int `json:"global_limit"`
	AgingMinutes int `json:"aging_minutes"`
	Version      int `json:"version"`
}

func (q *Queue) settings(ctx context.Context, db persistence.DBTX) (Settings, error) {
	var s Settings
	err := db.QueryRow(ctx, `SELECT global_limit, aging_minutes, version FROM scheduler_settings WHERE id = 1`).Scan(&s.GlobalLimit, &s.AgingMinutes, &s.Version)
	return s, persistence.MapError(err)
}

// Settings devolve o limite global e o envelhecimento (persistentes).
func (q *Queue) Settings(ctx context.Context) (Settings, error) { return q.settings(ctx, q.DB) }

// SetSettings altera o limite global (1 a 64) e o período de envelhecimento (minutos).
// A mudança vale para as próximas reclamações: nada em execução é preemptado.
func (q *Queue) SetSettings(ctx context.Context, limit, agingMinutes int) (Settings, error) {
	if limit < 1 || limit > 64 {
		return Settings{}, &FieldError{"global_limit", "o limite global vai de 1 a 64"}
	}
	if agingMinutes < 1 || agingMinutes > 10080 {
		return Settings{}, &FieldError{"aging_minutes", "o envelhecimento vai de 1 minuto a 7 dias"}
	}
	var s Settings
	err := q.DB.QueryRow(ctx, `UPDATE scheduler_settings SET global_limit = $1, aging_minutes = $2, version = version + 1, updated_at = $3 WHERE id = 1
		RETURNING global_limit, aging_minutes, version`, limit, agingMinutes, q.now()).Scan(&s.GlobalLimit, &s.AgingMinutes, &s.Version)
	return s, persistence.MapError(err)
}

// SetPriority muda a prioridade de um item que ainda espera na fila. Afeta só a
// próxima seleção; nunca preempta quem está em execução.
func (q *Queue) SetPriority(ctx context.Context, itemID string, priority int) (Item, error) {
	if priority < 0 || priority > 4 {
		return Item{}, &FieldError{"priority", "a prioridade vai de 0 a 4"}
	}
	it, err := scanItem(q.DB.QueryRow(ctx, `UPDATE queue_items SET priority = $2, version = version + 1 WHERE id = $1 AND state = 'queued' RETURNING `+cols, itemID, priority))
	if errors.Is(err, persistence.ErrNotFound) {
		return Item{}, fmt.Errorf("%w: só um item na fila muda de prioridade", ErrBadState)
	}
	return it, err
}

// Fence identifica o escritor que detém a vaga: quem reclamou e a geração.
type Fence struct {
	Owner      string
	Generation int64
}

type fenceKey struct{}

// WithFence anexa o token ao contexto das operações do escritor.
func WithFence(ctx context.Context, f Fence) context.Context {
	return context.WithValue(ctx, fenceKey{}, f)
}

func fenceOf(ctx context.Context) (Fence, bool) {
	f, ok := ctx.Value(fenceKey{}).(Fence)
	return f, ok
}

// checkFence recusa o escritor obsoleto. Item reclamado pelo scheduler (geração > 0)
// exige o token do dono atual e a lease viva; item iniciado sem scheduler (geração 0)
// segue as regras anteriores.
func (q *Queue) checkFence(ctx context.Context, it Item) error {
	if it.Generation == 0 {
		return nil
	}
	f, ok := fenceOf(ctx)
	switch {
	case !ok:
		return fmt.Errorf("%w: informe o token do dono da vaga", ErrStale)
	case f.Generation != it.Generation || f.Owner != it.LeaseOwner:
		return fmt.Errorf("%w: geração %d, dono %q; a vaga está na geração %d", ErrStale, f.Generation, f.Owner, it.Generation)
	case it.LeaseExpires == nil || !it.LeaseExpires.After(q.now()):
		return fmt.Errorf("%w: a lease venceu", ErrStale)
	}
	return nil
}

func (q *Queue) event(ctx context.Context, itemID, projectID, action, owner string, gen int64, detail string) {
	_, _ = q.DB.Exec(ctx, `INSERT INTO queue_events (item_id, project_id, action, owner, generation, detail) VALUES ($1, $2, $3, $4, $5, $6)`,
		itemID, projectID, action, owner, gen, detail)
}

// Heartbeat renova a lease do dono atual. Token antigo, lease vencida ou item que
// não está mais em execução: ErrStale (o escritor deve parar).
func (q *Queue) Heartbeat(ctx context.Context, itemID string, f Fence, lease time.Duration) (Item, error) {
	now := q.now()
	it, err := scanItem(q.DB.QueryRow(ctx, `UPDATE queue_items SET lease_expires_at = $5, heartbeat_at = $4
		WHERE id = $1 AND state = 'running' AND lease_owner = $2 AND generation = $3 AND lease_expires_at > $4 RETURNING `+cols,
		itemID, f.Owner, f.Generation, now, now.Add(lease)))
	if errors.Is(err, persistence.ErrNotFound) {
		var pid string
		_ = q.DB.QueryRow(ctx, `SELECT project_id::text FROM queue_items WHERE id = $1`, itemID).Scan(&pid)
		q.event(ctx, itemID, pid, "heartbeat_refused", f.Owner, f.Generation, "token antigo, lease vencida ou item fora de execução")
		return Item{}, ErrStale
	}
	return it, err
}

// ExpireLeases devolve à fila os itens em execução cuja lease venceu (o dono sumiu):
// a vaga é liberada e a próxima reclamação sobe a geração, então o dono antigo, se
// voltar, não consegue mais gravar.
func (q *Queue) ExpireLeases(ctx context.Context) ([]Item, error) {
	var out []Item
	err := persistence.WithTx(ctx, q.DB, func(tx pgx.Tx) error {
		var err error
		out, err = q.expireTx(ctx, tx)
		return err
	})
	return out, err
}

func (q *Queue) expireTx(ctx context.Context, tx pgx.Tx) ([]Item, error) {
	now := q.now()
	rows, err := tx.Query(ctx, `UPDATE queue_items SET state = 'queued', slot_held = false, lease_owner = NULL, lease_expires_at = NULL, version = version + 1
		WHERE state = 'running' AND lease_expires_at IS NOT NULL AND lease_expires_at <= $1 RETURNING `+cols, now)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	var out []Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, it)
	}
	rows.Close()
	for _, it := range out {
		if it.Kind == KindTask {
			if _, err := tx.Exec(ctx, `UPDATE plan_tasks SET state = 'pending', updated_at = $4 WHERE project_id = $1 AND feature_id = $2 AND task_key = $3 AND state = 'running'`,
				it.ProjectID, it.FeatureID, it.TaskKey, now); err != nil {
				return nil, persistence.MapError(err)
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO queue_events (item_id, project_id, action, owner, generation, detail) VALUES ($1, $2, 'lease_expired', '', $3, 'o dono não renovou a lease')`,
			it.ID, it.ProjectID, it.Generation); err != nil {
			return nil, persistence.MapError(err)
		}
	}
	return out, nil
}

// Claim reclama UMA vaga para o worker `owner`, em transação serializada: devolve à
// fila o que está com lease vencida, respeita o limite global, escolhe pelo critério
// justo (correção de PR, prioridade com envelhecimento, prioridade do projeto, chegada)
// e inicia o item com lease e geração nova. Sem capacidade ou sem candidato: (nil, nil).
func (q *Queue) Claim(ctx context.Context, owner, executionID string, lease time.Duration) (*Item, error) {
	if owner == "" || len(owner) > 100 {
		return nil, &FieldError{"owner", "informe o worker (até 100 caracteres)"}
	}
	if lease < time.Second || lease > time.Hour {
		return nil, &FieldError{"lease", "a lease vai de 1 segundo a 1 hora"}
	}
	var out *Item
	err := persistence.WithTx(ctx, q.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('scheduler', 15))`); err != nil {
			return persistence.MapError(err)
		}
		if _, err := q.expireTx(ctx, tx); err != nil {
			return err
		}
		st, err := q.settings(ctx, tx)
		if err != nil {
			return err
		}
		var running int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM queue_items WHERE state = 'running'`).Scan(&running); err != nil {
			return persistence.MapError(err)
		}
		if running >= st.GlobalLimit {
			return nil
		}
		cand, err := q.candidatesIn(ctx, tx)
		if err != nil {
			return err
		}
		for _, c := range cand {
			it, err := q.startTx(ctx, tx, c.ID, executionID, owner, lease)
			var ne *NotEligibleError
			if errors.As(err, &ne) {
				continue
			}
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO queue_events (item_id, project_id, action, owner, generation, detail) VALUES ($1, $2, 'claimed', $3, $4, '')`,
				it.ID, it.ProjectID, owner, it.Generation); err != nil {
				return persistence.MapError(err)
			}
			out = &it
			return nil
		}
		return nil
	})
	return out, err
}

// Guard devolve a guarda de gravação de checkpoint para o dono da vaga: dentro da
// transação do Save, confere (com bloqueio da linha) que o item segue em execução, com
// o mesmo dono, a mesma geração e a lease viva, e que o checkpoint é da execução dele.
// Escritor obsoleto: ErrStale e nada é gravado.
func (q *Queue) Guard(itemID string, f Fence) func(ctx context.Context, tx pgx.Tx, in checkpoints.Input) error {
	return func(ctx context.Context, tx pgx.Tx, in checkpoints.Input) error {
		it, err := scanItem(tx.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1 FOR SHARE`, itemID))
		if err != nil {
			return err
		}
		refuse := func(why string) error {
			q.event(ctx, itemID, it.ProjectID, "write_refused", f.Owner, f.Generation, why)
			return fmt.Errorf("%w: %s", ErrStale, why)
		}
		switch {
		case it.State != StateRunning:
			return refuse("o item não está em execução")
		case it.LeaseOwner != f.Owner || it.Generation != f.Generation:
			return refuse(fmt.Sprintf("o token é da geração %d e a vaga está na geração %d", f.Generation, it.Generation))
		case it.LeaseExpires == nil || !it.LeaseExpires.After(q.now()):
			return refuse("a lease venceu")
		case it.ExecutionID != "" && in.ExecutionID != it.ExecutionID:
			return refuse("o checkpoint é de outra execução")
		}
		return nil
	}
}

// Events lista a trilha do item (reclamações, recusas e vencimentos).
func (q *Queue) Events(ctx context.Context, itemID string) ([]string, error) {
	rows, err := q.DB.Query(ctx, `SELECT action, owner, generation FROM queue_events WHERE item_id = $1 ORDER BY id`, itemID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var a, o string
		var g int64
		if err := rows.Scan(&a, &o, &g); err != nil {
			return nil, persistence.MapError(err)
		}
		out = append(out, fmt.Sprintf("%s:%s:%d", a, o, g))
	}
	sort.Strings(out[:0])
	return out, persistence.MapError(rows.Err())
}
