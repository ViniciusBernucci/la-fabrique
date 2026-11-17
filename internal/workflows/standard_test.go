package workflows

import "testing"

func without(g Graph, id string) Graph {
	out := Graph{}
	for _, n := range g.Nodes {
		if n.ID != id {
			out.Nodes = append(out.Nodes, n)
		}
	}
	for _, e := range g.Edges {
		if e.From != id && e.To != id {
			out.Edges = append(out.Edges, e)
		}
	}
	return out
}

func TestStandardGraphHasBothPathsAndNoGuardIssues(t *testing.T) {
	g := StandardGraph()
	if is := GuardIssues(g); len(is) != 0 {
		t.Fatalf("%v", is)
	}
	have := map[string]bool{}
	for _, n := range g.Nodes {
		have[n.ID] = true
	}
	for _, rg := range RequiredGuards {
		if !have[rg.ID] {
			t.Fatalf("falta %s", rg.ID)
		}
	}
	// O caminho de correção existe: a entrada tem as duas saídas.
	outs := map[string]bool{}
	for _, e := range g.Edges {
		if e.From == correctionEntry {
			outs[e.FromPort] = true
		}
	}
	if !outs["true"] || !outs["false"] {
		t.Fatalf("%v", outs)
	}
}

// ADR-0068: o desenho padrão já vem completo (agentes padrão e checks), sem pendências de configuração.
func TestStandardGraphComesFilledWithDefaultAgentsAndChecks(t *testing.T) {
	g := StandardGraph()
	if is := Validate(g); len(is) != 0 {
		t.Fatalf("o fluxo padrão deveria nascer sem pendências: %+v", is)
	}
	for id, agent := range StandardAgents {
		found := false
		for _, n := range g.Nodes {
			if n.ID == id {
				found = true
				if n.Props["agent"] != agent {
					t.Errorf("%s deveria usar %q: %v", id, agent, n.Props["agent"])
				}
			}
		}
		if !found {
			t.Errorf("nó %s ausente", id)
		}
	}
}

func TestFillStandardDefaultsOnlyFillsEmptyFields(t *testing.T) {
	g := StandardGraph()
	for i := range g.Nodes {
		switch g.Nodes[i].ID {
		case GuardPlanning:
			g.Nodes[i].Props["agent"] = "meu-agente" // escolha da pessoa
		case GuardImplFront:
			delete(g.Nodes[i].Props, "agent")
		case GuardIntegrated:
			g.Nodes[i].Props["commands"] = "  "
		}
	}
	if n := FillStandardDefaults(&g); n != 2 {
		t.Fatalf("deveria preencher 2 campos vazios, preencheu %d", n)
	}
	for _, n := range g.Nodes {
		switch n.ID {
		case GuardPlanning:
			if n.Props["agent"] != "meu-agente" {
				t.Fatalf("escolha da pessoa foi sobrescrita: %v", n.Props["agent"])
			}
		case GuardImplFront:
			if n.Props["agent"] != AgentImplFront {
				t.Fatalf("agente não preenchido: %v", n.Props["agent"])
			}
		case GuardIntegrated:
			if n.Props["commands"] != ChecksCommands {
				t.Fatalf("checks não preenchidos: %v", n.Props["commands"])
			}
		}
	}
	if n := FillStandardDefaults(&g); n != 0 {
		t.Fatalf("segunda passada deveria ser vazia: %d", n)
	}
}

func TestRemovedOrRewiredGuardsAreCaught(t *testing.T) {
	for _, id := range []string{GuardPRD, GuardIntegrated, GuardDraftPR, GuardMergeWait, GuardReviewBack, GuardImplFront} {
		if is := GuardIssues(without(StandardGraph(), id)); len(is) == 0 || is[0].Code != "guard_missing" {
			t.Fatalf("%s: %v", id, is)
		}
	}
	// Troca de tipo: a guarda deixa de ser o que era.
	g := StandardGraph()
	for i := range g.Nodes {
		if g.Nodes[i].ID == GuardIntegrated {
			g.Nodes[i].Type = TScript
		}
	}
	if is := GuardIssues(g); len(is) == 0 || is[0].Code != "guard_type" {
		t.Fatalf("%v", is)
	}
}

func TestCustomNodeCannotBypassAMandatoryGuard(t *testing.T) {
	// Atalho: consolidação → fim de sucesso, pulando checks, Draft PR e espera do merge.
	g := StandardGraph()
	g.Nodes = append(g.Nodes, Node{ID: "atalho", Type: TEnd, Props: map[string]any{"outcome": "success"}})
	g.Edges = append(g.Edges, Edge{ID: "x1", From: GuardConsolidate, FromPort: "out", To: "atalho", ToPort: "in"})
	is := GuardIssues(g)
	got := map[string]bool{}
	for _, i := range is {
		got[i.Node] = true
		if i.Code != "guard_bypass" {
			t.Fatalf("%+v", i)
		}
	}
	if !got[GuardIntegrated] || !got[GuardDraftPR] || !got[GuardMergeWait] || got[GuardConsolidate] || got[GuardPlanning] {
		t.Fatalf("%v", is)
	}
	// Atalho que pula o gate do PRD fora da entrada de correção.
	h := StandardGraph()
	h.Edges = append(h.Edges, Edge{ID: "x2", From: "inicio", FromPort: "out", To: "pronto_planejar", ToPort: "in"})
	if is := GuardIssues(h); len(is) == 0 {
		t.Fatal("o desvio do gate do PRD deveria ser recusado")
	}
	// Customizar sem desviar é livre: um nó extra no meio do caminho não contorna nada.
	k := StandardGraph()
	k.Nodes = append(k.Nodes, Node{ID: "extra", Type: TWait, Props: map[string]any{"minutes": float64(5)}})
	for i, e := range k.Edges {
		if e.From == GuardPlanning && e.FromPort == "out" {
			k.Edges[i].To = "extra"
		}
	}
	k.Edges = append(k.Edges, Edge{ID: "x3", From: "extra", FromPort: "out", To: "ramificar", ToPort: "in"})
	if is := GuardIssues(k); len(is) != 0 {
		t.Fatalf("%v", is)
	}
}
