package workflows

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

var toolItem = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,79}$`)
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Validate verifica o grafo e devolve todos os problemas com nó e campo. Um
// rascunho pode ser salvo inválido; só a publicação (T07-03) exige zero problemas.
func Validate(g Graph) []Issue { return validate(g, idPattern) }

// flatIDPattern aceita os IDs gerados por Flatten ("<subfluxo>/<nó>").
var flatIDPattern = regexp.MustCompile(`^[A-Za-z0-9_/-]{1,512}$`)

// ValidateFlat valida um grafo já achatado (IDs prefixados por Flatten).
func ValidateFlat(g Graph) []Issue { return validate(g, flatIDPattern) }

func validate(g Graph, idRe *regexp.Regexp) []Issue {
	var issues []Issue
	add := func(node, field, code, msg string) {
		issues = append(issues, Issue{Node: node, Field: field, Code: code, Message: msg})
	}
	if len(g.Nodes) > MaxNodes {
		add("", "", "too_many_nodes", fmt.Sprintf("O fluxo tem mais de %d nós.", MaxNodes))
		return issues
	}
	if len(g.Edges) > MaxEdges {
		add("", "", "too_many_edges", fmt.Sprintf("O fluxo tem mais de %d conexões.", MaxEdges))
		return issues
	}

	byID := map[string]Node{}
	starts, ends := 0, 0
	for _, n := range g.Nodes {
		if !idRe.MatchString(n.ID) {
			add(n.ID, "", "invalid_id", "Identificador de nó inválido.")
			continue
		}
		if _, dup := byID[n.ID]; dup {
			add(n.ID, "", "duplicate_id", "Há dois nós com o mesmo identificador.")
			continue
		}
		byID[n.ID] = n
		if math.IsNaN(n.X) || math.IsInf(n.X, 0) || math.IsNaN(n.Y) || math.IsInf(n.Y, 0) {
			add(n.ID, "", "invalid_position", "Posição inválida.")
		}
		nt, ok := TypeOf(n.Type)
		if !ok {
			add(n.ID, "type", "unknown_type", "Tipo de nó desconhecido.")
			continue
		}
		switch n.Type {
		case TStart:
			starts++
		case TEnd:
			ends++
		}
		validateProps(n, nt, add)
	}
	if starts != 1 {
		add("", "", "start_count", "O fluxo precisa de exatamente um nó de início.")
	}
	if ends == 0 {
		add("", "", "end_missing", "O fluxo precisa de ao menos um nó de fim.")
	}

	// Conexões.
	outUsed := map[string]int{}
	inUsed := map[string]int{}
	seenEdge := map[string]bool{}
	valid := make([]Edge, 0, len(g.Edges))
	for _, e := range g.Edges {
		from, okF := byID[e.From]
		to, okT := byID[e.To]
		if !okF || !okT {
			add(firstNonEmpty(e.From, e.To), "", "edge_dangling", "Uma conexão aponta para um nó que não existe.")
			continue
		}
		ft, _ := TypeOf(from.Type)
		tt, _ := TypeOf(to.Type)
		if !contains(ft.Outputs, e.FromPort) {
			add(from.ID, e.FromPort, "invalid_port", fmt.Sprintf("O nó %q não tem a saída %q.", ft.Label, e.FromPort))
			continue
		}
		if !contains(tt.Inputs, e.ToPort) {
			add(to.ID, e.ToPort, "invalid_port", fmt.Sprintf("O nó %q não tem a entrada %q.", tt.Label, e.ToPort))
			continue
		}
		if e.From == e.To {
			add(from.ID, e.FromPort, "self_edge", "Um nó não pode se ligar a si mesmo.")
			continue
		}
		key := e.From + "|" + e.FromPort + "|" + e.To + "|" + e.ToPort
		if seenEdge[key] {
			add(from.ID, e.FromPort, "duplicate_edge", "Conexão duplicada.")
			continue
		}
		seenEdge[key] = true
		outUsed[e.From+"|"+e.FromPort]++
		inUsed[e.To+"|"+e.ToPort]++
		valid = append(valid, e)
	}
	for _, n := range g.Nodes {
		nt, ok := TypeOf(n.Type)
		if !ok || byID[n.ID].ID != n.ID {
			continue
		}
		for _, p := range nt.Outputs {
			c := outUsed[n.ID+"|"+p]
			switch {
			case c > 1:
				add(n.ID, p, "output_fanout", fmt.Sprintf("A saída %q só pode ter uma conexão; use um nó de condição para ramificar.", p))
			case c == 0 && p != PortError:
				add(n.ID, p, "output_unconnected", fmt.Sprintf("A saída %q não está conectada.", p))
			}
		}
		for _, p := range nt.Inputs {
			c := inUsed[n.ID+"|"+p]
			switch {
			case c == 0:
				add(n.ID, p, "input_unconnected", fmt.Sprintf("A entrada %q não está conectada.", p))
			case c > 1 && !nt.MultiIn:
				add(n.ID, p, "input_multiple", "Esta entrada aceita uma única conexão; use um nó de convergência.")
			}
		}
	}

	issues = append(issues, graphIssues(g, byID, valid)...)
	sortIssues(issues)
	return issues
}

func validateProps(n Node, nt NodeType, add func(node, field, code, msg string)) {
	known := map[string]bool{}
	for _, f := range nt.Fields {
		known[f.Name] = true
		raw, present := n.Props[f.Name]
		text := ""
		switch v := raw.(type) {
		case string:
			text = strings.TrimSpace(v)
		case float64:
			text = fmt.Sprint(v)
		case nil:
		default:
			add(n.ID, f.Name, "invalid_value", "Valor com tipo inválido.")
			continue
		}
		if !present || text == "" {
			if f.Required {
				add(n.ID, f.Name, "required", fmt.Sprintf("O campo %q é obrigatório.", f.Label))
			}
			continue
		}
		if len(text) > 4000 {
			add(n.ID, f.Name, "too_long", fmt.Sprintf("O campo %q é longo demais.", f.Label))
			continue
		}
		switch f.Kind {
		case KindInt:
			v, isNum := raw.(float64)
			if !isNum || v != math.Trunc(v) || (f.Min != 0 && v < float64(f.Min)) || (f.Max != 0 && v > float64(f.Max)) {
				add(n.ID, f.Name, "out_of_range", fmt.Sprintf("O campo %q deve ser um inteiro entre %d e %d.", f.Label, f.Min, f.Max))
			}
		case KindEnum:
			if !contains(f.Options, text) {
				add(n.ID, f.Name, "invalid_option", fmt.Sprintf("O campo %q deve ser um de: %s.", f.Label, strings.Join(f.Options, ", ")))
			}
		}
	}
	// Expressões precisam ser analisáveis (a avaliação em tempo de execução é do motor).
	for _, f := range []string{"expression", "condition"} {
		if (n.Type == TCondition && f == "expression") || (n.Type == TLoop && f == "condition") {
			if src, _ := n.Props[f].(string); strings.TrimSpace(src) != "" {
				if _, err := ParseExpr(src); err != nil {
					add(n.ID, f, "invalid_expression", err.Error())
				}
			}
		}
	}
	if n.Type == TAgent {
		if c, _ := n.Props["connection"].(string); strings.TrimSpace(c) != "" && !uuidPattern.MatchString(strings.TrimSpace(c)) {
			add(n.ID, "connection", "invalid_value", "A conta deve ser identificada por um ID de conexão válido.")
		}
		if t, _ := n.Props["tools"].(string); strings.TrimSpace(t) != "" {
			for _, item := range strings.Split(t, ",") {
				if !toolItem.MatchString(strings.TrimSpace(item)) {
					add(n.ID, "tools", "invalid_value", "Ferramenta inválida na lista (use nomes separados por vírgula).")
					break
				}
			}
		}
	}
	if n.Type == TSubflow {
		if id, _ := n.Props["workflow_id"].(string); strings.TrimSpace(id) != "" && !uuidPattern.MatchString(strings.TrimSpace(id)) {
			add(n.ID, "workflow_id", "invalid_value", "O subfluxo deve ser identificado por um ID de workflow válido.")
		}
	}
	for k := range n.Props {
		if !known[k] {
			add(n.ID, k, "unknown_field", "Propriedade desconhecida para este tipo de nó.")
		}
	}
}

// graphIssues detecta ciclos sem loop e nós inalcançáveis a partir do início.
func graphIssues(g Graph, byID map[string]Node, edges []Edge) []Issue {
	var issues []Issue
	// Aresta de retorno de loop (saída "body") não conta para ciclos: só um loop com
	// limite pode fechar um ciclo.
	adj := map[string][]string{}
	full := map[string][]string{}
	for _, e := range edges {
		full[e.From] = append(full[e.From], e.To)
		if byID[e.From].Type == TLoop && e.FromPort == "body" {
			continue
		}
		adj[e.From] = append(adj[e.From], e.To)
	}
	const (
		white = iota
		gray
		black
	)
	color := map[string]int{}
	cyclic := map[string]bool{}
	var dfs func(string)
	dfs = func(u string) {
		color[u] = gray
		for _, v := range adj[u] {
			switch color[v] {
			case white:
				dfs(v)
			case gray:
				cyclic[v] = true
			}
		}
		color[u] = black
	}
	for _, n := range g.Nodes {
		if _, ok := byID[n.ID]; ok && color[n.ID] == white {
			dfs(n.ID)
		}
	}
	for id := range cyclic {
		issues = append(issues, Issue{Node: id, Code: "cycle_without_loop", Message: "Há um ciclo que não passa por um nó de loop com limite."})
	}
	// Alcançabilidade (inclui o retorno do loop).
	var start string
	for _, n := range g.Nodes {
		if n.Type == TStart {
			start = n.ID
		}
	}
	if start != "" {
		seen := map[string]bool{start: true}
		stack := []string{start}
		for len(stack) > 0 {
			u := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, v := range full[u] {
				if !seen[v] {
					seen[v] = true
					stack = append(stack, v)
				}
			}
		}
		for _, n := range g.Nodes {
			if _, ok := byID[n.ID]; ok && !seen[n.ID] {
				issues = append(issues, Issue{Node: n.ID, Code: "unreachable", Message: "Este nó não é alcançado a partir do início."})
			}
		}
	}
	return issues
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
