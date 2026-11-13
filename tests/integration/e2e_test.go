package integration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checks"
	"github.com/ViniciusBernucci/le-fabrique/internal/e2e"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/risk"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// fakeBrowserEnv é um ambiente isolado em memória (sem Docker).
type fakeBrowserEnv struct {
	out      map[string]checks.Outcome
	files    map[string][]byte
	isolated bool
}

func (f fakeBrowserEnv) Run(_ context.Context, command string, _ time.Duration) (checks.Outcome, error) {
	return f.out[command], nil
}
func (f fakeBrowserEnv) Isolated() bool  { return f.isolated }
func (f fakeBrowserEnv) Browser() string { return "chromium-headless" }
func (f fakeBrowserEnv) Read(_ context.Context, p string) ([]byte, error) {
	b, ok := f.files[p]
	if !ok {
		return nil, errors.New("sem arquivo")
	}
	return b, nil
}

type e2eEnv struct {
	e       *authEnv
	s       *e2e.Store
	proj    string
	feature string
}

func newE2EEnv(t *testing.T, slug string) *e2eEnv {
	t.Helper()
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, slug)
	f, err := persistence.CreateFeature(context.Background(), e.db.Pool, p.ID, "E2E-1", "Feature E2E")
	if err != nil {
		t.Fatal(err)
	}
	red := security.NewRedactor()
	red.Register("segredo-do-e2e-xyz")
	cs := &checks.Store{DB: e.db.Pool, Redactor: red}
	return &e2eEnv{e: e, s: &e2e.Store{DB: e.db.Pool, Checks: cs, Redactor: red}, proj: p.ID, feature: f.ID}
}

func (x *e2eEnv) run(t *testing.T, rev, journey, cmd string, env fakeBrowserEnv, caps ...string) (e2e.Evidence, error) {
	t.Helper()
	return x.s.Run(context.Background(), e2e.Request{ProjectID: x.proj, FeatureID: x.feature, TaskKey: "t", Revision: rev, Journey: journey, Command: cmd,
		Fixtures: []string{"usuario-demo", "pedido-123"}, Captures: caps}, env, env)
}

func (x *e2eEnv) gate(t *testing.T, rev, level string, files []risk.File, journeys ...string) e2e.Result {
	t.Helper()
	r, err := x.s.Gate(context.Background(), x.proj, x.feature, "t", rev, level, files, e2e.DefaultConfig(), journeys)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

var okEnv = fakeBrowserEnv{isolated: true, out: map[string]checks.Outcome{"e2e checkout": {Output: "1 passed\n"}, "e2e login": {Output: "1 passed\n"}},
	files: map[string][]byte{"e2e/checkout.png": []byte("PNG-FAKE-BYTES")}}

var testedFiles = []risk.File{{Path: "internal/pedidos/pedidos.go", Added: 10}, {Path: "internal/pedidos/pedidos_test.go", Added: 20}}

// T14-04 — "Jornada de alto risco testável tem E2E real."
func TestAHighRiskTestableJourneyNeedsARealE2EOnTheTestedRevision(t *testing.T) {
	x := newE2EEnv(t, "e2e-high")
	// Sem E2E: alto risco com jornada configurada bloqueia.
	g := x.gate(t, "rev1", risk.High, testedFiles, "checkout", "login")
	if !g.Blocked || len(g.Journeys) != 2 || g.Journeys[0].Status != "missing" || !g.E2E.Required {
		t.Fatalf("%+v", g)
	}
	// Roda uma jornada: evidência com fixtures, navegador isolado e captura com hash.
	ev, err := x.run(t, "rev1", "checkout", "e2e checkout", okEnv, "e2e/checkout.png", "e2e/nao-existe.png")
	if err != nil || ev.Status != "pass" || !ev.Isolated || ev.Browser != "chromium-headless" || len(ev.Fixtures) != 2 || len(ev.Captures) != 1 || ev.CheckRunID == "" {
		t.Fatalf("%+v %v", ev, err)
	}
	data, c, err := x.s.Capture(context.Background(), ev.Captures[0].ID)
	if err != nil || string(data) != "PNG-FAKE-BYTES" || len(c.SHA256) != 64 || c.Size != 14 {
		t.Fatalf("%q %+v %v", data, c, err)
	}
	g = x.gate(t, "rev1", risk.High, testedFiles, "checkout", "login")
	if !g.Blocked || g.Journeys[0].Status != "passed" || g.Journeys[1].Status != "missing" {
		t.Fatalf("falta a outra jornada: %+v", g)
	}
	if _, err := x.run(t, "rev1", "login", "e2e login", okEnv); err != nil {
		t.Fatal(err)
	}
	if g := x.gate(t, "rev1", risk.High, testedFiles, "checkout", "login"); g.Blocked {
		t.Fatalf("%+v", g)
	}
	// Uma alteração depois do E2E muda a revisão: a evidência não vale.
	if g := x.gate(t, "rev2", risk.High, testedFiles, "checkout", "login"); !g.Blocked || g.Journeys[0].Status != "missing" {
		t.Fatalf("%+v", g)
	}
	// Jornada que falha bloqueia.
	failEnv := fakeBrowserEnv{isolated: true, out: map[string]checks.Outcome{"e2e checkout": {ExitCode: 1, Output: "FAIL checkout: botão não encontrado"}}}
	bad, err := x.run(t, "rev3", "checkout", "e2e checkout", failEnv)
	if err != nil || bad.Status != "fail" {
		t.Fatalf("%+v %v", bad, err)
	}
	g = x.gate(t, "rev3", risk.High, testedFiles, "checkout")
	if !g.Blocked || g.Journeys[0].Status != "failed" || !strings.Contains(strings.Join(g.Reasons, " "), "falhou") {
		t.Fatalf("%+v", g)
	}
	// Baixo risco não exige E2E; médio só com UI.
	if g := x.gate(t, "revx", risk.Low, testedFiles, "checkout"); g.Blocked || g.E2E.Required {
		t.Fatalf("%+v", g)
	}
	ui := []risk.File{{Path: "apps/web/src/pages/A.tsx"}, {Path: "apps/web/src/pages/A.test.tsx"}}
	if g := x.gate(t, "revx", risk.Medium, testedFiles, "checkout"); g.Blocked {
		t.Fatalf("%+v", g)
	}
	if g := x.gate(t, "revx", risk.Medium, ui, "checkout"); !g.Blocked || !g.E2E.Required {
		t.Fatalf("%+v", g)
	}
}

func TestE2ERefusesANonIsolatedEnvironmentAndValidatesInputs(t *testing.T) {
	x := newE2EEnv(t, "e2e-iso")
	notIso := fakeBrowserEnv{isolated: false, out: okEnv.out}
	if _, err := x.run(t, "r1", "checkout", "e2e checkout", notIso); !errors.Is(err, e2e.ErrNotIsolated) {
		t.Fatalf("%v", err)
	}
	var n int
	_ = x.e.db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM check_runs WHERE feature_id = $1`, x.feature).Scan(&n)
	if n != 0 {
		t.Fatalf("nada pode rodar fora do isolamento: %d", n)
	}
	for name, call := range map[string]func() error{
		"jornada":    func() error { _, e := x.run(t, "r1", "Checkout!", "e2e checkout", okEnv); return e },
		"captura":    func() error { _, e := x.run(t, "r1", "checkout", "e2e checkout", okEnv, "/etc/passwd"); return e },
		"captura ..": func() error { _, e := x.run(t, "r1", "checkout", "e2e checkout", okEnv, "../x.png"); return e },
		"fixture com segredo": func() error {
			_, e := x.s.Run(context.Background(), e2e.Request{ProjectID: x.proj, FeatureID: x.feature, TaskKey: "t", Revision: "r1", Journey: "checkout", Command: "e2e checkout",
				Fixtures: []string{"token segredo-do-e2e-xyz"}}, okEnv, okEnv)
			return e
		},
		"captura grande": func() error {
			big := fakeBrowserEnv{isolated: true, out: okEnv.out, files: map[string][]byte{"e2e/g.png": make([]byte, e2e.MaxCaptureBytes+1)}}
			_, e := x.run(t, "r1", "checkout", "e2e checkout", big, "e2e/g.png")
			return e
		},
	} {
		if err := call(); !errors.Is(err, persistence.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// T14-04 — "Caso inaplicável tem justificativa e evidência alternativa."
func TestAnInapplicableCaseNeedsAJustificationAndAnAlternativeEvidence(t *testing.T) {
	x := newE2EEnv(t, "e2e-waive")
	ctx := context.Background()
	// Sem nenhuma jornada de usuário testável (ex.: biblioteca interna): alto risco exige E2E, mas é inaplicável.
	g := x.gate(t, "rev1", risk.High, testedFiles)
	if !g.Blocked || !strings.Contains(strings.Join(g.Reasons, " "), "não tem jornada de usuário testável") {
		t.Fatalf("%+v", g)
	}
	in := e2e.WaiverInput{ProjectID: x.proj, FeatureID: x.feature, TaskKey: "t", Revision: "rev1", Subject: "e2e:sem-jornada",
		Reason: "É uma biblioteca interna sem jornada de usuário; o contrato é coberto por testes de integração.", AltKind: "check_run"}
	// Evidência alternativa inexistente, de outra revisão ou que falhou: recusada.
	in.AltRef = "00000000-0000-0000-0000-000000000000"
	if _, err := x.s.Waive(ctx, in); !errors.Is(err, e2e.ErrNoAlt) {
		t.Fatalf("inexistente: %v", err)
	}
	passRun, _ := x.s.Checks.RunSet(ctx, x.proj, x.feature, checks.RoleCandidate, "rev1", okEnv, []checks.Spec{{Name: "integracao", Command: "e2e login"}})
	failRun, _ := x.s.Checks.RunSet(ctx, x.proj, x.feature, checks.RoleCandidate, "rev1", fakeBrowserEnv{isolated: true, out: map[string]checks.Outcome{"falha": {ExitCode: 1}}}, []checks.Spec{{Name: "quebrado", Command: "falha"}})
	otherRev, _ := x.s.Checks.RunSet(ctx, x.proj, x.feature, checks.RoleCandidate, "rev-antiga", okEnv, []checks.Spec{{Name: "integracao", Command: "e2e login"}})
	for name, ref := range map[string]string{"falhou": failRun[0].ID, "outra revisão": otherRev[0].ID} {
		in.AltRef = ref
		if _, err := x.s.Waive(ctx, in); !errors.Is(err, e2e.ErrNoAlt) {
			t.Errorf("%s: %v", name, err)
		}
	}
	// Justificativa curta, assunto inválido, tipo desconhecido.
	short := in
	short.AltRef, short.Reason = passRun[0].ID, "não se aplica"
	if _, err := x.s.Waive(ctx, short); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("%v", err)
	}
	badSubject := in
	badSubject.AltRef, badSubject.Subject = passRun[0].ID, "qualquer"
	if _, err := x.s.Waive(ctx, badSubject); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("%v", err)
	}
	// Com justificativa e uma evidência alternativa REAL da mesma revisão: aceita e o gate libera.
	in.AltRef = passRun[0].ID
	w, err := x.s.Waive(ctx, in)
	if err != nil || w.ID == "" {
		t.Fatalf("%+v %v", w, err)
	}
	g = x.gate(t, "rev1", risk.High, testedFiles)
	if g.Blocked || g.Journeys[0].Status != "waived" || g.Journeys[0].WaiverID != w.ID {
		t.Fatalf("%+v", g)
	}
	// A justificativa é da revisão: em outra revisão volta a bloquear.
	if g := x.gate(t, "rev2", risk.High, testedFiles); !g.Blocked {
		t.Fatalf("%+v", g)
	}
	// Alternativas por revisão (texto) e por captura.
	rev := e2e.WaiverInput{ProjectID: x.proj, FeatureID: x.feature, TaskKey: "t", Revision: "rev2", Subject: "e2e:sem-jornada",
		Reason: "Sem jornada de usuário; a revisão manual cobre o comportamento.", AltKind: "review", AltRef: "revisão independente aprovou o comportamento (relatório 12)"}
	if _, err := x.s.Waive(ctx, rev); err != nil {
		t.Fatal(err)
	}
	if g := x.gate(t, "rev2", risk.High, testedFiles); g.Blocked {
		t.Fatalf("%+v", g)
	}
	rev.AltRef = "ok"
	if _, err := x.s.Waive(ctx, rev); !errors.Is(err, e2e.ErrNoAlt) {
		t.Fatalf("%v", err)
	}
	ev, _ := x.run(t, "rev3", "checkout", "e2e checkout", okEnv, "e2e/checkout.png")
	cap := e2e.WaiverInput{ProjectID: x.proj, FeatureID: x.feature, TaskKey: "t", Revision: "rev3", Subject: "e2e:checkout",
		Reason: "A jornada não roda no ambiente de teste; a captura manual da tela prova o resultado.", AltKind: "capture", AltRef: ev.Captures[0].ID}
	if _, err := x.s.Waive(ctx, cap); err != nil {
		t.Fatalf("%v", err)
	}
	cap.Revision, cap.AltRef = "rev-sem-captura", ev.Captures[0].ID
	if _, err := x.s.Waive(ctx, cap); !errors.Is(err, e2e.ErrNoAlt) {
		t.Fatalf("captura de outra revisão: %v", err)
	}
}

// T14-04 — "Orientar criação/atualização de testes".
func TestTestGuidanceBlocksHighRiskWithoutTestsUnlessWaived(t *testing.T) {
	x := newE2EEnv(t, "e2e-tests")
	ctx := context.Background()
	untested := []risk.File{{Path: "internal/pedidos/pedidos.go", Added: 40}}
	g := x.gate(t, "rev1", risk.High, untested, "checkout")
	if !g.Blocked || len(g.TestGaps) != 1 || !strings.Contains(strings.Join(g.Reasons, " "), "sem teste criado ou atualizado") {
		t.Fatalf("%+v", g)
	}
	if g := x.gate(t, "rev1", risk.Medium, untested, "checkout"); g.Blocked || len(g.TestGaps) != 1 {
		t.Fatalf("em risco médio a falta é só orientação: %+v", g)
	}
	run, _ := x.s.Checks.RunSet(ctx, x.proj, x.feature, checks.RoleCandidate, "rev1", okEnv, []checks.Spec{{Name: "integracao", Command: "e2e login"}})
	if _, err := x.s.Waive(ctx, e2e.WaiverInput{ProjectID: x.proj, FeatureID: x.feature, TaskKey: "t", Revision: "rev1", Subject: "tests",
		Reason: "Refatoração mecânica coberta pelos testes de integração existentes.", AltKind: "check_run", AltRef: run[0].ID}); err != nil {
		t.Fatal(err)
	}
	g = x.gate(t, "rev1", risk.High, untested)
	if !g.TestsWaived || len(g.TestGaps) != 1 || g.Blocked && !strings.Contains(strings.Join(g.Reasons, " "), "jornada") {
		t.Fatalf("%+v", g)
	}
}

// Execução real: navegador/jornada rodando num sandbox Docker isolado (sem rede), com captura.
func TestE2ERunsInTheRealIsolatedSandbox(t *testing.T) {
	x := newE2EEnv(t, "e2e-real")
	sb := ckSandbox(t, x.proj, "e2e-feat")
	env := e2e.SandboxEnv{Sandbox: sb, BrowserName: "simulado (alpine, sem navegador)"}
	if !env.Isolated() {
		t.Fatal("o perfil base não tem rede")
	}
	ev, err := x.s.Run(context.Background(), e2e.Request{ProjectID: x.proj, FeatureID: x.feature, TaskKey: "t", Revision: "real1", Journey: "checkout",
		Command: `mkdir -p e2e && echo "jornada ok" > e2e/saida.txt && echo passou`, Fixtures: []string{"usuario-demo"}, Captures: []string{"e2e/saida.txt"}}, env, env)
	if err != nil || ev.Status != "pass" || len(ev.Captures) != 1 || ev.Captures[0].Size != len("jornada ok\n") {
		t.Fatalf("%+v %v", ev, err)
	}
	data, _, err := x.s.Capture(context.Background(), ev.Captures[0].ID)
	if err != nil || string(data) != "jornada ok\n" {
		t.Fatalf("%q %v", data, err)
	}
}
