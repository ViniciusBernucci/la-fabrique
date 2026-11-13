package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

type stubExec struct {
	calls []string
	fn    map[string]func(n int) (ExecResult, error)
	count map[string]int
}

func (s *stubExec) Run(_ context.Context, n Node, _ map[string]any) (ExecResult, error) {
	if s.count == nil {
		s.count = map[string]int{}
	}
	s.count[n.ID]++
	s.calls = append(s.calls, n.ID)
	if f, ok := s.fn[n.ID]; ok {
		return f(s.count[n.ID])
	}
	return ExecResult{}, nil
}

func agent(id string) Node {
	return n(id, TAgent, map[string]any{"agent": "a", "prompt": "p"})
}

func end(id, outcome string) Node { return n(id, TEnd, map[string]any{"outcome": outcome}) }

func run(t *testing.T, g Graph, vars map[string]any, ex Executor, now func() time.Time) (*RunState, *Engine) {
	t.Helper()
	st, err := NewRun(g, vars)
	if err != nil {
		t.Fatal(err)
	}
	eng := &Engine{Exec: ex, Now: now}
	if err := eng.Advance(context.Background(), g, st); err != nil {
		t.Fatal(err)
	}
	return st, eng
}

func walked(st *RunState) string {
	var ids []string
	for _, s := range st.Trace {
		ids = append(ids, s.Node)
	}
	return strings.Join(ids, ",")
}

func branchGraph() Graph {
	return Graph{
		Nodes: []Node{
			n("s", TStart, nil), n("c", TCondition, map[string]any{"expression": "go_left"}),
			agent("left"), agent("right"),
			n("join", TConverge, map[string]any{"mode": "any"}), end("ok", "success"),
		},
		Edges: []Edge{
			e("1", "s", PortOut, "c", PortIn), e("2", "c", "true", "left", PortIn), e("3", "c", "false", "right", PortIn),
			e("4", "left", PortOut, "join", PortIn), e("5", "right", PortOut, "join", PortIn), e("6", "join", PortOut, "ok", PortIn),
		},
	}
}

func TestConditionTakesEachBranchAndConverges(t *testing.T) {
	for goLeft, want := range map[bool]string{true: "s,c,left,join,ok", false: "s,c,right,join,ok"} {
		ex := &stubExec{}
		st, _ := run(t, branchGraph(), map[string]any{"go_left": goLeft}, ex, nil)
		if st.Status != RunCompleted || st.Outcome != "success" || walked(st) != want {
			t.Errorf("go_left=%v: %s %s %q", goLeft, st.Status, st.Outcome, walked(st))
		}
		if len(ex.calls) != 1 {
			t.Errorf("só o ramo escolhido executa: %v", ex.calls)
		}
	}
}

func loopGraph() Graph {
	return Graph{
		Nodes: []Node{
			n("s", TStart, nil),
			n("loop", TLoop, map[string]any{"max_iterations": float64(3), "condition": "last.passed == false || last == null"}),
			agent("dev"), n("chk", TChecks, map[string]any{"commands": "make test", "profile": "go"}),
			end("ok", "success"),
		},
		Edges: []Edge{
			e("1", "s", PortOut, "loop", PortIn), e("2", "loop", "body", "dev", PortIn), e("3", "dev", PortOut, "chk", PortIn),
			e("4", "chk", PortOut, "loop", PortIn), e("5", "loop", "done", "ok", PortIn),
		},
	}
}

func TestBoundedLoopStopsAtLimitOrWhenConditionClears(t *testing.T) {
	failing := &stubExec{fn: map[string]func(int) (ExecResult, error){
		"chk": func(int) (ExecResult, error) { return ExecResult{Outputs: map[string]any{"passed": false}}, nil },
	}}
	st, _ := run(t, loopGraph(), nil, failing, nil)
	if st.Status != RunCompleted || st.Loops["loop"] != 3 || failing.count["dev"] != 3 || st.Vars["loop_loop_exhausted"] != true {
		t.Fatalf("o loop deve parar no limite de 3: %+v calls=%v", st.Loops, failing.calls)
	}
	exhausted := false
	for _, s := range st.Trace {
		exhausted = exhausted || strings.Contains(s.Note, "limite de 3 iterações")
	}
	if !exhausted {
		t.Fatal("o limite atingido deve constar no rastro")
	}

	passing := &stubExec{fn: map[string]func(int) (ExecResult, error){
		"chk": func(n int) (ExecResult, error) { return ExecResult{Outputs: map[string]any{"passed": n >= 2}}, nil },
	}}
	st, _ = run(t, loopGraph(), nil, passing, nil)
	if st.Loops["loop"] != 2 || passing.count["dev"] != 2 || st.Vars["loop_loop_exhausted"] != nil {
		t.Fatalf("o loop deve terminar quando a condição deixa de valer: %+v", st.Loops)
	}
}

func TestLoopWithoutLimitIsBlockedBeforeRunning(t *testing.T) {
	g := loopGraph()
	delete(g.Nodes[1].Props, "max_iterations")
	if _, err := NewRun(g, nil); err == nil || !strings.Contains(err.Error(), "max_iterations") && !strings.Contains(err.Error(), "obrigat") {
		t.Fatalf("loop sem limite deve ser recusado: %v", err)
	}
	g = loopGraph()
	g.Nodes[1].Props["max_iterations"] = float64(1000)
	if _, err := NewRun(g, nil); err == nil {
		t.Fatal("limite acima do máximo deve ser recusado")
	}
}

func waitGraph() Graph {
	return Graph{
		Nodes: []Node{n("s", TStart, nil), n("w", TWait, map[string]any{"minutes": float64(5)}), agent("after"), end("ok", "success")},
		Edges: []Edge{e("1", "s", PortOut, "w", PortIn), e("2", "w", PortOut, "after", PortIn), e("3", "after", PortOut, "ok", PortIn)},
	}
}

func TestDurableWaitResumesFromPersistedState(t *testing.T) {
	t0 := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	ex := &stubExec{}
	st, _ := run(t, waitGraph(), nil, ex, func() time.Time { return t0 })
	if st.Status != RunWaiting || len(ex.calls) != 0 {
		t.Fatalf("deve aguardar sem executar o que vem depois: %s %v", st.Status, ex.calls)
	}

	// Persiste e recarrega (outro processo, outro motor).
	raw, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	var loaded RunState
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*st, loaded) {
		t.Fatalf("o estado deve sobreviver ao JSON:\n%+v\n%+v", *st, loaded)
	}

	early := &Engine{Exec: ex, Now: func() time.Time { return t0.Add(4 * time.Minute) }}
	if err := early.Advance(context.Background(), waitGraph(), &loaded); err != nil || loaded.Status != RunWaiting || len(ex.calls) != 0 {
		t.Fatalf("antes do prazo continua aguardando: %s %v", loaded.Status, err)
	}
	due := &Engine{Exec: ex, Now: func() time.Time { return t0.Add(5 * time.Minute) }}
	if err := due.Advance(context.Background(), waitGraph(), &loaded); err != nil || loaded.Status != RunCompleted || !reflect.DeepEqual(ex.calls, []string{"after"}) {
		t.Fatalf("no prazo retoma e conclui: %s %v %v", loaded.Status, ex.calls, err)
	}
	// Concluída não executa de novo.
	if err := due.Advance(context.Background(), waitGraph(), &loaded); err != nil || len(ex.calls) != 1 {
		t.Fatalf("execução concluída não pode rodar outra vez: %v", ex.calls)
	}
}

func humanGraph() Graph {
	return Graph{
		Nodes: []Node{n("s", TStart, nil), n("h", THuman, map[string]any{"question": "ok?"}), end("good", "success"), end("bad", "failure")},
		Edges: []Edge{e("1", "s", PortOut, "h", PortIn), e("2", "h", "approved", "good", PortIn), e("3", "h", "rejected", "bad", PortIn)},
	}
}

func TestHumanDecisionPausesAndResumesOnce(t *testing.T) {
	for choice, outcome := range map[string]string{"approved": "success", "rejected": "failure"} {
		st, eng := run(t, humanGraph(), nil, nil, nil)
		if st.Status != RunAwaitingDecision || !reflect.DeepEqual(st.Pending, []string{"h"}) {
			t.Fatalf("deve aguardar a decisão: %s %v", st.Status, st.Pending)
		}
		if err := eng.Decide(humanGraph(), st, "h", "talvez"); err == nil {
			t.Fatal("decisão inválida deve ser recusada")
		}
		if err := eng.Decide(humanGraph(), st, "h", choice); err != nil {
			t.Fatal(err)
		}
		if err := eng.Decide(humanGraph(), st, "h", choice); err == nil {
			t.Fatal("decisão repetida deve ser recusada")
		}
		if err := eng.Advance(context.Background(), humanGraph(), st); err != nil || st.Status != RunCompleted || st.Outcome != outcome {
			t.Fatalf("%s: %s %s %v", choice, st.Status, st.Outcome, err)
		}
	}
}

func forkGraph(mode string) Graph {
	return Graph{
		Nodes: []Node{
			n("s", TStart, nil), n("f", TFork, nil), agent("A"), agent("B"),
			n("j", TConverge, map[string]any{"mode": mode}), agent("after"), end("ok", "success"),
		},
		Edges: []Edge{
			e("1", "s", PortOut, "f", PortIn), e("2", "f", "a", "A", PortIn), e("3", "f", "b", "B", PortIn),
			e("4", "A", PortOut, "j", PortIn), e("5", "B", PortOut, "j", PortIn), e("6", "j", PortOut, "after", PortIn),
			e("7", "after", PortOut, "ok", PortIn),
		},
	}
}

func TestForkRunsBranchesSequentiallyAndConvergeAllWaitsForActivePaths(t *testing.T) {
	ex := &stubExec{}
	st, _ := run(t, forkGraph("all"), nil, ex, nil)
	if st.Status != RunCompleted || !reflect.DeepEqual(ex.calls, []string{"A", "B", "after"}) {
		t.Fatalf("A e B rodam em sequência e 'after' só uma vez, depois de ambos: %v %s", ex.calls, st.Status)
	}

	// Em "any" cada chegada continua (o que roda duas vezes é do autor escolher).
	ex = &stubExec{}
	run(t, forkGraph("any"), nil, ex, nil)
	if !reflect.DeepEqual(ex.calls, []string{"A", "B", "after", "after"}) {
		t.Fatalf("modo any: %v", ex.calls)
	}

	// Caminho não percorrido não bloqueia: com condição só um ramo chega ao "all".
	g := branchGraph()
	g.Nodes[4].Props["mode"] = "all"
	ex = &stubExec{}
	st, _ = run(t, g, map[string]any{"go_left": false}, ex, nil)
	if st.Status != RunCompleted || !reflect.DeepEqual(ex.calls, []string{"right"}) {
		t.Fatalf("convergência all com ramo não percorrido: %s %v %s", st.Status, ex.calls, st.Failure)
	}
}

func errorGraph(withHandler bool) Graph {
	g := Graph{
		Nodes: []Node{n("s", TStart, nil), agent("work"), end("ok", "success")},
		Edges: []Edge{e("1", "s", PortOut, "work", PortIn), e("2", "work", PortOut, "ok", PortIn)},
	}
	if withHandler {
		g.Nodes = append(g.Nodes, end("handled", "failure"))
		g.Edges = append(g.Edges, e("3", "work", PortError, "handled", PortIn))
	}
	return g
}

func TestErrorRouteHandlesFailureOtherwiseRunFails(t *testing.T) {
	boom := &stubExec{fn: map[string]func(int) (ExecResult, error){"work": func(int) (ExecResult, error) { return ExecResult{}, errors.New("quebrou") }}}
	st, _ := run(t, errorGraph(true), nil, boom, nil)
	if st.Status != RunCompleted || st.Outcome != "failure" || !strings.Contains(walked(st), "work,handled") {
		t.Fatalf("a rota de erro deve tratar: %s %s %q", st.Status, st.Outcome, walked(st))
	}
	st, _ = run(t, errorGraph(false), nil, boom, nil)
	if st.Status != RunFailed || !strings.Contains(st.Failure, "work") {
		t.Fatalf("sem rota de erro a execução falha: %s %q", st.Status, st.Failure)
	}
	// Porta "error" devolvida pelo executor também segue a rota.
	viaPort := &stubExec{fn: map[string]func(int) (ExecResult, error){"work": func(int) (ExecResult, error) { return ExecResult{Port: PortError}, nil }}}
	st, _ = run(t, errorGraph(true), nil, viaPort, nil)
	if st.Outcome != "failure" {
		t.Fatalf("porta de erro: %+v", st)
	}
}

func TestBadExpressionsFailTheRunWithTheNode(t *testing.T) {
	g := branchGraph()
	g.Nodes[1].Props["expression"] = "x < 'a'"
	st, _ := run(t, g, map[string]any{"x": float64(1)}, &stubExec{}, nil)
	if st.Status != RunFailed || !strings.Contains(st.Failure, `"c"`) {
		t.Fatalf("expressão impossível de avaliar: %s %q", st.Status, st.Failure)
	}
}

const (
	idSub   = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	idOther = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
)

func subGraph(failing bool) Graph {
	outcome := "success"
	if failing {
		outcome = "failure"
	}
	return Graph{
		Nodes: []Node{n("s", TStart, nil), agent("inner"), end("e", outcome)},
		Edges: []Edge{e("1", "s", PortOut, "inner", PortIn), e("2", "inner", PortOut, "e", PortIn)},
	}
}

func rootWithSub(id string, withError bool) Graph {
	g := Graph{
		Nodes: []Node{n("s", TStart, nil), n("sub", TSubflow, map[string]any{"workflow_id": id}), agent("next"), end("ok", "success")},
		Edges: []Edge{e("1", "s", PortOut, "sub", PortIn), e("2", "sub", PortOut, "next", PortIn), e("3", "next", PortOut, "ok", PortIn)},
	}
	if withError {
		g.Nodes = append(g.Nodes, agent("fix"), end("fixed", "success"))
		g.Edges = append(g.Edges, e("4", "sub", PortError, "fix", PortIn), e("5", "fix", PortOut, "fixed", PortIn))
	}
	return g
}

func resolver(m map[string]Graph) Resolver {
	return func(id string) (Graph, error) {
		g, ok := m[id]
		if !ok {
			return Graph{}, errors.New("não encontrado")
		}
		return g, nil
	}
}

func TestSubflowIsInlinedAndRunsToCompletion(t *testing.T) {
	flat, err := Flatten(rootWithSub(idSub, false), "root", resolver(map[string]Graph{idSub: subGraph(false)}))
	if err != nil {
		t.Fatal(err)
	}
	ex := &stubExec{}
	st, _ := run(t, flat, nil, ex, nil)
	if st.Status != RunCompleted || !reflect.DeepEqual(ex.calls, []string{"sub/inner", "next"}) {
		t.Fatalf("o subfluxo executa em linha e o fluxo segue: %v %s %s", ex.calls, st.Status, st.Failure)
	}

	// Falha no subfluxo com rota de erro segue a rota; sem rota, falha o fluxo.
	flat, err = Flatten(rootWithSub(idSub, true), "root", resolver(map[string]Graph{idSub: subGraph(true)}))
	if err != nil {
		t.Fatal(err)
	}
	ex = &stubExec{}
	st, _ = run(t, flat, nil, ex, nil)
	if st.Status != RunCompleted || st.Outcome != "success" || !reflect.DeepEqual(ex.calls, []string{"sub/inner", "fix"}) {
		t.Fatalf("falha tratada pela rota de erro: %v %s %s", ex.calls, st.Status, st.Outcome)
	}
	flat, _ = Flatten(rootWithSub(idSub, false), "root", resolver(map[string]Graph{idSub: subGraph(true)}))
	st, _ = run(t, flat, nil, &stubExec{}, nil)
	if st.Outcome != "failure" {
		t.Fatalf("falha sem rota de erro deve falhar o fluxo: %+v", st)
	}
}

func TestRecursiveAndTooDeepSubflowsAreBlocked(t *testing.T) {
	// A → B → A.
	a, b := rootWithSub(idOther, false), rootWithSub(idSub, false)
	_, err := Flatten(a, idSub, resolver(map[string]Graph{idSub: a, idOther: b}))
	if !errors.Is(err, ErrSubflowRecursive) {
		t.Fatalf("recursão indireta deve ser bloqueada: %v", err)
	}
	// Chamada direta a si mesmo.
	self := rootWithSub(idSub, false)
	if _, err := Flatten(self, idSub, resolver(map[string]Graph{idSub: self})); !errors.Is(err, ErrSubflowRecursive) {
		t.Fatalf("auto-chamada: %v", err)
	}
	// Cadeia mais funda que o limite.
	chain := map[string]Graph{}
	id := func(i int) string { return fmt.Sprintf("%08d-0000-4000-8000-000000000000", i) }
	for i := 1; i <= MaxSubflowDepth+3; i++ {
		chain[id(i)] = rootWithSub(id(i+1), false)
	}
	chain[id(MaxSubflowDepth+4)] = subGraph(false)
	if _, err := Flatten(rootWithSub(id(1), false), "root", resolver(chain)); !errors.Is(err, ErrSubflowDepth) {
		t.Fatalf("profundidade: %v", err)
	}
	// Subfluxo inexistente ou inválido.
	if _, err := Flatten(rootWithSub(idSub, false), "root", resolver(nil)); !errors.Is(err, ErrSubflowInvalid) {
		t.Fatalf("inexistente: %v", err)
	}
	bad := subGraph(false)
	bad.Nodes[1].Props = map[string]any{}
	if _, err := Flatten(rootWithSub(idSub, false), "root", resolver(map[string]Graph{idSub: bad})); !errors.Is(err, ErrSubflowInvalid) {
		t.Fatalf("inválido: %v", err)
	}
	// O motor recusa um grafo que ainda tem subfluxo.
	if _, err := NewRun(rootWithSub(idSub, false), nil); err == nil {
		t.Fatal("NewRun exige o grafo achatado")
	}
}

func TestExecutionIsDeterministic(t *testing.T) {
	g := forkGraph("all")
	var traces []string
	for i := 0; i < 3; i++ {
		st, _ := run(t, g, map[string]any{"a": float64(1)}, &stubExec{}, nil)
		raw, _ := json.Marshal(st)
		traces = append(traces, string(raw))
	}
	if traces[0] != traces[1] || traces[1] != traces[2] {
		t.Fatal("a mesma entrada deve produzir exatamente o mesmo estado")
	}
}

func TestExpressions(t *testing.T) {
	vars := map[string]any{"checks": map[string]any{"passed": true, "count": float64(3)}, "name": "abc", "empty": ""}
	cases := map[string]bool{
		"checks.passed":                         true,
		"checks.passed == true":                 true,
		"checks.count >= 3 && checks.count < 4": true,
		"checks.count > 3 || name == 'abc'":     true,
		`name != "abc"`:                         false,
		"!checks.passed":                        false,
		"missing":                               false,
		"missing == null":                       true,
		"empty":                                 false,
		"checks.count == 3":                     true,
		"1 < 2":                                 true,
		"-1 < 0":                                true,
	}
	for src, want := range cases {
		x, err := ParseExpr(src)
		if err != nil {
			t.Errorf("%q: %v", src, err)
			continue
		}
		got, err := x.Eval(vars)
		if err != nil || got != want {
			t.Errorf("%q = %v (%v), esperado %v", src, got, err, want)
		}
	}
	for _, bad := range []string{"", "   ", "a ==", "== 1", "a b", "'aberto", "a..b", "a.", "(a)", "a && ", "1.2.3", strings.Repeat("a", 600)} {
		if _, err := ParseExpr(bad); err == nil {
			t.Errorf("%q deveria ser recusada", bad)
		}
	}
	if x, _ := ParseExpr("name < 1"); x != nil {
		if _, err := x.Eval(vars); err == nil {
			t.Error("comparar texto com < deve falhar")
		}
	}
}

func TestInvalidExpressionIsReportedByValidator(t *testing.T) {
	g := branchGraph()
	g.Nodes[1].Props["expression"] = "a =="
	if is := Validate(g); !has(is, "c", "expression", "invalid_expression") {
		t.Fatalf("expressão inválida deve ser apontada com nó e campo: %+v", is)
	}
}
