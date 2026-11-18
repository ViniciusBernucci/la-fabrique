package integration

import (
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/stdflow"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// ADR-0068 — os agentes padrão existem, são idempotentes, usam as skills padrão, e o fluxo padrão
// (novo ou já existente) aponta para eles e valida sem pendências.
func TestDefaultAgentsAndStandardFlowAreFilled(t *testing.T) {
	db := newMigratedDB(t)
	added, err := agents.InstallDefaults(ctx, db.Pool)
	if err != nil || len(added) != len(agents.DefaultNames()) || len(added) != 7 {
		t.Fatalf("instalação: %v %v", added, err)
	}
	for _, name := range agents.DefaultNames() {
		a, err := agents.GetByName(ctx, db.Pool, name)
		if err != nil || a.Provider != "claude" || len(a.Skills) != 1 || a.Instructions == "" {
			t.Fatalf("agente %s mal cadastrado: %+v %v", name, a, err)
		}
	}

	// quem editou um agente não o vê sobrescrito na subida seguinte
	a, _ := agents.GetByName(ctx, db.Pool, workflows.AgentPlanner)
	in := agents.Input{Name: a.Name, Description: "minha descrição", Instructions: a.Instructions, Provider: a.Provider, Skills: a.Skills, TimeoutMinutes: 15}
	if _, err := agents.Update(ctx, db.Pool, a.ID, a.Version, in); err != nil {
		t.Fatal(err)
	}
	added, err = agents.InstallDefaults(ctx, db.Pool)
	if err != nil || len(added) != 0 {
		t.Fatalf("idempotência: %v %v", added, err)
	}
	if cur, _ := agents.GetByName(ctx, db.Pool, a.Name); cur.Description != "minha descrição" || cur.TimeoutMinutes != 15 {
		t.Fatalf("edição da pessoa foi perdida: %+v", cur)
	}

	// fluxo de um projeto antigo, salvo com os campos vazios, é completado sem tocar no que foi escolhido
	p := newProjectDirect(t, db, "padrao-agentes")
	wf, _, err := stdflow.Provision(ctx, db.Pool, p.ID, p.Name)
	if err != nil {
		t.Fatal(err)
	}
	g := wf.Graph
	for i := range g.Nodes {
		switch g.Nodes[i].ID {
		case workflows.GuardImplBack:
			delete(g.Nodes[i].Props, "agent")
		case workflows.GuardReviewBack:
			g.Nodes[i].Props["agent"] = "escolha-da-pessoa"
		case workflows.GuardIntegrated:
			delete(g.Nodes[i].Props, "commands")
		}
	}
	if _, err := workflows.Save(ctx, db.Pool, wf.ID, wf.Version, wf.Name, wf.Description, g); err != nil {
		t.Fatal(err)
	}
	changed, err := stdflow.BackfillDefaults(ctx, db.Pool)
	if err != nil || len(changed) != 1 || changed[0] != wf.ID {
		t.Fatalf("backfill: %v %v", changed, err)
	}
	cur, _ := workflows.Get(ctx, db.Pool, wf.ID)
	for _, n := range cur.Graph.Nodes {
		if n.ID == workflows.GuardImplBack && n.Props["agent"] != workflows.AgentImplBack {
			t.Fatalf("agente não preenchido: %v", n.Props)
		}
		if n.ID == workflows.GuardReviewBack && n.Props["agent"] != "escolha-da-pessoa" {
			t.Fatalf("escolha sobrescrita: %v", n.Props)
		}
	}
	if is := workflows.Validate(cur.Graph); len(is) != 0 {
		t.Fatalf("pendências: %+v", is)
	}
	if again, err := stdflow.BackfillDefaults(ctx, db.Pool); err != nil || len(again) != 0 {
		t.Fatalf("segunda passada: %v %v", again, err)
	}
}
