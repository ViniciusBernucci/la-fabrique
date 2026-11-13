package integration

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/recovery"
	"github.com/ViniciusBernucci/le-fabrique/internal/retention"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/worker"
)

func jobWorker(t *testing.T) *worker.Worker {
	t.Helper()
	return worker.New(nopPinger{}, slogDiscard(), time.Hour)
}

// T10-02 / T13-02 — o worker devolve à fila o item cuja decisão foi resolvida, e só ele.
func TestTheWorkerReleasesItemsOnlyAfterTheirDecisionIsResolved(t *testing.T) {
	e := newQEnv(t)
	ctx := context.Background()
	p := e.project(t, "wk-decision", independent("g1", "g2"))
	g1, g2 := p.item(t, "g1", 0), p.item(t, "g2", 0)
	e.q.Gates[queue.WaitDecision] = queue.DecisionGate{DB: e.e.db.Pool}
	blockOnDecision := func(it queue.Item, node string) decisions.Request {
		exec := p.execution(t)
		if _, err := e.q.Start(ctx, it.ID, exec); err != nil {
			t.Fatal(err)
		}
		if _, err := e.q.Block(ctx, it.ID, queue.WaitDecision, "", p.checkpoint(t, exec), true); err != nil {
			t.Fatal(err)
		}
		if _, err := e.e.db.Pool.Exec(ctx, `UPDATE executions SET state = 'waiting' WHERE id = $1`, exec); err != nil {
			t.Fatal(err)
		}
		dec, _, err := decisions.Create(ctx, e.e.db.Pool, security.NewRedactor(), newRequest(p.proj, exec, node, decisions.KindConflict))
		if err != nil {
			t.Fatal(err)
		}
		return dec
	}
	d1 := blockOnDecision(g1, "g1")
	blockOnDecision(g2, "g2")
	w := jobWorker(t)
	w.AddJob(worker.ReleaseDecisionsJob(e.e.db.Pool, e.q, time.Hour))
	// Ninguém decidiu: nada é liberado.
	if note, _ := w.RunJobOnce(ctx, "liberar-decisoes"); note != "" {
		t.Fatalf("%s", note)
	}
	var admin string
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT id FROM admins LIMIT 1`).Scan(&admin)
	if _, err := decisions.Resolve(ctx, e.e.db.Pool, admin, d1.ID, "seguir-a", "ok"); err != nil {
		t.Fatal(err)
	}
	w.RunJobOnce(ctx, "liberar-decisoes")
	items, _ := e.q.List(ctx, p.proj)
	state := map[string]string{}
	for _, i := range items {
		state[i.TaskKey] = i.State
	}
	if state["g1"] != queue.StateQueued || state["g2"] != queue.StateWaiting {
		t.Fatalf("só a decisão resolvida libera o seu item: %v", state)
	}
	st := w.Jobs()
	if len(st) != 1 || st[0].Runs != 2 || st[0].LastError != "" {
		t.Fatalf("%+v", st)
	}
}

// T12-03 — o worker recupera tentativas interrompidas; sem inspeção possível, adia e não muda nada.
func TestTheWorkerRecoversStaleAttemptsAndDefersWhenItCannotInspect(t *testing.T) {
	e := newRcvEnv(t)
	ctx := context.Background()
	p := e.project(t, "wk-recover", independent("t"))
	it, exec := e.crashed(t, p, "t", "worker-1")
	cp := p.checkpoint(t, exec)
	e.expire()

	// Runtime de containers ausente: a inspeção falha e a recuperação ADIA sem consumir tentativa.
	e.r.Inspector = recovery.DockerInspector{Docker: &execution.Docker{Bin: "docker-que-nao-existe"}}
	w := jobWorker(t)
	w.AddJob(worker.RecoverJob(e.r, time.Hour))
	w.RunJobOnce(ctx, "recuperar-tentativas")
	items, _ := e.q.List(ctx, p.proj)
	if items[0].State != queue.StateRunning || items[0].ID != it.ID {
		t.Fatalf("sem inspeção nada muda: %+v", items[0])
	}
	var ev int
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM recovery_events WHERE item_id = $1 AND action = 'deferred'`, it.ID).Scan(&ev)
	if ev != 1 {
		t.Fatalf("o adiamento fica registrado: %d", ev)
	}

	// Inspeção que responde e prova quietude: volta à fila, retomando do checkpoint.
	e.r.Inspector = fakeInspector{}
	w.RunJobOnce(ctx, "recuperar-tentativas")
	items, _ = e.q.List(ctx, p.proj)
	if items[0].State != queue.StateQueued || items[0].CheckpointID != cp {
		t.Fatalf("%+v", items[0])
	}
	if st := w.Jobs()[0]; st.LastNote == "" {
		t.Fatalf("%+v", st)
	}
}

// T12-03 — o inspetor real enxerga o sandbox: container quieto ou ocupado.
func TestTheDockerInspectorTellsAQuietSandboxFromABusyOne(t *testing.T) {
	d := &execution.Docker{}
	if !d.Available(t.Context()) {
		t.Skip("Docker indisponível")
	}
	project, feature := "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"
	insp := recovery.DockerInspector{Docker: d}
	it := queue.Item{ProjectID: project, FeatureID: feature}
	if quiet, err := insp.Quiescent(t.Context(), it); err != nil || !quiet {
		t.Fatalf("sem container: quieto %v %v", quiet, err)
	}
	sb := ckSandbox(t, project, feature)
	if quiet, err := insp.Quiescent(t.Context(), it); err != nil || !quiet {
		t.Fatalf("container ocioso: quieto %v %v", quiet, err)
	}
	if _, err := sb.Start(t.Context(), execution.ExecSpec{Cmd: []string{"sleep", "60"}}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if quiet, err := insp.Quiescent(t.Context(), it); err != nil || quiet {
		t.Fatalf("processo vivo: não quieto %v %v", quiet, err)
	}
	// Outra feature não é afetada.
	if quiet, err := insp.Quiescent(t.Context(), queue.Item{ProjectID: project, FeatureID: "33333333-3333-4333-8333-333333333333"}); err != nil || !quiet {
		t.Fatalf("%v %v", quiet, err)
	}
}

// T16-04 — o worker só simula a retenção, a menos que a remoção real seja ligada.
func TestTheWorkerOnlySimulatesRetentionUnlessApplyIsOn(t *testing.T) {
	e := newQEnv(t)
	ctx := context.Background()
	w := jobWorker(t)
	ret := &retention.Retention{DB: e.e.db.Pool, Checkpoints: e.cps}
	w.AddJob(worker.RetentionJob(ret, retention.Policy{}, false, time.Hour))
	w.AddJob(worker.RetentionJob(ret, retention.Policy{}, true, time.Hour))
	if _, err := w.RunJobOnce(ctx, "retencao-simulada"); err != nil {
		t.Fatal(err)
	}
	if _, err := w.RunJobOnce(ctx, "retencao"); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, s := range w.Jobs() {
		names[s.Name] = s.LastError == ""
	}
	if !names["retencao-simulada"] || !names["retencao"] {
		t.Fatalf("%v", names)
	}
	var runs, dry int
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE dry_run) FROM retention_runs`).Scan(&runs, &dry)
	if runs != 2 || dry != 1 {
		t.Fatalf("execuções %d, simuladas %d", runs, dry)
	}
}

type nopPinger struct{}

func (nopPinger) Ping(context.Context) error { return nil }

func slogDiscard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
