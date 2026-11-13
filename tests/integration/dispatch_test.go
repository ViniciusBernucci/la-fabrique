package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/contract"
	"github.com/ViniciusBernucci/le-fabrique/internal/dispatch"
	"github.com/ViniciusBernucci/le-fabrique/internal/executor"
	"github.com/ViniciusBernucci/le-fabrique/internal/providers"
)

type recAdapter struct {
	kind   executor.Kind
	tasks  []executor.Task
	failed bool
}

func (r *recAdapter) Capabilities() executor.Capabilities {
	return executor.Capabilities{Kind: r.kind, StructuredEvents: true}
}
func (r *recAdapter) Run(_ context.Context, t executor.Task, _ *executor.Session, emit func(executor.Event)) (executor.Outcome, error) {
	r.tasks = append(r.tasks, t)
	if emit != nil {
		emit(executor.Event{Seq: 1, Kind: executor.EvDone})
	}
	return executor.Outcome{Final: "ok", Failed: r.failed, Failure: map[bool]string{true: "falhou", false: ""}[r.failed]}, nil
}

// AJ04-03 / CT04 — CLI e API recebem o contrato da sua execução; revisão dos insumos gera outro
// despacho com snapshot novo sem reescrever o histórico.
func TestDispatchCarriesTheImmutableContractAndSurvivesInputRevisions(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "dispatch-1")
	cs := &contract.Store{DB: db.Pool}
	dp := &dispatch.Dispatcher{DB: db.Pool, Contracts: cs}
	prio := 1
	d := contract.Deterministic{ProjectID: p.ID, IssueID: "PRJ-1", SubtaskID: "s1", Phase: "IMPLEMENTACAO", Status: "IMPLEMENTACAO", OfficialPriority: &prio,
		Repo: contract.RepoIdentity{Owner: "o", Name: "r"}, Branch: "lf/s1", BaseSHA: strings.Repeat("c", 40), Worktree: "/work/s1",
		Versions: contract.Versions{Workflow: "w"}, Limits: contract.Limits{TimeoutSeconds: 120, MaxRounds: 3}}
	scope := contract.ScopeKey(d)
	req := func(exec string) dispatch.Request {
		return dispatch.Request{ProjectID: p.ID, Scope: scope, ExecutionID: exec, Instructions: "faça a tarefa", Mode: providers.ModeWorkspaceWrite, Dir: "/workspace"}
	}
	cli, api := &recAdapter{kind: executor.CLI}, &recAdapter{kind: executor.API}

	if _, _, err := dp.Run(ctx, cli, req("e0"), nil, nil); !errors.Is(err, dispatch.ErrNoContract) {
		t.Fatalf("%v", err)
	}
	prop := contract.Proposal{Criteria: []string{"o módulo compila e testa"}, Risk: "low", RelevantFiles: []string{"internal/a.go"}}
	v1, _, err := cs.Publish(ctx, d, prop, "", contract.References{PlanVersion: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	d1, _, err := dp.Run(ctx, cli, req("e1"), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := dp.Run(ctx, api, req("e2"), nil, nil); err != nil {
		t.Fatal(err)
	}
	for _, task := range []executor.Task{cli.tasks[0], api.tasks[0]} {
		if task.Contract.Version != 1 || task.Contract.Hash != v1.Hash || task.Limits.TimeoutSeconds != 120 || !strings.HasPrefix(task.Prompt(), task.Header()) {
			t.Fatalf("%+v", task.Contract)
		}
	}
	// Insumo revisado em ponto seguro: nova versão para o próximo despacho.
	prop.Criteria = append(prop.Criteria, "a API mantém o contrato público")
	v2, _, err := cs.Publish(ctx, d, prop, "", contract.References{PlanVersion: 2}, nil)
	if err != nil || v2.Version != 2 {
		t.Fatalf("%+v %v", v2, err)
	}
	if stale, latest, err := dp.Stale(ctx, p.ID, "e1"); err != nil || !stale || latest != 2 {
		t.Fatalf("%v %d %v", stale, latest, err)
	}
	// A execução em andamento (retomada/reentrega) segue com a versão que recebeu.
	rerun := req("e1")
	rerun.Kind = executor.CLI
	again, err := dp.Begin(ctx, rerun)
	if err != nil || !again.Resumed || again.ID != d1.ID || again.Task.Contract.Version != 1 {
		t.Fatalf("%+v %v", again.Task.Contract, err)
	}
	if _, _, err := dp.Run(ctx, cli, req("e3"), nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := cli.tasks[len(cli.tasks)-1]; got.Contract.Version != 2 || got.Contract.Hash != v2.Hash {
		t.Fatalf("%+v", got.Contract)
	}
	// Evidência liga versão, execução e resultado; histórico preservado.
	ev, err := dp.EvidenceOf(ctx, p.ID, "e1")
	if err != nil || ev.ContractVersion != 1 || ev.ContractHash != v1.Hash || ev.State != "done" || ev.Executor != "cli" {
		t.Fatalf("%+v %v", ev, err)
	}
	if ev2, _ := dp.EvidenceOf(ctx, p.ID, "e3"); ev2.ContractVersion != 2 {
		t.Fatalf("%+v", ev2)
	}
	if chain, err := cs.Chain(ctx, p.ID, scope); err != nil || len(chain) != 2 {
		t.Fatalf("%v %v", chain, err)
	}
	// Falha do executor fica registrada no despacho.
	bad := &recAdapter{kind: executor.API, failed: true}
	if _, _, err := dp.Run(ctx, bad, req("e4"), nil, nil); err != nil {
		t.Fatal(err)
	}
	if ev4, _ := dp.EvidenceOf(ctx, p.ID, "e4"); ev4.State != "failed" {
		t.Fatalf("%+v", ev4)
	}
}
