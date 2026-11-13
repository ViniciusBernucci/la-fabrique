// Comando worker: processo que reclama e executa trabalho (a lógica de fila,
// leases e execução chega nos épicos E10–E12; aqui há configuração, saúde e
// ciclo de vida).
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/logging"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/recovery"
	"github.com/ViniciusBernucci/le-fabrique/internal/retention"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() { os.Exit(run()) }

func run() int {
	cfg, err := config.LoadWorker(os.LookupEnv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "worker: "+err.Error())
		return 2
	}
	log := logging.New(os.Stdout, cfg.LogLevel, "worker")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := connectWithRetry(ctx, cfg.DatabaseURL, log)
	if err != nil {
		log.Error("falha ao conectar ao PostgreSQL", "error", err.Error())
		return 1
	}
	defer pool.Close()

	w := worker.New(pool, log, cfg.HeartbeatInterval)
	registerJobs(w, pool, cfg)
	srv := &http.Server{Addr: cfg.HealthAddr, Handler: w.HealthHandler(), ReadHeaderTimeout: 5 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	go w.Run(ctx)
	log.Info("worker iniciado", "health_addr", cfg.HealthAddr, "env", string(cfg.Env))

	select {
	case err := <-errc:
		log.Error("servidor de saúde encerrou", "error", err.Error())
		return 1
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return 1
	}
	log.Info("worker encerrado")
	return 0
}

// registerJobs liga ao worker o que não depende do executor de agentes.
func registerJobs(w *worker.Worker, pool *pgxpool.Pool, cfg config.Worker) {
	cps := &checkpoints.Store{DB: pool}
	q := &queue.Queue{DB: pool, Checkpoints: cps, RequireRecovery: true, Gates: map[string]queue.Gate{queue.WaitDecision: queue.DecisionGate{DB: pool}}}
	rec := &recovery.Recoverer{DB: pool, Queue: q, Checkpoints: cps, Redactor: security.NewRedactor(),
		Inspector: recovery.DockerInspector{Docker: &execution.Docker{}}}
	w.AddJob(worker.ReleaseDecisionsJob(pool, q, cfg.JobsInterval))
	w.AddJob(worker.RecoverJob(rec, cfg.JobsInterval))
	// Sentinela: rotinas com relógio/cursor próprios por projeto. As rotinas concretas (sync, correções,
	// PRs, manutenção) são registradas pelos seus pacotes; sem registro o Tick não faz nada.
	sch := &sentinel.Scheduler{DB: pool}
	w.AddJob(worker.SentinelRoutinesJob(sch, func(ctx context.Context) ([]string, error) { return sentinel.ProjectScopes(ctx, pool) }, cfg.JobsInterval))
	w.AddJob(worker.RetentionJob(&retention.Retention{DB: pool, Checkpoints: cps}, retention.Policy{}, cfg.RetentionApply, cfg.RetentionInterval))
}
