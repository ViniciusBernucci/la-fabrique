package workflows

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Estados de uma execução.
const (
	RunRunning          = "running"
	RunWaiting          = "waiting"           // espera durável: retome depois do prazo
	RunAwaitingDecision = "awaiting_decision" // decisão humana pendente
	RunCompleted        = "completed"
	RunFailed           = "failed"
)

// MaxSteps protege contra execuções que não terminam.
const MaxSteps = 10000

// Token é um caminho ativo: o nó a executar e a conexão pela qual chegou.
type Token struct {
	Node string `json:"node"`
	Via  string `json:"via"`
}

// TraceStep registra um passo percorrido.
type TraceStep struct {
	Node string `json:"node"`
	Type string `json:"type"`
	Port string `json:"port,omitempty"`
	Note string `json:"note,omitempty"`
}

// RunState é o estado completo e serializável (JSON) de uma execução: com ele
// e o mesmo grafo congelado, a execução retoma exatamente de onde parou.
type RunState struct {
	Status    string               `json:"status"`
	Outcome   string               `json:"outcome,omitempty"` // success | failure (quando concluída)
	Failure   string               `json:"failure,omitempty"`
	Queue     []Token              `json:"queue"`
	Vars      map[string]any       `json:"vars"`
	Loops     map[string]int       `json:"loops"`
	Arrivals  map[string][]string  `json:"arrivals"`
	Visited   map[string]bool      `json:"visited"`
	WaitUntil map[string]time.Time `json:"wait_until"`
	Pending   []string             `json:"pending"`
	Ends      []string             `json:"ends"` // resultados dos fins alcançados
	Steps     int                  `json:"steps"`
	Trace     []TraceStep          `json:"trace"`
}

// ExecResult é o resultado de um nó de execução. Port vazio vale "out".
type ExecResult struct {
	Port    string
	Outputs map[string]any
}

// Executor executa os nós de trabalho (agente, skill, script, checks, git, docs).
// O motor não conhece como: isso é dos épicos de execução (E09, E11, E13, E15).
type Executor interface {
	Run(ctx context.Context, n Node, vars map[string]any) (ExecResult, error)
}

// Engine percorre o grafo de forma determinística, um nó por vez (escritor
// único por projeto). Não toca banco nem relógio: estado e `Now` são injetados.
type Engine struct {
	Exec Executor
	Now  func() time.Time
}

var nonIdent = regexp.MustCompile(`[^A-Za-z0-9_]`)

// VarKey é a chave sob a qual as saídas de um nó ficam em Vars (o ID com
// símbolos trocados por "_"), além de "last".
func VarKey(nodeID string) string { return nonIdent.ReplaceAllString(nodeID, "_") }

// NewRun cria o estado inicial. O grafo precisa estar válido e achatado
// (sem nós subflow; veja Flatten).
func NewRun(g Graph, vars map[string]any) (*RunState, error) {
	for _, n := range g.Nodes {
		if n.Type == TSubflow {
			return nil, fmt.Errorf("%w: o grafo tem subfluxo; use Flatten antes", ErrSubflowInvalid)
		}
	}
	// Nós inalcançáveis ou sem entrada são aceitos aqui: depois de achatar, um
	// subfluxo que sempre falha deixa a continuação sem caminho. A publicação
	// valida o grafo autoral.
	for _, is := range ValidateFlat(g) {
		if is.Code != "unreachable" && is.Code != "input_unconnected" {
			return nil, fmt.Errorf("grafo inválido: %s (%s)", is.Message, is.Node)
		}
	}
	start := ""
	for _, n := range g.Nodes {
		if n.Type == TStart {
			start = n.ID
		}
	}
	st := &RunState{
		Status: RunRunning, Queue: []Token{{Node: start}}, Vars: map[string]any{}, Loops: map[string]int{},
		Arrivals: map[string][]string{}, Visited: map[string]bool{}, WaitUntil: map[string]time.Time{},
		Pending: []string{}, Ends: []string{}, Trace: []TraceStep{},
	}
	for k, v := range vars {
		st.Vars[k] = v
	}
	return st, nil
}

type index struct {
	nodes map[string]Node
	out   map[string]map[string]Edge // nó → porta → conexão
}

func indexOf(g Graph) index {
	ix := index{nodes: map[string]Node{}, out: map[string]map[string]Edge{}}
	for _, n := range g.Nodes {
		ix.nodes[n.ID] = n
	}
	for _, e := range g.Edges {
		if ix.out[e.From] == nil {
			ix.out[e.From] = map[string]Edge{}
		}
		ix.out[e.From][e.FromPort] = e
	}
	return ix
}

func (st *RunState) fail(format string, a ...any) {
	st.Status, st.Failure = RunFailed, fmt.Sprintf(format, a...)
}

func (st *RunState) emit(ix index, node, port string) bool {
	e, ok := ix.out[node][port]
	if !ok {
		return false
	}
	st.Queue = append(st.Queue, Token{Node: e.To, Via: e.ID})
	return true
}

// Advance executa até a conclusão, uma falha, uma espera ou uma decisão humana.
// Pode ser chamado de novo (com o estado persistido) para retomar.
func (e *Engine) Advance(ctx context.Context, g Graph, st *RunState) error {
	if st.Status == RunCompleted || st.Status == RunFailed {
		return nil
	}
	ix := indexOf(g)
	st.Status = RunRunning
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(st.Queue) > 0 {
			tok := st.Queue[0]
			st.Queue = st.Queue[1:]
			if st.Steps++; st.Steps > MaxSteps {
				st.fail("limite de %d passos excedido", MaxSteps)
				return nil
			}
			e.step(ctx, ix, st, tok)
			if st.Status == RunFailed {
				return nil
			}
			continue
		}
		if len(st.Pending) > 0 {
			st.Status = RunAwaitingDecision
			return nil
		}
		if len(st.WaitUntil) > 0 {
			if e.wakeWaits(ix, st) {
				continue
			}
			st.Status = RunWaiting
			return nil
		}
		if e.fireConverges(ix, st) {
			continue
		}
		break
	}
	if len(st.Ends) == 0 {
		st.fail("o fluxo terminou sem alcançar um nó de fim")
		return nil
	}
	st.Status, st.Outcome = RunCompleted, "success"
	for _, o := range st.Ends {
		if o == "failure" {
			st.Outcome = "failure"
		}
	}
	return nil
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// wakeWaits enfileira as esperas vencidas (na ordem do ID) e informa se houve alguma.
func (e *Engine) wakeWaits(ix index, st *RunState) bool {
	ids := make([]string, 0, len(st.WaitUntil))
	for id := range st.WaitUntil {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	woke := false
	for _, id := range ids {
		if !e.now().Before(st.WaitUntil[id]) {
			delete(st.WaitUntil, id)
			st.Trace = append(st.Trace, TraceStep{Node: id, Type: TWait, Port: PortOut, Note: "espera concluída"})
			st.emit(ix, id, PortOut)
			woke = true
		}
	}
	return woke
}

// fireConverges libera convergências cujas chegadas se acumularam e já não há
// caminhos em andamento: convergência "all" segue assim que todos os caminhos
// ATIVOS chegaram; caminhos não percorridos não são esperados.
func (e *Engine) fireConverges(ix index, st *RunState) bool {
	ids := make([]string, 0, len(st.Arrivals))
	for id, a := range st.Arrivals {
		if len(a) > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return false
	}
	id := ids[0]
	n := len(st.Arrivals[id])
	delete(st.Arrivals, id)
	st.Trace = append(st.Trace, TraceStep{Node: id, Type: TConverge, Port: PortOut, Note: fmt.Sprintf("convergiu %d caminho(s)", n)})
	st.emit(ix, id, PortOut)
	return true
}

func (e *Engine) step(ctx context.Context, ix index, st *RunState, tok Token) {
	n, ok := ix.nodes[tok.Node]
	if !ok {
		st.fail("nó %q não existe", tok.Node)
		return
	}
	st.Visited[n.ID] = true
	trace := func(port, note string) {
		st.Trace = append(st.Trace, TraceStep{Node: n.ID, Type: n.Type, Port: port, Note: note})
	}
	route := func(port, note string) {
		trace(port, note)
		if !st.emit(ix, n.ID, port) && port != PortError {
			st.fail("o nó %q não tem conexão na saída %q", n.ID, port)
		}
	}
	switch n.Type {
	case TStart:
		route(PortOut, "")
	case TEnd:
		outcome, _ := n.Props["outcome"].(string)
		st.Ends = append(st.Ends, outcome)
		trace("", "fim: "+outcome)
	case TAgent, TSkill, TScript, TChecks, TGit, TDocs:
		res, err := e.runExec(ctx, n, st)
		if err != nil || res.Port == PortError {
			note := "falha"
			if err != nil {
				note = "falha: " + err.Error()
			}
			if _, has := ix.out[n.ID][PortError]; !has {
				trace(PortError, note)
				st.fail("o nó %q falhou e não tem rota de erro", n.ID)
				return
			}
			route(PortError, note)
			return
		}
		route(PortOut, "")
	case TCondition:
		expr, err := ParseExpr(propString(n, "expression"))
		if err == nil {
			var v bool
			if v, err = expr.Eval(st.Vars); err == nil {
				if v {
					route("true", "")
				} else {
					route("false", "")
				}
				return
			}
		}
		st.fail("condição do nó %q: %v", n.ID, err)
	case TLoop:
		expr, err := ParseExpr(propString(n, "condition"))
		if err != nil {
			st.fail("condição do loop %q: %v", n.ID, err)
			return
		}
		cont, err := expr.Eval(st.Vars)
		if err != nil {
			st.fail("condição do loop %q: %v", n.ID, err)
			return
		}
		max := propInt(n, "max_iterations")
		switch {
		case cont && st.Loops[n.ID] < max:
			st.Loops[n.ID]++
			route("body", fmt.Sprintf("iteração %d de %d", st.Loops[n.ID], max))
		case cont:
			st.Vars["loop_"+VarKey(n.ID)+"_exhausted"] = true
			route("done", fmt.Sprintf("limite de %d iterações atingido", max))
		default:
			route("done", "")
		}
	case TWait:
		st.WaitUntil[n.ID] = e.now().Add(time.Duration(propInt(n, "minutes")) * time.Minute)
		trace("", "aguardando")
	case THuman:
		st.Pending = append(st.Pending, n.ID)
		trace("", "decisão humana pendente")
	case TFork:
		trace("a,b", "")
		st.emit(ix, n.ID, "a")
		st.emit(ix, n.ID, "b")
	case TConverge:
		if propString(n, "mode") == "any" {
			route(PortOut, "")
			return
		}
		st.Arrivals[n.ID] = append(st.Arrivals[n.ID], tok.Via)
		trace("", "chegada registrada")
	default:
		st.fail("tipo de nó %q não é executável", n.Type)
	}
}

func (e *Engine) runExec(ctx context.Context, n Node, st *RunState) (ExecResult, error) {
	if e.Exec == nil {
		return ExecResult{}, errors.New("sem executor configurado")
	}
	res, err := e.Exec.Run(ctx, n, st.Vars)
	if err != nil {
		return res, err
	}
	if res.Outputs != nil {
		st.Vars[VarKey(n.ID)] = res.Outputs
		st.Vars["last"] = res.Outputs
	}
	return res, nil
}

// Decide registra a decisão humana de um nó pendente ("approved" ou "rejected")
// e deixa a execução pronta para Advance. Decisão repetida ou para nó sem
// pendência é recusada.
func (e *Engine) Decide(g Graph, st *RunState, node, choice string) error {
	if choice != "approved" && choice != "rejected" {
		return fmt.Errorf("decisão inválida %q", choice)
	}
	at := -1
	for i, p := range st.Pending {
		if p == node {
			at = i
		}
	}
	if at < 0 {
		return fmt.Errorf("o nó %q não tem decisão pendente", node)
	}
	st.Pending = append(st.Pending[:at:at], st.Pending[at+1:]...)
	ix := indexOf(g)
	st.Trace = append(st.Trace, TraceStep{Node: node, Type: THuman, Port: choice, Note: "decisão registrada"})
	if !st.emit(ix, node, choice) {
		st.fail("o nó %q não tem conexão na saída %q", node, choice)
	}
	st.Status = RunRunning
	return nil
}

func propString(n Node, k string) string {
	s, _ := n.Props[k].(string)
	return strings.TrimSpace(s)
}

func propInt(n Node, k string) int {
	switch v := n.Props[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}
