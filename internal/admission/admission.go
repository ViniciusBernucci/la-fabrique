// Package admission é o laço vivo de admissão da Sentinela (AJ03-03): reavalia o projeto por evento
// (conclusão, falha, vaga liberada, mudança de estado, observação) e por rotina periódica, admite por
// categoria, prioridade oficial, antiguidade e disponibilidade (vagas Front/Back, planejamento com limite
// próprio, capacidade global), reserva de forma atômica e entrega o trabalho ao Starter. Só age em projeto
// cujo despachante é a Sentinela (activation). Sem Starter o laço roda em observação (dry-run): decide e
// audita, mas não reserva nem muda estado, para ligar a admissão antes de ligar a execução.
package admission

import (
	"context"
	"fmt"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/activation"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/trackadmit"
)

// RoutineName é a rotina periódica da Sentinela.
const RoutineName = "admission.evaluate"

// Starter começa o trabalho admitido (provisiona worktree e dispara o agente). Erro desfaz a admissão.
type Starter interface {
	Start(ctx context.Context, projectID string, s trackadmit.Started) error
}

// DB é o que o laço precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Loop é o laço de admissão.
type Loop struct {
	DB            DB
	Starter       Starter // nil = observação (dry-run)
	PlanningSlots int     // limite próprio de planejamento (padrão 1)
}

// Report é o resultado de uma avaliação.
type Report struct {
	Skipped string               `json:"skipped,omitempty"`
	DryRun  bool                 `json:"dry_run"`
	Started []trackadmit.Started `json:"started"`
	Waiting []trackadmit.Waiting `json:"waiting"`
}

func (l *Loop) planning() int {
	if l.PlanningSlots > 0 {
		return l.PlanningSlots
	}
	return 1
}

// limits calcula a capacidade global restante: o limite global menos tudo o que roda (fila legada,
// implementações e reviews, planejamentos), em todos os projetos.
func (l *Loop) limits(ctx context.Context) (sentinel.Limits, error) {
	var global, legacy, subs, plans int
	if err := l.DB.QueryRow(ctx, `SELECT global_limit FROM scheduler_settings WHERE id = 1`).Scan(&global); err != nil {
		return sentinel.Limits{}, persistence.MapError(err)
	}
	_ = l.DB.QueryRow(ctx, `SELECT count(*) FROM queue_items WHERE state = 'running'`).Scan(&legacy)
	_ = l.DB.QueryRow(ctx, `SELECT count(*) FROM subtasks WHERE impl_state = 'running' OR review_state = 'running'`).Scan(&subs)
	_ = l.DB.QueryRow(ctx, `SELECT count(*) FROM track_reservations WHERE slot LIKE 'PLANNING:%'`).Scan(&plans)
	return sentinel.Limits{PlanningSlots: l.planning(), GlobalCapacity: max(global-legacy-subs-plans, 0)}, nil
}

// Evaluate reavalia o projeto agora.
func (l *Loop) Evaluate(ctx context.Context, projectID string) (Report, error) {
	rep := Report{DryRun: l.Starter == nil, Started: []trackadmit.Started{}, Waiting: []trackadmit.Waiting{}}
	ok, err := (&activation.Service{DB: l.DB}).CanAdmit(ctx, projectID, activation.Sentinel)
	if err != nil {
		return rep, err
	}
	if !ok {
		rep.Skipped = "o despachante do projeto não é a Sentinela"
		return rep, nil
	}
	lim, err := l.limits(ctx)
	if err != nil {
		return rep, err
	}
	ta := &trackadmit.Service{DB: l.DB, DryRun: l.Starter == nil}
	res, err := ta.Evaluate(ctx, projectID, lim)
	if err != nil {
		return rep, err
	}
	l.audit(ctx, projectID, res.Audit, rep.DryRun)
	rep.Waiting = append(rep.Waiting, res.Waiting...)
	for _, st := range res.Started {
		if l.Starter == nil {
			rep.Started = append(rep.Started, st)
			continue
		}
		if err := l.Starter.Start(ctx, projectID, st); err != nil {
			_ = ta.Revert(ctx, projectID, st)
			rep.Waiting = append(rep.Waiting, trackadmit.Waiting{FeatureID: st.FeatureID, TaskKey: st.TaskKey, Stage: st.Stage, Reason: "executor: " + err.Error()})
			continue
		}
		rep.Started = append(rep.Started, st)
	}
	return rep, nil
}

// audit grava só o que mudou desde a última decisão do mesmo candidato (a ordem da fila fica auditável sem
// repetir a mesma linha a cada avaliação).
func (l *Loop) audit(ctx context.Context, projectID string, items []trackadmit.Audit, dry bool) {
	for _, a := range items {
		_, _ = l.DB.Exec(ctx, `INSERT INTO admission_decisions (project_id, candidate, stage, position, admitted, wait, dry_run)
			SELECT $1,$2,$3,$4,$5,$6,$7 WHERE NOT EXISTS (
				SELECT 1 FROM admission_decisions d WHERE d.project_id = $1 AND d.candidate = $2 AND d.stage = $3
				AND d.id = (SELECT max(id) FROM admission_decisions WHERE project_id = $1 AND candidate = $2 AND stage = $3)
				AND d.admitted = $5 AND d.wait = $6 AND d.position = $4)`, projectID, a.Candidate, a.Stage, a.Position, a.Admitted, a.Wait, dry)
	}
}

// Decision é uma linha de auditoria.
type Decision struct {
	Candidate string    `json:"candidate"`
	Stage     string    `json:"stage"`
	Position  int       `json:"position"`
	Admitted  bool      `json:"admitted"`
	Wait      string    `json:"wait"`
	DryRun    bool      `json:"dry_run"`
	At        time.Time `json:"at"`
}

// History devolve as decisões mais recentes do projeto.
func (l *Loop) History(ctx context.Context, projectID string, limit int) ([]Decision, error) {
	rows, err := l.DB.Query(ctx, `SELECT candidate, stage, position, admitted, wait, dry_run, created_at FROM admission_decisions WHERE project_id = $1 ORDER BY id DESC LIMIT $2`, projectID, limit)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []Decision
	for rows.Next() {
		var d Decision
		if err := rows.Scan(&d.Candidate, &d.Stage, &d.Position, &d.Admitted, &d.Wait, &d.DryRun, &d.At); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// HandleEvent reavalia o projeto quando um evento durável o exige (sem esperar a varredura).
func (l *Loop) HandleEvent(ctx context.Context, ev sentinel.Event) error {
	switch ev.Kind {
	case sentinel.EventCompleted, sentinel.EventFailed, sentinel.EventSlotReleased, sentinel.EventStateChanged, sentinel.EventObservation:
	default:
		return nil
	}
	if ev.ProjectID == "" {
		return nil
	}
	if _, err := l.Evaluate(ctx, ev.ProjectID); err != nil {
		return fmt.Errorf("reavaliar a admissão: %w", err)
	}
	return nil
}

// Routine é a varredura periódica de segurança (a reavaliação por evento é o caminho rápido).
func (l *Loop) Routine(every time.Duration) sentinel.Routine {
	return sentinel.Routine{Name: RoutineName, Every: every, Lease: time.Minute, Backoff: 30 * time.Second,
		Run: func(ctx context.Context, scope, cursor string) (string, error) {
			if scope == "" {
				return "", nil
			}
			_, err := l.Evaluate(ctx, scope)
			return "", err
		}}
}
