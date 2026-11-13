package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/contract"
	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/prdgate"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// AJ04-02 / CT03 — gate do PRD: pendências, evento forjado de agente e exatamente um contrato inicial.
func TestPRDGateBlocksUntilCompleteThenPublishesOneInitialContract(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	if err := (&statusflow.Store{DB: db.Pool}).EnsureCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	p := mustProject(t, db.Pool, "prd-gate")
	co := &sentinel.Coordinator{DB: db.Pool}
	cs := &contract.Store{DB: db.Pool}
	g := &prdgate.Gate{DB: db.Pool, Contracts: cs, Sentinel: co}
	if _, err := co.Track(ctx, sentinel.NewIssue{ProjectID: p.ID, Key: "PRJ-7", Kind: "feature", Status: statusflow.PRDDoc}); err != nil {
		t.Fatal(err)
	}
	cfg := prdgate.Config{ProjectID: p.ID, RequiredFields: []string{"Critérios de aceite"}, Repo: contract.RepoIdentity{Owner: "org", Name: "repo"},
		Versions: contract.Versions{Workflow: "w1"}, Limits: contract.Limits{TimeoutSeconds: 600}}
	prd := prdgate.PRD{Key: "PRJ-7", Revision: "r1", Title: "Cadastro de clientes", Description: "Permitir cadastrar e editar clientes pelo painel.",
		Accessible: true, Criteria: []string{"Cadastro salva o cliente"}}

	res, err := g.Run(ctx, cfg, prd) // falta o campo obrigatório do projeto
	if err != nil || res.Complete || len(res.Pending) != 1 || res.Pending[0].Field != "field:Critérios de aceite" {
		t.Fatalf("%+v %v", res, err)
	}
	// Agente forja a conclusão: recusado; Issue segue em PRD-DOC e sem contrato.
	_, err = co.Transition(ctx, sentinel.Request{ProjectID: p.ID, IssueKey: "PRJ-7", To: statusflow.PRDDocConcluido, Actor: sentinel.ActorAgent,
		ObservationKey: "forjado", Evidence: statusflow.Evidence{PRDComplete: true, InitialContract: true}})
	if !errors.Is(err, sentinel.ErrNotAuthority) {
		t.Fatalf("%v", err)
	}
	if iss, _ := co.Get(ctx, p.ID, "PRJ-7"); iss.Status != statusflow.PRDDoc {
		t.Fatalf("%+v", iss)
	}
	if _, err := cs.Get(ctx, p.ID, "issue:PRJ-7:initial", 0); err == nil {
		t.Fatal("não deveria haver contrato")
	}
	// PRD corrigido; avaliar duas vezes.
	prd.Fields = map[string]string{"Critérios de aceite": "ver lista"}
	prd.Revision = "r2"
	first, err := g.Run(ctx, cfg, prd)
	if err != nil || !first.Complete || first.Transition != sentinel.DecisionApplied || first.Contract.Version != 1 {
		t.Fatalf("%+v %v", first, err)
	}
	second, err := g.Run(ctx, cfg, prd)
	if err != nil || !second.Repeated || second.Contract.ID != first.Contract.ID {
		t.Fatalf("%+v %v", second, err)
	}
	var n int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM execution_contracts WHERE scope_key='issue:PRJ-7:initial'`).Scan(&n)
	if iss, _ := co.Get(ctx, p.ID, "PRJ-7"); n != 1 || iss.Status != statusflow.PRDDocConcluido || iss.Version != 2 {
		t.Fatalf("contratos=%d %+v", n, iss)
	}
}
