package integration

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
)

type plEnv struct {
	e       *authEnv
	s       *planning.Store
	proj    string
	feature string
}

func newPlEnv(t *testing.T, slug string) *plEnv {
	t.Helper()
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, slug)
	f, err := persistence.CreateFeature(context.Background(), e.db.Pool, p.ID, "PL-1", "Feature de planejamento")
	if err != nil {
		t.Fatal(err)
	}
	return &plEnv{e: e, s: &planning.Store{DB: e.db.Pool}, proj: p.ID, feature: f.ID}
}

func plTask(key string, deps ...string) planning.Task {
	return planning.Task{Key: key, Title: "Tarefa " + key, Description: "Faz " + key, Risk: "medium", DependsOn: deps,
		Criteria: []planning.Criterion{{Text: "Os testes de " + key + " passam", Check: planning.CheckTest, Target: "./internal/" + key}}}
}

func basePlan() planning.Plan {
	return planning.Plan{Summary: "decomposição", Tasks: []planning.Task{plTask("schema"), plTask("api", "schema"), plTask("tela", "api")}}
}

func (p *plEnv) propose(t *testing.T, plan planning.Plan) planning.Outcome {
	t.Helper()
	o, err := p.s.Propose(context.Background(), p.proj, p.feature, plan)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return o
}

// T10-01 — "Feature vira tarefas verificáveis, sem ciclo..."
func TestFeatureBecomesVerifiableTasksInAnAcyclicPlan(t *testing.T) {
	p := newPlEnv(t, "pl-ok")
	ctx := context.Background()
	if _, err := p.s.Current(ctx, p.proj, p.feature); !errors.Is(err, planning.ErrNoPlan) {
		t.Fatalf("%v", err)
	}
	o := p.propose(t, basePlan())
	if !o.Accepted || o.Stored.Version != 1 || o.Stored.Status != "current" || len(o.Problems) != 0 {
		t.Fatalf("%+v", o)
	}
	cur, _ := p.s.Current(ctx, p.proj, p.feature)
	if got := strings.Join(planning.Order(cur.Plan), ","); got != "schema,api,tela" || cur.Hash == "" {
		t.Fatalf("%s", got)
	}
	tasks, _ := p.s.Tasks(ctx, p.proj, p.feature)
	if len(tasks) != 3 || tasks[0].State != planning.StatePending || tasks[0].Hash == "" {
		t.Fatalf("%+v", tasks)
	}
}

// T10-01 — "Plano conflitante preserva versão anterior."
func TestInvalidOrConflictingPlansPreserveThePreviousVersion(t *testing.T) {
	p := newPlEnv(t, "pl-conflict")
	ctx := context.Background()
	p.propose(t, basePlan())
	rejected := func(name string, plan planning.Plan, want string) {
		t.Helper()
		o, err := p.s.Propose(ctx, p.proj, p.feature, plan)
		var re *planning.RejectedError
		if !errors.As(err, &re) || !errors.Is(err, planning.ErrRejected) || o.Accepted || o.Stored.Status != "rejected" || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: %v %+v", name, err, o)
		}
		cur, _ := p.s.Current(ctx, p.proj, p.feature)
		if cur.Version != 1 || len(cur.Plan.Tasks) != 3 {
			t.Fatalf("%s: a versão vigente mudou: %+v", name, cur)
		}
	}
	cyc := basePlan()
	cyc.Tasks[0].DependsOn = []string{"tela"}
	rejected("ciclo", cyc, "ciclo")
	rejected("referência ausente", planning.Plan{Tasks: []planning.Task{plTask("a", "fantasma")}}, "referência ausente")
	vague := basePlan()
	vague.Tasks[1].Criteria = []planning.Criterion{{Text: "Funciona bem para todo mundo"}}
	rejected("critério inverificável", vague, "verificável")

	// Uma tarefa que já está em execução não pode ser alterada nem removida.
	if err := p.s.SetTaskState(ctx, p.proj, p.feature, "schema", planning.StateRunning); err != nil {
		t.Fatal(err)
	}
	changed := basePlan()
	changed.Tasks[0].Title = "Outro título"
	rejected("alterar tarefa em execução", changed, "não pode ser alterada")
	removed := planning.Plan{Tasks: []planning.Task{plTask("api", "schema"), plTask("tela", "api")}}
	rejected("remover tarefa em execução (dependência órfã)", removed, "referência ausente")
	removed2 := planning.Plan{Tasks: []planning.Task{plTask("api"), plTask("tela", "api")}}
	rejected("remover tarefa em execução (plano válido)", removed2, "não pode ser removida")

	vs, _ := p.s.Versions(ctx, p.proj, p.feature)
	rej := 0
	for _, v := range vs {
		if v.Status == "rejected" {
			rej++
			if len(v.Problems) == 0 {
				t.Fatal("a rejeição registra os problemas")
			}
		}
	}
	if len(vs) != 7 || rej != 6 {
		t.Fatalf("versões: %d rejeitadas: %d", len(vs), rej)
	}

	// Mudar o que ainda está pendente é aceito, e a identidade da tarefa em execução fica intacta.
	ok := basePlan()
	ok.Tasks[2].Title = "Tela renomeada"
	ok.Tasks = append(ok.Tasks, plTask("docs", "tela"))
	o := p.propose(t, ok)
	if !o.Accepted || o.Stored.Version != 8 {
		t.Fatalf("%+v", o)
	}
	tasks, _ := p.s.Tasks(ctx, p.proj, p.feature)
	st := map[string]planning.StoredTask{}
	for _, tk := range tasks {
		st[tk.Key] = tk
	}
	if len(tasks) != 4 || st["schema"].State != planning.StateRunning || st["tela"].Title != "Tela renomeada" || st["docs"].State != planning.StatePending {
		t.Fatalf("%+v", tasks)
	}
	// Remover uma tarefa pendente marca como removida (sem apagar a identidade).
	o = p.propose(t, planning.Plan{Tasks: ok.Tasks[:3]})
	tasks, _ = p.s.Tasks(ctx, p.proj, p.feature)
	removedFlag := false
	for _, tk := range tasks {
		if tk.Key == "docs" {
			removedFlag = tk.Removed
		}
	}
	if !o.Accepted || !removedFlag {
		t.Fatalf("%+v %+v", o, tasks)
	}
	if err := p.s.SetTaskState(ctx, p.proj, p.feature, "schema", planning.StatePending); err != nil {
		t.Fatal(err)
	}
	if err := p.s.SetTaskState(ctx, p.proj, p.feature, "schema", planning.StateDone); err == nil {
		t.Fatal("pendente não vai direto a concluída")
	}
}

func TestTheSamePlanDoesNotCreateANewVersion(t *testing.T) {
	p := newPlEnv(t, "pl-same")
	p.propose(t, basePlan())
	rev := basePlan()
	rev.Tasks[0], rev.Tasks[2] = rev.Tasks[2], rev.Tasks[0] // outra ordem de escrita
	o := p.propose(t, rev)
	if !o.Unchanged || o.Stored.Version != 1 {
		t.Fatalf("%+v", o)
	}
	// Quatro propostas simultâneas do mesmo plano novo: uma versão só.
	next := basePlan()
	next.Tasks = append(next.Tasks, plTask("docs", "tela"))
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := p.s.Propose(context.Background(), p.proj, p.feature, next); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	vs, _ := p.s.Versions(context.Background(), p.proj, p.feature)
	if len(vs) != 2 {
		t.Fatalf("versões: %d", len(vs))
	}
}

// T10-01 — "... sem efeito duplicado no retry."
func TestPublishingThePlanToTheOriginNeverDuplicatesOnRetry(t *testing.T) {
	p := newPlEnv(t, "pl-publish")
	ctx := context.Background()
	p.propose(t, basePlan())
	tr := newMem()
	// A primeira criação é aplicada na origem, mas a resposta se perde.
	tr.createFn = func(call int) (bool, error) {
		if call == 1 {
			return true, errTrackerTimeout
		}
		return true, nil
	}
	pub := publisher(p.e, tr, nil)
	rep, err := p.s.Publish(ctx, p.proj, p.feature, "P1", pub)
	if err != nil || len(rep.Failed) != 0 || len(rep.Subtasks) != 3 || rep.Dependencies != 2 {
		t.Fatalf("%+v %v", rep, err)
	}
	if tr.count("P1") != 3 || tr.depAdds != 2 {
		t.Fatalf("subtarefas %d, dependências %d (a resposta perdida não pode duplicar)", tr.count("P1"), tr.depAdds)
	}
	first := rep.Subtasks
	// Repetir não cria nada de novo.
	creates, deps := tr.creates, tr.depAdds
	rep2, err := p.s.Publish(ctx, p.proj, p.feature, "P1", pub)
	if err != nil || rep2.Created != 0 || rep2.AlreadyThere != 3 || tr.creates != creates || tr.depAdds != deps || tr.count("P1") != 3 {
		t.Fatalf("%+v %v creates %d→%d deps %d→%d", rep2, err, creates, tr.creates, deps, tr.depAdds)
	}
	// As dependências ficaram na origem, ligando os IDs certos.
	if got := tr.items[first["api"]].Dependencies; len(got) != 1 || got[0] != first["schema"] {
		t.Fatalf("%v", got)
	}
	// Replanejar adicionando uma tarefa cria só ela; as identidades remotas se mantêm.
	next := basePlan()
	next.Tasks = append(next.Tasks, plTask("docs", "tela"))
	p.propose(t, next)
	rep3, err := p.s.Publish(ctx, p.proj, p.feature, "P1", pub)
	if err != nil || rep3.Created != 1 || rep3.AlreadyThere != 3 || tr.count("P1") != 4 || tr.depAdds != deps+1 {
		t.Fatalf("%+v %v", rep3, err)
	}
	for k, id := range first {
		if rep3.Subtasks[k] != id {
			t.Fatalf("a identidade remota de %s mudou: %s → %s", k, id, rep3.Subtasks[k])
		}
	}
	tasks, _ := p.s.Tasks(ctx, p.proj, p.feature)
	for _, tk := range tasks {
		if tk.RemoteID == "" || tk.RemoteID != rep3.Subtasks[tk.Key] {
			t.Fatalf("%+v", tk)
		}
	}
	// A descrição leva os critérios.
	if d := tr.items[first["schema"]].Description; !strings.Contains(d, "Os testes de schema passam") || !strings.Contains(d, "Risco inicial: medium") {
		t.Fatalf("%q", d)
	}
}

func TestPublishFailureIsReportedAndCompletedOnTheNextRun(t *testing.T) {
	p := newPlEnv(t, "pl-fail")
	ctx := context.Background()
	p.propose(t, basePlan())
	tr := newMem()
	tr.createFn = func(call int) (bool, error) { return false, errTrackerTimeout } // nada chega à origem
	pub := publisher(p.e, tr, nil)
	rep, err := p.s.Publish(ctx, p.proj, p.feature, "P1", pub)
	if err != nil || len(rep.Failed) == 0 || rep.Created != 0 {
		t.Fatalf("%+v %v", rep, err)
	}
	tr.createFn = nil
	rep, err = p.s.Publish(ctx, p.proj, p.feature, "P1", pub)
	if err != nil || len(rep.Failed) != 0 || tr.count("P1") != 3 {
		t.Fatalf("%+v %v n=%d", rep, err, tr.count("P1"))
	}
	if _, err := p.s.Publish(ctx, p.proj, "00000000-0000-0000-0000-000000000000", "P1", pub); !errors.Is(err, planning.ErrNoPlan) {
		t.Fatalf("%v", err)
	}
}

func TestPlanThroughTheAPI(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "pl-api")
	f, err := persistence.CreateFeature(context.Background(), e.db.Pool, p.ID, "PL-API", "Feature")
	if err != nil {
		t.Fatal(err)
	}
	other := mustProject(t, e.db.Pool, "pl-api-b")
	url := "/api/projects/" + p.ID + "/features/" + f.ID + "/plan"
	var view struct {
		Current  *planning.Stored      `json:"current"`
		Tasks    []planning.StoredTask `json:"tasks"`
		Versions []planning.Stored     `json:"versions"`
	}
	c.expect(c.get(url), 200).JSON(&view)
	if view.Current != nil || len(view.Versions) != 0 {
		t.Fatalf("%+v", view)
	}
	var out planning.Outcome
	c.expect(c.do("PUT", url, basePlan(), nil), 200).JSON(&out)
	if !out.Accepted || out.Stored.Version != 1 {
		t.Fatalf("%+v", out)
	}
	c.expect(c.do("PUT", url, basePlan(), nil), 200).JSON(&out)
	if !out.Unchanged {
		t.Fatalf("%+v", out)
	}
	bad := basePlan()
	bad.Tasks[0].DependsOn = []string{"tela"}
	resp := c.do("PUT", url, bad, nil)
	if resp.StatusCode != 422 || !strings.Contains(string(resp.Body), "plan_rejected") || !strings.Contains(string(resp.Body), "ciclo") {
		t.Fatalf("%d %s", resp.StatusCode, resp.Body)
	}
	c.expect(c.get(url), 200).JSON(&view)
	if view.Current.Version != 1 || len(view.Versions) != 2 || len(view.Tasks) != 3 {
		t.Fatalf("%+v", view)
	}
	// Feature de outro projeto.
	if r := c.get("/api/projects/" + other.ID + "/features/" + f.ID + "/plan"); r.StatusCode != 404 {
		t.Fatalf("%d", r.StatusCode)
	}
}
