package issuesync

import (
	"context"
	"fmt"

	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// Resolver entrega a origem e o adaptador do projeto (a conexão sai do cofre, no chamador).
type Resolver func(ctx context.Context, projectID string) (Source, integrations.Tracker, error)

// Routine é a rotina do sync matinal. Falha total ou parcial devolve erro à agenda (retentativa
// com backoff); o progresso já gravado e o histórico do Run permanecem.
func (s *Syncer) Routine(resolve Resolver, at sentinel.DailyAt) sentinel.Routine {
	return sentinel.Routine{Name: sentinel.RoutineIssueSync, Daily: &at, Run: func(ctx context.Context, scope, _ string) (string, error) {
		if scope == "" {
			return "", nil
		}
		src, tr, err := resolve(ctx, scope)
		if err != nil {
			return "", fmt.Errorf("origem indisponível para o sync: %w", err)
		}
		run, err := s.General(ctx, scope, TriggerDaily, tr, src)
		if err != nil {
			return "", err
		}
		if run.Outcome != "ok" {
			return "", fmt.Errorf("sync %s: %s", run.Outcome, run.Error)
		}
		return "", nil
	}}
}
