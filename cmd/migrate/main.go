// Comando migrate: aplica migrations com identidade própria e carrega fixtures
// de desenvolvimento. Uso: migrate up | down | status | seed-dev
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/logging"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/tests/fixtures"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "uso: migrate up|down|status|seed-dev")
		return 2
	}
	cfg, err := config.LoadMigrate(os.LookupEnv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migrate: "+err.Error())
		return 2
	}
	log := logging.New(os.Stdout, cfg.LogLevel, "migrate")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if args[0] == "seed-dev" {
		return seedDev(ctx, cfg, log)
	}
	m, err := persistence.NewMigrator(cfg.DatabaseURL)
	if err != nil {
		log.Error("falha ao preparar migrações", "error", err.Error())
		return 1
	}
	defer m.Close()

	switch args[0] {
	case "up":
		res, err := m.Up(ctx)
		if err != nil {
			log.Error("migrate up falhou", "error", err.Error())
			return 1
		}
		log.Info("migrate up concluído", "aplicadas", len(res))
	case "down":
		res, err := m.Down(ctx)
		if err != nil {
			log.Error("migrate down falhou", "error", err.Error())
			return 1
		}
		log.Info("migrate down concluído", "versao", res.Source.Version)
	case "status":
		st, err := m.Status(ctx)
		if err != nil {
			log.Error("migrate status falhou", "error", err.Error())
			return 1
		}
		for _, s := range st {
			fmt.Printf("%05d  %-10s  %s\n", s.Source.Version, s.State, s.Source.Path)
		}
	default:
		fmt.Fprintln(os.Stderr, "comando desconhecido: "+args[0])
		return 2
	}
	return 0
}

func seedDev(ctx context.Context, cfg config.Migrate, log interface {
	Info(string, ...any)
	Error(string, ...any)
}) int {
	if cfg.Env != config.Development {
		log.Error("seed-dev só é permitido com APP_ENV=development")
		return 2
	}
	pool, err := persistence.Connect(ctx, cfg.DatabaseURL, 10*time.Second)
	if err != nil {
		log.Error("falha ao conectar", "error", err.Error())
		return 1
	}
	defer pool.Close()
	items, err := fixtures.Projects()
	if err != nil {
		log.Error("fixtures inválidas", "error", err.Error())
		return 1
	}
	created := 0
	for _, f := range items {
		tag, err := pool.Exec(ctx, `
			INSERT INTO projects (slug, name, state, priority) VALUES ($1, $2, $3, $4)
			ON CONFLICT (slug) DO NOTHING`, f.Slug, f.Name, f.State, f.Priority)
		if err != nil {
			log.Error("falha ao inserir fixture", "slug", f.Slug, "error", persistence.MapError(err).Error())
			return 1
		}
		created += int(tag.RowsAffected())
	}
	log.Info("seed-dev concluído", "criados", created, "total", len(items))
	return 0
}
