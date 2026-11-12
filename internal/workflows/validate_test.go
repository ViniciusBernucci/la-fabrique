package workflows

import (
	"strings"
	"testing"
)

func n(id, typ string, props map[string]any) Node {
	if props == nil {
		props = map[string]any{}
	}
	return Node{ID: id, Type: typ, Props: props}
}

func e(id, from, fp, to, tp string) Edge {
	return Edge{ID: id, From: from, FromPort: fp, To: to, ToPort: tp}
}

func has(is []Issue, node, field, code string) bool {
	for _, i := range is {
		if i.Node == node && i.Field == field && i.Code == code {
			return true
		}
	}
	return false
}

func TestStarterGraphIsValid(t *testing.T) {
	if is := Validate(Starter()); len(is) != 0 {
		t.Fatalf("o grafo inicial deve ser válido: %+v", is)
	}
}

// Fluxo complexo com todos os tipos de controle e um loop limitado.
func complexGraph() Graph {
	return Graph{
		Nodes: []Node{
			n("s", TStart, nil),
			n("plan", TAgent, map[string]any{"agent": "planejador", "prompt": "Planeje", "timeout_minutes": float64(30)}),
			n("cond", TCondition, map[string]any{"expression": "plan.ok == true"}),
			n("loop", TLoop, map[string]any{"max_iterations": float64(3), "condition": "checks.passed == false"}),
			n("dev", TAgent, map[string]any{"agent": "dev", "prompt": "Implemente"}),
			n("chk", TChecks, map[string]any{"commands": "make test", "profile": "go"}),
			n("wait", TWait, map[string]any{"minutes": float64(5)}),
			n("hum", THuman, map[string]any{"question": "Aprovar?"}),
			n("docs", TDocs, map[string]any{"target": "docs/", "instruction": "Atualize"}),
			n("git", TGit, map[string]any{"operation": "draft_pr"}),
			n("join", TConverge, map[string]any{"mode": "any"}),
			n("ok", TEnd, map[string]any{"outcome": "success"}),
			n("bad", TEnd, map[string]any{"outcome": "failure"}),
		},
		Edges: []Edge{
			e("1", "s", PortOut, "plan", PortIn),
			e("2", "plan", PortOut, "cond", PortIn),
			e("3", "cond", "true", "loop", PortIn),
			e("4", "cond", "false", "wait", PortIn),
			e("5", "loop", "body", "dev", PortIn),
			e("6", "dev", PortOut, "chk", PortIn),
			e("7", "chk", PortOut, "loop", PortIn), // fecha o ciclo através do loop
			e("8", "loop", "done", "docs", PortIn),
			e("9", "wait", PortOut, "hum", PortIn),
			e("10", "hum", "approved", "join", PortIn),
			e("11", "hum", "rejected", "bad", PortIn),
			e("12", "docs", PortOut, "git", PortIn),
			e("13", "git", PortOut, "join", PortIn),
			e("14", "join", PortOut, "ok", PortIn),
		},
	}
}

func TestComplexGraphWithBoundedLoopIsValid(t *testing.T) {
	if is := Validate(complexGraph()); len(is) != 0 {
		t.Fatalf("esperado válido: %+v", is)
	}
}

func TestIssuesLocateNodeAndField(t *testing.T) {
	g := complexGraph()
	// Remove campos obrigatórios e quebra limites.
	for i := range g.Nodes {
		switch g.Nodes[i].ID {
		case "plan":
			g.Nodes[i].Props["prompt"] = "  "
			g.Nodes[i].Props["timeout_minutes"] = float64(9999)
		case "loop":
			delete(g.Nodes[i].Props, "max_iterations")
		case "chk":
			g.Nodes[i].Props["profile"] = "windows"
		case "git":
			g.Nodes[i].Props["extra"] = "x"
		}
	}
	is := Validate(g)
	for _, want := range [][3]string{
		{"plan", "prompt", "required"}, {"plan", "timeout_minutes", "out_of_range"},
		{"loop", "max_iterations", "required"}, {"chk", "profile", "invalid_option"}, {"git", "extra", "unknown_field"},
	} {
		if !has(is, want[0], want[1], want[2]) {
			t.Errorf("faltou %v em %+v", want, is)
		}
	}
	for _, i := range is {
		if i.Message == "" {
			t.Errorf("toda pendência precisa de mensagem: %+v", i)
		}
	}
}

func TestStructuralIssues(t *testing.T) {
	cases := map[string]struct {
		mut  func(*Graph)
		node string
		fld  string
		code string
	}{
		"sem início":        {func(g *Graph) { g.Nodes = g.Nodes[1:]; g.Edges = nil }, "", "", "start_count"},
		"dois inícios":      {func(g *Graph) { g.Nodes = append(g.Nodes, n("s2", TStart, nil)) }, "", "", "start_count"},
		"sem fim":           {func(g *Graph) { g.Nodes = g.Nodes[:1]; g.Edges = nil }, "", "", "end_missing"},
		"tipo desconhecido": {func(g *Graph) { g.Nodes = append(g.Nodes, n("x", "mágico", nil)) }, "x", "type", "unknown_type"},
		"id duplicado":      {func(g *Graph) { g.Nodes = append(g.Nodes, n("fim", TEnd, map[string]any{"outcome": "success"})) }, "fim", "", "duplicate_id"},
		"id inválido":       {func(g *Graph) { g.Nodes = append(g.Nodes, n("a b", TEnd, nil)) }, "a b", "", "invalid_id"},
		"porta inexistente": {func(g *Graph) { g.Edges[0].FromPort = "nada" }, "inicio", "nada", "invalid_port"},
		"conexão solta":     {func(g *Graph) { g.Edges[0].To = "ninguem" }, "inicio", "", "edge_dangling"},
		"auto-conexão": {func(g *Graph) {
			g.Nodes = append(g.Nodes, n("w", TWait, map[string]any{"minutes": float64(1)}))
			g.Edges = append(g.Edges, e("e9", "w", PortOut, "w", PortIn))
		}, "w", PortOut, "self_edge"},
		"nó órfão": {func(g *Graph) {
			g.Nodes = append(g.Nodes, n("o", TDocs, map[string]any{"target": "a", "instruction": "b"}))
		}, "o", "", "unreachable"},
		"saída sem conexão": {func(g *Graph) {
			g.Nodes = append(g.Nodes, n("c", TCondition, map[string]any{"expression": "x"}))
			g.Edges = append(g.Edges, e("e2", "inicio", PortOut, "c", PortIn))
		}, "c", "true", "output_unconnected"},
		"duas conexões na mesma saída": {func(g *Graph) {
			g.Nodes = append(g.Nodes, n("f2", TEnd, map[string]any{"outcome": "failure"}))
			g.Edges = append(g.Edges, e("e3", "inicio", PortOut, "f2", PortIn))
		}, "inicio", PortOut, "output_fanout"},
		"subfluxo com ID ruim": {func(g *Graph) {
			g.Nodes = append(g.Nodes, n("sf", TSubflow, map[string]any{"workflow_id": "123"}))
		}, "sf", "workflow_id", "invalid_value"},
	}
	for name, c := range cases {
		g := Starter()
		c.mut(&g)
		if is := Validate(g); !has(is, c.node, c.fld, c.code) {
			t.Errorf("%s: faltou (%q,%q,%s) em %+v", name, c.node, c.fld, c.code, is)
		}
	}
}

func TestCycleWithoutLoopIsRejectedButLoopCycleIsAllowed(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			n("s", TStart, nil),
			n("a", TWait, map[string]any{"minutes": float64(1)}),
			n("b", TWait, map[string]any{"minutes": float64(1)}),
			n("c", TConverge, map[string]any{"mode": "any"}),
			n("end", TEnd, map[string]any{"outcome": "success"}),
			n("d", TCondition, map[string]any{"expression": "x"}),
		},
		Edges: []Edge{
			e("1", "s", PortOut, "c", PortIn),
			e("2", "c", PortOut, "a", PortIn),
			e("3", "a", PortOut, "d", PortIn),
			e("4", "d", "true", "c", PortIn), // volta para c sem loop: ciclo ilegal
			e("5", "d", "false", "end", PortIn),
			e("6", "b", PortOut, "end", PortIn),
		},
	}
	is := Validate(g)
	found := false
	for _, i := range is {
		found = found || i.Code == "cycle_without_loop"
	}
	if !found {
		t.Fatalf("ciclo sem loop deveria ser rejeitado: %+v", is)
	}
}

func TestGraphLimitsAndMessages(t *testing.T) {
	g := Starter()
	for i := 0; i < MaxNodes+1; i++ {
		g.Nodes = append(g.Nodes, n("n"+strings.Repeat("x", i%5)+string(rune('a'+i%26)), TWait, nil))
	}
	if is := Validate(g); len(is) != 1 || is[0].Code != "too_many_nodes" {
		t.Fatalf("limite de nós: %+v", is)
	}
}

func TestCatalogIsConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, nt := range Catalog() {
		if seen[nt.Type] || nt.Label == "" {
			t.Fatalf("tipo duplicado ou sem rótulo: %+v", nt)
		}
		seen[nt.Type] = true
		for _, f := range nt.Fields {
			if f.Name == "" || f.Label == "" || (f.Kind == KindEnum && len(f.Options) == 0) {
				t.Errorf("campo inconsistente em %s: %+v", nt.Type, f)
			}
		}
	}
	for _, want := range []string{TStart, TEnd, TAgent, TSkill, TScript, TChecks, TGit, TDocs, TCondition, TLoop, TWait, THuman, TSubflow, TConverge, TFork} {
		if !seen[want] {
			t.Errorf("tipo %s ausente do catálogo", want)
		}
	}
}
