package integration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/authorizations"
	"github.com/ViniciusBernucci/le-fabrique/internal/compensation"
	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// remoteState é o "destino" externo: o que existe lá.
type remoteState struct {
	mu        sync.Mutex
	exists    map[string]bool
	undos     int
	failUndo  bool
	loseReply bool
}

func (r *remoteState) has(k string) bool { r.mu.Lock(); defer r.mu.Unlock(); return r.exists[k] }
func (r *remoteState) undoCount() int    { r.mu.Lock(); defer r.mu.Unlock(); return r.undos }

type originalAdapter struct{ r *remoteState }

func (a originalAdapter) Apply(_ context.Context, op effects.Op) (any, error) {
	a.r.mu.Lock()
	a.r.exists[op.Key] = true
	a.r.mu.Unlock()
	return map[string]any{"remote": op.Key}, nil
}
func (a originalAdapter) Verify(context.Context, effects.Op) (effects.Verdict, error) {
	return effects.Verdict{Status: effects.VerdictAbsent}, nil
}

type undoAdapter struct {
	r    *remoteState
	orig persistence.OutboxEntry
}

func (a undoAdapter) Apply(context.Context, effects.Op) (any, error) {
	a.r.mu.Lock()
	defer a.r.mu.Unlock()
	if a.r.failUndo {
		return nil, errors.New("o destino recusou desfazer")
	}
	a.r.undos++
	delete(a.r.exists, a.orig.BusinessKey)
	if a.r.loseReply {
		return nil, errors.New("timeout aguardando a resposta")
	}
	return map[string]any{"undone": a.orig.BusinessKey}, nil
}
func (a undoAdapter) Verify(context.Context, effects.Op) (effects.Verdict, error) {
	a.r.mu.Lock()
	defer a.r.mu.Unlock()
	if a.r.exists[a.orig.BusinessKey] {
		return effects.Verdict{Status: effects.VerdictAbsent}, nil
	}
	return effects.Verdict{Status: effects.VerdictConfirmed, Proof: map[string]any{"gone": true}}, nil
}

type compEnv struct {
	*authEnvT
	remote *remoteState
	comp   *compensation.Compensator
	ledger *effects.Ledger
}

func newCompEnv(t *testing.T) *compEnv {
	t.Helper()
	a := newAuthzEnv(t)
	r := &remoteState{exists: map[string]bool{}}
	ledger := &effects.Ledger{DB: a.e.db.Pool, Lease: time.Minute, MaxAttempts: 2}
	c := &compEnv{authEnvT: a, remote: r, ledger: ledger}
	c.comp = &compensation.Compensator{DB: a.e.db.Pool, Ledger: ledger, Auth: a.store,
		Adapter: func(orig persistence.OutboxEntry, _ compensation.Spec) effects.Adapter {
			return undoAdapter{r: r, orig: orig}
		}}
	return c
}

// confirmed cria um efeito original confirmado no ledger.
func (c *compEnv) confirmed(t *testing.T, key, op string) persistence.OutboxEntry {
	t.Helper()
	res, err := c.ledger.Run(context.Background(), effects.Spec{ProjectID: c.proj, BusinessKey: key, Operation: op, Request: map[string]any{"k": key}}, originalAdapter{c.remote})
	if err != nil || res.Outcome != effects.OutcomeApplied {
		t.Fatalf("%+v %v", res, err)
	}
	return res.Entry
}

func (c *compEnv) req(e persistence.OutboxEntry, auth string) compensation.Request {
	return compensation.Request{ProjectID: c.proj, ExecutionID: c.exec, OutboxID: e.ID, AuthorizationID: auth}
}

func eventTypes(t *testing.T, c *compEnv, id string) []string {
	t.Helper()
	evs, err := c.comp.Events(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range evs {
		out = append(out, e.Type)
	}
	return out
}

// T13-05 — baixo risco: compensação idempotente, com prova, auditável.
func TestLowRiskCompensationIsIdempotentAndProvenAndAudited(t *testing.T) {
	c := newCompEnv(t)
	ctx := context.Background()
	e := c.confirmed(t, "P:F:1:subtask:t1", "create_subtask")
	rec, err := c.comp.Compensate(ctx, c.req(e, ""))
	if err != nil || rec.State != "applied" || rec.Risk != "low" || c.remote.has(e.BusinessKey) || c.remote.undoCount() != 1 {
		t.Fatalf("%+v %v", rec, err)
	}
	again, err := c.comp.Compensate(ctx, c.req(e, ""))
	if err != nil || again.ID != rec.ID || again.State != "applied" || c.remote.undoCount() != 1 {
		t.Fatalf("repetir não desfaz de novo: %+v %v undos=%d", again, err, c.remote.undoCount())
	}
	if got := eventTypes(t, c, rec.ID); len(got) != 1 || got[0] != "applied" {
		t.Fatalf("%v", got)
	}
	// Resposta perdida depois de desfazer: a prova reconcilia, sem desfazer duas vezes.
	c.remote.loseReply = true
	e2 := c.confirmed(t, "P:F:1:subtask:t2", "create_subtask")
	rec2, err := c.comp.Compensate(ctx, c.req(e2, ""))
	if err != nil || rec2.State != "applied" || c.remote.has(e2.BusinessKey) || c.remote.undoCount() != 2 {
		t.Fatalf("%+v %v undos=%d", rec2, err, c.remote.undoCount())
	}
	// Só se compensa o que foi confirmado.
	pending, _, _ := persistence.EnqueueOutbox(ctx, c.e.db.Pool, c.proj, "P:F:1:subtask:t3", "create_subtask", map[string]any{})
	if _, err := c.comp.Compensate(ctx, c.req(pending, "")); !errors.Is(err, compensation.ErrNotConfirmed) {
		t.Fatalf("%v", err)
	}
}

// T13-05 — sensível: só com autorização presa à compensação.
func TestSensitiveCompensationNeedsAMatchingAuthorization(t *testing.T) {
	c := newCompEnv(t)
	ctx := context.Background()
	e := c.confirmed(t, "P:F:1:push:lf/f-1-a1:abc", "push")
	spec := compensation.DefaultSpecs["push"]
	rec, err := c.comp.Compensate(ctx, c.req(e, ""))
	if !errors.Is(err, compensation.ErrNeedsAuth) || rec.State != "refused" || c.remote.undoCount() != 0 || !c.remote.has(e.BusinessKey) {
		t.Fatalf("%+v %v", rec, err)
	}
	// Autorização para OUTRA compensação: recusada, sem gastar uso.
	other := c.confirmed(t, "P:F:1:push:lf/f-1-a1:def", "push")
	wrong, err := c.store.Create(ctx, authorizations.NewAuthorization{ProjectID: c.proj, ExecutionID: c.exec, Action: compensation.ActionFor(other, spec), MaxUses: 1, TTL: time.Hour, GrantedBy: c.admin})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.comp.Compensate(ctx, c.req(e, wrong.ID)); !errors.Is(err, compensation.ErrNeedsAuth) || c.remote.undoCount() != 0 || c.uses(t, wrong.ID) != 0 {
		t.Fatalf("%v undos=%d usos=%d", err, c.remote.undoCount(), c.uses(t, wrong.ID))
	}
	// Autorização correta: aplica e gasta o único uso.
	good, _ := c.store.Create(ctx, authorizations.NewAuthorization{ProjectID: c.proj, ExecutionID: c.exec, Action: compensation.ActionFor(e, spec), MaxUses: 1, TTL: time.Hour, GrantedBy: c.admin})
	rec, err = c.comp.Compensate(ctx, c.req(e, good.ID))
	if err != nil || rec.State != "applied" || rec.Risk != "sensitive" || c.remote.has(e.BusinessKey) || c.uses(t, good.ID) != 1 {
		t.Fatalf("%+v %v", rec, err)
	}
	// A mesma autorização não serve a outro efeito.
	if _, err := c.comp.Compensate(ctx, c.req(other, good.ID)); !errors.Is(err, compensation.ErrNeedsAuth) || !c.remote.has(other.BusinessKey) {
		t.Fatalf("%v", err)
	}
	if got := eventTypes(t, c, rec.ID); len(got) == 0 || got[len(got)-1] != "applied" {
		t.Fatalf("%v", got)
	}
}

// T13-05 — irreversível: preservado e recusado, de forma auditável.
func TestIrreversibleEffectsArePreservedAndTheRefusalIsAudited(t *testing.T) {
	c := newCompEnv(t)
	ctx := context.Background()
	for _, op := range []string{"merge", "send_message", "operação-desconhecida"} {
		e := c.confirmed(t, "P:F:1:"+op, op)
		rec, err := c.comp.Compensate(ctx, c.req(e, ""))
		if !errors.Is(err, compensation.ErrIrreversible) || rec.State != "refused" || rec.Risk != "irreversible" || !c.remote.has(e.BusinessKey) {
			t.Fatalf("%s: %+v %v", op, rec, err)
		}
		if again, err := c.comp.Compensate(ctx, c.req(e, "")); !errors.Is(err, compensation.ErrIrreversible) || again.ID != rec.ID {
			t.Fatalf("%v", err)
		}
		if got := eventTypes(t, c, rec.ID); len(got) != 1 || got[0] != "refused" {
			t.Fatalf("%v", got)
		}
	}
	if c.remote.undoCount() != 0 {
		t.Fatal("nada irreversível pode ser desfeito")
	}
}

// T13-05 — "Falha conserva trabalho e bloqueia nova operação agravante."
func TestFailedCompensationKeepsTheWorkAndBlocksAggravatingOperations(t *testing.T) {
	c := newCompEnv(t)
	ctx := context.Background()
	e := c.confirmed(t, "P:F:1:subtask:t9", "create_subtask")
	c.remote.failUndo = true
	rec, err := c.comp.Compensate(ctx, c.req(e, ""))
	if !errors.Is(err, compensation.ErrFailed) || rec.State != "failed" || !c.remote.has(e.BusinessKey) {
		t.Fatalf("o efeito segue lá e a falha é registrada: %+v %v", rec, err)
	}
	// Bloqueio: operações que alteram estado são recusadas; leitura e compensação, não.
	for _, op := range []string{"push", "create_subtask", "transition", "merge"} {
		if err := c.comp.CheckAggravating(ctx, c.proj, c.exec, op); !errors.Is(err, compensation.ErrBlocked) {
			t.Errorf("%s deveria ser bloqueada: %v", op, err)
		}
	}
	for _, op := range []string{"read_file", "list_dir", "get_item", "search", "compensate:delete_subtask"} {
		if err := c.comp.CheckAggravating(ctx, c.proj, c.exec, op); err != nil {
			t.Errorf("%s deve seguir permitida: %v", op, err)
		}
	}
	// Outra execução não é afetada.
	other := waitingExecution(t, c.e, c.proj, "CP-2")
	if err := c.comp.CheckAggravating(ctx, c.proj, other.ID, "push"); err != nil {
		t.Fatalf("%v", err)
	}
	// Corrigido o problema, repetir a compensação aplica e libera o bloqueio.
	c.remote.failUndo = false
	rec2, err := c.comp.Compensate(ctx, c.req(e, ""))
	if err != nil || rec2.ID != rec.ID || rec2.State != "applied" || rec2.Attempts < 2 || c.remote.has(e.BusinessKey) {
		t.Fatalf("%+v %v", rec2, err)
	}
	if err := c.comp.CheckAggravating(ctx, c.proj, c.exec, "push"); err != nil {
		t.Fatalf("bloqueio resolvido: %v", err)
	}
	if got := eventTypes(t, c, rec.ID); len(got) < 4 || got[0] != "failed" || got[1] != "blocked" {
		t.Fatalf("%v", got)
	}
	// Resolução humana do bloqueio, sem compensar.
	e2 := c.confirmed(t, "P:F:1:subtask:t10", "create_subtask")
	c.remote.failUndo = true
	rec3, err := c.comp.Compensate(ctx, c.req(e2, ""))
	if !errors.Is(err, compensation.ErrFailed) {
		t.Fatalf("%v", err)
	}
	if err := c.comp.ResolveBlock(ctx, rec3.ID, c.admin, "tratado manualmente na origem"); err != nil {
		t.Fatal(err)
	}
	if err := c.comp.CheckAggravating(ctx, c.proj, c.exec, "push"); err != nil {
		t.Fatalf("%v", err)
	}
	if err := c.comp.ResolveBlock(ctx, rec3.ID, c.admin, "de novo"); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("%v", err)
	}
	if got := eventTypes(t, c, rec3.ID); got[len(got)-1] != "block_resolved" {
		t.Fatalf("%v", got)
	}
}
