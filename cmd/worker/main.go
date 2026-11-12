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

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/logging"
	"github.com/ViniciusBernucci/le-fabrique/internal/worker"
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
