package contract

import (
	"errors"
	"strings"
	"testing"
)

func det() Deterministic {
	prio := 3
	return Deterministic{ProjectID: "p1", IssueID: "i1", SubtaskID: "s1", ExecutionID: "e1", Phase: "IMPLEMENTACAO", Status: "IMPLEMENTACAO",
		OfficialPriority: &prio, Repo: RepoIdentity{"org", "repo"}, Branch: "lf/i1-s1", BaseSHA: strings.Repeat("a", 40),
		Worktree: "/work/i1-s1", Versions: Versions{"w1", "a1", "k1"}, Limits: Limits{TimeoutSeconds: 600, MaxRounds: 2},
		Permissions: []string{"fs.write:worktree", "net:none"}}
}

func prop() Proposal {
	return Proposal{Criteria: []string{"os testes do módulo passam"}, Risk: "low", Dependencies: []string{"s0"}, SharedResources: []string{"schema"}, RelevantFiles: []string{"internal/x/x.go"}}
}

func TestUnknownProposalFieldsAreRejected(t *testing.T) {
	for _, raw := range []string{
		`{"criteria":["abcdefgh"],"risk":"low","permissions":["root"]}`,
		`{"criteria":["abcdefgh"],"risk":"low","project_id":"outro"}`,
		`{"criteria":["abcdefgh"],"risk":"low","worktree":"/"}`,
		`not json`,
	} {
		_, err := ParseProposal([]byte(raw))
		var rej *RejectedError
		if !errors.As(err, &rej) {
			t.Fatalf("%s: %v", raw, err)
		}
	}
	p, err := ParseProposal([]byte(`{"criteria":["abcdefgh"],"risk":"low","dependencies":[],"shared_resources":[],"relevant_files":[]}`))
	if err != nil || p.Risk != "low" {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestValidateRejectsEscapesCyclesAndBadRisk(t *testing.T) {
	if ps := Validate(det(), prop(), Graph{}); len(ps) != 0 {
		t.Fatalf("%v", ps)
	}
	p := prop()
	p.RelevantFiles = []string{"../etc/passwd", "/abs", "ok/file.go"}
	p.Risk = "extreme"
	ps := Validate(det(), p, Graph{})
	if len(ps) != 3 {
		t.Fatalf("%v", ps)
	}
	cyc := prop()
	cyc.Dependencies = []string{"s2"}
	if ps := Validate(det(), cyc, Graph{"s2": {"s3"}, "s3": {"s1"}}); len(ps) != 1 || ps[0].Field != "dependencies" {
		t.Fatalf("%v", ps)
	}
	bad := det()
	bad.BaseSHA = "xyz"
	bad.Worktree = "../fora"
	if ps := Validate(bad, prop(), Graph{}); len(ps) != 2 {
		t.Fatalf("%v", ps)
	}
}

func TestBuildAppliesRiskFloorAndHashIsDeterministic(t *testing.T) {
	a := Build(det(), prop(), "high", References{PlanVersion: 2})
	if a.Proposal.Risk != "high" || a.Provenance["risk"] != FromSentinel || a.Provenance["criteria"] != FromTechLead || a.Provenance["worktree"] != FromBackend {
		t.Fatalf("%+v", a)
	}
	d2 := det()
	d2.Permissions = []string{"net:none", "fs.write:worktree"} // outra ordem, mesmo conteúdo
	b := Build(d2, prop(), "high", References{PlanVersion: 2})
	if a.Hash() != b.Hash() || len(a.Hash()) != 64 {
		t.Fatal("hash não determinístico")
	}
	if Build(det(), prop(), "low", References{PlanVersion: 3}).Hash() == a.Hash() {
		t.Fatal("hash deve mudar com o conteúdo")
	}
}
