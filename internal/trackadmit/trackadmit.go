// Package trackadmit decide, a partir das subtarefas da fábrica, quais trilhas entram agora (AJ06-02).
// Dependências verificáveis, recursos compartilhados e vagas por trilha passam pela política pura da
// Sentinela e, ao admitir, viram reservas atômicas (vaga + recursos) com token de geração.
package trackadmit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// Etapas de uma subtarefa.
const (
	StageImplementation = "implementation"
	StageReview         = "review"
)

// DB é o que o admissor precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Service admite trilhas.
type Service struct {
	DB  DB
	Now func() time.Time
}

// Started é um trabalho admitido com as reservas que o protegem.
type Started struct {
	FeatureID    string                 `json:"feature_id"`
	TaskKey      string                 `json:"task_key"`
	Stage        string                 `json:"stage"`
	Track        string                 `json:"track"`
	Holder       string                 `json:"holder"`
	Reservations []sentinel.Reservation `json:"reservations"`
}

// Waiting é um trabalho que segue aguardando e o motivo.
type Waiting struct {
	FeatureID string `json:"feature_id"`
	TaskKey   string `json:"task_key"`
	Stage     string `json:"stage"`
	Reason    string `json:"reason"`
}

// Holder identifica quem segura as reservas de um trabalho.
func Holder(featureID, taskKey, stage string) string { return featureID + "/" + taskKey + "#" + stage }

func resSlot(r string) string { return "RES:" + r }

type row struct {
	feature, key, track, impl, review, class string
	deps, resources                          []string
	both                                     bool
	created                                  time.Time
	priority                                 int
}

func (s *Service) load(ctx context.Context, projectID string) ([]row, error) {
	rows, err := s.DB.Query(ctx, `SELECT st.feature_id::text, st.task_key, st.track, st.impl_state, st.review_state, st.work_class, st.depends_on, st.resources, st.exclusive_both, st.created_at, COALESCE(fi.priority, 1000000)
		FROM subtasks st
		JOIN (SELECT feature_id, MAX(plan_version) v FROM subtasks WHERE project_id = $1 GROUP BY feature_id) cur ON cur.feature_id = st.feature_id AND cur.v = st.plan_version
		LEFT JOIN flow_issues fi ON fi.feature_id = st.feature_id AND fi.project_id = st.project_id AND fi.kind IN ('feature','correction')
		WHERE st.project_id = $1 AND st.state <> 'blocked'
		ORDER BY st.created_at, st.feature_id, st.task_key`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []row
	for rows.Next() {
		var r row
		var d, rs []byte
		if err := rows.Scan(&r.feature, &r.key, &r.track, &r.impl, &r.review, &r.class, &d, &rs, &r.both, &r.created, &r.priority); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(d, &r.deps)
		_ = json.Unmarshal(rs, &r.resources)
		out = append(out, r)
	}
	return out, rows.Err()
}

// Tick avalia o projeto e admite o que couber. A política decide; a reserva atômica confirma (uma
// corrida perdida volta a esperar). Nada em execução é interrompido.
func (s *Service) Tick(ctx context.Context, projectID string, lim sentinel.Limits) ([]Started, []Waiting, error) {
	rs, err := s.load(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	done := map[string]bool{}
	var cands []sentinel.Candidate
	var running []sentinel.Running
	byID := map[string]row{}
	stageOf := map[string]string{}
	for _, r := range rs {
		q := r.feature + "/" + r.key
		if r.impl == "done" {
			done[q] = true
		}
		var deps []string
		for _, d := range r.deps {
			deps = append(deps, r.feature+"/"+d)
		}
		track := flow.Track(r.track)
		switch {
		case r.impl == "running":
			running = append(running, sentinel.Running{ID: q, Work: sentinel.WorkImplementation, Track: track, NeedsBoth: r.both, Resources: r.resources})
		case r.review == "running":
			running = append(running, sentinel.Running{ID: q + "#review", Work: sentinel.WorkReview, Track: track})
		case r.impl == "pending":
			cands = append(cands, sentinel.Candidate{ID: q, Work: sentinel.WorkImplementation, Track: track, NeedsBoth: r.both, Correction: r.class == "correction",
				Priority: r.priority, EnqueuedAt: r.created, DependsOn: deps, Resources: r.resources,
				WorkflowPublished: true, BudgetOK: true, Authorized: true})
			byID[q], stageOf[q] = r, StageImplementation
		case r.impl == "done" && r.review == "pending":
			// O review começa assim que a implementação da própria tarefa termina, sem esperar a outra trilha.
			id := q + "#review"
			cands = append(cands, sentinel.Candidate{ID: id, Work: sentinel.WorkReview, Track: track, Correction: r.class == "correction",
				Priority: r.priority, EnqueuedAt: r.created, WorkflowPublished: true, BudgetOK: true, Authorized: true})
			byID[id], stageOf[id] = r, StageReview
		}
	}
	decisions := sentinel.Admit(cands, running, done, lim)
	res := &sentinel.Reserver{DB: s.DB}
	var started []Started
	var waiting []Waiting
	for _, d := range decisions {
		r := byID[d.ID]
		stage := stageOf[d.ID]
		if !d.Admitted {
			waiting = append(waiting, Waiting{FeatureID: r.feature, TaskKey: r.key, Stage: stage, Reason: d.Wait})
			continue
		}
		holder := Holder(r.feature, r.key, stage)
		slots := []string{string(r.track)}
		if r.both && stage == StageImplementation {
			slots = []string{string(flow.Frontend), string(flow.Backend)}
		}
		if stage == StageImplementation {
			for _, x := range r.resources {
				slots = append(slots, resSlot(x))
			}
		}
		got, err := res.Acquire(ctx, projectID, holder, slots...)
		if errors.Is(err, sentinel.ErrSlotBusy) {
			waiting = append(waiting, Waiting{FeatureID: r.feature, TaskKey: r.key, Stage: stage, Reason: sentinel.WaitSlot})
			continue
		}
		if err != nil {
			return started, waiting, err
		}
		col := "impl_state"
		if stage == StageReview {
			col = "review_state"
		}
		if _, err := s.DB.Exec(ctx, `UPDATE subtasks SET `+col+` = 'running' WHERE feature_id::text = $1 AND task_key = $2 AND plan_version = (SELECT MAX(plan_version) FROM subtasks WHERE feature_id::text = $1)`, r.feature, r.key); err != nil {
			_, _ = res.ReleaseHolder(ctx, projectID, holder)
			return started, waiting, persistence.MapError(err)
		}
		started = append(started, Started{FeatureID: r.feature, TaskKey: r.key, Stage: stage, Track: r.track, Holder: holder, Reservations: got})
	}
	return started, waiting, nil
}

// Complete encerra uma etapa e libera vaga e recursos do titular. Para review sem exigência use SkipReview.
func (s *Service) Complete(ctx context.Context, projectID, featureID, taskKey, stage string) error {
	col := "impl_state"
	switch stage {
	case StageImplementation:
	case StageReview:
		col = "review_state"
	default:
		return fmt.Errorf("%w: etapa %q", persistence.ErrInvalidInput, stage)
	}
	tag, err := s.DB.Exec(ctx, `UPDATE subtasks SET `+col+` = 'done' WHERE feature_id::text = $1 AND task_key = $2 AND `+col+` = 'running'
		AND plan_version = (SELECT MAX(plan_version) FROM subtasks WHERE feature_id::text = $1)`, featureID, taskKey)
	if err != nil {
		return persistence.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return persistence.ErrNotFound
	}
	_, err = (&sentinel.Reserver{DB: s.DB}).ReleaseHolder(ctx, projectID, Holder(featureID, taskKey, stage))
	return err
}

// Progress resume o andamento da feature por trilha. O status agregado segue a etapa obrigatória mais
// atrasada: um Front revisado não leva a feature a REVIEW enquanto o Back ainda implementa.
func (s *Service) Progress(ctx context.Context, featureID string) (status string, behind flow.Track, tracks map[flow.Track]flow.Stage, err error) {
	rows, err := s.DB.Query(ctx, `SELECT track, impl_state, review_state FROM subtasks
		WHERE feature_id::text = $1 AND plan_version = (SELECT MAX(plan_version) FROM subtasks WHERE feature_id::text = $1)`, featureID)
	if err != nil {
		return "", "", nil, persistence.MapError(err)
	}
	defer rows.Close()
	tracks = map[flow.Track]flow.Stage{}
	for rows.Next() {
		var tr, impl, rev string
		if err := rows.Scan(&tr, &impl, &rev); err != nil {
			return "", "", nil, err
		}
		st := flow.StageNone
		switch {
		case impl == "running":
			st = flow.StageImplementing
		case impl == "done" && (rev == "done" || rev == "not_required"):
			st = flow.StageReviewed
		case impl == "done":
			st = flow.StageImplemented
		}
		if cur, ok := tracks[flow.Track(tr)]; !ok || st < cur {
			tracks[flow.Track(tr)] = st
		}
	}
	if err := rows.Err(); err != nil {
		return "", "", nil, err
	}
	status, behind = flow.Aggregate(len(tracks) > 0, tracks)
	return status, behind, tracks, nil
}
