package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/risk"
)

type rkEnv struct {
	e       *authEnv
	s       *risk.Store
	proj    string
	feature string
}

func newRkEnv(t *testing.T, slug string) *rkEnv {
	t.Helper()
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, slug)
	f, err := persistence.CreateFeature(context.Background(), e.db.Pool, p.ID, "RK-1", "Feature de risco")
	if err != nil {
		t.Fatal(err)
	}
	return &rkEnv{e: e, s: &risk.Store{DB: e.db.Pool}, proj: p.ID, feature: f.ID}
}

// T14-02 — "Fixture de arquivo crítico eleva nível apesar de sugestão baixa" e reavaliação do diff.
func TestAssessmentsAreJustifiedAndReevaluatedFromTheRealDiff(t *testing.T) {
	r := newRkEnv(t, "risk-flow")
	ctx := context.Background()
	task := plTask("schema")
	task.Risk = "low"
	task.Paths = []string{"db/migrations/0042_pedidos.sql", "internal/pedidos/pedidos.go"}
	suggested, files := risk.FromTask(task)

	// Planejamento: a IA disse baixo, mas o caminho previsto é uma migration.
	plan, err := r.s.Assess(ctx, r.proj, r.feature, "schema", risk.PhasePlanning, "", suggested, files)
	if err != nil || plan.Final != risk.High || !plan.Raised || plan.Suggested != risk.Low {
		t.Fatalf("%+v %v", plan, err)
	}
	// Depois da implementação o diff real não tocou a migration: o nível é reavaliado (sem o piso).
	diff, err := r.s.Assess(ctx, r.proj, r.feature, "schema", risk.PhaseDiff, "abc123", suggested,
		[]risk.File{{Path: "internal/pedidos/pedidos.go", Added: 30, Removed: 4}})
	if err != nil || diff.Final != risk.Low || diff.Raised {
		t.Fatalf("%+v %v", diff, err)
	}
	// Uma correção passa a tocar um arquivo crítico: sobe de novo, com a razão registrada.
	fix, err := r.s.Assess(ctx, r.proj, r.feature, "schema", risk.PhaseFix, "def456", suggested,
		[]risk.File{{Path: "internal/pedidos/pedidos.go", Added: 5}, {Path: ".github/workflows/ci.yml", Added: 2}})
	if err != nil || fix.Final != risk.High || !fix.Raised {
		t.Fatalf("%+v %v", fix, err)
	}
	found := false
	for _, rs := range fix.Reasons {
		found = found || (rs.Source == "rule" && rs.Level == risk.High)
	}
	if !found {
		t.Fatalf("a justificativa deve citar a regra: %+v", fix.Reasons)
	}
	hist, _ := r.s.History(ctx, r.proj, r.feature, "schema")
	if len(hist) != 3 || hist[0].Phase != risk.PhaseFix || hist[1].Phase != risk.PhaseDiff || hist[2].Phase != risk.PhasePlanning || hist[1].Revision != "abc123" {
		t.Fatalf("%+v", hist)
	}
	if latest, _ := r.s.Latest(ctx, r.proj, r.feature, "schema"); latest.ID != fix.ID || latest.Final != risk.High {
		t.Fatalf("%+v", latest)
	}
	if _, err := r.s.Latest(ctx, r.proj, r.feature, "outra"); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("%v", err)
	}
	// Cada nível escolhe verificações previstas (aqui, com os checks do projeto).
	avail := []string{"lint", "build", "test", "secret_scan"}
	if p := risk.Plan(diff.Final, avail); len(p.Required) != 3 || p.Review != "none" {
		t.Fatalf("%+v", p)
	}
	if p := risk.Plan(fix.Final, avail); len(p.Required) != 4 || len(p.NotApplicable) != 2 || p.Review != "independent_security" || p.E2E != "required" {
		t.Fatalf("%+v", p)
	}
}

func TestEachProjectHasItsOwnRiskPolicy(t *testing.T) {
	a := newRkEnv(t, "risk-a")
	b := newRkEnv(t, "risk-b")
	ctx := context.Background()
	files := []risk.File{{Path: "billing/invoice.go", Added: 10}}
	def, err := a.s.Policy(ctx, a.proj)
	if err != nil || len(def.Rules) == 0 || def.MediumLines != 200 {
		t.Fatalf("a política padrão vale sem configuração: %+v %v", def, err)
	}
	if _, err := a.s.SetPolicy(ctx, a.proj, risk.Policy{Rules: []risk.Rule{{Pattern: "billing/**", Floor: risk.High, Reason: "cobrança"}}}); err != nil {
		t.Fatal(err)
	}
	ra, _ := a.s.Assess(ctx, a.proj, a.feature, "t", risk.PhaseDiff, "r1", risk.Low, files)
	rb, _ := b.s.Assess(ctx, b.proj, b.feature, "t", risk.PhaseDiff, "r1", risk.Low, files)
	if ra.Final != risk.High || rb.Final != risk.Low {
		t.Fatalf("a política de A não vale para B: %s × %s", ra.Final, rb.Final)
	}
	got, _ := a.s.Policy(ctx, a.proj)
	if len(got.Rules) != 1 || got.Rules[0].Pattern != "billing/**" {
		t.Fatalf("%+v", got)
	}
	// Entradas inválidas.
	for name, err := range map[string]error{
		"fase": func() error { _, e := a.s.Assess(ctx, a.proj, a.feature, "t", "x", "", risk.Low, nil); return e }(),
		"tarefa": func() error {
			_, e := a.s.Assess(ctx, a.proj, a.feature, " ", risk.PhaseDiff, "", risk.Low, nil)
			return e
		}(),
		"caminho": func() error {
			_, e := a.s.Assess(ctx, a.proj, a.feature, "t", risk.PhaseDiff, "", risk.Low, []risk.File{{Path: "/etc/passwd"}})
			return e
		}(),
		"política": func() error {
			_, e := a.s.SetPolicy(ctx, a.proj, risk.Policy{Rules: []risk.Rule{{Pattern: "x", Floor: "low"}}})
			return e
		}(),
		"tamanho": func() error {
			_, e := a.s.Assess(ctx, a.proj, a.feature, "t", risk.PhaseDiff, "", risk.Low, []risk.File{{Path: "a", Added: -1}})
			return e
		}(),
	} {
		if !errors.Is(err, persistence.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	// Sugestão ausente: falha fechada, registrada como alta.
	rc, err := b.s.Assess(ctx, b.proj, b.feature, "u", risk.PhasePlanning, "", "", nil)
	if err != nil || rc.Final != risk.High || rc.Suggested != risk.High {
		t.Fatalf("%+v %v", rc, err)
	}
}
