package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/issuesync"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// registerIssueSync registra o sync matinal das Issues (opt-in: SENTINEL_ISSUE_SYNC=true). O segredo da
// conexão sai do cofre a cada execução e nunca é logado.
func registerIssueSync(sch *sentinel.Scheduler, pool *pgxpool.Pool, cfg config.Worker) error {
	if !cfg.IssueSync {
		return nil
	}
	sealer, err := security.NewSealer(cfg.VaultKey, cfg.VaultKeyID)
	if err != nil {
		return fmt.Errorf("cofre do worker: %w", err)
	}
	vault := security.NewVault(pool, sealer, security.NewRedactor())
	loc, _ := time.LoadLocation(cfg.IssueSyncTZ)
	fs := &flow.Store{DB: pool}
	resolve := func(ctx context.Context, projectID string) (issuesync.Source, integrations.Tracker, error) {
		p, err := projects.Get(ctx, pool, projectID)
		if err != nil {
			return issuesync.Source{}, nil, err
		}
		if p.State != "active" {
			return issuesync.Source{}, nil, fmt.Errorf("projeto não está ativo")
		}
		if p.Source.Kind == "" || p.Source.ConnectionID == nil || p.Source.Container == "" {
			return issuesync.Source{}, nil, projects.ErrSourceIncomplete
		}
		m, ok, err := fs.GetMap(ctx, projectID)
		if err != nil {
			return issuesync.Source{}, nil, err
		}
		if !ok {
			return issuesync.Source{}, nil, fmt.Errorf("mapa de status não configurado")
		}
		secret, conn, err := vault.Resolve(ctx, *p.Source.ConnectionID)
		if err != nil {
			return issuesync.Source{}, nil, err
		}
		tr, err := integrations.NewTracker(integrations.Config{Kind: conn.Kind, Target: conn.Target, Secret: secret})
		if err != nil {
			return issuesync.Source{}, nil, err
		}
		return issuesync.Source{Kind: p.Source.Kind, Container: p.Source.Container, Map: m.ProjectMap}, tr, nil
	}
	sy := &issuesync.Syncer{DB: pool, Sentinel: &sentinel.Coordinator{DB: pool}}
	if err := sch.Register(sy.Routine(resolve, sentinel.DailyAt{Hour: cfg.IssueSyncHour, Loc: loc})); err != nil {
		return err
	}
	return sch.Register(sy.TargetedRoutine(resolve, cfg.CorrectionsPollInterval))
}
