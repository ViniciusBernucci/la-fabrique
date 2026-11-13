package integration

import (
	"context"
	"strings"
	"testing"

	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/qualitypolicy"
	"github.com/ViniciusBernucci/le-fabrique/internal/risk"
	"github.com/ViniciusBernucci/le-fabrique/internal/trackgates"
)

func trackEvidence(r qualitypolicy.Requirement, rev string) qualitypolicy.Evidence {
	e := qualitypolicy.Evidence{Tests: &qualitypolicy.TestRun{Revision: rev, Commands: []string{"make test"}, Passed: true}, ChecksPassed: map[string]string{}, Justifications: map[string]string{}}
	for _, c := range r.Checks {
		e.ChecksPassed[c] = rev
	}
	for _, c := range r.NotApplicable {
		e.Justifications[c] = "n/a"
	}
	if r.BrowserReview {
		e.Browser = &qualitypolicy.Review{Revision: rev, Approved: true, Independent: true}
	}
	if r.AIReview {
		e.AI = &qualitypolicy.Review{Revision: rev, Approved: true, Independent: true}
	}
	return e
}

// AJ08-05 / CT06, CT17 — Front conclui cedo; faltas no Back, nos testes integrados ou o review visual
// impedem a publicação; a consolidação que muda a cabeça exige a evidência da cabeça publicada.
func TestPublishGateNeedsEveryTrackIntegratedTestsAndTheRightHead(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	st := &trackgates.Store{DB: db.Pool}
	feature := "00000000-0000-0000-0000-0000000000f1"
	avail := []string{"lint", "build", "test"}
	front := qualitypolicy.Decide(qualitypolicy.Input{Track: statusflow.Frontend, Suggested: risk.Low, Policy: risk.DefaultPolicy(), Available: avail})
	back := qualitypolicy.Decide(qualitypolicy.Input{Track: statusflow.Backend, Suggested: risk.Low, Policy: risk.DefaultPolicy(), Available: avail})
	in := trackgates.Input{FeatureID: feature, Head: "headaaaa1111", Required: []statusflow.Track{statusflow.Frontend, statusflow.Backend}}
	has := func(res trackgates.Result, sub string) bool {
		return strings.Contains(strings.Join(res.Problems, "|"), sub)
	}

	// O Front termina cedo, com review visual.
	if err := st.RecordTrack(ctx, feature, statusflow.Frontend, "frontrev1", front, trackEvidence(front, "frontrev1"), []trackgates.Artifact{{Kind: "screenshot", Name: "home", SHA256: "abc"}}); err != nil {
		t.Fatal(err)
	}
	res, err := st.Gate(ctx, in)
	if err != nil || res.Allowed || !has(res, "BACKEND sem resultado") || !has(res, "testes integrados ainda não rodaram") {
		t.Fatalf("falta do Back e dos testes integrados bloqueia: %+v %v", res, err)
	}
	// Back pronto, mas os testes integrados ainda não.
	_ = st.RecordTrack(ctx, feature, statusflow.Backend, "backrev1", back, trackEvidence(back, "backrev1"), nil)
	if res, _ = st.Gate(ctx, in); res.Allowed || len(res.Problems) != 1 || !has(res, "testes integrados") {
		t.Fatalf("%+v", res)
	}
	// Testes integrados na cabeça: libera.
	_ = st.RecordIntegrated(ctx, feature, "headaaaa1111", qualitypolicy.TestRun{Commands: []string{"make test"}, Passed: true})
	if res, _ = st.Gate(ctx, in); !res.Allowed {
		t.Fatalf("tudo comprovado deve liberar: %+v", res)
	}
	// Cabeça nova depois dos testes: exige revalidação.
	in2 := in
	in2.Head = "headbbbb2222"
	if res, _ = st.Gate(ctx, in2); res.Allowed || !has(res, "testes integrados ainda não rodaram na cabeça headbbbb") {
		t.Fatalf("%+v", res)
	}
	// A consolidação alterou os commits do Back: o resultado precisa ser da cabeça publicada.
	in.Changed = map[statusflow.Track]bool{statusflow.Backend: true}
	if res, _ = st.Gate(ctx, in); res.Allowed || !has(res, "a consolidação alterou os commits") {
		t.Fatalf("%+v", res)
	}
	_ = st.RecordTrack(ctx, feature, statusflow.Backend, "headaaaa1111", back, trackEvidence(back, "headaaaa1111"), nil)
	if res, _ = st.Gate(ctx, in); !res.Allowed {
		t.Fatalf("%+v", res)
	}
	// Review visual do Front pendente (MCP/navegador indisponível) impede a publicação.
	ev := trackEvidence(front, "frontrev2")
	ev.Browser = &qualitypolicy.Review{Revision: "frontrev2", Pending: true}
	_ = st.RecordTrack(ctx, feature, statusflow.Frontend, "frontrev2", front, ev, nil)
	if res, _ = st.Gate(ctx, in); res.Allowed || !has(res, "browser_pending") {
		t.Fatalf("%+v", res)
	}
	// Testes integrados com falha também bloqueiam.
	_ = st.RecordTrack(ctx, feature, statusflow.Frontend, "frontrev3", front, trackEvidence(front, "frontrev3"), nil)
	_ = st.RecordIntegrated(ctx, feature, "headaaaa1111", qualitypolicy.TestRun{Commands: []string{"make test"}, Passed: false, Errors: []string{"TestX"}})
	if res, _ = st.Gate(ctx, in); res.Allowed || !has(res, "falharam") {
		t.Fatalf("%+v", res)
	}
}
