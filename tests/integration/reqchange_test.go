package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/reqchange"
	"github.com/ViniciusBernucci/le-fabrique/internal/resume"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type rcEnv struct {
	e       *authEnv
	h       *reqchange.Handler
	plans   *planning.Store
	proj    string
	feature string
	admin   string
}

func newRcEnv(t *testing.T, slug string) *rcEnv {
	t.Helper()
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, slug)
	f, err := persistence.CreateFeature(context.Background(), e.db.Pool, p.ID, "ABC-1", "Feature")
	if err != nil {
		t.Fatal(err)
	}
	var admin string
	if err := e.db.Pool.QueryRow(context.Background(), `SELECT id FROM admins LIMIT 1`).Scan(&admin); err != nil {
		t.Fatal(err)
	}
	return &rcEnv{e: e, h: &reqchange.Handler{DB: e.db.Pool, Redactor: security.NewRedactor()}, plans: &planning.Store{DB: e.db.Pool}, proj: p.ID, feature: f.ID, admin: admin}
}

func rcReq() resume.Requirement {
	return resume.Requirement{Title: "Exportar CSV", Description: "Relatórios mensais", Status: "To Do", Priority: "High", Labels: []string{"lf-ready"}}
}

func (r *rcEnv) handle(t *testing.T, q resume.Requirement) reqchange.Outcome {
	t.Helper()
	o, err := r.h.Handle(context.Background(), r.proj, r.feature, q)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func (r *rcEnv) startWork(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if _, err := r.plans.Propose(ctx, r.proj, r.feature, basePlan()); err != nil {
		t.Fatal(err)
	}
	// Existe execução (a feature já foi executada) e a primeira tarefa começou.
	if _, err := persistence.CreateExecution(ctx, r.e.db.Pool, r.proj, r.feature); err != nil {
		t.Fatal(err)
	}
	if err := r.plans.SetTaskState(ctx, r.proj, r.feature, "schema", planning.StateRunning); err != nil {
		t.Fatal(err)
	}
}

// T06-05 — "Mudança compatível mantém histórico."
func TestACompatibleChangeKeepsTheHistory(t *testing.T) {
	r := newRcEnv(t, "rc-compat")
	ctx := context.Background()
	first := r.handle(t, rcReq())
	if first.Revision.Impact != reqchange.ImpactInitial || first.Revision.State != reqchange.StateApplied || first.Revision.Seq != 1 {
		t.Fatalf("%+v", first)
	}
	if again := r.handle(t, rcReq()); !again.Unchanged || again.Revision.ID != first.Revision.ID {
		t.Fatalf("o mesmo conteúdo não gera revisão: %+v", again)
	}
	chg := rcReq()
	chg.Priority, chg.Labels = "Low", []string{"lf-ready", "urgente"}
	o := r.handle(t, chg)
	if o.Revision.Impact != reqchange.ImpactMetadata || o.Revision.Action != reqchange.ActionUpdate || o.Revision.State != reqchange.StateApplied || len(o.Revision.Diff) != 2 {
		t.Fatalf("%+v", o.Revision)
	}
	hist, _ := r.h.History(ctx, r.proj, r.feature)
	if len(hist) != 2 || hist[1].State != reqchange.StateSuperseded || hist[1].Content.Priority != "High" || hist[0].Content.Priority != "Low" {
		t.Fatalf("o histórico preserva a revisão antiga: %+v", hist)
	}
	cur, _ := r.h.Current(ctx, r.proj, r.feature)
	var rev string
	_ = r.e.db.Pool.QueryRow(ctx, `SELECT external_revision FROM features WHERE id = $1`, r.feature).Scan(&rev)
	if cur.Seq != 2 || rev != cur.Hash {
		t.Fatalf("%+v %s", cur, rev)
	}
}

func TestMetadataChangesWaitForASafeMilestone(t *testing.T) {
	r := newRcEnv(t, "rc-defer")
	ctx := context.Background()
	r.handle(t, rcReq())
	r.startWork(t) // tarefa "schema" em execução
	a := rcReq()
	a.Priority = "Low"
	b := rcReq()
	b.Priority = "Highest"
	o1 := r.handle(t, a)
	o2 := r.handle(t, b)
	if o1.Revision.State != reqchange.StateDeferred || o1.Revision.Action != reqchange.ActionDefer || o2.Revision.State != reqchange.StateDeferred {
		t.Fatalf("%+v %+v", o1.Revision, o2.Revision)
	}
	if cur, _ := r.h.Current(ctx, r.proj, r.feature); cur.Seq != 1 {
		t.Fatalf("a vigente não muda durante a execução: %+v", cur)
	}
	if got, err := r.h.AtMilestone(ctx, r.proj, r.feature); err != nil || got != nil {
		t.Fatalf("ainda há tarefa em execução: %+v %v", got, err)
	}
	if err := r.plans.SetTaskState(ctx, r.proj, r.feature, "schema", planning.StateDone); err != nil {
		t.Fatal(err)
	}
	got, err := r.h.AtMilestone(ctx, r.proj, r.feature)
	if err != nil || got == nil || got.Seq != 3 || got.Content.Priority != "Highest" || got.State != reqchange.StateApplied {
		t.Fatalf("a adiada mais recente vence: %+v %v", got, err)
	}
	hist, _ := r.h.History(ctx, r.proj, r.feature)
	states := map[int]string{}
	for _, h := range hist {
		states[h.Seq] = h.State
	}
	if len(hist) != 3 || states[1] != "superseded" || states[2] != "superseded" || states[3] != "applied" {
		t.Fatalf("%v", states)
	}
}

func TestContentChangeBeforeAnyWorkAsksForANewPlan(t *testing.T) {
	r := newRcEnv(t, "rc-replan")
	ctx := context.Background()
	r.handle(t, rcReq())
	if _, err := r.plans.Propose(ctx, r.proj, r.feature, basePlan()); err != nil { // plano pronto, nada iniciado
		t.Fatal(err)
	}
	chg := rcReq()
	chg.Description = "Relatórios mensais e semanais"
	o := r.handle(t, chg)
	if o.Revision.Impact != reqchange.ImpactContent || o.Revision.Action != reqchange.ActionReplan || o.Revision.State != reqchange.StateNeedsReplan || o.Decision != nil {
		t.Fatalf("%+v", o)
	}
	if cur, _ := r.h.Current(ctx, r.proj, r.feature); cur.Seq != 1 {
		t.Fatalf("a vigente fica até o novo plano: %+v", cur)
	}
	// Nova leitura antes do replanejamento substitui a pendente.
	chg2 := chg
	chg2.Title = "Exportar CSV e XML"
	o2 := r.handle(t, chg2)
	if o2.Revision.State != reqchange.StateNeedsReplan || o2.Revision.Seq != 3 {
		t.Fatalf("%+v", o2.Revision)
	}
	got, err := r.h.CompleteReplan(ctx, r.proj, r.feature)
	if err != nil || got.Seq != 3 || got.State != reqchange.StateApplied {
		t.Fatalf("%+v %v", got, err)
	}
	hist, _ := r.h.History(ctx, r.proj, r.feature)
	if len(hist) != 3 {
		t.Fatalf("o histórico não perde revisões: %d", len(hist))
	}
	if _, err := r.h.CompleteReplan(ctx, r.proj, r.feature); !errors.Is(err, reqchange.ErrNoRevision) {
		t.Fatalf("%v", err)
	}
}

// T06-05 — "Conflito preserva alterações e cria decisão."
func TestAConflictWithWorkInProgressPreservesEverythingAndOpensADecision(t *testing.T) {
	r := newRcEnv(t, "rc-conflict")
	ctx := context.Background()
	r.handle(t, rcReq())
	r.startWork(t)
	planBefore, _ := r.plans.Current(ctx, r.proj, r.feature)
	chg := rcReq()
	chg.Description = "Relatórios em outro formato (mudou)"
	chg.Dependencies = []string{"ABC-9"}
	o := r.handle(t, chg)
	if o.Revision.State != reqchange.StateConflict || o.Revision.Action != reqchange.ActionDecision || o.Decision == nil ||
		o.Decision.Kind != decisions.KindConflict || o.Decision.State != decisions.StatePending || len(o.Decision.Evidence) != 2 {
		t.Fatalf("%+v", o)
	}
	// Plano e requisito vigente intactos; as tarefas seguem como estavam.
	planAfter, _ := r.plans.Current(ctx, r.proj, r.feature)
	cur, _ := r.h.Current(ctx, r.proj, r.feature)
	tasks, _ := r.plans.Tasks(ctx, r.proj, r.feature)
	if planAfter.Hash != planBefore.Hash || cur.Seq != 1 || cur.Content.Description != "Relatórios mensais" {
		t.Fatalf("o conflito não pode alterar nada: %+v %+v", planAfter.Version, cur)
	}
	running := 0
	for _, tk := range tasks {
		if tk.State == planning.StateRunning {
			running++
		}
	}
	if running != 1 {
		t.Fatalf("%+v", tasks)
	}
	// Repetir a leitura não abre outra decisão.
	if o2 := r.handle(t, chg); !o2.Unchanged || o2.Revision.ID != o.Revision.ID {
		t.Fatalf("%+v", o2)
	}
	// Antes de resolver, não se conclui.
	if _, err := r.h.Settle(ctx, r.proj, o.Revision.ID); !errors.Is(err, reqchange.ErrNotResolved) {
		t.Fatalf("%v", err)
	}
	// Manter o plano: a mudança fica rejeitada no histórico e o vigente permanece.
	if _, err := decisions.Resolve(ctx, r.e.db.Pool, r.admin, o.Decision.ID, "keep_work", "seguir"); err != nil {
		t.Fatal(err)
	}
	got, err := r.h.Settle(ctx, r.proj, o.Revision.ID)
	if err != nil || got.State != reqchange.StateRejected {
		t.Fatalf("%+v %v", got, err)
	}
	if cur, _ := r.h.Current(ctx, r.proj, r.feature); cur.Seq != 1 {
		t.Fatalf("%+v", cur)
	}
	if _, err := r.h.Settle(ctx, r.proj, o.Revision.ID); !errors.Is(err, reqchange.ErrNoRevision) {
		t.Fatalf("%v", err)
	}
	// O mesmo conteúdo, já decidido, não reabre a pergunta.
	if o3 := r.handle(t, chg); !o3.Unchanged {
		t.Fatalf("%+v", o3)
	}
}

func TestAConflictResolvedWithReplanBecomesPendingReplan(t *testing.T) {
	r := newRcEnv(t, "rc-conflict-replan")
	ctx := context.Background()
	r.handle(t, rcReq())
	r.startWork(t)
	chg := rcReq()
	chg.Title = "Outro título"
	o := r.handle(t, chg)
	if _, err := decisions.Resolve(ctx, r.e.db.Pool, r.admin, o.Decision.ID, "replan", "refazer"); err != nil {
		t.Fatal(err)
	}
	got, err := r.h.Settle(ctx, r.proj, o.Revision.ID)
	if err != nil || got.State != reqchange.StateNeedsReplan {
		t.Fatalf("%+v %v", got, err)
	}
	done, err := r.h.CompleteReplan(ctx, r.proj, r.feature)
	if err != nil || done.Content.Title != "Outro título" || done.State != reqchange.StateApplied {
		t.Fatalf("%+v %v", done, err)
	}
	hist, _ := r.h.History(ctx, r.proj, r.feature)
	if len(hist) != 2 || hist[1].State != reqchange.StateSuperseded {
		t.Fatalf("%+v", hist)
	}
}

// --- troca de plataforma --------------------------------------------------------

type newPrEnvLite struct {
	e     *authEnv
	c     *client
	p     projects.Project
	gh    security.Connection
	jira  security.Connection
	click security.Connection
	sw    *reqchange.Switcher
	feats map[string]string // external_id → id
}

func newSwEnv(t *testing.T, slug string) *newPrEnvLite {
	t.Helper()
	e, c, _ := newProjectEnv(t)
	p := createProject(t, c, slug)
	gh := conn(t, c, "github", "gh-"+slug)
	jira := conn(t, c, "jira", "jira-"+slug)
	click := conn(t, c, "clickup", "click-"+slug)
	p = completeConfig(t, c, p, gh, jira)
	env := &newPrEnvLite{e: e, c: c, p: p, gh: gh, jira: jira, click: click, sw: &reqchange.Switcher{DB: e.db.Pool}, feats: map[string]string{}}
	for _, ext := range []string{"ABC-1", "ABC-2"} {
		f, err := persistence.CreateFeature(context.Background(), e.db.Pool, p.ID, ext, "Feature "+ext)
		if err != nil {
			t.Fatal(err)
		}
		env.feats[ext] = f.ID
	}
	return env
}

func clickSource(connID string) projects.Source {
	return projects.Source{Kind: "clickup", Container: "9001", ConnectionID: &connID, EligibilityMarker: "lf-ready", EligibleStatuses: []string{"to do"}, PriorityMap: map[string]int{"high": 1}, Access: "read"}
}

func newOrigin(ids ...string) *srcTracker {
	t := &srcTracker{}
	for _, id := range ids {
		t.items = append(t.items, issue(id, "To Do", "todo", "High"))
	}
	return t
}

// T06-05 — "Troca mantém vínculos históricos e uma origem ativa."
func TestSwitchingThePlatformKeepsHistoricalLinksAndASingleActiveOrigin(t *testing.T) {
	e := newSwEnv(t, "rc-switch")
	ctx := context.Background()
	tr := newOrigin("901", "902")
	admin := ""
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT id FROM admins LIMIT 1`).Scan(&admin)

	// Projeto ativo: não troca.
	suspendedBefore := e.p
	e.c.expect(e.c.post("/api/projects/"+e.p.ID+"/activate", map[string]any{"expected_version": suspendedBefore.Version}), 200)
	rec, err := e.sw.Plan(ctx, e.p.ID, clickSource(e.click.ID), reqchange.Mapping{"ABC-1": "901", "ABC-2": "902"}, tr)
	if err != nil || rec.State != reqchange.SwitchRejected || !strings.Contains(strings.Join(rec.Problems, " "), "ativo") {
		t.Fatalf("%+v %v", rec, err)
	}
	var cur projects.Project
	e.c.expect(e.c.get("/api/projects/"+e.p.ID), 200).JSON(&cur)
	e.c.expect(e.c.post("/api/projects/"+e.p.ID+"/suspend", map[string]any{"expected_version": cur.Version}), 200)

	bad := map[string]struct {
		m    reqchange.Mapping
		want string
	}{
		"feature sem mapa":     {reqchange.Mapping{"ABC-1": "901"}, "não tem identidade"},
		"identidade repetida":  {reqchange.Mapping{"ABC-1": "901", "ABC-2": "901"}, "uma identidade remota por item"},
		"feature desconhecida": {reqchange.Mapping{"ABC-1": "901", "ABC-2": "902", "ZZZ-9": "903"}, "não é uma feature"},
		"item inexistente":     {reqchange.Mapping{"ABC-1": "901", "ABC-2": "999"}, "não existe"},
		"colide com a atual":   {reqchange.Mapping{"ABC-1": "ABC-2", "ABC-2": "902"}, "colide"},
	}
	for name, c := range bad {
		r, err := e.sw.Plan(ctx, e.p.ID, clickSource(e.click.ID), c.m, tr)
		if err != nil || r.State != reqchange.SwitchRejected || !strings.Contains(strings.Join(r.Problems, " "), c.want) {
			t.Errorf("%s: %+v %v", name, r, err)
		}
		if _, err := e.sw.Activate(ctx, admin, e.p.ID, r.ID); !errors.Is(err, reqchange.ErrSwitchState) {
			t.Errorf("%s: rejeitada não ativa: %v", name, err)
		}
	}
	// Mesma origem.
	same := e.p.Source
	if r, _ := e.sw.Plan(ctx, e.p.ID, same, reqchange.Mapping{}, tr); r.State != reqchange.SwitchRejected {
		t.Fatalf("%+v", r)
	}
	// Plataforma sem acesso (item não legível) também recusa; nada mudou no projeto até aqui.
	e.c.expect(e.c.get("/api/projects/"+e.p.ID), 200).JSON(&cur)
	if cur.Source.Kind != "jira" || cur.Source.Container != "ABC" {
		t.Fatalf("a validação não altera o projeto: %+v", cur.Source)
	}

	ok, err := e.sw.Plan(ctx, e.p.ID, clickSource(e.click.ID), reqchange.Mapping{"ABC-1": "901", "ABC-2": "902"}, tr)
	if err != nil || ok.State != reqchange.SwitchValidated || len(ok.Problems) != 0 {
		t.Fatalf("%+v %v", ok, err)
	}
	// Dados que valem para a origem antiga: itens sincronizados e um vínculo de histórico.
	if _, err := e.e.db.Pool.Exec(ctx, `INSERT INTO source_items (project_id, remote_id, content_hash, first_seen_at, last_seen_at, last_changed_at)
		VALUES ($1, 'ABC-1', repeat('a', 64), now(), now(), now())`, e.p.ID); err != nil {
		t.Fatal(err)
	}
	updated, err := e.sw.Activate(ctx, admin, e.p.ID, ok.ID)
	if err != nil || updated.Source.Kind != "clickup" || updated.Source.Container != "9001" || updated.Source.ConnectionID == nil || *updated.Source.ConnectionID != e.click.ID {
		t.Fatalf("%+v %v", updated.Source, err)
	}
	// Features na nova identidade; a antiga fica no histórico.
	for old, nu := range map[string]string{"ABC-1": "901", "ABC-2": "902"} {
		var ext string
		_ = e.e.db.Pool.QueryRow(ctx, `SELECT external_id FROM features WHERE id = $1`, e.feats[old]).Scan(&ext)
		hist, _ := e.sw.IdentityHistory(ctx, e.p.ID, e.feats[old])
		if ext != nu || len(hist) != 1 || hist[0].RemoteID != old || hist[0].SupersededBy != nu || hist[0].SourceKind != "jira" || hist[0].SwitchID != ok.ID {
			t.Fatalf("%s → %s: %q %+v", old, nu, ext, hist)
		}
	}
	var items int
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM source_items WHERE project_id = $1`, e.p.ID).Scan(&items)
	if items != 0 {
		t.Fatal("o estado da origem antiga não vale para a nova")
	}
	// A troca não se repete e a trilha registra a mudança de origem.
	if _, err := e.sw.Activate(ctx, admin, e.p.ID, ok.ID); !errors.Is(err, reqchange.ErrSwitchState) {
		t.Fatalf("%v", err)
	}
	changes, _ := projects.Changes(ctx, e.e.db.Pool, e.p.ID, 20)
	found := false
	for _, c := range changes {
		found = found || c.Action == projects.ChangeSourceChanged
	}
	if !found {
		t.Fatalf("%+v", changes)
	}
	// Uma origem só: o projeto passa a exigir nova verificação antes de reativar.
	rep := validation(t, e.c, e.p.ID)
	pend := pending(rep)
	if _, ok := pend["source.remote"]; !ok {
		t.Fatalf("a nova origem só ativa depois de verificada: %+v", pend)
	}
	hist, _ := e.sw.Switches(ctx, e.p.ID)
	states := map[string]int{}
	for _, h := range hist {
		states[h.State]++
	}
	if states["activated"] != 1 || states["rejected"] < 6 {
		t.Fatalf("%v", states)
	}
}

func TestAStaleValidatedSwitchIsRefused(t *testing.T) {
	e := newSwEnv(t, "rc-switch-stale")
	ctx := context.Background()
	tr := newOrigin("901", "902")
	var cur projects.Project
	e.c.expect(e.c.get("/api/projects/"+e.p.ID), 200).JSON(&cur)
	ok, _ := e.sw.Plan(ctx, e.p.ID, clickSource(e.click.ID), reqchange.Mapping{"ABC-1": "901", "ABC-2": "902"}, tr)
	if ok.State != reqchange.SwitchValidated {
		t.Fatalf("%+v", ok)
	}
	// Depois da validação surge uma feature nova: a troca precisa ser validada de novo.
	if _, err := persistence.CreateFeature(ctx, e.e.db.Pool, e.p.ID, "ABC-3", "Nova"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.sw.Activate(ctx, "", e.p.ID, ok.ID); !errors.Is(err, reqchange.ErrSwitchStale) {
		t.Fatalf("%v", err)
	}
	var kind string
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT source_kind FROM projects WHERE id = $1`, e.p.ID).Scan(&kind)
	if kind != "jira" {
		t.Fatalf("a origem não pode ter mudado: %s", kind)
	}
}
