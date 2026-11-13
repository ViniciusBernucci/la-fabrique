package availability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/budget"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Estados e classes de falha.
const (
	Available   = "available"
	Auth        = "auth"
	Quota       = "quota"
	Unavailable = "unavailable"
	Billing     = "billing"
)

// Fontes do sinal.
const (
	SourceOfficial = "official"
	SourceHistory  = "history"
)

// Parâmetros do backoff e do histórico.
const (
	BackoffBase      = 30 * time.Second
	BackoffMax       = 15 * time.Minute
	DefaultQuotaWait = time.Hour
	// HistoryThreshold: falhas de indisponibilidade seguidas para marcar a conta.
	HistoryThreshold = 2
)

// Backoff devolve a espera da n-ésima tentativa (1,2,3…): base·2^(n-1), com teto.
func Backoff(n int) time.Duration {
	if n < 1 {
		n = 1
	}
	d := BackoffBase
	for i := 1; i < n && d < BackoffMax; i++ {
		d *= 2
	}
	if d > BackoffMax {
		d = BackoffMax
	}
	return d
}

// Failure é uma falha observada numa chamada.
type Failure struct {
	Class    string     // auth | quota | unavailable | billing
	Official bool       // o provedor informou (código/cabeçalho), não é inferência
	RetryAt  *time.Time // quando o provedor diz que volta (ex.: reset da cota)
	Detail   string     // sem segredos
}

func (f Failure) valid() bool {
	switch f.Class {
	case Auth, Quota, Unavailable, Billing:
		return true
	}
	return false
}

// Health é a saúde calculada de uma conta.
type Health struct {
	ConnectionID string     `json:"connection_id"`
	State        string     `json:"state"`
	Source       string     `json:"source"`
	Reason       string     `json:"reason"`
	Until        *time.Time `json:"until"`
	Failures     int        `json:"consecutive_failures"`
}

// Usable informa se a conta pode receber trabalho agora.
func (h Health) Usable() bool { return h.State == Available }

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Tracker registra falhas e calcula a saúde.
type Tracker struct {
	DB  DB
	Now func() time.Time
}

func (t *Tracker) now() time.Time {
	if t.Now != nil {
		return t.Now().UTC()
	}
	return time.Now().UTC()
}

type stored struct {
	state, source, reason string
	until                 *time.Time
	failures              int
}

func (t *Tracker) load(ctx context.Context, q persistence.DBTX, conn string) (stored, error) {
	var s stored
	err := q.QueryRow(ctx, `SELECT state, source, reason, until, consecutive_failures FROM provider_health WHERE connection_id = $1`, conn).
		Scan(&s.state, &s.source, &s.reason, &s.until, &s.failures)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return stored{state: Available, source: SourceHistory}, nil
	}
	return s, persistence.MapError(err)
}

// Record registra uma falha e atualiza a saúde da conta.
func (t *Tracker) Record(ctx context.Context, conn string, f Failure) (Health, error) {
	if conn == "" || !f.valid() {
		return Health{}, &FieldError{"failure", "conta e classe de falha (auth, quota, unavailable, billing) são obrigatórias"}
	}
	now := t.now()
	err := persistence.WithTx(ctx, t.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 9))`, conn); err != nil {
			return persistence.MapError(err)
		}
		cur, err := t.load(ctx, tx, conn)
		if err != nil {
			return err
		}
		n := cur.failures + 1
		if f.Class != Unavailable {
			n = cur.failures // só indisponibilidade acumula histórico
		}
		state, source, until := f.Class, SourceHistory, f.RetryAt
		if f.Official {
			source = SourceOfficial
		}
		switch f.Class {
		case Unavailable:
			if n < HistoryThreshold && !f.Official {
				// Uma falha isolada é ruído: a conta segue disponível, só conta no histórico.
				state, until = Available, nil
			} else {
				step := n - HistoryThreshold + 1
				if f.Official {
					step = n
				}
				if until == nil {
					u := now.Add(Backoff(step))
					until = &u
				}
			}
		case Quota:
			if until == nil {
				u := now.Add(DefaultQuotaWait)
				until = &u
			}
		case Auth, Billing:
			until = nil // só sai com revalidação (Clear) ou sucesso explícito
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO provider_health (connection_id, state, source, reason, until, consecutive_failures, last_failure_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
			ON CONFLICT (connection_id) DO UPDATE SET state = $2, source = $3, reason = $4, until = $5,
				consecutive_failures = $6, last_failure_at = $7, updated_at = $7`,
			conn, state, source, f.Detail, until, n, now); err != nil {
			return persistence.MapError(err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO provider_events (connection_id, kind, class, source, detail, created_at) VALUES ($1, 'failure', $2, $3, $4, $5)`,
			conn, f.Class, source, f.Detail, now)
		return persistence.MapError(err)
	})
	if err != nil {
		return Health{}, err
	}
	return t.Get(ctx, conn)
}

// Success registra uma chamada bem-sucedida: zera o histórico e devolve a conta
// a disponível (um sucesso é evidência oficial de que ela funciona).
func (t *Tracker) Success(ctx context.Context, conn string) error {
	return t.reset(ctx, conn, "success", "")
}

// Clear devolve a conta a disponível após revalidação (ex.: credencial renovada).
func (t *Tracker) Clear(ctx context.Context, conn, why string) error {
	return t.reset(ctx, conn, "cleared", why)
}

func (t *Tracker) reset(ctx context.Context, conn, kind, detail string) error {
	if conn == "" {
		return &FieldError{"connection_id", "obrigatório"}
	}
	now := t.now()
	return persistence.WithTx(ctx, t.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 9))`, conn); err != nil {
			return persistence.MapError(err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO provider_health (connection_id, state, source, reason, until, consecutive_failures, last_success_at, updated_at)
			VALUES ($1, 'available', 'official', $2, NULL, 0, $3, $3)
			ON CONFLICT (connection_id) DO UPDATE SET state = 'available', source = 'official', reason = $2, until = NULL,
				consecutive_failures = 0, last_success_at = $3, updated_at = $3`, conn, detail, now); err != nil {
			return persistence.MapError(err)
		}
		_, err := tx.Exec(ctx, `INSERT INTO provider_events (connection_id, kind, detail, created_at) VALUES ($1, $2, $3, $4)`, conn, kind, detail, now)
		return persistence.MapError(err)
	})
}

// Get calcula a saúde ATUAL: o registrado, ajustado pelo tempo (o prazo vencido
// libera quota/unavailable) e pela cota oficial observada pelo adaptador.
func (t *Tracker) Get(ctx context.Context, conn string) (Health, error) {
	cur, err := t.load(ctx, t.DB, conn)
	if err != nil {
		return Health{}, err
	}
	now := t.now()
	h := Health{ConnectionID: conn, State: cur.state, Source: cur.source, Reason: cur.reason, Until: cur.until, Failures: cur.failures}
	// Prazo vencido: quota e unavailable voltam; auth e billing só saem por Clear/Success.
	if (h.State == Quota || h.State == Unavailable) && h.Until != nil && !h.Until.After(now) {
		h.State, h.Until = Available, nil
	}
	// Sinal oficial de cota (assinatura): esgotada e ainda sem reset → indisponível por cota;
	// observação posterior com folga libera uma cota marcada por falha.
	bs := &budget.Store{DB: t.DB}
	q, qerr := bs.LatestQuota(ctx, conn)
	switch {
	case qerr == nil && q.Limit > 0 && q.Used >= q.Limit && (q.ResetsAt == nil || q.ResetsAt.After(now)) && h.State == Available:
		h.State, h.Source, h.Until = Quota, SourceOfficial, q.ResetsAt
		h.Reason = fmt.Sprintf("cota observada esgotada (%d/%d %s, fonte %s)", q.Used, q.Limit, q.Unit, q.Source)
	case qerr == nil && h.State == Quota && q.Used < q.Limit && cur.until != nil && q.ObservedAt.After(lastFailure(ctx, t.DB, conn)):
		h.State, h.Until = Available, nil
	case qerr != nil && !errors.Is(qerr, persistence.ErrNotFound):
		return Health{}, qerr
	}
	return h, nil
}

func lastFailure(ctx context.Context, q persistence.DBTX, conn string) time.Time {
	var at *time.Time
	_ = q.QueryRow(ctx, `SELECT last_failure_at FROM provider_health WHERE connection_id = $1`, conn).Scan(&at)
	if at == nil {
		return time.Time{}
	}
	return *at
}

// FieldError é um erro de entrada.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }
