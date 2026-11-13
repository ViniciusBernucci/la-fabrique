package resume

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

func node(id, typ string, props map[string]any) workflows.Node {
	if props == nil {
		props = map[string]any{}
	}
	return workflows.Node{ID: id, Type: typ, Props: props}
}

func edge(id, from, port, to string) workflows.Edge {
	return workflows.Edge{ID: id, From: from, FromPort: port, To: to, ToPort: "in"}
}

// start → fork ─a→ x → y → end1
//
//	└b→ z → end2
func graph() workflows.Graph {
	return workflows.Graph{
		Nodes: []workflows.Node{
			node("s", workflows.TStart, nil), node("f", workflows.TFork, nil),
			node("x", workflows.TAgent, map[string]any{"agent": "dev", "prompt": "faça"}), node("y", workflows.TAgent, map[string]any{"agent": "dev", "prompt": "faça"}), node("z", workflows.TAgent, map[string]any{"agent": "dev", "prompt": "faça"}),
			node("e1", workflows.TEnd, map[string]any{"outcome": "success"}), node("e2", workflows.TEnd, map[string]any{"outcome": "success"}),
		},
		Edges: []workflows.Edge{
			edge("1", "s", "out", "f"), edge("2", "f", "a", "x"), edge("3", "f", "b", "z"),
			edge("4", "x", "out", "y"), edge("5", "y", "out", "e1"), edge("6", "z", "out", "e2"),
		},
	}
}

type fakeExec struct {
	calls map[string]int
	out   map[string]any // saída por nó
	fail  map[string]bool
}

func (f *fakeExec) Run(_ context.Context, n workflows.Node, _ map[string]any) (workflows.ExecResult, error) {
	f.calls[n.ID]++
	if f.fail[n.ID] {
		return workflows.ExecResult{Port: workflows.PortError}, errors.New("falhou")
	}
	return workflows.ExecResult{Outputs: map[string]any{"v": f.out[n.ID]}}, nil
}

func finished(t *testing.T, f *fakeExec) workflows.RunState {
	t.Helper()
	g := graph()
	st, err := workflows.NewRun(g, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := (&workflows.Engine{Exec: f}).Advance(context.Background(), g, st); err != nil || st.Status != workflows.RunCompleted {
		t.Fatalf("%v %+v", err, st)
	}
	return *st
}

func newFake() *fakeExec {
	return &fakeExec{calls: map[string]int{}, out: map[string]any{"x": 1, "y": 1, "z": 1}, fail: map[string]bool{}}
}

func TestSelectComputesTheAffectedSetFromTheFrozenGraph(t *testing.T) {
	f := newFake()
	st := finished(t, f)
	sel, err := Select(graph(), st, "x")
	if err != nil || strings.Join(sel.Affected, ",") != "e1,y" || strings.Join(sel.Reusable, ",") != "e2,f,s,z" {
		t.Fatalf("%+v %v", sel, err)
	}
	for _, bad := range []string{"fantasma", "s", "e1", "f"} {
		if _, err := Select(graph(), st, bad); err == nil {
			t.Errorf("%q não deveria ser reexecutável", bad)
		}
	}
	notRun := workflows.RunState{Visited: map[string]bool{}}
	if _, err := Select(graph(), notRun, "x"); !errors.Is(err, ErrNotRerunnable) {
		t.Fatal(err)
	}
}

// T12-04 — "Reexecutar nó altera só o conjunto afetado."
func TestRerunChangesOnlyTheAffectedSetWhenTheOutputChanges(t *testing.T) {
	f := newFake()
	before := finished(t, f)
	beforeJSON, _ := json.Marshal(before)
	f.out["x"] = 2 // a saída de x muda
	f.calls = map[string]int{}
	next, res, err := Rerun(context.Background(), f, graph(), before, "x")
	if err != nil || !res.OutputChanged || strings.Join(res.Invalidated, ",") != "e1,y" {
		t.Fatalf("%+v %v", res, err)
	}
	if afterJSON, _ := json.Marshal(before); string(afterJSON) != string(beforeJSON) {
		t.Fatal("o estado de entrada não pode ser alterado")
	}
	// z, e2, s, f ficam idênticos.
	for _, k := range []string{"z"} {
		a, _ := json.Marshal(next.Vars[k])
		b, _ := json.Marshal(before.Vars[k])
		if string(a) != string(b) {
			t.Fatalf("%s mudou", k)
		}
	}
	for _, id := range []string{"s", "f", "z", "e2"} {
		if !next.Visited[id] {
			t.Fatalf("%s deveria continuar percorrido", id)
		}
	}
	if next.Visited["y"] || next.Visited["e1"] || next.Vars["y"] != nil || next.Status != workflows.RunRunning || len(next.Queue) != 1 || next.Queue[0].Node != "y" {
		t.Fatalf("%+v", next)
	}
	if len(next.Ends) != 1 {
		t.Fatalf("o fim invalidado não conta mais: %v", next.Ends)
	}
	// O motor refaz só o que foi invalidado.
	f.calls = map[string]int{}
	if err := (&workflows.Engine{Exec: f}).Advance(context.Background(), graph(), &next); err != nil || next.Status != workflows.RunCompleted {
		t.Fatalf("%v %s", err, next.Status)
	}
	if f.calls["y"] != 1 || f.calls["x"] != 0 || f.calls["z"] != 0 || len(next.Ends) != 2 {
		t.Fatalf("calls %v ends %v", f.calls, next.Ends)
	}
}

func TestRerunWithTheSameOutputKeepsTheDependents(t *testing.T) {
	f := newFake()
	before := finished(t, f)
	f.calls = map[string]int{}
	next, res, err := Rerun(context.Background(), f, graph(), before, "x")
	if err != nil || res.OutputChanged || len(res.Invalidated) != 0 {
		t.Fatalf("%+v %v", res, err)
	}
	if !next.Visited["y"] || !next.Visited["e1"] || next.Status != workflows.RunCompleted || f.calls["x"] != 1 || f.calls["y"] != 0 {
		t.Fatalf("%+v calls %v", next, f.calls)
	}
}

func TestAFailedRerunKeepsThePreviousState(t *testing.T) {
	f := newFake()
	before := finished(t, f)
	f.fail["x"] = true
	next, _, err := Rerun(context.Background(), f, graph(), before, "x")
	if !errors.Is(err, ErrRerunFailed) || !reflect.DeepEqual(next, before) {
		t.Fatalf("%v", err)
	}
	if _, _, err := Rerun(context.Background(), nil, graph(), before, "x"); !errors.Is(err, ErrNoExecutor) {
		t.Fatal(err)
	}
}

func TestReconcileClassifiesCompatibleAndConflictingChanges(t *testing.T) {
	base := Inputs{Refs: map[string]string{"branch": "feat", "head": "aaa"},
		Requirement: Requirement{Title: "T", Description: "D", Status: "To Do", Priority: "High", Labels: []string{"a"}}, ConfigHash: "h1"}
	cur := base
	cur.Refs = map[string]string{"branch": "feat", "head": "bbb"}
	cur.Requirement = Requirement{Title: "T", Description: "D", Status: "In Progress", Priority: "Low", Labels: []string{"b", "a"}}
	ff := func(string, string, string) (bool, error) { return true, nil }
	ds := Reconcile(base, cur, Options{FastForward: ff})
	if HasConflict(ds) || len(ds) != 4 {
		t.Fatalf("%v", ds)
	}
	// Sem prova de fast-forward, a mudança de ref é conflito (falha fechada).
	if !HasConflict(Reconcile(base, cur, Options{})) {
		t.Fatal("sem prova, conflito")
	}
	div := func(string, string, string) (bool, error) { return false, nil }
	if d := Reconcile(base, cur, Options{FastForward: div}); !HasConflict(d) {
		t.Fatalf("%v", d)
	}
	for name, mut := range map[string]func(*Inputs){
		"título":       func(i *Inputs) { i.Requirement.Title = "Outro" },
		"descrição":    func(i *Inputs) { i.Requirement.Description = "Outra" },
		"dependências": func(i *Inputs) { i.Requirement.Dependencies = []string{"X-1"} },
		"config":       func(i *Inputs) { i.ConfigHash = "h2" },
	} {
		c := base
		mut(&c)
		if !HasConflict(Reconcile(base, c, Options{})) {
			t.Errorf("%s deveria ser conflito", name)
		}
	}
	if ds := Reconcile(base, base, Options{}); len(ds) != 0 {
		t.Fatalf("%v", ds)
	}
}
