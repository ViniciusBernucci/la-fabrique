package workflows

import (
	"errors"
	"fmt"
	"strings"
)

// Limites e erros da expansão de subfluxos.
const MaxSubflowDepth = 5

var (
	// ErrSubflowRecursive: o subfluxo (direta ou indiretamente) chama a si mesmo.
	ErrSubflowRecursive = errors.New("subfluxo recursivo")
	// ErrSubflowDepth: profundidade acima de MaxSubflowDepth.
	ErrSubflowDepth = errors.New("subfluxos aninhados demais")
	// ErrSubflowInvalid: o subfluxo referenciado não existe ou tem problemas.
	ErrSubflowInvalid = errors.New("subfluxo inválido")
)

// Resolver devolve o grafo de um subfluxo pelo ID.
type Resolver func(id string) (Graph, error)

// Flatten expande os nós `subflow` em linha, produzindo um grafo único que o
// motor executa. Assim o estado de uma execução é um só e pode ser persistido.
// rootID identifica o workflow raiz para detectar recursão. Cada nó interno
// ganha o prefixo "<id do nó subflow>/". O início do subfluxo vira uma
// convergência de passagem; um fim de sucesso continua na saída `out` do nó
// subfluxo; um fim de falha continua em `error` se existir, senão permanece
// como fim de falha.
func Flatten(root Graph, rootID string, resolve Resolver) (Graph, error) {
	return expand(root, "", []string{rootID}, resolve)
}

func expand(g Graph, prefix string, stack []string, resolve Resolver) (Graph, error) {
	out := Graph{Nodes: []Node{}, Edges: []Edge{}}
	subflows := map[string]Node{}
	for _, n := range g.Nodes {
		if n.Type == TSubflow {
			subflows[n.ID] = n
			continue
		}
		c := n
		c.ID = prefix + n.ID
		out.Nodes = append(out.Nodes, c)
	}
	edgeSeq := 0
	newEdge := func(from, fp, to, tp string) Edge {
		edgeSeq++
		return Edge{ID: fmt.Sprintf("%sx%d", prefix, edgeSeq), From: from, FromPort: fp, To: to, ToPort: tp}
	}
	// Entradas e saídas já expandidas de cada nó subflow.
	type ends struct {
		entry    string
		ok, fail []string
	}
	expanded := map[string]ends{}
	for id, sn := range subflows {
		wid, _ := sn.Props["workflow_id"].(string)
		wid = strings.TrimSpace(wid)
		for _, s := range stack {
			if s == wid {
				return Graph{}, fmt.Errorf("%w: %s chama %s", ErrSubflowRecursive, strings.Join(stack, " → "), wid)
			}
		}
		if len(stack) > MaxSubflowDepth {
			return Graph{}, fmt.Errorf("%w (máximo %d)", ErrSubflowDepth, MaxSubflowDepth)
		}
		sub, err := resolve(wid)
		if err != nil {
			return Graph{}, fmt.Errorf("%w: %s: %v", ErrSubflowInvalid, wid, err)
		}
		if issues := Validate(sub); len(issues) > 0 {
			return Graph{}, fmt.Errorf("%w: %s tem %d problema(s)", ErrSubflowInvalid, wid, len(issues))
		}
		inner, err := expand(sub, prefix+id+"/", append(append([]string{}, stack...), wid), resolve)
		if err != nil {
			return Graph{}, err
		}
		var e ends
		for _, n := range inner.Nodes {
			switch n.Type {
			case TStart:
				e.entry = n.ID
				n.Type, n.Props = TConverge, map[string]any{"mode": "any"}
			case TEnd:
				if n.Props["outcome"] == "failure" {
					e.fail = append(e.fail, n.ID)
				} else {
					e.ok = append(e.ok, n.ID)
					n.Type, n.Props = TConverge, map[string]any{"mode": "any"}
				}
			}
			out.Nodes = append(out.Nodes, n)
		}
		out.Edges = append(out.Edges, inner.Edges...)
		expanded[id] = e
	}
	// Religa as conexões do nível atual.
	for _, e := range g.Edges {
		from, fp, to, tp := e.From, e.FromPort, e.To, e.ToPort
		_, fromSub := subflows[from]
		_, toSub := subflows[to]
		switch {
		case toSub:
			tp = PortIn
			to = expanded[to].entry
		default:
			to = prefix + to
		}
		if !fromSub {
			from = prefix + from
			out.Edges = append(out.Edges, Edge{ID: prefix + e.ID, From: from, FromPort: fp, To: to, ToPort: tp})
			continue
		}
		// Saída de um nó subflow: cada fim do subfluxo continua no destino.
		ex := expanded[e.From]
		switch fp {
		case PortOut:
			for _, end := range ex.ok {
				out.Edges = append(out.Edges, newEdge(end, PortOut, to, tp))
			}
		case PortError:
			for _, end := range ex.fail {
				out.Edges = append(out.Edges, newEdge(end, PortOut, to, tp))
			}
			markHandled(&out, ex.fail)
		}
	}
	return out, nil
}

// markHandled transforma fins de falha tratados (com rota de erro) em
// convergências de passagem, para que continuem em vez de terminar o fluxo.
func markHandled(g *Graph, ids []string) {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	for i := range g.Nodes {
		if set[g.Nodes[i].ID] {
			g.Nodes[i].Type, g.Nodes[i].Props = TConverge, map[string]any{"mode": "any"}
		}
	}
}
