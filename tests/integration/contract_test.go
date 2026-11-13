package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/contract"
)

// AJ04-01 — snapshot validado, versionado, com hash e imutável; alteração cria outra versão.
func TestContractSnapshotsAreVersionedHashedAndImmutable(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "contract-1")
	s := &contract.Store{DB: db.Pool}
	prio := 2
	d := contract.Deterministic{ProjectID: p.ID, IssueID: "PRJ-1", SubtaskID: "s1", ExecutionID: "e1", Phase: "IMPLEMENTACAO", Status: "IMPLEMENTACAO",
		OfficialPriority: &prio, Repo: contract.RepoIdentity{Owner: "org", Name: "repo"}, Branch: "lf/s1", BaseSHA: strings.Repeat("b", 40),
		Worktree: "/work/s1", Versions: contract.Versions{Workflow: "w1", Agent: "a1", Skill: "k1"}, Limits: contract.Limits{TimeoutSeconds: 300, MaxRounds: 2},
		Permissions: []string{"fs.write:worktree"}}
	prop := contract.Proposal{Criteria: []string{"os testes do módulo passam"}, Risk: "low", RelevantFiles: []string{"internal/a.go"}}

	// Escapar do worktree não é publicado.
	bad := prop
	bad.RelevantFiles = []string{"../../etc/passwd"}
	var rej *contract.RejectedError
	if _, _, err := s.Publish(ctx, d, bad, "", contract.References{}, nil); !errors.As(err, &rej) {
		t.Fatalf("%v", err)
	}
	if _, err := s.Get(ctx, p.ID, contract.ScopeKey(d), 0); err == nil {
		t.Fatal("nada deveria ter sido publicado")
	}
	v1, unchanged, err := s.Publish(ctx, d, prop, "medium", contract.References{PlanVersion: 1}, nil)
	if err != nil || unchanged || v1.Version != 1 || len(v1.Hash) != 64 || v1.Contract.Proposal.Risk != "medium" {
		t.Fatalf("%+v %v", v1, err)
	}
	if again, unchanged, err := s.Publish(ctx, d, prop, "medium", contract.References{PlanVersion: 1}, nil); err != nil || !unchanged || again.ID != v1.ID {
		t.Fatalf("%+v %v %v", again, unchanged, err)
	}
	// Insumo alterado: nova versão encadeada; a anterior segue legível.
	prop.Criteria = append(prop.Criteria, "a API responde 200 no caminho feliz")
	v2, unchanged, err := s.Publish(ctx, d, prop, "medium", contract.References{PlanVersion: 2}, nil)
	if err != nil || unchanged || v2.Version != 2 || v2.Contract.ParentHash != v1.Hash {
		t.Fatalf("%+v %v", v2, err)
	}
	old, err := s.Get(ctx, p.ID, contract.ScopeKey(d), 1)
	if err != nil || old.Hash != v1.Hash || len(old.Contract.Proposal.Criteria) != 1 {
		t.Fatalf("%+v %v", old, err)
	}
	chain, err := s.Chain(ctx, p.ID, contract.ScopeKey(d))
	if err != nil || len(chain) != 2 {
		t.Fatalf("%v %v", chain, err)
	}
	// Imutável no banco.
	if _, err := db.Pool.Exec(ctx, `UPDATE execution_contracts SET body='{}'::jsonb`); err == nil {
		t.Fatal("update deveria falhar")
	}
	if _, err := db.Pool.Exec(ctx, `DELETE FROM execution_contracts`); err == nil {
		t.Fatal("delete deveria falhar")
	}
}
