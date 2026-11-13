package flow

import (
	"errors"
	"testing"
)

func TestCatalogIDsAreStableAndUnique(t *testing.T) {
	ids, codes := map[string]bool{}, map[string]bool{}
	for _, s := range Catalog {
		if ids[s.ID] || codes[s.Code] {
			t.Fatalf("duplicado: %+v", s)
		}
		ids[s.ID], codes[s.Code] = true, true
	}
	for _, tr := range Transitions {
		if !codes[tr.From] || !codes[tr.To] {
			t.Fatalf("transição fora do catálogo: %+v", tr)
		}
	}
	if len(Catalog) != 9 {
		t.Fatalf("catálogo: %d", len(Catalog))
	}
}

func TestLegacyAliasResolvesToCanonicalAndKeepsRaw(t *testing.T) {
	a, ok := Normalize("reprovED_pr")
	b, _ := Normalize(ReprovadoPR)
	if !ok || a.ID != b.ID || a.Code != ReprovadoPR || !a.Alias || a.Raw != "reprovED_pr" {
		t.Fatalf("%+v %+v", a, b)
	}
	if !Allowed(LegacyReprovedPR, CorrecaoPR) {
		t.Fatal("alias deve ter a mesma transição")
	}
	if _, ok := Normalize("OUTRO"); ok {
		t.Fatal("desconhecido")
	}
}

func okEvidence() Evidence {
	return Evidence{PRDComplete: true, InitialContract: true, HumanRelease: true, LinkedToFeaturePR: true,
		ProjectActive: true, WorkflowPublished: true, InputsValid: true, PlanningSlot: true, FocusedPlanning: true,
		PlanValid: true, SubtasksMaterial: true, GatesVerified: true, ReviewsPassed: true, CommitsMerged: true,
		IntegratedHead: true, GitHubConfirms: true, MergeConfirmed: true,
		Tracks: map[Track]Stage{Frontend: StageReviewed, Backend: StageReviewed}}
}

func TestValidFlowsPass(t *testing.T) {
	ev := okEvidence()
	for _, tr := range Transitions {
		if err := Check(tr.From, tr.To, ev); err != nil {
			t.Fatalf("%s→%s: %v", tr.From, tr.To, err)
		}
	}
}

func TestInvalidTransitionsAreRejectedWithReasons(t *testing.T) {
	cases := []struct {
		from, to string
		mut      func(*Evidence)
		code     string
	}{
		{PRDDoc, PRDDocConcluido, func(e *Evidence) { e.PRDComplete = false }, "prd_incomplete"},
		{PRDDoc, PRDDocConcluido, func(e *Evidence) { e.InitialContract = false }, "no_initial_contract"},
		{ReprovadoPR, CorrecaoPR, func(e *Evidence) { e.HumanRelease = false }, "no_human_release"},
		{LegacyReprovedPR, CorrecaoPR, func(e *Evidence) { e.LinkedToFeaturePR = false }, "unlinked"},
		{CorrecaoPR, Planejamento, func(e *Evidence) { e.FocusedPlanning = false }, "not_focused"},
		{DraftPR, Concluido, func(e *Evidence) { e.MergeConfirmed = false }, "no_merge"},
		{Review, DraftPR, func(e *Evidence) { e.IntegratedHead = false }, "stale_head"},
		{PRDDoc, Planejamento, func(e *Evidence) {}, "not_allowed"},
	}
	for _, c := range cases {
		ev := okEvidence()
		c.mut(&ev)
		var rej *Rejection
		err := Check(c.from, c.to, ev)
		if !errors.As(err, &rej) || rej.Problems[0].Code != c.code {
			t.Fatalf("%s→%s: %v", c.from, c.to, err)
		}
	}
}

func TestFrontReviewedWithBackImplementingKeepsImplementation(t *testing.T) {
	tr := map[Track]Stage{Frontend: StageReviewed, Backend: StageImplementing}
	st, who := Aggregate(true, tr)
	if st != Implementacao || who != Backend {
		t.Fatalf("%s %s", st, who)
	}
	ev := okEvidence()
	ev.Tracks = tr
	if err := Check(Implementacao, Review, ev); err == nil {
		t.Fatal("agregado não pode ir a REVIEW")
	}
	tr[Backend] = StageImplemented
	if st, _ := Aggregate(true, tr); st != Review {
		t.Fatalf("%s", st)
	}
	if st, _ := Aggregate(false, nil); st != Planejamento {
		t.Fatal(st)
	}
}

func TestProjectMapValidationPointsToTheBlockingField(t *testing.T) {
	m := ProjectMap{Source: "jira", Remote: map[string]string{"To Do": PRDDoc, "Reprovado": LegacyReprovedPR, "X": "NADA"},
		Moves: map[string]string{PRDDoc: "11"}}
	ps := m.Validate([]string{PRDDoc, Concluido})
	if len(ps) != 2 || ps[0].Field != "remote[X]" || ps[1].Field != "moves["+Concluido+"]" {
		t.Fatalf("%v", ps)
	}
	r, ok := m.Resolve("reprovado")
	if !ok || r.Code != ReprovadoPR || r.Raw != "reprovado" {
		t.Fatalf("%+v", r)
	}
	if (ProjectMap{Source: "x"}).Validate(nil)[0].Field != "source" {
		t.Fatal("source")
	}
}
