package qualitypolicy

import (
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/risk"
)

var avail = []string{"lint", "build", "test"}

func okEvidence(r Requirement, rev string) Evidence {
	e := Evidence{Tests: &TestRun{Revision: rev, Commands: []string{"make test"}, Passed: true}, ChecksPassed: map[string]string{}, Justifications: map[string]string{}}
	for _, c := range r.Checks {
		e.ChecksPassed[c] = rev
	}
	for _, c := range r.NotApplicable {
		e.Justifications[c] = "sem comando no projeto"
	}
	if r.BrowserReview {
		e.Browser = &Review{Revision: rev, Approved: true, Independent: true}
	}
	if r.AIReview {
		e.AI = &Review{Revision: rev, Approved: true, Independent: true}
	}
	return e
}

// CT17 — Front de baixo risco sempre exige navegador; Back baixo/médio roda checks sem review de IA;
// Back alto (após reclassificação pelo diff) exige review independente; todos têm testes reais.
func TestPolicyByTrackAndRisk(t *testing.T) {
	front := Decide(Input{Track: flow.Frontend, Suggested: risk.Low, Policy: risk.DefaultPolicy(), Files: []risk.File{{Path: "apps/web/a.tsx", Added: 3}}, Available: avail})
	if !front.BrowserReview || front.AIReview || !front.Tests || front.Assessment.Final != risk.Low {
		t.Fatalf("%+v", front)
	}
	back := Decide(Input{Track: flow.Backend, Suggested: risk.Medium, Policy: risk.DefaultPolicy(), Files: []risk.File{{Path: "internal/x/a.go", Added: 3}}, Available: avail})
	if back.BrowserReview || back.AIReview || len(back.Checks) == 0 {
		t.Fatalf("Back médio: checks, sem review de IA e sem navegador: %+v", back)
	}
	// Sugerido baixo, mas o diff toca migração: reclassifica para alto e exige review independente.
	high := Decide(Input{Track: flow.Backend, Suggested: risk.Low, Policy: risk.DefaultPolicy(), Files: []risk.File{{Path: "db/migrations/0001_x.sql", Added: 5}}, Available: avail})
	if !high.AIReview || high.Assessment.Final != risk.High || !high.Assessment.Raised {
		t.Fatalf("%+v", high)
	}
	for _, r := range []Requirement{front, back, high} {
		if p := Verify(r, okEvidence(r, "rev1"), "rev1"); len(p) != 0 {
			t.Fatalf("evidência completa deve passar: %v", p)
		}
		if p := Verify(r, Evidence{}, "rev1"); len(p) == 0 || p[0].Code != "tests_missing" {
			t.Fatalf("sem testes reais nenhum gate passa: %v", p)
		}
	}
}

func TestEvidenceIsBoundToTheRevisionAndRejectionsBlock(t *testing.T) {
	r := Decide(Input{Track: flow.Frontend, Suggested: risk.Low, Policy: risk.DefaultPolicy(), Available: avail})
	e := okEvidence(r, "rev1")
	if p := Verify(r, e, "rev2"); len(p) < 3 || !strings.Contains(p[0].Code, "stale") {
		t.Fatalf("a revisão mudou: toda evidência antiga é inválida: %v", p)
	}
	e.Browser = &Review{Revision: "rev1", Pending: true}
	if p := Verify(r, e, "rev1"); len(p) != 1 || p[0].Code != "browser_pending" {
		t.Fatalf("navegador indisponível deixa o gate pendente: %v", p)
	}
	e.Browser = &Review{Revision: "rev1", Approved: false, Independent: true}
	if p := Verify(r, e, "rev1"); len(p) != 1 || p[0].Code != "browser_rejected" {
		t.Fatalf("reprovação impede a publicação: %v", p)
	}
	e.Browser = &Review{Revision: "rev1", Approved: true, Independent: false}
	if p := Verify(r, e, "rev1"); len(p) != 1 || p[0].Code != "browser_not_independent" {
		t.Fatalf("%v", p)
	}
	e = okEvidence(r, "rev1")
	e.Tests.Passed, e.Tests.Errors = false, []string{"TestX"}
	if p := Verify(r, e, "rev1"); len(p) != 1 || p[0].Code != "tests_failed" {
		t.Fatalf("%v", p)
	}
	// Check sem comando no projeto exige justificativa.
	r2 := Decide(Input{Track: flow.Backend, Suggested: risk.Low, Policy: risk.DefaultPolicy(), Available: []string{"test"}})
	e2 := okEvidence(r2, "rev1")
	delete(e2.Justifications, r2.NotApplicable[0])
	if p := Verify(r2, e2, "rev1"); len(p) != 1 || p[0].Code != "check_unjustified" {
		t.Fatalf("%v", p)
	}
}

func TestCorrectionReappliesAffectedGates(t *testing.T) {
	r := Decide(Input{Track: flow.Backend, Suggested: risk.High, Policy: risk.DefaultPolicy(), Available: avail})
	got := Affected(r, []string{"internal/x/a.go"})
	joined := strings.Join(got, ",")
	if got[0] != "tests" || !strings.Contains(joined, "ai") || strings.Contains(joined, "browser") {
		t.Fatalf("%v", got)
	}
	if Affected(r, nil) != nil {
		t.Fatal("sem alteração nada é repetido")
	}
}
