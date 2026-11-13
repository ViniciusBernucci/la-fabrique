package integration

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/contract"
	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/issuesync"
	"github.com/ViniciusBernucci/le-fabrique/internal/prdgate"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

type listTracker struct {
	integrations.Tracker
	mu   sync.Mutex
	item integrations.Item
}

func (l *listTracker) Provider() string { return "jira" }
func (l *listTracker) ListItems(context.Context, string, integrations.ListOptions) (integrations.ItemPage, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return integrations.ItemPage{Items: []integrations.Item{l.item}}, nil
}
func (l *listTracker) GetItem(context.Context, string) (integrations.Item, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.item, nil
}

const incompletePRD = "Permitir cadastrar e editar clientes pelo painel administrativo.\n\n## Contexto\nPrecisamos de cadastro."
const completePRD = "Permitir cadastrar e editar clientes pelo painel administrativo.\n\n## Contexto\nPrecisamos de cadastro.\n\n## Critérios de aceite\n- O cadastro salva o cliente\n- A edição preserva o histórico\n\n## Referências\n- https://wiki.example.com/clientes\n"

func TestFromItemParsesSectionsCriteriaAndReferences(t *testing.T) {
	p := prdgate.FromItem(integrations.Item{Key: "PRJ-1", Title: "Clientes", Version: "v3", Description: completePRD}, []string{"Critérios de aceite", "Contexto"})
	if len(p.Criteria) != 2 || p.Criteria[0] != "O cadastro salva o cliente" || len(p.References) != 1 || !p.Accessible || p.Revision != "v3" {
		t.Fatalf("%+v", p)
	}
	if p.Fields["Contexto"] == "" || !strings.HasPrefix(p.Description, "Permitir cadastrar") {
		t.Fatalf("%+v", p)
	}
	if got := prdgate.FromItem(integrations.Item{Key: "X", Title: "t", Version: "v1", Description: strings.Repeat("a", 31000)}, nil); got.Accessible {
		t.Fatal("conteúdo longo demais deve ser tratado como possivelmente truncado")
	}
}

// AJ04-02 / CT03 — o PRD lido da origem pelo sync passa pelo gate: incompleto fica pendente em PRD-DOC;
// completo (nova revisão) publica exatamente um contrato inicial e a Sentinela libera o planejamento.
func TestPRDReadFromTheOriginGatesTheIssueDuringSync(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	if err := (&statusflow.Store{DB: db.Pool}).EnsureCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	p := mustProject(t, db.Pool, "prd-origin")
	co := &sentinel.Coordinator{DB: db.Pool}
	gate := &prdgate.Gate{DB: db.Pool, Contracts: &contract.Store{DB: db.Pool}, Sentinel: co}
	cfg := prdgate.Config{ProjectID: p.ID, RequiredFields: []string{"Critérios de aceite"}, Repo: contract.RepoIdentity{Owner: "org", Name: "repo"},
		Versions: contract.Versions{Workflow: "w1"}, Limits: contract.Limits{TimeoutSeconds: 600}}
	sy := &issuesync.Syncer{DB: db.Pool, Sentinel: co, PRD: func(ctx context.Context, _ string, it integrations.Item) error {
		_, err := gate.RunItem(ctx, cfg, it)
		return err
	}}
	tr := &listTracker{item: integrations.Item{Provider: "jira", ID: "1", Key: "PRJ-1", Title: "Cadastro de clientes", Status: "To Do", Version: "v1", Description: incompletePRD}}
	src := issuesync.Source{Kind: "jira", Container: "C", Map: syncMap("jira")}
	if _, err := sy.General(ctx, p.ID, issuesync.TriggerManual, tr, src); err != nil {
		t.Fatal(err)
	}
	iss, _ := co.Get(ctx, p.ID, "PRJ-1")
	var problems string
	_ = db.Pool.QueryRow(ctx, `SELECT problems::text FROM prd_gate_results WHERE issue_key='PRJ-1' AND revision='v1' AND complete=false`).Scan(&problems)
	if iss.Status != statusflow.PRDDoc || !strings.Contains(problems, "criteria") {
		t.Fatalf("PRD incompleto: pendências e a Issue em PRD-DOC: %s %s", iss.Status, problems)
	}
	// O responsável completa o PRD na origem (revisão nova): o próximo sync publica o contrato e libera.
	tr.item.Version, tr.item.Description = "v2", completePRD
	for i := 0; i < 2; i++ { // repetir não duplica
		if _, err := sy.General(ctx, p.ID, issuesync.TriggerManual, tr, src); err != nil {
			t.Fatal(err)
		}
	}
	iss, _ = co.Get(ctx, p.ID, "PRJ-1")
	var contracts int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM execution_contracts WHERE project_id=$1 AND scope_key='issue:PRJ-1:initial'`, p.ID).Scan(&contracts)
	if iss.Status != statusflow.PRDDocConcluido || contracts != 1 {
		t.Fatalf("%s contratos=%d", iss.Status, contracts)
	}
}
