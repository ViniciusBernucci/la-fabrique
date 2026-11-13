// Package trackadmit decide, a partir das subtarefas da fábrica, quais trilhas entram agora (AJ06-02).
// Dependências verificáveis, recursos compartilhados e vagas por trilha passam pela política pura da
// Sentinela e, ao admitir, viram reservas atômicas (vaga + recursos) com token de geração.
package trackadmit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// Etapas de uma subtarefa.
const (
	StageImplementation = "implementation"
	StageReview         = "review"
	StagePlanning       = "planning" // planejamento (Tech Lead) de uma Issue: TaskKey é a chave da Issue
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
	// DryRun decide e audita, mas não reserva nem altera estado (modo de observação antes de ligar a execução).
	DryRun bool
}

// Audit é uma decisão de admissão registrada na ordem da fila vigente.
type Audit struct {
	Candidate string `json:"candidate"`
	Stage     string `json:"stage"`
	Position  int    `json:"position"`
	Admitted  bool   `json:"admitted"`
	Wait      string `json:"wait,omitempty"`
}

// Result é o desfecho de uma avaliação.
type Result struct {
	Started []Started `json:"started"`
	Waiting []Waiting `json:"waiting"`
	Audit   []Audit   `json:"audit"`
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

type planRow struct {
	key        string
	correction bool
	priority   int
	at         time.Time
}

// loadPlanning lista as Issues prontas para planejar (PRD concluído ou correção liberada) e quantas
// vagas de planejamento estão ocupadas.
func (s *Service) loadPlanning(ctx context.Context, projectID string) (cands []planRow, held map[string]bool, busy int, err error) {
	held = map[string]bool{}
	rows, err := s.DB.Query(ctx, `SELECT holder FROM track_reservations WHERE project_id = $1 AND slot LIKE 'PLANNING:%'`, projectID)
	if err != nil {
		return nil, nil, 0, persistence.MapError(err)
	}
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			rows.Close()
			return nil, nil, 0, err
		}
		held[strings.TrimSuffix(strings.TrimPrefix(h, "plan/"), "#planning")] = true
		busy++
	}
	rows.Close()
	cr, err := s.DB.Query(ctx, `SELECT i.external_key, c.code = 'CORRECAO_PR', COALESCE(i.priority, 1000000), i.updated_at
		FROM flow_issues i JOIN status_catalog c ON c.id = i.status_id
		WHERE i.project_id = $1 AND c.code IN ('PRD-DOC-CONCLUIDO', 'CORRECAO_PR') ORDER BY i.updated_at, i.external_key`, projectID)
	if err != nil {
		return nil, nil, 0, persistence.MapError(err)
	}
	defer cr.Close()
	for cr.Next() {
		var r planRow
		if err := cr.Scan(&r.key, &r.correction, &r.priority, &r.at); err != nil {
			return nil, nil, 0, err
		}
		cands = append(cands, r)
	}
	return cands, held, busy, cr.Err()
}

// Tick avalia o projeto e admite o que couber (compatível com chamadas antigas; use Evaluate para a auditoria).
func (s *Service) Tick(ctx context.Context, projectID string, lim sentinel.Limits) ([]Started, []Waiting, error) {
	r, err := s.Evaluate(ctx, projectID, lim)
	return r.Started, r.Waiting, err
}

// Evaluate avalia o projeto e admite o que couber. A política decide; a reserva atômica confirma (uma
// corrida perdida volta a esperar). Nada em execução é interrompido. Planejamento concorre sob o limite próprio.
func (s *Service) Evaluate(ctx context.Context, projectID string, lim sentinel.Limits) (Result, error) {
	var out Result
	rs, err := s.load(ctx, projectID)
	if err != nil {
		return out, err
	}
	planCands, planHeld, planBusy, err := s.loadPlanning(ctx, projectID)
	if err != nil {
		return out, err
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
	planKey := map[string]string{}
	for _, pr := range planCands {
		if planHeld[pr.key] {
			continue
		}
		id := "plan/" + pr.key
		cands = append(cands, sentinel.Candidate{ID: id, Work: sentinel.WorkPlanning, Correction: pr.correction, Priority: pr.priority, EnqueuedAt: pr.at,
			WorkflowPublished: true, BudgetOK: true, Authorized: true})
		planKey[id], stageOf[id] = pr.key, StagePlanning
	}
	for i := 0; i < planBusy; i++ {
		running = append(running, sentinel.Running{ID: fmt.Sprintf("planning-%d", i), Work: sentinel.WorkPlanning})
	}
	decisions := sentinel.Admit(cands, running, done, lim)
	res := &sentinel.Reserver{DB: s.DB}
	var started []Started
	var waiting []Waiting
	for _, d := range decisions {
		stage := stageOf[d.ID]
		out.Audit = append(out.Audit, Audit{Candidate: d.ID, Stage: stage, Position: d.Position, Admitted: d.Admitted, Wait: d.Wait})
		if stage == StagePlanning {
			key := planKey[d.ID]
			if !d.Admitted {
				waiting = append(waiting, Waiting{TaskKey: key, Stage: stage, Reason: d.Wait})
				continue
			}
			holder := "plan/" + key + "#planning"
			if s.DryRun {
				started = append(started, Started{TaskKey: key, Stage: stage, Holder: holder})
				continue
			}
			var got []sentinel.Reservation
			var aerr error
			for slot := 1; slot <= max(lim.PlanningSlots, 1); slot++ {
				got, aerr = res.Acquire(ctx, projectID, holder, fmt.Sprintf("PLANNING:%d", slot))
				if aerr == nil {
					break
				}
			}
			if aerr != nil {
				waiting = append(waiting, Waiting{TaskKey: key, Stage: stage, Reason: sentinel.WaitPlanning})
				continue
			}
			started = append(started, Started{TaskKey: key, Stage: stage, Holder: holder, Reservations: got})
			continue
		}
		r := byID[d.ID]
		if !d.Admitted {
			waiting = append(waiting, Waiting{FeatureID: r.feature, TaskKey: r.key, Stage: stage, Reason: d.Wait})
			continue
		}
		if s.DryRun {
			started = append(started, Started{FeatureID: r.feature, TaskKey: r.key, Stage: stage, Track: r.track, Holder: Holder(r.feature, r.key, stage)})
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
			out.Started, out.Waiting = started, waiting
			return out, err
		}
		col := "impl_state"
		if stage == StageReview {
			col = "review_state"
		}
		if _, err := s.DB.Exec(ctx, `UPDATE subtasks SET `+col+` = 'running' WHERE feature_id::text = $1 AND task_key = $2 AND plan_version = (SELECT MAX(plan_version) FROM subtasks WHERE feature_id::text = $1)`, r.feature, r.key); err != nil {
			_, _ = res.ReleaseHolder(ctx, projectID, holder)
			out.Started, out.Waiting = started, waiting
			return out, persistence.MapError(err)
		}
		started = append(started, Started{FeatureID: r.feature, TaskKey: r.key, Stage: stage, Track: r.track, Holder: holder, Reservations: got})
	}
	out.Started, out.Waiting = started, waiting
	return out, nil
}

// Revert desfaz uma admissão que não pôde começar (ex.: o executor recusou): libera as reservas e devolve a
// etapa a pendente para a próxima avaliação.
func (s *Service) Revert(ctx context.Context, projectID string, st Started) error {
	if _, err := (&sentinel.Reserver{DB: s.DB}).ReleaseHolder(ctx, projectID, st.Holder); err != nil {
		return err
	}
	var col string
	switch st.Stage {
	case StageImplementation:
		col = "impl_state"
	case StageReview:
		col = "review_state"
	default:
		return nil // planejamento só guarda a reserva
	}
	_, err := s.DB.Exec(ctx, `UPDATE subtasks SET `+col+` = 'pending' WHERE feature_id::text = $1 AND task_key = $2 AND `+col+` = 'running'
		AND plan_version = (SELECT MAX(plan_version) FROM subtasks WHERE feature_id::text = $1)`, st.FeatureID, st.TaskKey)
	return persistence.MapError(err)
}

// CompletePlanning encerra o planejamento da Issue e libera a vaga própria.
func (s *Service) CompletePlanning(ctx context.Context, projectID, issueKey string) error {
	n, err := (&sentinel.Reserver{DB: s.DB}).ReleaseHolder(ctx, projectID, "plan/"+issueKey+"#planning")
	if err == nil && n == 0 {
		return persistence.ErrNotFound
	}
	return err
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
	return ProgressOf(ctx, s.DB, featureID)
}

// ProgressOf é Progress sobre qualquer leitor do banco.
func ProgressOf(ctx context.Context, q persistence.DBTX, featureID string) (status string, behind flow.Track, tracks map[flow.Track]flow.Stage, err error) {
	rows, err := q.Query(ctx, `SELECT track, impl_state, review_state FROM subtasks
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
