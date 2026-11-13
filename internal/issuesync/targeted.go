package issuesync

import (
	"context"
	"fmt"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// RegisterCorrection registra uma Issue de correção criada pela fábrica na origem (REPROVADO_PR),
// vinculada à Issue da feature. Idempotente pela identidade externa.
func (s *Syncer) RegisterCorrection(ctx context.Context, projectID string, src Source, it integrations.Item, parentKey string) error {
	if s.Sentinel == nil {
		return fmt.Errorf("%w: a Sentinela é necessária para rastrear a correção", persistence.ErrInvalidInput)
	}
	if _, err := s.DB.Exec(ctx, `INSERT INTO issue_registry (project_id, source, remote_id, issue_key, title, description, raw_status, status_code, revision, priority, kind, url, last_seen_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'correction',$11,$12) ON CONFLICT (project_id, source, remote_id) DO NOTHING`,
		projectID, src.Kind, it.ID, it.Key, it.Title, it.Description, it.Status, flow.ReprovadoPR, it.Version, it.Priority, it.URL, s.now()); err != nil {
		return persistence.MapError(err)
	}
	parent, err := s.Sentinel.Get(ctx, projectID, parentKey)
	if err != nil {
		return err
	}
	_, err = s.Sentinel.Track(ctx, sentinel.NewIssue{ProjectID: projectID, Key: it.Key, Kind: "correction", Status: flow.ReprovadoPR, ParentID: parent.ID, FeatureID: parent.FeatureID})
	return err
}

// forwardRelease encaminha à Sentinela a observação de liberação (CORRECAO_PR). A Sentinela valida:
// vínculo inequívoco com a feature e liberação humana. A chave da observação é a revisão observada,
// então sync geral, consulta direcionada e reentregas concorrentes viram uma só decisão.
func (s *Syncer) forwardRelease(ctx context.Context, projectID string, it integrations.Item) error {
	if s.Sentinel == nil {
		return nil
	}
	iss, err := s.Sentinel.Get(ctx, projectID, it.Key)
	if err != nil {
		if err == persistence.ErrNotFound {
			return nil // ainda não rastreada: o sync geral a registra
		}
		return err
	}
	if iss.Status != flow.ReprovadoPR {
		return nil
	}
	_, err = s.Sentinel.Transition(ctx, sentinel.Request{ProjectID: projectID, IssueKey: it.Key, To: flow.CorrecaoPR, Actor: sentinel.ActorSentinel,
		Observer: "origem:" + it.Provider, ObservationKey: "release:" + it.ID + ":" + it.Version, RawStatus: it.Status,
		Evidence: flow.Evidence{HumanRelease: true, LinkedToFeaturePR: iss.ParentID != "" && iss.FeatureID != ""}})
	return err
}

// waiting lista os IDs remotos das correções que aguardam liberação humana (alvo do ciclo direcionado).
func (s *Syncer) waiting(ctx context.Context, projectID, source string, limit int) ([]string, error) {
	rows, err := s.DB.Query(ctx, `SELECT remote_id FROM issue_registry WHERE project_id=$1 AND source=$2 AND kind='correction' AND status_code=$3
		ORDER BY first_seen_at, remote_id LIMIT $4`, projectID, source, flow.ReprovadoPR, limit)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// Targeted é o ciclo direcionado: consulta POR ID apenas as correções em REPROVADO_PR aguardando
// liberação (nunca lista a origem), atualiza o registro e encaminha à Sentinela a observação de
// CORRECAO_PR. Funciona mesmo que o filtro remoto de REPROVADO_PR já não encontre a Issue.
func (s *Syncer) Targeted(ctx context.Context, projectID string, tr integrations.Tracker, src Source) (Run, error) {
	run := Run{Trigger: TriggerTargeted, Scope: "corrections", Started: s.now()}
	ids, err := s.waiting(ctx, projectID, src.Kind, maxKnown)
	if err != nil {
		return Run{}, err
	}
	finish := func(msg string) (Run, error) {
		run.Finished = s.now()
		switch {
		case msg == "":
			run.Outcome = "ok"
		case run.Read > 0:
			run.Outcome, run.Error = "partial", msg
		default:
			run.Outcome, run.Error = "failed", msg
		}
		return s.record(ctx, projectID, TriggerTargeted, "corrections", run)
	}
	if len(ids) == 0 {
		return finish("") // nada aguardando: nada é consultado nem gravado como execução vazia
	}
	for _, id := range ids {
		it, err := tr.GetItem(ctx, id)
		if err != nil {
			msg := "falha ao ler a origem"
			if ie, ok := err.(*integrations.Error); ok {
				msg = ie.Error()
			}
			return finish(msg)
		}
		run.Read++
		o, err := s.apply(ctx, projectID, src, it)
		if err != nil {
			return Run{}, err
		}
		switch o {
		case oUpdated:
			run.Updated++
		case oIgnored:
			run.Ignored++
		}
	}
	return finish("")
}

// TargetedRoutine é a rotina periódica (intervalo próprio) do ciclo direcionado.
func (s *Syncer) TargetedRoutine(resolve Resolver, every time.Duration) sentinel.Routine {
	return sentinel.Routine{Name: sentinel.RoutineCorrections, Every: every, Run: func(ctx context.Context, scope, _ string) (string, error) {
		if scope == "" {
			return "", nil
		}
		src, tr, err := resolve(ctx, scope)
		if err != nil {
			return "", fmt.Errorf("origem indisponível: %w", err)
		}
		run, err := s.Targeted(ctx, scope, tr, src)
		if err != nil {
			return "", err
		}
		if run.Outcome != "ok" {
			return "", fmt.Errorf("consulta direcionada %s: %s", run.Outcome, run.Error)
		}
		return "", nil
	}}
}
