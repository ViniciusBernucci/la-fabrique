package workflows

import (
	"fmt"
	"sort"
	"strings"
)

// Guardas obrigatórias do fluxo padrão (LF-AJ-05). Identificadas pelo ID do nó; o editor aceita
// personalizar tudo, menos permitir um caminho até o sucesso que as contorne.
const (
	GuardPRD          = "guard_prd"
	GuardPlanning     = "guard_planning"
	GuardImplFront    = "guard_impl_front"
	GuardImplBack     = "guard_impl_back"
	GuardReviewFront  = "guard_review_front"
	GuardReviewBack   = "guard_review_back"
	GuardConsolidate  = "guard_consolidation"
	GuardIntegrated   = "guard_integrated_checks"
	GuardDraftPR      = "guard_draft_pr"
	GuardMergeWait    = "guard_merge_wait"
	correctionEntry   = "entrada"
	correctionPortOut = "true"
)

// Agentes padrão do fluxo padrão (ADR-0068): o catálogo os cria (agents.InstallDefaults) e o desenho os usa.
const (
	AgentPRD             = "redator-prd"
	AgentPlanner         = "tech-lead"
	AgentImplFront       = "implementador-frontend"
	AgentImplBack        = "implementador-backend"
	AgentReviewFront     = "revisor-frontend"
	AgentReviewBack      = "revisor-backend"
	AgentVisualDiscovery = "analista-ambiente-visual"
)

// ChecksCommands são os comandos padrão do nó de checks integrados (convenção do projeto: make).
const ChecksCommands = "make lint\nmake test"

// StandardAgents diz qual agente padrão atende cada nó-guarda de agente.
var StandardAgents = map[string]string{
	GuardPlanning: AgentPlanner, GuardImplFront: AgentImplFront, GuardReviewFront: AgentReviewFront,
	GuardImplBack: AgentImplBack, GuardReviewBack: AgentReviewBack,
}

// FillStandardDefaults preenche, num grafo já salvo, só o que está vazio nos nós-guarda: o agente de cada
// nó de agente e os comandos do nó de checks. Nunca sobrescreve o que o administrador escolheu.
// Devolve quantos campos preencheu.
func FillStandardDefaults(g *Graph) int {
	n := 0
	for i := range g.Nodes {
		nd := &g.Nodes[i]
		if nd.Props == nil {
			nd.Props = map[string]any{}
		}
		empty := func(k string) bool { v, _ := nd.Props[k].(string); return strings.TrimSpace(v) == "" }
		if a, ok := StandardAgents[nd.ID]; ok && nd.Type == TAgent && empty("agent") {
			nd.Props["agent"], n = a, n+1
		}
		if nd.ID == GuardIntegrated && nd.Type == TChecks && empty("commands") {
			nd.Props["commands"], n = ChecksCommands, n+1
		}
	}
	return n
}

type guardSpec struct {
	ID   string
	Type string
}

// RequiredGuards são as guardas e o tipo de nó esperado em cada uma, na ordem do fluxo.
var RequiredGuards = []guardSpec{
	{GuardPRD, TCondition}, {GuardPlanning, TAgent},
	{GuardImplFront, TAgent}, {GuardReviewFront, TAgent}, {GuardImplBack, TAgent}, {GuardReviewBack, TAgent},
	{GuardConsolidate, TGit}, {GuardIntegrated, TChecks}, {GuardDraftPR, TGit}, {GuardMergeWait, TWait},
}

// StandardGraph é o rascunho estrutural entregue a todo projeto novo: gate do PRD, planejamento,
// implementação e review por trilha, consolidação, checks integrados, Draft PR, espera por merge
// e o caminho de correção (entra direto no planejamento focalizado). Cada nó de agente já aponta para o
// agente padrão do catálogo e os checks trazem make lint/test; modelos, contas e ferramentas ficam
// vazios (valem os do agente) e o administrador ajusta o que quiser antes de publicar.
func StandardGraph() Graph {
	n := func(id, typ string, x, y float64, props map[string]any) Node {
		if props == nil {
			props = map[string]any{}
		}
		return Node{ID: id, Type: typ, X: x, Y: y, Props: props}
	}
	e := func(from, fp, to, tp string) Edge {
		return Edge{ID: from + "-" + fp + "-" + to, From: from, FromPort: fp, To: to, ToPort: tp}
	}
	g := Graph{Nodes: []Node{
		n("inicio", TStart, 40, 280, nil),
		n(correctionEntry, TCondition, 360, 280, map[string]any{"expression": "fabrica.correcao == true"}),
		n(GuardPRD, TCondition, 680, 480, map[string]any{"expression": "prd.concluido == true"}),
		n("pronto_planejar", TConverge, 1000, 280, map[string]any{"mode": "any"}),
		n(GuardPlanning, TAgent, 1320, 280, map[string]any{"prompt": "Planejar a feature (ou só o escopo afetado, em correção) e propor o plano técnico."}),
		n("ramificar", TFork, 1640, 280, nil),
		n(GuardImplFront, TAgent, 1960, 80, map[string]any{"prompt": "Implementar as subtarefas FRONTEND."}),
		n(GuardReviewFront, TAgent, 2280, 80, map[string]any{"prompt": "Revisar visualmente a interface em navegador isolado."}),
		n(GuardImplBack, TAgent, 1960, 480, map[string]any{"prompt": "Implementar as subtarefas BACKEND."}),
		n(GuardReviewBack, TAgent, 2280, 480, map[string]any{"prompt": "Revisar o Backend (obrigatório em alto risco)."}),
		n("juntar_trilhas", TConverge, 2600, 280, map[string]any{"mode": "all"}),
		n(GuardConsolidate, TGit, 2920, 280, map[string]any{"operation": "commit", "message": "Consolidar commits da feature"}),
		n(GuardIntegrated, TChecks, 3240, 280, map[string]any{"profile": "base"}),
		n(GuardDraftPR, TGit, 3560, 280, map[string]any{"operation": "draft_pr"}),
		n(GuardMergeWait, TWait, 3880, 280, map[string]any{"minutes": float64(1440)}),
		n("fim_ok", TEnd, 4200, 280, map[string]any{"outcome": "success"}),
		n("fim_pendente", TEnd, 680, 680, map[string]any{"outcome": "failure"}),
		n("fim_falha", TEnd, 2600, 680, map[string]any{"outcome": "failure"}),
	}}
	g.Edges = []Edge{
		e("inicio", PortOut, correctionEntry, PortIn),
		e(correctionEntry, "false", GuardPRD, PortIn),
		e(correctionEntry, "true", "pronto_planejar", PortIn),
		e(GuardPRD, "true", "pronto_planejar", PortIn),
		e(GuardPRD, "false", "fim_pendente", PortIn),
		e("pronto_planejar", PortOut, GuardPlanning, PortIn),
		e(GuardPlanning, PortOut, "ramificar", PortIn),
		e("ramificar", "a", GuardImplFront, PortIn),
		e("ramificar", "b", GuardImplBack, PortIn),
		e(GuardImplFront, PortOut, GuardReviewFront, PortIn),
		e(GuardImplBack, PortOut, GuardReviewBack, PortIn),
		e(GuardReviewFront, PortOut, "juntar_trilhas", PortIn),
		e(GuardReviewBack, PortOut, "juntar_trilhas", PortIn),
		e("juntar_trilhas", PortOut, GuardConsolidate, PortIn),
		e(GuardConsolidate, PortOut, GuardIntegrated, PortIn),
		e(GuardIntegrated, PortOut, GuardDraftPR, PortIn),
		e(GuardDraftPR, PortOut, GuardMergeWait, PortIn),
		e(GuardMergeWait, PortOut, "fim_ok", PortIn),
	}
	FillStandardDefaults(&g)
	for _, id := range []string{GuardPlanning, GuardImplFront, GuardReviewFront, GuardImplBack, GuardReviewBack, GuardConsolidate, GuardIntegrated, GuardDraftPR} {
		g.Edges = append(g.Edges, e(id, PortError, "fim_falha", PortIn))
	}
	return g
}

// GuardIssues confere que o grafo mantém as guardas obrigatórias e que nenhum caminho até um fim de
// sucesso as contorna. Pura. Convergência "all" só avança com todos os ramos; a entrada de correção
// (o ramo "true" do nó "entrada") é o único desvio permitido do gate do PRD.
func GuardIssues(g Graph) []Issue {
	var out []Issue
	add := func(node, code, msg string) { out = append(out, Issue{Node: node, Code: code, Message: msg}) }
	byID := map[string]Node{}
	for _, nd := range g.Nodes {
		byID[nd.ID] = nd
	}
	for _, rg := range RequiredGuards {
		nd, ok := byID[rg.ID]
		switch {
		case !ok:
			add(rg.ID, "guard_missing", fmt.Sprintf("A guarda obrigatória %q foi removida.", rg.ID))
		case nd.Type != rg.Type:
			add(rg.ID, "guard_type", fmt.Sprintf("A guarda %q deve ser um nó do tipo %q.", rg.ID, rg.Type))
		}
	}
	if len(out) > 0 {
		return out
	}
	for _, rg := range RequiredGuards {
		skipEdge := func(Edge) bool { return false }
		if rg.ID == GuardPRD {
			skipEdge = func(e Edge) bool { return e.From == correctionEntry && e.FromPort == correctionPortOut }
		}
		if reachesSuccess(g, byID, rg.ID, skipEdge) {
			add(rg.ID, "guard_bypass", fmt.Sprintf("Há um caminho até o sucesso que contorna a guarda %q.", rg.ID))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Node < out[j].Node })
	return out
}

// reachesSuccess diz se, sem passar pelo nó `blocked` nem pelas arestas filtradas, algum fim de
// sucesso é alcançável a partir do início (com a semântica de convergência).
func reachesSuccess(g Graph, byID map[string]Node, blocked string, skip func(Edge) bool) bool {
	in := map[string][]Edge{}
	out := map[string][]Edge{}
	for _, e := range g.Edges {
		if e.From == blocked || e.To == blocked || skip(e) {
			continue
		}
		in[e.To] = append(in[e.To], e)
		out[e.From] = append(out[e.From], e)
	}
	// total de entradas do grafo completo (uma convergência "all" precisa de todas, inclusive as que
	// passariam pelo nó bloqueado, que por isso nunca chegam).
	total := map[string]int{}
	for _, e := range g.Edges {
		total[e.To]++
	}
	reached := map[string]bool{}
	arrived := map[string]int{}
	var queue []string
	for _, nd := range g.Nodes {
		if nd.Type == TStart && nd.ID != blocked {
			reached[nd.ID] = true
			queue = append(queue, nd.ID)
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, e := range out[cur] {
			to := byID[e.To]
			if reached[to.ID] {
				continue
			}
			arrived[to.ID]++
			if to.Type == TConverge {
				if mode, _ := to.Props["mode"].(string); mode == "all" && arrived[to.ID] < total[to.ID] {
					continue
				}
			}
			reached[to.ID] = true
			queue = append(queue, to.ID)
		}
	}
	for _, nd := range g.Nodes {
		if nd.Type == TEnd && reached[nd.ID] {
			if outcome, _ := nd.Props["outcome"].(string); outcome == "success" {
				return true
			}
		}
	}
	return false
}
