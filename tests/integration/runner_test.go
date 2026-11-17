package integration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

type fakeExec struct {
	mu    sync.Mutex
	calls []string
	block bool
}

func (f *fakeExec) Run(ctx context.Context, n workflows.Node, vars map[string]any) (workflows.ExecResult, error) {
	f.mu.Lock()
	f.calls = append(f.calls, n.ID)
	block := f.block
	f.mu.Unlock()
	if info, ok := runner.InfoFrom(ctx); ok {
		info.Emit(runner.EventNode, map[string]any{"node": n.ID, "line": "olá do terminal"})
	}
	if block {
		<-ctx.Done()
		return workflows.ExecResult{}, ctx.Err()
	}
	return workflows.ExecResult{Outputs: map[string]any{"status": "ok", "tarefa": vars["fabrica"].(map[string]any)["tarefa"]}}, nil
}

func humanFlow(r *relEnv, t *testing.T) wfResp {
	g := linear(agentNode("faça"))
	// agente → decisão humana → fim; recusa → fim de falha
	g.Nodes = g.Nodes[:2]
	g.Edges = g.Edges[:1]
	g.Nodes = append(g.Nodes,
		wfNode("h", "human_decision", 0, 0, map[string]any{"question": "Posso seguir?"}),
		wfNode("ok", "end", 0, 0, map[string]any{"outcome": "success"}),
		wfNode("no", "end", 0, 0, map[string]any{"outcome": "failure"}),
		wfNode("bad", "end", 0, 0, map[string]any{"outcome": "failure"}))
	g.Edges = append(g.Edges,
		wfEdge("e2", "dev", "out", "h", "in"),
		wfEdge("e3", "h", "approved", "ok", "in"),
		wfEdge("e4", "h", "rejected", "no", "in"),
		wfEdge("e5", "dev", "error", "bad", "in"))
	r.agent(t, "dev", nil)
	return r.draft(t, "fluxo-runner", "feature", g)
}

// ADR-0069 — a execução percorre o fluxo publicado, para na decisão humana, retoma com a resposta e conclui.
func TestRunnerAdvancesPausesOnHumanDecisionAndCompletes(t *testing.T) {
	r := newRelEnv(t)
	db := r.e.db.Pool
	w := humanFlow(r, t)

	// rascunho não executa
	if _, err := runner.Start(context.Background(), db, runner.StartInput{ProjectID: r.proj, WorkflowID: w.ID, Title: "t"}); !errors.Is(err, releases.ErrNotPublished) {
		t.Fatalf("rascunho deveria recusar: %v", err)
	}
	r.c.expect(r.publish(w), 201)

	run, err := runner.Start(context.Background(), db, runner.StartInput{ProjectID: r.proj, WorkflowID: w.ID,
		Title: "Cadastro de pedidos", Task: "criar a tela", PRDDone: true})
	if err != nil || run.Status != "queued" {
		t.Fatalf("start: %+v %v", run, err)
	}

	claimed, err := runner.Claim(context.Background(), db, "worker-1", time.Minute)
	if err != nil || claimed == nil || claimed.ExecutionID != run.ExecutionID {
		t.Fatalf("claim: %+v %v", claimed, err)
	}
	if again, _ := runner.Claim(context.Background(), db, "worker-2", time.Minute); again != nil {
		t.Fatal("a execução com posse vigente não pode ser reclamada por outro worker")
	}

	fe := &fakeExec{}
	run, err = runner.Advance(context.Background(), db, *claimed, runner.Options{Exec: fe})
	if err != nil || run.Status != "awaiting_decision" || len(run.Pending) != 1 || run.Pending[0] != "h" {
		t.Fatalf("deveria esperar a decisão: %+v %v", run, err)
	}
	if len(fe.calls) != 1 || fe.calls[0] != "dev" {
		t.Fatalf("o agente deveria rodar uma vez: %v", fe.calls)
	}
	var exState string
	_ = db.QueryRow(context.Background(), `SELECT state FROM executions WHERE id = $1`, run.ExecutionID).Scan(&exState)
	if exState != "waiting" {
		t.Fatalf("execução deveria estar waiting: %s", exState)
	}
	// o terminal do agente e os passos viram eventos do painel
	var steps, lines int
	_ = db.QueryRow(context.Background(), `SELECT count(*) FILTER (WHERE type = 'run_step'), count(*) FILTER (WHERE type = 'node_event') FROM execution_events WHERE execution_id = $1`, run.ExecutionID).Scan(&steps, &lines)
	if steps < 3 || lines != 1 {
		t.Fatalf("eventos: passos=%d terminal=%d", steps, lines)
	}
	if next, _ := runner.Claim(context.Background(), db, "worker-1", time.Minute); next != nil {
		t.Fatal("esperando decisão não há o que reclamar")
	}

	if _, err := runner.Decide(context.Background(), db, r.proj, run.ExecutionID, "h", "talvez", "admin"); err == nil {
		t.Fatal("decisão inválida deveria ser recusada")
	}
	if _, err := runner.Decide(context.Background(), db, r.proj, run.ExecutionID, "h", "approved", "admin"); err != nil {
		t.Fatal(err)
	}
	claimed, err = runner.Claim(context.Background(), db, "worker-1", time.Minute)
	if err != nil || claimed == nil {
		t.Fatalf("claim após decisão: %v %v", claimed, err)
	}
	run, err = runner.Advance(context.Background(), db, *claimed, runner.Options{Exec: fe})
	if err != nil || run.Status != "completed" || run.Outcome != "success" {
		t.Fatalf("deveria concluir com sucesso: %+v %v", run, err)
	}
	if len(fe.calls) != 1 {
		t.Fatalf("o agente não pode rodar de novo ao retomar: %v", fe.calls)
	}
	_ = db.QueryRow(context.Background(), `SELECT state FROM executions WHERE id = $1`, run.ExecutionID).Scan(&exState)
	if exState != "succeeded" {
		t.Fatalf("execução deveria estar succeeded: %s", exState)
	}
	if _, err := runner.Decide(context.Background(), db, r.proj, run.ExecutionID, "h", "approved", "admin"); err == nil {
		t.Fatal("decidir de novo uma execução concluída deveria falhar")
	}
}

func TestRunnerCancelStopsARunningNode(t *testing.T) {
	r := newRelEnv(t)
	db := r.e.db.Pool
	w := humanFlow(r, t)
	r.c.expect(r.publish(w), 201)
	run, err := runner.Start(context.Background(), db, runner.StartInput{ProjectID: r.proj, WorkflowID: w.ID, Title: "cancelar"})
	if err != nil {
		t.Fatal(err)
	}
	claimed, _ := runner.Claim(context.Background(), db, "w", time.Minute)
	fe := &fakeExec{block: true}
	done := make(chan error, 1)
	go func() {
		_, err := runner.Advance(context.Background(), db, *claimed, runner.Options{Exec: fe, WatchEvery: 50 * time.Millisecond})
		done <- err
	}()
	for i := 0; i < 100; i++ {
		fe.mu.Lock()
		n := len(fe.calls)
		fe.mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := runner.Cancel(context.Background(), db, r.proj, run.ExecutionID, "admin"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, runner.ErrCancelled) {
			t.Fatalf("deveria devolver ErrCancelled: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("o nó em andamento não foi interrompido")
	}
	got, _ := runner.Get(context.Background(), db, r.proj, run.ExecutionID)
	if got.Status != "cancelled" {
		t.Fatalf("status: %s", got.Status)
	}
	if again, _ := runner.Claim(context.Background(), db, "w", time.Minute); again != nil {
		t.Fatal("execução cancelada não é reclamada")
	}
}
