package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/contract"
	"github.com/ViniciusBernucci/le-fabrique/internal/focusplan"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
	"github.com/ViniciusBernucci/le-fabrique/internal/risk"
	"github.com/ViniciusBernucci/le-fabrique/internal/techplan"
)

func fixTask(key, track string, risk string, paths []string, refs ...string) planning.Task {
	t := tpTask(key, track, paths)
	t.Risk, t.Refs = risk, refs
	return t
}

// AJ05-04 / CT11, CT12 — plano focalizado: cada subtarefa aponta para os comentários; retém o que ainda vale,
// reavalia o risco, gera contratos novos; mudança material da cabeça exige reconciliação antes do despacho.
func TestFocusedPlanFromRoundKeepsValidReferencesAndRequiresReconciliation(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "focus")
	f, err := persistence.CreateFeature(ctx, db.Pool, p.ID, "FP-1", "Feature")
	if err != nil {
		t.Fatal(err)
	}
	tech := &techplan.Service{DB: db.Pool, Plans: &planning.Store{DB: db.Pool}}
	prev := planning.Plan{Summary: "original", Tasks: []planning.Task{
		tpTask("api", "BACKEND", []string{"internal/x/x.go"}), tpTask("tela", "FRONTEND", []string{"apps/web/src/pages/X.tsx"}, "api")}}
	if _, err := tech.Accept(ctx, p.ID, f.ID, prev); err != nil {
		t.Fatal(err)
	}
	svc := &focusplan.Service{DB: db.Pool, Tech: tech, Contracts: &contract.Store{DB: db.Pool}}
	round := focusplan.Round{Key: "r1", HeadSHA: "aaaaaaa1", Refs: []string{"comment:1", "line_comment:2", "comment:3"}}
	proposed := planning.Plan{Summary: "correção da rodada 1", Tasks: []planning.Task{
		fixTask("fix-tela", "FRONTEND", "low", []string{"apps/web/src/pages/X.tsx"}, "comment:1", "comment:3"),
		fixTask("fix-api", "BACKEND", "low", []string{"internal/auth/x.go"}, "line_comment:2"),
	}}
	base := func(key string) contract.Deterministic {
		return contract.Deterministic{ProjectID: p.ID, IssueID: "FP-1", Phase: "implementation", Status: "IMPLEMENTACAO", Repo: contract.RepoIdentity{Owner: "org", Name: "app"},
			Branch: "lf-tmp/fp/" + key, BaseSHA: strings.Repeat("a", 40), Worktree: "wt/" + key, Limits: contract.Limits{TimeoutSeconds: 600, MaxRounds: 2, BudgetCents: 100}}
	}
	in := focusplan.Input{Round: round, CurrentHead: "aaaaaaa1", Previous: prev, Proposed: proposed, Policy: risk.DefaultPolicy()}

	// Validação: sem referência, referência estranha e comentário sem tratamento são recusados.
	bad := in
	bad.Proposed = planning.Plan{Tasks: []planning.Task{fixTask("fix-a", "BACKEND", "low", []string{"internal/a.go"}, "comment:999"), fixTask("fix-b", "BACKEND", "low", []string{"internal/b.go"})}}
	_, ps := focusplan.Prepare(bad)
	all := ""
	for _, x := range ps {
		all += x.String() + "|"
	}
	for _, want := range []string{"referência fora da rodada: comment:999", "deve apontar para os comentários", "sem tratamento"} {
		if !strings.Contains(all, want) {
			t.Fatalf("faltou %q: %s", want, all)
		}
	}
	// Retenção e risco reavaliado.
	in.ChangedSinceRound = []string{"apps/web/src/pages/X.tsx"}
	in.CurrentHead = "bbbbbbb2"
	res, ps := focusplan.Prepare(in)
	if len(ps) != 0 {
		t.Fatalf("%v", ps)
	}
	if strings.Join(res.Retained, ",") != "api" || strings.Join(res.Invalidated, ",") != "tela" {
		t.Fatalf("retidas=%v invalidadas=%v", res.Retained, res.Invalidated)
	}
	for _, tk := range res.Plan.Tasks {
		if tk.Key == "fix-api" && tk.Risk != risk.High {
			t.Fatalf("o risco deve ser reavaliado sobre os arquivos (auth ⇒ alto): %s", tk.Risk)
		}
	}
	// Mudança material da cabeça (o arquivo da correção foi alterado depois da rodada): reconciliar, nada gravado.
	if !res.Reconcile.Required || len(res.Reconcile.Files) != 1 {
		t.Fatalf("%+v", res.Reconcile)
	}
	if _, err := svc.Accept(ctx, p.ID, f.ID, in, base, contract.References{}); !errors.Is(err, focusplan.ErrNeedsReconciliation) {
		t.Fatalf("%v", err)
	}
	var n int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM subtasks WHERE feature_id=$1`, f.ID).Scan(&n)
	if n != 2 {
		t.Fatalf("nada novo pode ser gravado antes da reconciliação: %d", n)
	}
	// Mudança não material (outro arquivo): materializa subtarefas de correção e um contrato por subtarefa.
	in.ChangedSinceRound = []string{"README.md"}
	if _, err := svc.Accept(ctx, p.ID, f.ID, in, base, contract.References{RequirementsRevision: "rev-pr"}); err != nil {
		t.Fatal(err)
	}
	var corr int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM subtasks WHERE feature_id=$1 AND work_class='correction'`, f.ID).Scan(&corr)
	if corr != 2 {
		t.Fatalf("subtarefas de correção: %d", corr)
	}
	ch, err := (&contract.Store{DB: db.Pool}).Chain(ctx, p.ID, "subtask:fix-api")
	if err != nil || len(ch) != 1 || ch[0].Contract.Proposal.Risk != risk.High || ch[0].Contract.References.PlanVersion != 2 {
		t.Fatalf("%+v %v", ch, err)
	}
}
