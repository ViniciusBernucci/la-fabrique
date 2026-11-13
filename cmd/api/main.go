// Comando api: API HTTP interna, autenticação e leitura de estado.
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
	"github.com/ViniciusBernucci/le-fabrique/internal/httpapi"
	"github.com/ViniciusBernucci/le-fabrique/internal/logging"
)

func main() { os.Exit(run()) }

func run() int {
	cfg, err := config.LoadAPI(os.LookupEnv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "api: "+err.Error())
		return 2 // 2 = configuração inválida
	}
	log := logging.New(os.Stdout, cfg.LogLevel, "api")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := connectWithRetry(ctx, cfg.DatabaseURL, log)
	if err != nil {
		log.Error("falha ao conectar ao PostgreSQL", "error", err.Error())
		return 1
	}
	defer pool.Close()

	api, err := httpapi.NewServer(cfg, pool, log)
	if err != nil {
		log.Error("falha ao montar a API", "error", err.Error())
		return 2
	}
	// Sem carregar os segredos já gravados a redação de logs/respostas seria parcial.
	primeCtx, cancelPrime := context.WithTimeout(ctx, 10*time.Second)
	err = api.Prime(primeCtx)
	cancelPrime()
	if err != nil {
		log.Error("falha ao carregar o cofre", "error", err.Error())
		return 1
	}
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errc := make(chan error, 1)
	go func() {
		log.Info("api iniciada", "addr", cfg.Addr, "env", string(cfg.Env))
		errc <- srv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		log.Error("servidor HTTP encerrou", "error", err.Error())
		return 1
	case <-ctx.Done():
	}
	log.Info("encerrando api")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("falha no shutdown", "error", err.Error())
		return 1
	}
	return 0
}
