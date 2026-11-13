package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/consolidate"
	"github.com/ViniciusBernucci/le-fabrique/internal/contract"
	"github.com/ViniciusBernucci/le-fabrique/internal/correction"
	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/focusplan"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
	"github.com/ViniciusBernucci/le-fabrique/internal/publication"
	"github.com/ViniciusBernucci/le-fabrique/internal/qualitypolicy"
	"github.com/ViniciusBernucci/le-fabrique/internal/risk"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/techplan"
	"github.com/ViniciusBernucci/le-fabrique/internal/trackadmit"
	"github.com/ViniciusBernucci/le-fabrique/internal/trackgates"
	"github.com/ViniciusBernucci/le-fabrique/internal/worktrees"
)

type fakePusher struct{ pushed []gitops.PushRequest }

func (f *fakePusher) Push(_ context.Context, r gitops.PushRequest) (effects.Result, error) {
	f.pushed = append(f.pushed, r)
	return effects.Result{}, nil
}

type fakePRs struct{ opened int }

func (f *fakePRs) OpenDraft(_ context.Context, r gitops.PRRequest) (effects.Result, gitops.PullRequest, error) {
	f.opened++
	return effects.Result{}, gitops.PullRequest{Number: 7, URL: "https://gh/pull/7", Draft: true, Head: r.Branch, Base: r.Base}, nil
}

type fakeState struct{ st gitops.PRState }

func (f *fakeState) ReadPR(context.Context, int) (gitops.PRState, error) { return f.st, nil }

func writeCommit(t *testing.T, dir, file, content string) {
	t.Helper()
	_ = os.MkdirAll(filepath.Dir(filepath.Join(dir, file)), 0o755)
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "agente: "+file)
}

// AJ06-04 / CT06 — a Draft PR só abre com os gates e os testes integrados na cabeça publicada; a mesma
// PR é reutilizada; cabeça nova depois dos testes exige revalidação.
func TestDraftPRPublishesOnlyTheGatedHeadAndOpensOnce(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "pub")
	var feature string
	_ = db.Pool.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'PB-1','f') RETURNING id::text`, p.ID).Scan(&feature)
	repo, base := newRepoWithBase(t)
	cons := &consolidate.Consolidator{DB: db.Pool, Repo: repo, Root: t.TempDir()}
	branch, _ := gitops.FeatureBranch(feature, 1)
	commitOnBranch(t, repo, "lf-tmp/f/frontend-impl-1", base, "apps/web/new.ts", "front\n")
	commitOnBranch(t, repo, "lf-tmp/f/backend-impl-1", base, "internal/api/new.go", "package api\n")
	if _, err := cons.Run(ctx, consolidate.Request{ProjectID: p.ID, FeatureID: feature, DeliveryBranch: branch, StartSHA: base, Attempts: []consolidate.Attempt{
		{Ref: "front", Branch: "lf-tmp/f/frontend-impl-1", BaseSHA: base}, {Ref: "back", Branch: "lf-tmp/f/backend-impl-1", BaseSHA: base}}}); err != nil {
		t.Fatal(err)
	}
	head, reval, _ := cons.Candidate(ctx, feature)
	gates := &trackgates.Store{DB: db.Pool}
	push, prs := &fakePusher{}, &fakePRs{}
	pub := &publication.Service{DB: db.Pool, Gates: gates, Cands: cons, Push: push, PRs: prs, State: &fakeState{}}
	req := publication.Request{ProjectID: p.ID, FeatureID: feature, Attempt: 1, Base: "main", Title: "Feature", Body: "corpo", Required: []statusflow.Track{statusflow.Frontend, statusflow.Backend},
		AttemptTrack: map[string]statusflow.Track{"front": statusflow.Frontend, "back": statusflow.Backend}}

	var blocked *publication.BlockedError
	if _, err := pub.PublishDraft(ctx, req); !errors.As(err, &blocked) || len(push.pushed) != 0 || prs.opened != 0 {
		t.Fatalf("sem gates nada é enviado: %v pushes=%d", err, len(push.pushed))
	}
	avail := []string{"lint", "build", "test"}
	fr := qualitypolicy.Decide(qualitypolicy.Input{Track: statusflow.Frontend, Suggested: risk.Low, Policy: risk.DefaultPolicy(), Available: avail})
	bk := qualitypolicy.Decide(qualitypolicy.Input{Track: statusflow.Backend, Suggested: risk.Low, Policy: risk.DefaultPolicy(), Available: avail})
	_ = gates.RecordTrack(ctx, feature, statusflow.Frontend, "frontrev", fr, trackEvidence(fr, "frontrev"), nil) // o Front não mudou de SHA (fast-forward)
	_ = gates.RecordTrack(ctx, feature, statusflow.Backend, head, bk, trackEvidence(bk, head), nil)              // o Back foi reescrito pela consolidação
	_ = gates.RecordIntegrated(ctx, feature, head, qualitypolicy.TestRun{Commands: []string{"make test"}, Passed: true})
	if len(reval) != 1 || reval[0] != "back" {
		t.Fatalf("%v", reval)
	}
	res, err := pub.PublishDraft(ctx, req)
	if err != nil || res.Head != head || res.PRNumber != 7 || len(push.pushed) != 1 || push.pushed[0].SHA != head || push.pushed[0].Branch != branch {
		t.Fatalf("%+v %v %+v", res, err, push.pushed)
	}
	again, err := pub.PublishDraft(ctx, req)
	if err != nil || !again.Existing || again.ID != res.ID {
		t.Fatalf("a Draft PR é única por feature: %+v %v", again, err)
	}
	var n int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM publications WHERE feature_id::text=$1 AND kind='draft'`, feature).Scan(&n)
	if n != 1 {
		t.Fatalf("%d", n)
	}
	// Cabeça nova depois dos testes: revalidação.
	_, _ = db.Pool.Exec(ctx, `INSERT INTO delivery_candidates (feature_id, head_sha, base_head, run_id, revalidate)
		SELECT feature_id, 'abcdef0123456789abcdef0123456789abcdef01', head_sha, run_id, '[]' FROM delivery_candidates WHERE feature_id::text=$1 LIMIT 1`, feature)
	if _, err := pub.PublishDraft(ctx, req); !errors.As(err, &blocked) || !strings.Contains(blocked.Error(), "testes integrados ainda não rodaram") {
		t.Fatalf("%v", err)
	}
}

// AJ09-03 / CT02, CT12 — a correção só começa após a liberação humana, assume a próxima vaga compatível com
// prioridade sem tirar o trabalho ativo, planeja só o afetado, executa em worktrees, publica na PR original
// com evidência ligando rodada, Issue, contrato, commits e cabeça; PR integrada antes cancela o avanço.
func TestCorrectionIsReleasedPrioritizedPlannedAndPublishedToTheOriginalPR(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	if err := (&statusflow.Store{DB: db.Pool}).EnsureCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	p := mustProject(t, db.Pool, "corr")
	co := &sentinel.Coordinator{DB: db.Pool}
	var feature string
	_ = db.Pool.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'CR-1','f') RETURNING id::text`, p.ID).Scan(&feature)
	repo, base := newRepoWithBase(t)
	cons := &consolidate.Consolidator{DB: db.Pool, Repo: repo, Root: t.TempDir()}
	branch, _ := gitops.FeatureBranch(feature, 1)
	// PR original já publicada na cabeça `base`.
	gitIn(t, repo, "branch", branch, base)
	_, _ = db.Pool.Exec(ctx, `INSERT INTO publications (project_id, feature_id, kind, branch, head_sha, pr_number, pr_url) VALUES ($1,$2,'draft',$3,$4,7,'https://gh/pull/7')`, p.ID, feature, branch, base)
	_, _ = db.Pool.Exec(ctx, `INSERT INTO delivery_candidates (feature_id, head_sha, base_head, run_id)
		SELECT $1, $2, $2, id FROM (SELECT id FROM consolidation_runs LIMIT 1) r`, feature, base)
	parent, _ := co.Track(ctx, sentinel.NewIssue{ProjectID: p.ID, Key: "CR-1", Kind: "feature", Status: statusflow.DraftPR, FeatureID: feature})
	corr, err := co.Track(ctx, sentinel.NewIssue{ProjectID: p.ID, Key: "FIX-1", Kind: "correction", Status: statusflow.ReprovadoPR, ParentID: parent.ID, FeatureID: feature})
	if err != nil {
		t.Fatal(err)
	}
	var watch string
	_ = db.Pool.QueryRow(ctx, `INSERT INTO pr_watch (project_id, pr_number, feature_id, issue_key, branch, state) VALUES ($1,7,$2,'CR-1',$3,'open') RETURNING id::text`, p.ID, feature, branch).Scan(&watch)
	_, _ = db.Pool.Exec(ctx, `INSERT INTO pr_rounds (watch_id, round_key, seq, closing_kind, closing_id, status, issue_key) VALUES ($1,'r1',1,'review',10,'issue_created','FIX-1')`, watch)

	tech := &techplan.Service{DB: db.Pool, Plans: &planning.Store{DB: db.Pool}}
	gates := &trackgates.Store{DB: db.Pool}
	state := &fakeState{st: gitops.PRState{Number: 7, State: "open", Draft: true}}
	push := &fakePusher{}
	d := &correction.Dispatcher{Sentinel: co, Admit: &trackadmit.Service{DB: db.Pool}, Wt: &worktrees.Manager{DB: db.Pool, Repo: repo, Root: t.TempDir()}, Cons: cons,
		Focus: &focusplan.Service{DB: db.Pool, Tech: tech, Contracts: &contract.Store{DB: db.Pool}},
		Pub:   &publication.Service{DB: db.Pool, Gates: gates, Cands: cons, Push: push, PRs: &fakePRs{}, State: state}}

	prev := planning.Plan{Summary: "original", Tasks: []planning.Task{tpTask("api", "BACKEND", []string{"internal/api"}), tpTask("tela", "FRONTEND", []string{"apps/web/src"}, "api")}}
	if _, err := tech.Accept(ctx, p.ID, feature, prev); err != nil {
		t.Fatal(err)
	}
	_, _ = db.Pool.Exec(ctx, `UPDATE subtasks SET impl_state='done', review_state='done', state='done' WHERE feature_id=$1 AND plan_version=1`, feature)
	in := focusplan.Input{Round: focusplan.Round{Key: "r1", HeadSHA: base, Refs: []string{"comment:1", "line_comment:2"}}, CurrentHead: base, Previous: prev, Policy: risk.DefaultPolicy(),
		Proposed: planning.Plan{Summary: "correção", Tasks: []planning.Task{
			fixTask("fix-front", "FRONTEND", "low", []string{"apps/web/src"}, "comment:1"), fixTask("fix-back", "BACKEND", "low", []string{"internal/api"}, "line_comment:2")}}}
	cbase := func(key string) contract.Deterministic {
		return contract.Deterministic{ProjectID: p.ID, IssueID: "FIX-1", Phase: "implementation", Status: "IMPLEMENTACAO", Repo: contract.RepoIdentity{Owner: "org", Name: "app"},
			Branch: "lf-tmp/" + key, BaseSHA: base, Worktree: "wt/" + key, Limits: contract.Limits{TimeoutSeconds: 600, MaxRounds: 2, BudgetCents: 100}}
	}
	preq := correction.PlanRequest{ProjectID: p.ID, FeatureID: feature, IssueKey: "FIX-1", Input: in, Base: cbase}
	// Sem liberação humana: nada é planejado nem despachado.
	if _, err := d.Plan(ctx, preq); !errors.Is(err, correction.ErrAwaitingRelease) {
		t.Fatalf("%v", err)
	}
	var subs int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM subtasks WHERE feature_id=$1`, feature).Scan(&subs)
	if subs != 2 {
		t.Fatalf("nada é materializado antes da liberação: %d", subs)
	}
	// O humano libera na origem: a Sentinela efetiva REPROVADO_PR → CORRECAO_PR.
	if _, err := co.Transition(ctx, sentinel.Request{ProjectID: p.ID, IssueKey: "FIX-1", To: statusflow.CorrecaoPR, Actor: sentinel.ActorSentinel, Observer: "origem:jira", ObservationKey: "rel1",
		Evidence: statusflow.Evidence{HumanRelease: true, LinkedToFeaturePR: true}}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Plan(ctx, preq); err != nil {
		t.Fatal(err)
	}
	if iss, _ := co.Get(ctx, p.ID, "FIX-1"); iss.Status != statusflow.Implementacao || iss.ID != corr.ID {
		t.Fatalf("%+v", iss)
	}
	// Outro trabalho normal já ocupa o Front e há um Back normal mais antigo na fila.
	var other string
	_ = db.Pool.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'OT-1','o') RETURNING id::text`, p.ID).Scan(&other)
	seedSubtask(t, db, p.ID, other, "o-front", "FRONTEND", `[]`, `[]`)
	seedSubtask(t, db, p.ID, other, "o-back", "BACKEND", `[]`, `[]`)
	_, _ = db.Pool.Exec(ctx, `UPDATE subtasks SET impl_state='running', created_at = now() - interval '2 hours' WHERE feature_id=$1 AND task_key='o-front'`, other)
	_, _ = db.Pool.Exec(ctx, `UPDATE subtasks SET created_at = now() - interval '1 hour' WHERE feature_id=$1 AND task_key='o-back'`, other)
	_, _ = db.Pool.Exec(ctx, `INSERT INTO track_reservations (project_id, slot, holder, generation) VALUES ($1,'FRONTEND',$2,nextval('reservation_generation_seq'))`, p.ID, other+"/o-front#implementation")
	pathsOf := func(_, key string) []string {
		if key == "fix-front" {
			return []string{"apps/web/src"}
		}
		return []string{"internal/api"}
	}
	lim := sentinel.Limits{PlanningSlots: 1, GlobalCapacity: 5}
	run1, wait1, err := d.Dispatch(ctx, p.ID, lim, base, 1, pathsOf)
	if err != nil {
		t.Fatal(err)
	}
	if len(run1) != 1 || run1[0].TaskKey != "fix-back" {
		t.Fatalf("a correção assume a vaga compatível antes do Back normal mais antigo; o Front segue ocupado: %+v", run1)
	}
	w := reasons(wait1)
	if w["fix-front/implementation"] != sentinel.WaitSlot || w["o-back/implementation"] != sentinel.WaitSlot {
		t.Fatalf("%v", w)
	}
	// O trabalho ativo do Front não foi interrompido.
	var running string
	_ = db.Pool.QueryRow(ctx, `SELECT impl_state FROM subtasks WHERE feature_id=$1 AND task_key='o-front'`, other).Scan(&running)
	if running != "running" {
		t.Fatalf("trabalho ativo foi tocado: %s", running)
	}
	// O Front termina: a correção Front entra na vaga liberada.
	_, _ = db.Pool.Exec(ctx, `UPDATE subtasks SET impl_state='done' WHERE feature_id=$1 AND task_key='o-front'`, other)
	_, _ = db.Pool.Exec(ctx, `DELETE FROM track_reservations WHERE project_id=$1 AND slot='FRONTEND'`, p.ID)
	run2, _, err := d.Dispatch(ctx, p.ID, lim, base, 1, pathsOf)
	if err != nil {
		t.Fatal(err)
	}
	var fixFront correction.Dispatched
	for _, r := range run2 {
		if r.TaskKey == "fix-front" {
			fixFront = r
		}
	}
	if fixFront.Attempt.ID == "" {
		t.Fatalf("%+v", run2)
	}
	fixBack := run1[0]
	if fixBack.Attempt.Branch == fixFront.Attempt.Branch || fixBack.Attempt.Path == fixFront.Attempt.Path || fixBack.Attempt.BaseSHA != base {
		t.Fatalf("worktrees distintas a partir da cabeça da PR: %+v %+v", fixBack.Attempt, fixFront.Attempt)
	}
	// Os agentes produzem commits nas worktrees temporárias.
	writeCommit(t, fixFront.Attempt.Path, "apps/web/src/fix.ts", "corrigido\n")
	writeCommit(t, fixBack.Attempt.Path, "internal/api/fix.go", "package api\n")
	for _, a := range []worktrees.Attempt{fixFront.Attempt, fixBack.Attempt} {
		if bad, _ := d.Wt.Violations(ctx, a.ID); len(bad) != 0 {
			t.Fatalf("fora do escopo: %v", bad)
		}
		_ = d.Wt.SetWriter(ctx, a.ID, false)
	}
	attempts := []consolidate.Attempt{{Ref: fixFront.Attempt.ID, Branch: fixFront.Attempt.Branch, BaseSHA: base}, {Ref: fixBack.Attempt.ID, Branch: fixBack.Attempt.Branch, BaseSHA: base}}
	cr, err := cons.Run(ctx, consolidate.Request{ProjectID: p.ID, FeatureID: feature, DeliveryBranch: branch, Attempts: attempts})
	if err != nil || cr.Head == base {
		t.Fatalf("%+v %v", cr, err)
	}
	// Gates da cabeça (as trilhas foram alteradas pela consolidação) e testes integrados.
	avail := []string{"lint", "build", "test"}
	fr := qualitypolicy.Decide(qualitypolicy.Input{Track: statusflow.Frontend, Suggested: risk.Low, Policy: risk.DefaultPolicy(), Available: avail})
	bk := qualitypolicy.Decide(qualitypolicy.Input{Track: statusflow.Backend, Suggested: risk.Low, Policy: risk.DefaultPolicy(), Available: avail})
	_ = gates.RecordTrack(ctx, feature, statusflow.Frontend, cr.Head, fr, trackEvidence(fr, cr.Head), nil)
	_ = gates.RecordTrack(ctx, feature, statusflow.Backend, cr.Head, bk, trackEvidence(bk, cr.Head), nil)
	finish := correction.FinishRequest{ProjectID: p.ID, FeatureID: feature, IssueKey: "FIX-1", RoundKey: "r1", DeliveryBranch: branch, Attempts: attempts,
		Publish: publication.Correction{ProjectID: p.ID, FeatureID: feature, Attempt: 1, RoundKey: "r1", IssueKey: "FIX-1", Contracts: []string{"hash-a"}, Required: []statusflow.Track{statusflow.Frontend, statusflow.Backend},
			AttemptTrack: map[string]statusflow.Track{fixFront.Attempt.ID: statusflow.Frontend, fixBack.Attempt.ID: statusflow.Backend}}}
	var blocked *publication.BlockedError
	if _, err := d.Finish(ctx, finish); !errors.As(err, &blocked) || len(push.pushed) != 0 {
		t.Fatalf("sem testes integrados na cabeça a PR não recebe nada: %v", err)
	}
	_ = gates.RecordIntegrated(ctx, feature, cr.Head, qualitypolicy.TestRun{Commands: []string{"make test"}, Passed: true})
	// PR integrada de forma concorrente: o avanço é cancelado e o trabalho fica preservado.
	state.st = gitops.PRState{Number: 7, State: "closed", Merged: true}
	if _, err := d.Finish(ctx, finish); !errors.Is(err, publication.ErrPRClosed) || len(push.pushed) != 0 {
		t.Fatalf("%v", err)
	}
	var cancelled int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM publications WHERE feature_id::text=$1 AND state='cancelled'`, feature).Scan(&cancelled)
	if cancelled != 1 || gitIn(t, repo, "rev-parse", "refs/heads/"+branch) != cr.Head {
		t.Fatalf("cancelada=%d; o trabalho segue na branch de entrega", cancelled)
	}
	// PR segue aberta: publica na MESMA PR e liga tudo.
	state.st = gitops.PRState{Number: 7, State: "open", Draft: true}
	res, err := d.Finish(ctx, finish)
	if err != nil || res.PRNumber != 7 || res.Head != cr.Head || len(push.pushed) != 1 || push.pushed[0].Branch != branch || push.pushed[0].SHA != cr.Head {
		t.Fatalf("%+v %v %+v", res, err, push.pushed)
	}
	var round, issue, commits, contracts, previous string
	_ = db.Pool.QueryRow(ctx, `SELECT round_key, issue_key, commits::text, contracts::text, previous_head FROM publications WHERE feature_id::text=$1 AND kind='correction' AND state='published'`, feature).Scan(&round, &issue, &commits, &contracts, &previous)
	if round != "r1" || issue != "FIX-1" || previous != base || !strings.Contains(commits, cr.Commits[0].Integrated) || !strings.Contains(contracts, "hash-a") {
		t.Fatalf("%s %s %s %s %s", round, issue, commits, contracts, previous)
	}
	var published string
	_ = db.Pool.QueryRow(ctx, `SELECT published_head FROM pr_rounds WHERE round_key='r1'`).Scan(&published)
	if published != cr.Head {
		t.Fatalf("a rodada guarda a cabeça publicada: %s", published)
	}
	if iss, _ := co.Get(ctx, p.ID, "FIX-1"); iss.Status != statusflow.DraftPR {
		t.Fatalf("a Issue da correção avança pela Sentinela: %s", iss.Status)
	}
}
