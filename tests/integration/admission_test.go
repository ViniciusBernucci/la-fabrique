package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/admission"
	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/trackadmit"
)

type recStarter struct {
	started []string
	fail    map[string]bool
}

func (r *recStarter) Start(_ context.Context, _ string, s trackadmit.Started) error {
	if r.fail[s.TaskKey] {
		return errors.New("sem executor disponível")
	}
	r.started = append(r.started, s.Stage+":"+s.TaskKey)
	return nil
}

func admissionEnv(t *testing.T) (testDB, string, string, *sentinel.Coordinator) {
	t.Helper()
	db := newMigratedDB(t)
	ctx := context.Background()
	if err := (&statusflow.Store{DB: db.Pool}).EnsureCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	p := mustProject(t, db.Pool, "adm")
	var feature string
	_ = db.Pool.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'AD-1','f') RETURNING id::text`, p.ID).Scan(&feature)
	_, _ = db.Pool.Exec(ctx, `INSERT INTO flow_activation (project_id, state, dispatcher) VALUES ($1,'active','sentinel')`, p.ID)
	_, _ = db.Pool.Exec(ctx, `UPDATE scheduler_settings SET global_limit = 8 WHERE id = 1`)
	return db, p.ID, feature, &sentinel.Coordinator{DB: db.Pool}
}

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// AJ03-03 / CT05, CT12 — laço vivo: duas trilhas independentes em paralelo, segunda tarefa Front espera,
// planejamento sob limite próprio com a correção na frente, falha do executor desfaz a admissão, e a conclusão
// reavalia por evento sem esperar a varredura.
func TestAdmissionLoopAdmitsByPolicyAuditsAndReactsToEvents(t *testing.T) {
	db, project, feature, co := admissionEnv(t)
	ctx := context.Background()
	seedSubtask(t, db, project, feature, "front-a", "FRONTEND", `[]`, `[]`)
	seedSubtask(t, db, project, feature, "front-b", "FRONTEND", `[]`, `[]`)
	seedSubtask(t, db, project, feature, "back-a", "BACKEND", `[]`, `[]`)
	// Duas Issues prontas para planejar: uma feature mais antiga e uma correção liberada mais nova.
	if _, err := co.Track(ctx, sentinel.NewIssue{ProjectID: project, Key: "FEAT-9", Kind: "feature", Status: statusflow.PRDDocConcluido}); err != nil {
		t.Fatal(err)
	}
	if _, err := co.Track(ctx, sentinel.NewIssue{ProjectID: project, Key: "FIX-9", Kind: "correction", Status: statusflow.CorrecaoPR}); err != nil {
		t.Fatal(err)
	}
	_, _ = db.Pool.Exec(ctx, `UPDATE flow_issues SET updated_at = now() - interval '2 hours' WHERE external_key = 'FEAT-9'`)

	// Observação (sem Starter): audita e não toca em nada.
	dry := &admission.Loop{DB: db.Pool, PlanningSlots: 1}
	rep, err := dry.Evaluate(ctx, project)
	if err != nil || !rep.DryRun || len(rep.Started) != 3 {
		t.Fatalf("%+v %v", rep, err)
	}
	var res, run int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM track_reservations WHERE project_id=$1`, project).Scan(&res)
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM subtasks WHERE project_id=$1 AND impl_state='running'`, project).Scan(&run)
	if res != 0 || run != 0 {
		t.Fatalf("o dry-run não pode reservar nem mudar estado: %d %d", res, run)
	}
	if _, err := dry.Evaluate(ctx, project); err != nil {
		t.Fatal(err)
	}
	var rows int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM admission_decisions WHERE project_id=$1 AND dry_run`, project).Scan(&rows)
	first := rows
	if _, err := dry.Evaluate(ctx, project); err != nil || first == 0 {
		t.Fatal(err)
	}
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM admission_decisions WHERE project_id=$1`, project).Scan(&rows)
	if rows != first {
		t.Fatalf("a auditoria só grava o que mudou: %d → %d", first, rows)
	}

	// Ligado: Front A, Back A e o planejamento da CORREÇÃO (antes da feature mais antiga); front-b e FEAT-9 esperam.
	st := &recStarter{fail: map[string]bool{"back-a": true}}
	loop := &admission.Loop{DB: db.Pool, Starter: st, PlanningSlots: 1}
	rep, err = loop.Evaluate(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	if !has(st.started, "implementation:front-a") || !has(st.started, "planning:FIX-9") || has(st.started, "planning:FEAT-9") || has(st.started, "implementation:front-b") {
		t.Fatalf("admitidos: %v", st.started)
	}
	// A falha do executor desfez a admissão do Back: a vaga foi liberada e o item voltou a pendente.
	var backState string
	_ = db.Pool.QueryRow(ctx, `SELECT impl_state FROM subtasks WHERE task_key='back-a'`).Scan(&backState)
	var backHolder int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM track_reservations WHERE project_id=$1 AND slot='BACKEND'`, project).Scan(&backHolder)
	if backState != "pending" || backHolder != 0 {
		t.Fatalf("revert: %s %d", backState, backHolder)
	}
	reasons := map[string]string{}
	for _, w := range rep.Waiting {
		reasons[w.TaskKey+"/"+w.Stage] = w.Reason
	}
	if reasons["FEAT-9/planning"] != sentinel.WaitPlanning || reasons["front-b/implementation"] != sentinel.WaitSlot {
		t.Fatalf("%v", reasons)
	}
	// A conclusão do Front A libera a vaga: o evento durável reavalia e admite o Front B sem esperar a varredura.
	st.fail = map[string]bool{}
	if err := (&trackadmit.Service{DB: db.Pool}).Complete(ctx, project, feature, "front-a", trackadmit.StageImplementation); err != nil {
		t.Fatal(err)
	}
	if _, err := sentinel.Publish(ctx, db.Pool, project, sentinel.EventCompleted, "front-a-done", map[string]string{"task": "front-a"}); err != nil {
		t.Fatal(err)
	}
	before := len(st.started)
	if h, f, err := sentinel.Drain(ctx, db.Pool, 50, loop.HandleEvent); err != nil || h == 0 || f != 0 {
		t.Fatalf("%d %d %v", h, f, err)
	}
	got := st.started[before:]
	if !has(got, "review:front-a") || !has(got, "implementation:back-a") {
		t.Fatalf("a conclusão reavalia por evento (review antecipado na vaga do Front e o Back que voltou): %v", got)
	}
	// A auditoria tem a ordem da fila e quem foi admitido.
	h, err := loop.History(ctx, project, 50)
	if err != nil || len(h) == 0 {
		t.Fatalf("%v %v", h, err)
	}
}

// AJ03-03 — só a Sentinela admite projeto ativado; o limite global regula novas admissões.
func TestAdmissionLoopRespectsDispatcherAndGlobalCapacity(t *testing.T) {
	db, project, feature, _ := admissionEnv(t)
	ctx := context.Background()
	seedSubtask(t, db, project, feature, "front-a", "FRONTEND", `[]`, `[]`)
	seedSubtask(t, db, project, feature, "back-a", "BACKEND", `[]`, `[]`)
	st := &recStarter{}
	loop := &admission.Loop{DB: db.Pool, Starter: st}
	// Capacidade global 1: só um dos dois independentes entra.
	_, _ = db.Pool.Exec(ctx, `UPDATE scheduler_settings SET global_limit = 1 WHERE id = 1`)
	if rep, err := loop.Evaluate(ctx, project); err != nil || len(rep.Started) != 1 {
		t.Fatalf("%+v %v", rep, err)
	}
	// Outro despachante ativo: a Sentinela não admite.
	_, _ = db.Pool.Exec(ctx, `UPDATE flow_activation SET dispatcher='legacy' WHERE project_id=$1`, project)
	_, _ = db.Pool.Exec(ctx, `UPDATE scheduler_settings SET global_limit = 8 WHERE id = 1`)
	rep, err := loop.Evaluate(ctx, project)
	if err != nil || rep.Skipped == "" || len(rep.Started) != 0 {
		t.Fatalf("%+v %v", rep, err)
	}
}
