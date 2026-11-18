package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/factory"
	"github.com/ViniciusBernucci/le-fabrique/internal/oauth"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/worker"
)

// registerFactory liga a execução dos workflows publicados (opt-in: FACTORY_RUN=true, ADR-0069). Cada tarefa
// reclama uma execução (uma por vez neste worker), conduz o fluxo até concluir, dormir ou esperar uma decisão
// e devolve um resumo. O terminal do agente precisa do docker.sock e da imagem de CLIs (make cli-image).
func registerFactory(w *worker.Worker, pool *pgxpool.Pool, cfg config.Worker) {
	if !cfg.FactoryRun {
		return
	}
	sealer, err := security.NewSealer(cfg.VaultKey, cfg.VaultKeyID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "worker: cofre da fábrica: "+err.Error())
		return
	}
	rd := security.NewRedactor()
	vault := security.NewVault(pool, sealer, rd)
	loadCtx, cancelLoad := context.WithTimeout(context.Background(), 15*time.Second)
	if _, err := vault.LoadRedactor(loadCtx); err != nil {
		fmt.Fprintln(os.Stderr, "worker: redator da fábrica: "+err.Error())
	}
	cancelLoad()
	vault.SetRefresher(oauth.NewRefresher(oauth.FromEnv(os.LookupEnv)))
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "worker", "job", "fabrica")
	ex := &factory.Executor{DB: pool, Cfg: factory.Config{
		Workspaces: cfg.WorkspacesDir, CLIImage: cfg.FactoryImage, ToolImage: cfg.FactoryToolImage, ChecksNetwork: cfg.FactoryChecksNetwork, Docker: &execution.Docker{}, Vault: vault, AllowPush: cfg.FactoryPush,
		Log: log, Redact: rd.Redact,
	}}
	owner := fmt.Sprintf("worker-%d", os.Getpid())
	w.AddJob(worker.Job{Name: "fabrica", Every: 3 * time.Second, Run: func(ctx context.Context) (string, error) {
		run, err := runner.Claim(ctx, pool, owner, 2*time.Minute)
		if err != nil || run == nil {
			return "", err
		}
		log.Info("execução reclamada", "execution", run.ExecutionID, "title", run.Title)
		done, err := runner.Advance(ctx, pool, *run, runner.Options{Exec: ex, Lease: 2 * time.Minute, Before: ex.Preflight, Log: log})
		switch {
		case errors.Is(err, runner.ErrCancelled):
			return fmt.Sprintf("execução %s cancelada", run.ExecutionID), nil
		case err != nil:
			// erro de infraestrutura: a posse vence e a execução volta a ser reclamada do último passo salvo
			return "", fmt.Errorf("execução %s: %w", run.ExecutionID, err)
		}
		return fmt.Sprintf("execução %s: %s", done.ExecutionID, done.Status), nil
	}})
	log.Info("fábrica ligada", "push", cfg.FactoryPush, "imagem", cfg.FactoryImage)
}
