package integration

import (
	"context"
	"strings"
	"testing"

	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

func tstep(node, typ, port string) workflows.TraceStep {
	return workflows.TraceStep{Node: node, Type: typ, Port: port}
}

// ADR-0069 — a execução de uma Issue pede à Sentinela as transições do catálogo com a evidência real; a Draft PR de
// verdade é registrada para o ciclo da PR; PR simulada não confirma nada.
func TestIssueBridgeRequestsTransitionsWithRealEvidence(t *testing.T) {
	r := newRelEnv(t)
	db := r.e.db.Pool
	ctx := context.Background()
	if _, err := db.Exec(ctx, `UPDATE projects SET state = 'active' WHERE id = $1`, r.proj); err != nil {
		t.Fatal(err)
	}
	w := humanFlow(r, t)
	r.c.expect(r.publish(w), 201)

	if err := (&statusflow.Store{DB: db}).EnsureCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	co := &sentinel.Coordinator{DB: db}
	if _, err := co.Track(ctx, sentinel.NewIssue{ProjectID: r.proj, Key: "ABC-7", Kind: "feature", Status: statusflow.PRDDocConcluido}); err != nil {
		t.Fatal(err)
	}
	run, err := runner.Start(ctx, db, runner.StartInput{ProjectID: r.proj, WorkflowID: w.ID, Title: "ABC-7 — Selo", Task: "x", IssueKey: "ABC-7", PRDDone: true})
	if err != nil || run.IssueKey != "ABC-7" {
		t.Fatalf("start: %+v %v", run, err)
	}
	// a feature da execução é a da Issue (nova tentativa da mesma Issue reaproveita a feature)
	var extID string
	_ = db.QueryRow(ctx, `SELECT f.external_id FROM features f JOIN executions e ON e.feature_id = f.id WHERE e.id = $1`, run.ExecutionID).Scan(&extID)
	if extID != "ABC-7" {
		t.Fatalf("a feature deveria ter a chave da Issue: %q", extID)
	}
	if again, err := runner.Start(ctx, db, runner.StartInput{ProjectID: r.proj, WorkflowID: w.ID, Title: "ABC-7 — Selo", IssueKey: "ABC-7"}); err != nil || again.ExecutionID == run.ExecutionID {
		t.Fatalf("segunda tentativa da mesma Issue: %+v %v", again, err)
	}

	b := &runner.IssueBridge{DB: db, Coord: co}
	status := func() string {
		iss, err := co.Get(ctx, r.proj, "ABC-7")
		if err != nil {
			t.Fatal(err)
		}
		return iss.Status
	}
	st := workflows.RunState{Vars: map[string]any{}}

	// planejamento
	st.Trace = append(st.Trace, tstep("inicio", "start", "out"), tstep("pronto_planejar", "converge", "out"))
	b.Progress(ctx, run, st)
	if status() != statusflow.Planejamento {
		t.Fatalf("deveria estar em PLANEJAMENTO: %s", status())
	}
	// o plano ainda não foi aceito: sem evidência não avança
	st.Trace = append(st.Trace, tstep(workflows.GuardPlanning, "agent", "out"))
	b.Progress(ctx, run, st) // vars do planejamento ausentes: não pede
	if status() != statusflow.Planejamento {
		t.Fatalf("sem plano aceito não pode ir a IMPLEMENTACAO: %s", status())
	}
	b2 := &runner.IssueBridge{DB: db, Coord: co} // outro processo: tenta de novo, agora com o plano
	st.Vars[workflows.VarKey(workflows.GuardPlanning)] = map[string]any{"plano_versao": float64(1), "subtarefas": float64(2), "subtarefas_frontend": float64(1), "subtarefas_backend": float64(1)}
	b2.Progress(ctx, run, st)
	if status() != statusflow.Implementacao {
		t.Fatalf("plano aceito deveria levar a IMPLEMENTACAO: %s", status())
	}
	// implementações, revisões, consolidação e checks integrados
	for _, n := range []string{workflows.GuardImplFront, workflows.GuardReviewFront, workflows.GuardImplBack, workflows.GuardReviewBack, workflows.GuardConsolidate, workflows.GuardIntegrated} {
		st.Trace = append(st.Trace, tstep(n, "x", "out"))
	}
	b2.Progress(ctx, run, st)
	if status() != statusflow.Review {
		t.Fatalf("checks integrados aprovados deveriam levar a REVIEW: %s", status())
	}
	// Draft PR simulada não confirma nada
	st.Trace = append(st.Trace, tstep(workflows.GuardDraftPR, "git", "out"))
	st.Vars[workflows.VarKey(workflows.GuardDraftPR)] = map[string]any{"simulado": true, "pr_url": "(simulada)"}
	b2.Progress(ctx, run, st)
	if status() != statusflow.Review {
		t.Fatalf("PR simulada não pode ir a DRAFT_PR: %s", status())
	}
	// Draft PR de verdade
	b3 := &runner.IssueBridge{DB: db, Coord: co}
	st.Vars[workflows.VarKey(workflows.GuardDraftPR)] = map[string]any{"pr_numero": float64(42), "pr_url": "https://github.example/acme/app/pull/42", "branch": "lf/abc"}
	b3.Progress(ctx, run, st)
	if status() != statusflow.DraftPR {
		t.Fatalf("PR aberta deveria levar a DRAFT_PR: %s", status())
	}
	var linked int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM pr_watch WHERE project_id = $1 AND pr_number = 42 AND issue_key = 'ABC-7'`, r.proj).Scan(&linked)
	if linked != 1 {
		t.Fatalf("a PR deveria estar registrada para o ciclo da PR: %d", linked)
	}
	// as projeções ao Jira ficaram como intenção (outbox) para a rotina de projeção
	var intents int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE project_id = $1`, r.proj).Scan(&intents)
	if intents == 0 {
		t.Error("as transições deveriam deixar intenções de projeção na outbox")
	}
	// o diagnóstico da execução registra as decisões
	var diags int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM execution_events WHERE execution_id = $1 AND type = 'run_diag' AND payload->>'step' = 'issue.transition'`, run.ExecutionID).Scan(&diags)
	if diags < 4 {
		t.Errorf("as decisões deveriam estar no diagnóstico: %d", diags)
	}
	// execução de teste e pedido manual nunca tocam a Sentinela
	manual := run
	manual.IssueKey = ""
	b3.Progress(ctx, manual, st)
	test := run
	test.Kind = "test"
	b3.Progress(ctx, test, st)
	if !strings.Contains(status(), "DRAFT_PR") {
		t.Fatalf("estado alterado indevidamente: %s", status())
	}
}
