package resume

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// Erros.
var (
	ErrUnknownNode   = errors.New("o nó não existe no workflow congelado da execução")
	ErrNotRerunnable = errors.New("só nós de trabalho já percorridos podem ser reexecutados")
	ErrRerunFailed   = errors.New("a reexecução do nó falhou: o estado anterior foi mantido")
	ErrNoExecutor    = errors.New("sem executor para reexecutar o nó")
)

var executable = map[string]bool{workflows.TAgent: true, workflows.TSkill: true, workflows.TScript: true, workflows.TChecks: true, workflows.TGit: true, workflows.TDocs: true}

// Selection é o conjunto afetado por reexecutar um nó.
type Selection struct {
	Node     string   `json:"node"`
	Affected []string `json:"affected"` // descendentes (pelo grafo congelado), se a saída mudar
	Reusable []string `json:"reusable"` // nós percorridos que NÃO são afetados
}

// Select calcula o conjunto afetado a partir do grafo congelado e do estado.
func Select(g workflows.Graph, st workflows.RunState, node string) (Selection, error) {
	by := map[string]workflows.Node{}
	out := map[string][]string{}
	for _, n := range g.Nodes {
		by[n.ID] = n
	}
	for _, e := range g.Edges {
		out[e.From] = append(out[e.From], e.To)
	}
	n, ok := by[node]
	if !ok {
		return Selection{}, fmt.Errorf("%w: %q", ErrUnknownNode, node)
	}
	if !executable[n.Type] || !st.Visited[node] {
		return Selection{}, fmt.Errorf("%w: %q", ErrNotRerunnable, node)
	}
	seen := map[string]bool{node: true}
	queue := []string{node}
	aff := []string{}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		next := append([]string{}, out[cur]...)
		sort.Strings(next)
		for _, nx := range next {
			if !seen[nx] {
				seen[nx] = true
				aff = append(aff, nx)
				queue = append(queue, nx)
			}
		}
	}
	sort.Strings(aff)
	reusable := []string{}
	for id := range st.Visited {
		if st.Visited[id] && !seen[id] {
			reusable = append(reusable, id)
		}
	}
	sort.Strings(reusable)
	return Selection{Node: node, Affected: aff, Reusable: reusable}, nil
}

func hashJSON(v any) string {
	b, _ := json.Marshal(v)
	var g any
	if json.Unmarshal(b, &g) == nil {
		b, _ = json.Marshal(g)
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// RerunResult descreve o que a reexecução fez.
type RerunResult struct {
	OutputChanged bool     `json:"output_changed"`
	Invalidated   []string `json:"invalidated"` // descendentes que voltam a rodar
	Reused        []string `json:"reused"`      // percorridos e mantidos como estavam
}

func cloneState(st workflows.RunState) workflows.RunState {
	b, _ := json.Marshal(st)
	var out workflows.RunState
	_ = json.Unmarshal(b, &out)
	if out.Vars == nil {
		out.Vars = map[string]any{}
	}
	if out.Loops == nil {
		out.Loops = map[string]int{}
	}
	if out.Arrivals == nil {
		out.Arrivals = map[string][]string{}
	}
	if out.Visited == nil {
		out.Visited = map[string]bool{}
	}
	if out.WaitUntil == nil {
		out.WaitUntil = map[string]time.Time{}
	}
	return out
}

// Rerun refaz SÓ o nó escolhido. O estado de entrada não é alterado; devolve o novo.
// Saída igual: dependentes reaproveitados. Saída diferente: só os descendentes são
// invalidados e a fila é retomada logo após o nó, para o motor refazê-los.
func Rerun(ctx context.Context, exec workflows.Executor, g workflows.Graph, st workflows.RunState, node string) (workflows.RunState, RerunResult, error) {
	if exec == nil {
		return st, RerunResult{}, ErrNoExecutor
	}
	sel, err := Select(g, st, node)
	if err != nil {
		return st, RerunResult{}, err
	}
	var n workflows.Node
	for _, x := range g.Nodes {
		if x.ID == node {
			n = x
		}
	}
	key := workflows.VarKey(node)
	oldHash := hashJSON(st.Vars[key])
	res, err := exec.Run(ctx, n, st.Vars)
	if err != nil || res.Port == workflows.PortError {
		return st, RerunResult{}, fmt.Errorf("%w: %v", ErrRerunFailed, err)
	}
	next := cloneState(st)
	out := RerunResult{Invalidated: []string{}, Reused: append([]string{}, sel.Reusable...)}
	if res.Outputs != nil {
		next.Vars[key] = res.Outputs
	}
	if hashJSON(next.Vars[key]) == oldHash {
		next.Trace = append(next.Trace, workflows.TraceStep{Node: node, Type: n.Type, Port: workflows.PortOut, Note: "reexecutado: saída igual, dependentes mantidos"})
		return next, out, nil
	}
	out.OutputChanged = true
	invalid := map[string]bool{}
	for _, d := range sel.Affected {
		invalid[d] = true
	}
	byID := map[string]workflows.Node{}
	for _, x := range g.Nodes {
		byID[x.ID] = x
	}
	for _, d := range sel.Affected {
		if next.Visited[d] {
			out.Invalidated = append(out.Invalidated, d)
		}
		delete(next.Visited, d)
		delete(next.Vars, workflows.VarKey(d))
		delete(next.Vars, "loop_"+workflows.VarKey(d)+"_exhausted")
		delete(next.Arrivals, d)
		delete(next.Loops, d)
		delete(next.WaitUntil, d)
	}
	// Reuse = percorridos que não foram invalidados.
	out.Reused = []string{}
	for id, v := range next.Visited {
		if v && id != node {
			out.Reused = append(out.Reused, id)
		}
	}
	sort.Strings(out.Reused)
	sort.Strings(out.Invalidated)
	q := next.Queue[:0:0]
	for _, t := range next.Queue {
		if !invalid[t.Node] {
			q = append(q, t)
		}
	}
	pend := []string{}
	for _, p := range next.Pending {
		if !invalid[p] {
			pend = append(pend, p)
		}
	}
	next.Pending = pend
	// Fins ainda válidos: um por nó de fim percorrido e não invalidado.
	ends := []string{}
	ids := make([]string, 0, len(next.Visited))
	for id := range next.Visited {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if byID[id].Type == workflows.TEnd {
			o, _ := byID[id].Props["outcome"].(string)
			ends = append(ends, o)
		}
	}
	next.Ends = ends
	// Continua logo após o nó: o motor só refaz o que foi invalidado.
	for _, e := range g.Edges {
		if e.From == node && e.FromPort == workflows.PortOut {
			q = append(q, workflows.Token{Node: e.To, Via: e.ID})
		}
	}
	next.Queue = q
	next.Status, next.Outcome, next.Failure = workflows.RunRunning, "", ""
	next.Trace = append(next.Trace, workflows.TraceStep{Node: node, Type: n.Type, Port: workflows.PortOut,
		Note: fmt.Sprintf("reexecutado: saída mudou, %d dependente(s) invalidado(s)", len(out.Invalidated))})
	return next, out, nil
}
