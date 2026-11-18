package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ViniciusBernucci/le-fabrique/internal/clilogin"
	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/worker"
)

// registerCLILogin liga o login oficial das contas de assinatura de IA (opt-in: CLI_LOGIN=true, ADR-0065).
// O worker precisa do docker.sock; sem ele, cada tentativa falha com uma mensagem que diz o que montar.
func registerCLILogin(w *worker.Worker, pool *pgxpool.Pool, cfg config.Worker) {
	if !cfg.CLILogin {
		return
	}
	sealer, err := security.NewSealer(cfg.VaultKey, cfg.VaultKeyID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "worker: cofre do login de contas: "+err.Error())
		return
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "worker", "job", "clilogin")
	r := &clilogin.Runner{
		Store: &clilogin.Store{DB: pool}, Vault: security.NewVault(pool, sealer, security.NewRedactor()), Log: log,
		Image: cfg.CLILoginImage, ContextDir: cfg.CLILoginContext,
	}
	// Logins que ficaram abertos de uma execução anterior não têm mais processo: encerra e limpa.
	sweepCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := r.Sweep(sweepCtx); err != nil {
		log.Warn("varredura dos logins pendentes falhou", "error", err.Error())
	}
	w.AddJob(worker.Job{Name: "login-contas-ia", Every: 2 * time.Second, Run: r.Tick})
}
