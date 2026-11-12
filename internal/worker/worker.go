// Package worker implementa o ciclo de vida do processo worker: sondas de
// saúde e heartbeat. Leases, fila e execução chegam com os épicos E10–E12.
package worker

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// Pinger verifica a disponibilidade do banco.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Worker mantém o estado de saúde e emite heartbeats.
type Worker struct {
	db       Pinger
	log      *slog.Logger
	interval time.Duration
	healthy  atomic.Bool
	beats    atomic.Int64
}

// New cria o worker.
func New(db Pinger, log *slog.Logger, interval time.Duration) *Worker {
	w := &Worker{db: db, log: log, interval: interval}
	w.healthy.Store(true)
	return w
}

// Beats informa quantos heartbeats foram emitidos.
func (w *Worker) Beats() int64 { return w.beats.Load() }

// Healthy informa se o último ping ao banco funcionou.
func (w *Worker) Healthy() bool { return w.healthy.Load() }

// Run emite heartbeats até ctx ser cancelado.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.beat(ctx)
	for {
		select {
		case <-ctx.Done():
			w.log.Info("worker parando")
			return
		case <-ticker.C:
			w.beat(ctx)
		}
	}
}

func (w *Worker) beat(ctx context.Context) {
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	err := w.db.Ping(pingCtx)
	w.healthy.Store(err == nil)
	w.beats.Add(1)
	if err != nil {
		w.log.Warn("heartbeat: banco indisponível", "error", "ping falhou")
		return
	}
	w.log.Debug("heartbeat", "beats", w.beats.Load())
}

// HealthHandler expõe /healthz (processo vivo) e /readyz (banco acessível).
func (w *Worker) HealthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /readyz", func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		if !w.Healthy() {
			rw.WriteHeader(http.StatusServiceUnavailable)
			_, _ = rw.Write([]byte(`{"status":"database_unavailable"}`))
			return
		}
		_, _ = rw.Write([]byte(`{"status":"ready"}`))
	})
	return mux
}
