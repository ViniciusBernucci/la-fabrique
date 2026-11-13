package integration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/recovery"
	"github.com/ViniciusBernucci/le-fabrique/internal/resume"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type fakeInspector struct {
	quiet  func(queue.Item) (bool, error)
	verify func(queue.Item, *checkpoints.Checkpoint) ([]string, error)
}

func (f fakeInspector) Quiescent(_ context.Context, it queue.Item) (bool, error) {
	if f.quiet == nil {
		return true, nil
	}
	return f.quiet(it)
}

func (f fakeInspector) Verify(_ context.Context, it queue.Item, cp *checkpoints.Checkpoint) ([]string, error) {
	if f.verify == nil {
		return nil, nil
	}
	return f.verify(it, cp)
}

type rcvEnv struct {
	*schedEnv
	r  *recovery.Recoverer
	in *fakeInspector
}

func newRcvEnv(t *testing.T) *rcvEnv {
	t.Helper()
	s := newSchedEnv(t)
	s.q.RequireRecovery = true
	in := &fakeInspector{}
	return &rcvEnv{schedEnv: s, in: in, r: &recovery.Recoverer{DB: s.e.db.Pool, Queue: s.q, Checkpoints: s.cps, Inspector: in, Redactor: security.NewRedactor()}}
}

// crashed deixa um item "em execução" com checkpoint, e a lease vence (worker/VPS sumiu).
func (e *rcvEnv) crashed(t *testing.T, p qProj, key, owner string) (queue.Item, string) {
	t.Helper()
	it := p.item(t, key, 0)
	_ = it
	exec := p.execution(t)
	got, err := e.q.Claim(context.Background(), owner, exec, time.Minute)
	if err != nil || got == nil || got.TaskKey != key {
		t.Fatalf("claim %s: %+v %v", key, got, err)
	}
	return *got, exec
}

func (e *rcvEnv) expire() { e.advance(5 * time.Minute) }

// T12-03 — "Reboot e crash retomam automaticamente quando seguros."
func TestACrashIsRecoveredAutomaticallyWhenItIsSafe(t *testing.T) {
	e := newRcvEnv(t)
	ctx := context.Background()
	p := e.project(t, "rcv-safe", independent("t"))
	it, exec := e.crashed(t, p, "t", "worker-1")
	cp := p.checkpoint(t, exec)
	// Ainda com a lease viva, ninguém mexe.
	if stale, _ := e.r.Stale(ctx); len(stale) != 0 {
		t.Fatalf("%+v", stale)
	}
	if _, err := e.q.RecoverRequeue(ctx, it.ID, cp); !errors.Is(err, queue.ErrNotStale) {
		t.Fatalf("um worker vivo não pode ser tocado: %v", err)
	}
	e.expire()
	// Com RequireRecovery, a reclamação não devolve às cegas o que está interrompido.
	if got := e.claim(t, "worker-2"); got != nil {
		t.Fatalf("o item interrompido segue ocupando a vaga até ser diagnosticado: %+v", got)
	}
	outs, err := e.r.RecoverAll(ctx, nil)
	if err != nil || len(outs) != 1 {
		t.Fatalf("%+v %v", outs, err)
	}
	o := outs[0]
	if o.Action != recovery.ActionRecovered || o.CheckpointID != cp || !o.SlotReleased || o.Attempts != 1 || o.Class != recovery.Infra {
		t.Fatalf("%+v", o)
	}
	items, _ := e.q.List(ctx, p.proj)
	if items[0].State != queue.StateQueued || items[0].SlotHeld || items[0].CheckpointID != cp || items[0].LeaseOwner != "" {
		t.Fatalf("%+v", items[0])
	}
	if tasks, _ := p.plans.Tasks(ctx, p.proj, p.feature); tasks[0].State != "pending" {
		t.Fatalf("%+v", tasks)
	}
	// O novo worker reclama (geração nova): o antigo, se voltar, não conclui.
	again := e.claim(t, "worker-2")
	if again == nil || again.ID != it.ID || again.Generation != it.Generation+1 || again.CheckpointID != cp {
		t.Fatalf("%+v", again)
	}
	if _, err := e.q.Complete(fenced(&it), it.ID); !errors.Is(err, queue.ErrStale) {
		t.Fatalf("%v", err)
	}
	var ev int
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM recovery_events WHERE item_id = $1 AND action = 'recovered'`, it.ID).Scan(&ev)
	if ev != 1 {
		t.Fatalf("%d", ev)
	}
	// Sem checkpoint e sem efeitos: recomeça do início, também automaticamente.
	p2 := e.project(t, "rcv-fresh", independent("f"))
	it2, _ := e.crashed(t, p2, "f", "worker-3")
	e.expire()
	outs, _ = e.r.RecoverAll(ctx, nil)
	var fresh *recovery.Outcome
	for i := range outs {
		if outs[i].ItemID == it2.ID {
			fresh = &outs[i]
		}
	}
	if fresh == nil || fresh.Action != recovery.ActionRecovered || fresh.CheckpointID != "" || !strings.Contains(strings.Join(fresh.Reasons, " "), "recomeça do início") {
		t.Fatalf("%+v", fresh)
	}
}

// T12-03 — "Incerteza abre decisão e outras tarefas elegíveis seguem."
func TestUncertaintyOpensADecisionAndOtherEligibleTasksKeepGoing(t *testing.T) {
	e := newRcvEnv(t)
	ctx := context.Background()
	if _, err := e.q.SetSettings(ctx, 8, 30); err != nil {
		t.Fatal(err)
	}
	a := e.project(t, "rcv-a", independent("a1", "a2"))
	b := e.project(t, "rcv-b", independent("b1"))
	c := e.project(t, "rcv-c", independent("c1"))
	a.item(t, "a2", 4)
	aIt, aExec := e.crashed(t, a, "a1", "w-a")
	bIt, bExec := e.crashed(t, b, "b1", "w-b")
	cIt, cExec := e.crashed(t, c, "c1", "w-c")
	_ = bExec
	cpA := a.checkpoint(t, aExec)
	_ = b.checkpoint(t, bExec)
	_ = c.checkpoint(t, cExec)
	// A: efeito externo incerto da execução. B: sandbox com processo vivo. C: tudo certo.
	if _, err := e.e.db.Pool.Exec(ctx, `INSERT INTO outbox (project_id, business_key, destination, state) VALUES ($1, $2, 'github', 'uncertain')`,
		a.proj, resume.EffectPrefix(aExec, "a1")+"create_pr"); err != nil {
		t.Fatal(err)
	}
	e.in.quiet = func(it queue.Item) (bool, error) { return it.ID != bIt.ID, nil }
	// Outra tarefa independente, de outro projeto, que chega depois e deve seguir.
	d := e.project(t, "rcv-d", independent("d1"))
	d.item(t, "d1", 1)
	e.expire()
	outs, err := e.r.RecoverAll(ctx, nil)
	if err != nil || len(outs) != 3 {
		t.Fatalf("%+v %v", outs, err)
	}
	by := map[string]recovery.Outcome{}
	for _, o := range outs {
		by[o.ItemID] = o
	}
	oa, ob, oc := by[aIt.ID], by[bIt.ID], by[cIt.ID]
	if oa.Action != recovery.ActionDecision || oa.Decision == nil || oa.Decision.Kind != decisions.KindUncertainResult || !oa.SlotReleased ||
		!strings.Contains(strings.Join(oa.Reasons, " "), "sem confirmação") {
		t.Fatalf("A: %+v", oa)
	}
	if ob.Action != recovery.ActionDecision || ob.SlotReleased || !strings.Contains(strings.Join(ob.Reasons, " "), "processo vivo") {
		t.Fatalf("B (quietude não provada mantém a vaga): %+v", ob)
	}
	if oc.Action != recovery.ActionRecovered {
		t.Fatalf("C: %+v", oc)
	}
	items, _ := e.q.List(ctx, a.proj)
	var a1 queue.Item
	for _, it := range items {
		if it.TaskKey == "a1" {
			a1 = it
		}
	}
	if a1.State != queue.StateWaiting || a1.WaitReason != queue.WaitDecision || a1.WaitRef != oa.Decision.ID || a1.SlotHeld || a1.CheckpointID != cpA {
		t.Fatalf("%+v", a1)
	}
	// Bloqueia só o afetado: a2 (mesmo projeto de A, vaga liberada com checkpoint seguro) e d1 seguem.
	var started []string
	for i := 0; i < 4; i++ {
		got := e.claim(t, "w-new")
		if got == nil {
			break
		}
		started = append(started, got.TaskKey)
	}
	joined := strings.Join(started, ",")
	for _, want := range []string{"a2", "d1", "c1"} {
		if !strings.Contains(joined, want) {
			t.Errorf("a tarefa %s deveria seguir; iniciadas: %s", want, joined)
		}
	}
	if strings.Contains(joined, "b1") || strings.Contains(joined, "a1") {
		t.Fatalf("as tarefas incertas não podem reiniciar sozinhas: %s", joined)
	}
	// B segurou a vaga: nada mais do projeto B inicia.
	if e2, _ := e.q.Explain(ctx, bIt.ID); len(e2) == 0 {
		t.Fatal("B está esperando decisão")
	}
}

func TestCorruptCheckpointAndVerifierProblemsRequireADecision(t *testing.T) {
	e := newRcvEnv(t)
	ctx := context.Background()
	p := e.project(t, "rcv-bad", independent("k", "v"))
	k, kExec := e.crashed(t, p, "k", "w1")
	cp := p.checkpoint(t, kExec)
	if _, err := e.e.db.Pool.Exec(ctx, `UPDATE checkpoints SET tree_hash = repeat('0', 64) WHERE id = $1`, cp); err != nil {
		t.Fatal(err)
	}
	e.expire()
	o, err := e.r.Recover(ctx, k, recovery.Infra)
	if err != nil || o.Action != recovery.ActionDecision || o.SlotReleased || !strings.Contains(strings.Join(o.Reasons, " "), "corrompido") {
		t.Fatalf("%+v %v", o, err)
	}
	// Problema de Git/artefato/política apontado pelo verificador.
	p2 := e.project(t, "rcv-git", independent("g"))
	g, gExec := e.crashed(t, p2, "g", "w2")
	_ = p2.checkpoint(t, gExec)
	e.in.verify = func(queue.Item, *checkpoints.Checkpoint) ([]string, error) {
		return []string{"o HEAD do workspace não é o do checkpoint"}, nil
	}
	e.expire()
	o2, _ := e.r.Recover(ctx, g, recovery.Infra)
	if o2.Action != recovery.ActionDecision || !strings.Contains(strings.Join(o2.Reasons, " "), "HEAD") {
		t.Fatalf("%+v", o2)
	}
	// Efeitos confirmados mas sem checkpoint: não recomeça às cegas.
	p3 := e.project(t, "rcv-eff", independent("e"))
	it3, exec3 := e.crashed(t, p3, "e", "w3")
	if _, err := e.e.db.Pool.Exec(ctx, `INSERT INTO outbox (project_id, business_key, destination, state) VALUES ($1, $2, 'jira', 'confirmed')`,
		p3.proj, resume.EffectPrefix(exec3, "e")+"create_subtask"); err != nil {
		t.Fatal(err)
	}
	e.in.verify = nil
	e.expire()
	o3, _ := e.r.Recover(ctx, it3, recovery.Infra)
	if o3.Action != recovery.ActionDecision || !strings.Contains(strings.Join(o3.Reasons, " "), "nenhum checkpoint") {
		t.Fatalf("%+v", o3)
	}
}

// T12-03 — "aplicar limites separados de infraestrutura/IA/revisão".
func TestRecoveryLimitsAreSeparatedPerFailureClass(t *testing.T) {
	e := newRcvEnv(t)
	ctx := context.Background()
	e.r.Limits = recovery.Limits{recovery.Infra: 2, recovery.AI: 1, recovery.Review: 1}
	p := e.project(t, "rcv-limits", independent("l"))
	item := p.item(t, "l", 0)
	exec := p.execution(t)
	cycle := func(owner string, class recovery.Class) recovery.Outcome {
		got, err := e.q.Claim(ctx, owner, exec, time.Minute)
		if err != nil || got == nil || got.ID != item.ID {
			t.Fatalf("%s: %+v %v", owner, got, err)
		}
		e.expire()
		o, err := e.r.Recover(ctx, *got, class)
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	// Infraestrutura: duas recuperações automáticas, a terceira pede decisão.
	if o := cycle("w1", recovery.Infra); o.Action != recovery.ActionRecovered || o.Attempts != 1 {
		t.Fatalf("%+v", o)
	}
	if o := cycle("w2", recovery.Infra); o.Action != recovery.ActionRecovered || o.Attempts != 2 {
		t.Fatalf("%+v", o)
	}
	// A classe de IA tem contador próprio: ainda tem a sua tentativa.
	if o := cycle("w3", recovery.AI); o.Action != recovery.ActionRecovered || o.Attempts != 1 {
		t.Fatalf("%+v", o)
	}
	// Esgotou a de IA (limite 1) e a de infraestrutura (limite 2): ambas pedem decisão.
	got, _ := e.q.Claim(ctx, "w4", exec, time.Minute)
	e.expire()
	o, err := e.r.Recover(ctx, *got, recovery.AI)
	if err != nil || o.Action != recovery.ActionLimit || o.Decision == nil || !strings.Contains(strings.Join(o.Reasons, " "), `classe "ai"`) {
		t.Fatalf("%+v %v", o, err)
	}
	// A revisão tem o seu contador, intacto, mas o item já espera decisão.
	if items, _ := e.q.List(ctx, p.proj); items[0].State != queue.StateWaiting {
		t.Fatalf("%+v", items[0])
	}
	if _, err := e.r.Recover(ctx, *got, "outra"); !errors.Is(err, errInvalid()) {
		t.Fatalf("%v", err)
	}
	var n int
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM recovery_attempts WHERE item_id = $1`, item.ID).Scan(&n)
	if n != 2 {
		t.Fatalf("contadores por classe: %d", n)
	}
}

func errInvalid() error { return persistence.ErrInvalidInput }

// Inspeção indisponível: nada muda, nada é consumido; o item é recuperado quando a inspeção volta.
func TestAnUnavailableInspectionDefersWithoutChangingAnything(t *testing.T) {
	e := newRcvEnv(t)
	ctx := context.Background()
	p := e.project(t, "rcv-defer", independent("d"))
	it, exec := e.crashed(t, p, "d", "w")
	_ = p.checkpoint(t, exec)
	e.in.quiet = func(queue.Item) (bool, error) { return false, errors.New("docker: daemon indisponível") }
	e.expire()
	o, err := e.r.Recover(ctx, it, recovery.Infra)
	if err != nil || o.Action != recovery.ActionDeferred || o.Attempts != 0 || !strings.Contains(strings.Join(o.Reasons, " "), "quietude") {
		t.Fatalf("%+v %v", o, err)
	}
	items, _ := e.q.List(ctx, p.proj)
	if items[0].State != queue.StateRunning || !items[0].SlotHeld {
		t.Fatalf("nada pode mudar: %+v", items[0])
	}
	var n int
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM recovery_attempts WHERE item_id = $1`, it.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("a tentativa não é consumida: %d", n)
	}
	e.in.quiet = nil
	if o, _ := e.r.Recover(ctx, it, recovery.Infra); o.Action != recovery.ActionRecovered {
		t.Fatalf("%+v", o)
	}
}
