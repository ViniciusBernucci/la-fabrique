package integration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checks"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type ckExec struct {
	out map[string]checks.Outcome
	err map[string]error
}

func (f ckExec) Run(_ context.Context, command string, _ time.Duration) (checks.Outcome, error) {
	if e := f.err[command]; e != nil {
		return checks.Outcome{}, e
	}
	return f.out[command], nil
}

type chEnv struct {
	e       *authEnv
	s       *checks.Store
	red     *security.Redactor
	proj    string
	feature string
}

func newChEnv(t *testing.T) *chEnv {
	t.Helper()
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "checks")
	f, err := persistence.CreateFeature(context.Background(), e.db.Pool, p.ID, "CH-1", "Feature de checks")
	if err != nil {
		t.Fatal(err)
	}
	red := security.NewRedactor()
	red.Register("segredo-dos-checks-xyz")
	return &chEnv{e: e, red: red, s: &checks.Store{DB: e.db.Pool, Redactor: red}, proj: p.ID, feature: f.ID}
}

var specs = []checks.Spec{{Name: "build", Command: "make build"}, {Name: "test", Command: "make test"}, {Name: "lint", Command: "make lint"}}

func okOut() checks.Outcome { return checks.Outcome{Output: "ok\n"} }

func failOut(msg string) checks.Outcome {
	return checks.Outcome{ExitCode: 1, Output: "FAIL TestCheckout (0.31s)\n    checkout_test.go:88: " + msg + "\n"}
}

func (c *chEnv) run(t *testing.T, role, rev string, out map[string]checks.Outcome, only ...string) []checks.Run {
	t.Helper()
	var sp []checks.Spec
	for _, s := range specs {
		if len(only) == 0 {
			sp = append(sp, s)
			continue
		}
		for _, o := range only {
			if o == s.Name {
				sp = append(sp, s)
			}
		}
	}
	rs, err := c.s.RunSet(context.Background(), c.proj, c.feature, role, rev, ckExec{out: out}, sp)
	if err != nil {
		t.Fatal(err)
	}
	return rs
}

func byName(ev checks.Evaluation) map[string]checks.Item {
	m := map[string]checks.Item{}
	for _, it := range ev.Items {
		m[it.Check] = it
	}
	return m
}

// T14-01 — "Regressão bloqueia."
func TestARegressionBlocksTheRevision(t *testing.T) {
	c := newChEnv(t)
	ctx := context.Background()
	c.run(t, checks.RoleBaseline, "base1", map[string]checks.Outcome{"make build": okOut(), "make test": okOut(), "make lint": okOut()})
	c.run(t, checks.RoleCandidate, "cand1", map[string]checks.Outcome{"make build": okOut(), "make test": failOut("esperava 5, veio 7"), "make lint": okOut()})
	ev, err := c.s.Evaluate(ctx, c.proj, c.feature, "cand1", "base1")
	if err != nil || !ev.Blocked || len(ev.Reasons) != 1 {
		t.Fatalf("%+v %v", ev, err)
	}
	m := byName(ev)
	if m["test"].Class != checks.ClassRegression || m["build"].Class != checks.ClassPass || m["lint"].Class != checks.ClassPass || m["test"].BaselineRunID == "" {
		t.Fatalf("%+v", m)
	}
	gate, _ := c.s.Gate(ctx, c.feature, "cand1")
	if !gate.Blocked || byName(gate)["test"].Class != checks.ClassRegression {
		t.Fatalf("o veredito persistido bloqueia: %+v", gate)
	}
	// Tudo passando libera.
	c.run(t, checks.RoleCandidate, "cand2", map[string]checks.Outcome{"make build": okOut(), "make test": okOut(), "make lint": okOut()})
	if ev, _ := c.s.Evaluate(ctx, c.proj, c.feature, "cand2", "base1"); ev.Blocked {
		t.Fatalf("%+v", ev)
	}
	if g, _ := c.s.Gate(ctx, c.feature, "cand2"); g.Blocked {
		t.Fatalf("%+v", g)
	}
}

// T14-01 — "Falha anterior comprovada é registrada."
func TestAProvenPreexistingFailureIsRecordedAndDoesNotBlock(t *testing.T) {
	c := newChEnv(t)
	ctx := context.Background()
	c.run(t, checks.RoleBaseline, "base1", map[string]checks.Outcome{"make build": okOut(), "make test": failOut("timeout de rede 30s"), "make lint": okOut()})
	// Mesma falha na candidata (números e tempos diferentes não mudam a assinatura).
	cand := map[string]checks.Outcome{"make build": okOut(), "make lint": okOut(),
		"make test": {ExitCode: 1, Output: "FAIL TestCheckout (9.99s)\n    checkout_test.go:91: timeout de rede 45s\n"}}
	c.run(t, checks.RoleCandidate, "cand1", cand)
	ev, err := c.s.Evaluate(ctx, c.proj, c.feature, "cand1", "base1")
	if err != nil || ev.Blocked || byName(ev)["test"].Class != checks.ClassPreexisting {
		t.Fatalf("%+v %v", ev, err)
	}
	kf, _ := c.s.KnownFailures(ctx, c.proj)
	base := byName(ev)["test"].BaselineRunID
	if len(kf) != 1 || kf[0].CheckName != "test" || kf[0].RunID != base || kf[0].Revision != "base1" {
		t.Fatalf("%+v", kf)
	}
	// Uma falha DIFERENTE no mesmo check, com o baseline falhando, é falha nova (regressão).
	c.run(t, checks.RoleCandidate, "cand2", map[string]checks.Outcome{"make build": okOut(), "make lint": okOut(),
		"make test": {ExitCode: 1, Output: "FAIL TestOther\n    other_test.go:5: panic: nil pointer\n"}}, "build", "test", "lint")
	ev, _ = c.s.Evaluate(ctx, c.proj, c.feature, "cand2", "base1")
	if !ev.Blocked || byName(ev)["test"].Class != checks.ClassRegression || !strings.Contains(byName(ev)["test"].Reason, "falha nova") {
		t.Fatalf("%+v", ev)
	}
	// Sem baseline utilizável, a mesma falha já comprovada antes continua sendo anterior.
	if err := c.s.RecordNotViable(ctx, c.proj, c.feature, "base-novo", "dependência indisponível no ambiente", specs); err != nil {
		t.Fatal(err)
	}
	c.run(t, checks.RoleCandidate, "cand3", cand)
	ev, _ = c.s.Evaluate(ctx, c.proj, c.feature, "cand3", "base-novo")
	if ev.Blocked || byName(ev)["test"].Class != checks.ClassPreexisting || !strings.Contains(byName(ev)["test"].Reason, "já comprovada") {
		t.Fatalf("%+v", ev)
	}
}

// T14-01 — "Origem desconhecida fica em investigação e não vira baseline por suposição."
func TestAnUnknownOriginStaysUnderInvestigationAndNeverBecomesBaseline(t *testing.T) {
	c := newChEnv(t)
	ctx := context.Background()
	// Só build e test foram executados no baseline; lint nunca rodou lá.
	c.run(t, checks.RoleBaseline, "base1", map[string]checks.Outcome{"make build": okOut(), "make test": okOut()}, "build", "test")
	c.run(t, checks.RoleCandidate, "cand1", map[string]checks.Outcome{"make build": okOut(), "make test": okOut(), "make lint": failOut("estilo quebrado")})
	ev, _ := c.s.Evaluate(ctx, c.proj, c.feature, "cand1", "base1")
	lint := byName(ev)["lint"]
	if !ev.Blocked || lint.Class != checks.ClassUnknown || !strings.Contains(lint.Reason, "investigação") {
		t.Fatalf("%+v", ev)
	}
	if kf, _ := c.s.KnownFailures(ctx, c.proj); len(kf) != 0 {
		t.Fatalf("a falha desconhecida não pode virar baseline: %+v", kf)
	}
	// Baseline inviável também deixa a falha em investigação, com o motivo.
	if err := c.s.RecordNotViable(ctx, c.proj, c.feature, "base2", "o baseline não compila neste ambiente", specs[2:]); err != nil {
		t.Fatal(err)
	}
	c.run(t, checks.RoleCandidate, "cand2", map[string]checks.Outcome{"make lint": failOut("estilo quebrado")}, "lint")
	ev, _ = c.s.Evaluate(ctx, c.proj, c.feature, "cand2", "base2")
	if !ev.Blocked || byName(ev)["lint"].Class != checks.ClassUnknown || !strings.Contains(byName(ev)["lint"].Reason, "não compila") {
		t.Fatalf("%+v", ev)
	}
	// Resolver como anterior exige PROVA de baseline com a mesma assinatura.
	cr := lint.CandidateRunID
	if _, err := c.s.Resolve(ctx, c.proj, cr, checks.ClassPreexisting, "já era assim", ""); !errors.Is(err, checks.ErrNoProof) {
		t.Fatalf("sem prova: %v", err)
	}
	passRun := c.run(t, checks.RoleBaseline, "base3", map[string]checks.Outcome{"make lint": okOut()}, "lint")[0]
	if _, err := c.s.Resolve(ctx, c.proj, cr, checks.ClassPreexisting, "já era assim", passRun.ID); !errors.Is(err, checks.ErrNoProof) {
		t.Fatalf("baseline que passa não prova: %v", err)
	}
	otherFail := c.run(t, checks.RoleBaseline, "base4", map[string]checks.Outcome{"make lint": failOut("outra coisa totalmente diferente")}, "lint")[0]
	if _, err := c.s.Resolve(ctx, c.proj, cr, checks.ClassPreexisting, "já era assim", otherFail.ID); !errors.Is(err, checks.ErrNoProof) {
		t.Fatalf("assinatura diferente não prova: %v", err)
	}
	if _, err := c.s.Resolve(ctx, c.proj, cr, checks.ClassPreexisting, "", otherFail.ID); err == nil {
		t.Fatal("exige justificativa")
	}
	proof := c.run(t, checks.RoleBaseline, "base5", map[string]checks.Outcome{"make lint": failOut("estilo quebrado")}, "lint")[0]
	got, err := c.s.Resolve(ctx, c.proj, cr, checks.ClassPreexisting, "falha já existia na base5", proof.ID)
	if err != nil || got.Classification != checks.ClassPreexisting || got.BaselineRunID != proof.ID {
		t.Fatalf("%+v %v", got, err)
	}
	if kf, _ := c.s.KnownFailures(ctx, c.proj); len(kf) != 1 || kf[0].RunID != proof.ID {
		t.Fatalf("%+v", kf)
	}
	gate, _ := c.s.Gate(ctx, c.feature, "cand1")
	if gate.Blocked {
		t.Fatalf("resolvida com prova, não bloqueia: %+v", gate)
	}
	// Reavaliar não desfaz a resolução humana.
	if ev, _ := c.s.Evaluate(ctx, c.proj, c.feature, "cand1", "base1"); ev.Blocked {
		t.Fatalf("%+v", ev)
	}
	// Resolver de novo (já não está em investigação) é recusado; confirmar defeito mantém o bloqueio.
	if _, err := c.s.Resolve(ctx, c.proj, cr, checks.ClassRegression, "x", ""); !errors.Is(err, checks.ErrNotResolvable) {
		t.Fatalf("%v", err)
	}
	// Com a falha antiga já comprovada, a mesma falha sem baseline passa a ser "anterior"; uma falha NOVA
	// sem baseline continua desconhecida, e confirmá-la como defeito mantém o bloqueio.
	if ev := mustEval(t, c, "cand2", "base2"); byName(ev)["lint"].Class != checks.ClassPreexisting || ev.Blocked {
		t.Fatalf("%+v", ev)
	}
	c.run(t, checks.RoleCandidate, "cand9", map[string]checks.Outcome{"make lint": failOut("regra nova violada no arquivo x")}, "lint")
	ev9 := mustEval(t, c, "cand9", "base2")
	if !ev9.Blocked || byName(ev9)["lint"].Class != checks.ClassUnknown {
		t.Fatalf("%+v", ev9)
	}
	r2, err := c.s.Resolve(ctx, c.proj, byName(ev9)["lint"].CandidateRunID, checks.ClassRegression, "defeito confirmado", "")
	if err != nil || r2.Classification != checks.ClassRegression {
		t.Fatalf("%+v %v", r2, err)
	}
	if g, _ := c.s.Gate(ctx, c.feature, "cand9"); !g.Blocked {
		t.Fatalf("regressão confirmada bloqueia: %+v", g)
	}
}

func mustEval(t *testing.T, c *chEnv, cand, base string) checks.Evaluation {
	t.Helper()
	ev, err := c.s.Evaluate(context.Background(), c.proj, c.feature, cand, base)
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

// A evidência vale só para a revisão testada; erro de infraestrutura não é falha do código.
func TestEvidenceBelongsToTheTestedRevisionAndInfrastructureErrorsAreInconclusive(t *testing.T) {
	c := newChEnv(t)
	ctx := context.Background()
	c.run(t, checks.RoleBaseline, "base1", map[string]checks.Outcome{"make build": okOut(), "make test": okOut(), "make lint": okOut()})
	c.run(t, checks.RoleCandidate, "cand1", map[string]checks.Outcome{"make build": okOut(), "make test": okOut(), "make lint": okOut()})
	if ev, _ := c.s.Evaluate(ctx, c.proj, c.feature, "cand1", "base1"); ev.Blocked {
		t.Fatalf("%+v", ev)
	}
	// Uma alteração depois dos checks muda a revisão: nada é herdado.
	g, _ := c.s.Gate(ctx, c.feature, "cand1-alterada")
	if !g.Blocked || !strings.Contains(strings.Join(g.Reasons, " "), "sem evidência") {
		t.Fatalf("%+v", g)
	}
	if ev, _ := c.s.Evaluate(ctx, c.proj, c.feature, "cand1-alterada", "base1"); !ev.Blocked {
		t.Fatalf("%+v", ev)
	}
	// Antes de avaliar, o veredito persistido bloqueia.
	c.run(t, checks.RoleCandidate, "cand2", map[string]checks.Outcome{"make build": okOut(), "make test": okOut(), "make lint": okOut()})
	if g, _ := c.s.Gate(ctx, c.feature, "cand2"); !g.Blocked || !strings.Contains(strings.Join(g.Reasons, " "), "ainda não foi avaliado") {
		t.Fatalf("%+v", g)
	}
	// Erro de infraestrutura e tempo limite: inconclusivo, nunca "regressão".
	rs, err := c.s.RunSet(ctx, c.proj, c.feature, checks.RoleCandidate, "cand3",
		ckExec{out: map[string]checks.Outcome{"make test": {TimedOut: true}, "make lint": okOut()}, err: map[string]error{"make build": errors.New("docker: sandbox indisponível")}}, specs)
	if err != nil || rs[0].Status != checks.StatusError || rs[1].Status != checks.StatusTimeout || rs[2].Status != checks.StatusPass {
		t.Fatalf("%+v %v", rs, err)
	}
	ev, _ := c.s.Evaluate(ctx, c.proj, c.feature, "cand3", "base1")
	m := byName(ev)
	if !ev.Blocked || m["build"].Class != checks.ClassInconclusive || m["test"].Class != checks.ClassInconclusive || m["lint"].Class != checks.ClassPass {
		t.Fatalf("%+v", ev)
	}
	// Corrigir o ambiente e rodar de novo na MESMA revisão substitui o resultado.
	c.run(t, checks.RoleCandidate, "cand3", map[string]checks.Outcome{"make build": okOut(), "make test": okOut(), "make lint": okOut()})
	if ev, _ := c.s.Evaluate(ctx, c.proj, c.feature, "cand3", "base1"); ev.Blocked {
		t.Fatalf("%+v", ev)
	}
}

func TestOutputsAreRedactedTruncatedAndInputsValidated(t *testing.T) {
	c := newChEnv(t)
	ctx := context.Background()
	big := strings.Repeat("linha de log sem importância\n", 2000) // ~58 KiB
	out := checks.Outcome{ExitCode: 1, Output: "token=segredo-dos-checks-xyz\n" + big + "FAIL final: erro no fim\n"}
	rs, err := c.s.RunSet(ctx, c.proj, c.feature, checks.RoleCandidate, "r1", ckExec{out: map[string]checks.Outcome{"make test": out}}, specs[1:2])
	if err != nil || len(rs) != 1 {
		t.Fatal(err)
	}
	r := rs[0]
	if strings.Contains(r.OutputExcerpt, "segredo-dos-checks-xyz") || !r.Truncated || len(r.OutputExcerpt) > 16<<10 || !strings.Contains(r.OutputExcerpt, "erro no fim") || r.OutputHash == "" || r.Signature == "" {
		t.Fatalf("truncated=%v len=%d", r.Truncated, len(r.OutputExcerpt))
	}
	for name, call := range map[string]func() error{
		"papel": func() error { _, e := c.s.RunSet(ctx, c.proj, c.feature, "x", "r", ckExec{}, specs); return e },
		"revisão": func() error {
			_, e := c.s.RunSet(ctx, c.proj, c.feature, checks.RoleBaseline, " ", ckExec{}, specs)
			return e
		},
		"sem checks": func() error {
			_, e := c.s.RunSet(ctx, c.proj, c.feature, checks.RoleBaseline, "r", ckExec{}, nil)
			return e
		},
		"repetido": func() error {
			_, e := c.s.RunSet(ctx, c.proj, c.feature, checks.RoleBaseline, "r", ckExec{}, []checks.Spec{specs[0], specs[0]})
			return e
		},
		"multi-linha": func() error {
			_, e := c.s.RunSet(ctx, c.proj, c.feature, checks.RoleBaseline, "r", ckExec{}, []checks.Spec{{Name: "a", Command: "x\ny"}})
			return e
		},
		"com segredo": func() error {
			_, e := c.s.RunSet(ctx, c.proj, c.feature, checks.RoleBaseline, "r", ckExec{}, []checks.Spec{{Name: "a", Command: "curl segredo-dos-checks-xyz"}})
			return e
		},
	} {
		if err := call(); !errors.Is(err, persistence.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Execução real no sandbox (Docker): código de saída e saída de erro viram evidência.
func TestChecksRunInTheRealSandbox(t *testing.T) {
	c := newChEnv(t)
	ctx := context.Background()
	sb := ckSandbox(t, c.proj, "checks-feat")
	rs, err := c.s.RunSet(ctx, c.proj, c.feature, checks.RoleCandidate, "real1", checks.SandboxExecer{Sandbox: sb}, []checks.Spec{
		{Name: "ok", Command: "echo tudo certo"},
		{Name: "falha", Command: "echo 'error: algo quebrou' >&2; exit 3"},
		{Name: "lento", Command: "sleep 30", Timeout: 1 * time.Second},
	})
	if err != nil || len(rs) != 3 {
		t.Fatalf("%+v %v", rs, err)
	}
	if rs[0].Status != checks.StatusPass || rs[1].Status != checks.StatusFail || rs[1].ExitCode == nil || *rs[1].ExitCode != 3 ||
		!strings.Contains(rs[1].OutputExcerpt, "algo quebrou") || rs[1].Signature == "" || rs[2].Status != checks.StatusTimeout {
		t.Fatalf("%+v", rs)
	}
}
