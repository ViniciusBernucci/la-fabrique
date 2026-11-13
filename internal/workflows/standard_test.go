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

func TestStandardGraphIsADraftWithNodeLevelConfigurationGaps(t *testing.T) {
	is := Validate(StandardGraph())
	missingAgent := 0
	for _, i := range is {
		if i.Code == "required" && i.Field == "agent" {
			missingAgent++
		}
		if i.Code != "required" {
			t.Fatalf("o rascunho padrão só deve ter pendências de configuração: %+v", i)
		}
	}
	if missingAgent != 5 || len(is) != 6 { // 5 agentes sem escolha + os comandos dos checks integrados
		t.Fatalf("agentes sem escolha: %d (%v)", missingAgent, is)
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
