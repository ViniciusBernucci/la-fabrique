package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/availability"
	"github.com/ViniciusBernucci/le-fabrique/internal/budget"
	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/models"
)

type avEnv struct {
	cxEnv
	now  time.Time
	h    *availability.Tracker
	r    *availability.Resolver
	bud  *budget.Store
	a, b string // contas da cadeia (principal, alternativa)
	c    string // conta saudável FORA da cadeia
}

func newAvEnv(t *testing.T) *avEnv {
	t.Helper()
	e := &avEnv{cxEnv: newCxEnv(t, "av")}
	e.now = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	clock := func() time.Time { return e.now }
	pool := e.e.db.Pool
	e.h = &availability.Tracker{DB: pool, Now: clock}
	e.bud = &budget.Store{DB: pool, Now: clock}
	e.r = &availability.Resolver{DB: pool, Health: e.h, Checkpoints: e.store, Budget: e.bud, Now: clock}
	c := e.cl
	for _, n := range []string{"m-a", "m-b", "m-x"} {
		in := models.Input{Provider: "api", Gateway: models.GatewayOpenRouter, Name: n, Label: n, SupportsTools: true, ContextWindow: 100000, MaxOutputTokens: 4000}
		if _, err := models.Register(context.Background(), pool, in); err != nil {
			t.Fatal(err)
		}
	}
	e.a = connWith(t, c, "api", "av-a", "openrouter").ID
	e.b = connWith(t, c, "api", "av-b", "openrouter").ID
	e.c = connWith(t, c, "api", "av-c", "openrouter").ID
	return e
}

func (e *avEnv) req() availability.Request {
	return availability.Request{ProjectID: e.proj, ExecutionID: e.exec, NodeKey: "dev", Provider: "api", NeedsTools: true,
		Chain: []availability.Step{{ConnectionID: e.a, Model: "m-a"}, {ConnectionID: e.b, Model: "m-b"}}}
}

func (e *avEnv) fail(r availability.Request, f availability.Failure, safe bool, cpID string) availability.Request {
	cur := r.Chain[0]
	r.Current, r.Failure, r.Quiescent, r.CheckpointID = &cur, &f, safe, cpID
	return r
}

// T09-04 — "Cota no principal leva só à alternativa autorizada."
func TestQuotaOnThePrimaryLeadsOnlyToTheAuthorizedAlternative(t *testing.T) {
	e := newAvEnv(t)
	ctx := context.Background()
	cp := e.checkpoint(t, "antes da troca", "package main")
	d, err := e.r.Choose(ctx, e.fail(e.req(), availability.Failure{Class: availability.Quota, Official: true, Detail: "429 quota"}, true, cp.ID))
	if err != nil || d.Action != availability.ActionSwitch || d.Step.ConnectionID != e.b || d.Step.Model != "m-b" {
		t.Fatalf("%+v %v", d, err)
	}
	if len(d.Skipped) != 1 || d.Skipped[0].Step.ConnectionID != e.a {
		t.Fatalf("o principal deve constar como descartado: %+v", d.Skipped)
	}
	// A alternativa também cai: a conta C está saudável, mas NÃO está na cadeia → nunca é usada.
	r2 := e.req()
	r2.Current = &availability.Step{ConnectionID: e.b, Model: "m-b"}
	r2.Failure = &availability.Failure{Class: availability.Billing, Official: true}
	r2.Quiescent, r2.CheckpointID = true, cp.ID
	d, err = e.r.Choose(ctx, r2)
	if err != nil || d.Step != nil || d.Action != availability.ActionWait {
		t.Fatalf("%+v %v", d, err)
	}
	ev, _ := e.r.Events(ctx, e.proj, e.exec)
	if len(ev) != 2 || ev[0].Action != "switch" || ev[0].ToConn != e.b || ev[0].FromConn != e.a || ev[1].Action != "wait" {
		t.Fatalf("trilha: %+v", ev)
	}
}

func TestSwitchingRequiresAnIntactCheckpointAndAQuietSandbox(t *testing.T) {
	e := newAvEnv(t)
	ctx := context.Background()
	f := availability.Failure{Class: availability.Quota, Official: true}
	for name, tc := range map[string]struct {
		safe bool
		cp   string
	}{"sem nada": {false, ""}, "sem quiescência": {false, e.checkpoint(t, "x", "package a").ID}, "sem checkpoint": {true, ""}} {
		if _, err := e.r.Choose(ctx, e.fail(e.req(), f, tc.safe, tc.cp)); !errors.Is(err, availability.ErrNotSafe) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// Checkpoint de outra execução.
	other := newCxEnv(t, "av-other")
	foreign := other.checkpoint(t, "alheio", "package o")
	if _, err := e.r.Choose(ctx, e.fail(e.req(), f, true, foreign.ID)); !errors.Is(err, availability.ErrNotSafe) {
		t.Fatalf("alheio: %v", err)
	}
	// Checkpoint corrompido.
	cp := e.checkpoint(t, "c", "package c")
	if _, err := e.e.db.Pool.Exec(ctx, `UPDATE checkpoints SET tree_hash = repeat('0', 64) WHERE id = $1`, cp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.r.Choose(ctx, e.fail(e.req(), f, true, cp.ID)); !errors.Is(err, availability.ErrNotSafe) && !errors.Is(err, checkpoints.ErrCorrupt) {
		t.Fatalf("corrompido: %v", err)
	}
	ev, _ := e.r.Events(ctx, e.proj, e.exec)
	for _, x := range ev {
		if x.Action == "switch" {
			t.Fatalf("nenhuma troca pode ter sido registrada: %+v", ev)
		}
	}
}

// T09-04 — "Esgotamento preserva tarefa e outra avança."
func TestExhaustionPreservesTheTaskWhileAnotherAdvances(t *testing.T) {
	e := newAvEnv(t)
	ctx := context.Background()
	cp := e.checkpoint(t, "seguro", "package main")
	for _, c := range []string{e.a, e.b} {
		if _, err := e.h.Record(ctx, c, availability.Failure{Class: availability.Quota, Official: true}); err != nil {
			t.Fatal(err)
		}
	}
	r := e.req()
	r.Quiescent, r.CheckpointID = true, cp.ID
	d, err := e.r.Choose(ctx, r)
	if err != nil || d.Action != availability.ActionWait || d.RetryAt == nil || !d.RetryAt.Equal(e.now.Add(availability.DefaultQuotaWait)) || len(d.Skipped) != 2 {
		t.Fatalf("%+v %v", d, err)
	}
	var released bool
	if err := e.e.db.Pool.QueryRow(ctx, `SELECT slot_released FROM waiting_tasks WHERE execution_id = $1`, e.exec).Scan(&released); err != nil || !released {
		t.Fatalf("com checkpoint seguro a vaga é liberada: %v %v", released, err)
	}
	// Sem checkpoint seguro a tarefa mantém a vaga (hold).
	r.Quiescent, r.CheckpointID = false, ""
	d, _ = e.r.Choose(ctx, r)
	if d.Action != availability.ActionHold {
		t.Fatalf("%+v", d)
	}
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT slot_released FROM waiting_tasks WHERE execution_id = $1`, e.exec).Scan(&released)
	if released {
		t.Fatal("sem checkpoint seguro a vaga não pode ser liberada")
	}
	// Outra execução, com outra cadeia saudável, avança normalmente.
	other := waitingExecution(t, e.e, e.proj, "AV-2")
	ro := e.req()
	ro.ExecutionID = other.ID
	ro.Chain = []availability.Step{{ConnectionID: e.c, Model: "m-x"}}
	d, err = e.r.Choose(ctx, ro)
	if err != nil || d.Action != availability.ActionUse || d.Step.ConnectionID != e.c {
		t.Fatalf("%+v %v", d, err)
	}
	// O horário chega: a tarefa aparece como devida e, recuperada a conta, volta ao principal.
	if due, _ := e.r.Due(ctx); len(due) != 0 {
		t.Fatalf("ainda não venceu: %+v", due)
	}
	e.now = e.now.Add(availability.DefaultQuotaWait + time.Second)
	due, _ := e.r.Due(ctx)
	if len(due) != 1 || due[0].ExecutionID != e.exec {
		t.Fatalf("%+v", due)
	}
	d, err = e.r.Choose(ctx, e.req())
	if err != nil || d.Action != availability.ActionUse || d.Step.ConnectionID != e.a {
		t.Fatalf("%+v %v", d, err)
	}
	if due, _ := e.r.Due(ctx); len(due) != 0 {
		t.Fatalf("a espera deve terminar: %+v", due)
	}
}

func TestWaitBacksOffExponentiallyWhenNothingIsOfficial(t *testing.T) {
	e := newAvEnv(t)
	ctx := context.Background()
	// Conta fora do ar sem prazo: duas falhas seguidas; nova escolha com cadeia só dela.
	for i := 0; i < 2; i++ {
		_, _ = e.h.Record(ctx, e.a, availability.Failure{Class: availability.Unavailable})
	}
	r := e.req()
	r.Chain = r.Chain[:1]
	var prev time.Time
	d, _ := e.r.Choose(ctx, r)
	if d.Action != availability.ActionHold || d.RetryAt == nil {
		t.Fatalf("%+v", d)
	}
	prev = *d.RetryAt
	if !prev.Equal(e.now.Add(availability.BackoffBase)) {
		t.Fatalf("primeira espera: %v", prev)
	}
	if availability.Backoff(1) != 30*time.Second || availability.Backoff(2) != time.Minute || availability.Backoff(3) != 2*time.Minute || availability.Backoff(50) != availability.BackoffMax {
		t.Fatal("backoff exponencial com teto")
	}
}

// T09-04 — "Disponibilidade volta sem rodízio automático."
func TestAvailabilityReturnsWithoutAutomaticRotation(t *testing.T) {
	e := newAvEnv(t)
	ctx := context.Background()
	cp := e.checkpoint(t, "x", "package main")
	if _, err := e.r.Choose(ctx, e.fail(e.req(), availability.Failure{Class: availability.Quota, RetryAt: ptrTime(e.now.Add(10 * time.Minute))}, true, cp.ID)); err != nil {
		t.Fatal(err)
	}
	e.now = e.now.Add(11 * time.Minute) // o principal voltou
	for i := 0; i < 10; i++ {
		d, err := e.r.Choose(ctx, e.req())
		if err != nil || d.Action != availability.ActionUse || d.Step.ConnectionID != e.a {
			t.Fatalf("sempre o principal, sem rodízio: %+v %v", d, err)
		}
	}
	ev, _ := e.r.Events(ctx, e.proj, e.exec)
	if len(ev) != 1 {
		t.Fatalf("a recuperação não gera troca automática: %+v", ev)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

// T09-04 — autenticação, cota, indisponibilidade e bloqueio financeiro são diferentes.
func TestFailureClassesHaveDistinctRecoveryRules(t *testing.T) {
	e := newAvEnv(t)
	ctx := context.Background()
	// Indisponibilidade isolada é ruído; a segunda marca a conta, com backoff.
	h, _ := e.h.Record(ctx, e.a, availability.Failure{Class: availability.Unavailable})
	if !h.Usable() || h.Failures != 1 {
		t.Fatalf("%+v", h)
	}
	h, _ = e.h.Record(ctx, e.a, availability.Failure{Class: availability.Unavailable})
	if h.State != availability.Unavailable || h.Until == nil || !h.Until.Equal(e.now.Add(availability.BackoffBase)) {
		t.Fatalf("%+v", h)
	}
	h, _ = e.h.Record(ctx, e.a, availability.Failure{Class: availability.Unavailable})
	if !h.Until.Equal(e.now.Add(2 * availability.BackoffBase)) {
		t.Fatalf("backoff cresce: %+v", h)
	}
	e.now = e.now.Add(3 * time.Minute)
	if h, _ = e.h.Get(ctx, e.a); !h.Usable() {
		t.Fatalf("o prazo vencido libera: %+v", h)
	}
	if err := e.h.Success(ctx, e.a); err != nil {
		t.Fatal(err)
	}
	if h, _ = e.h.Get(ctx, e.a); h.Failures != 0 {
		t.Fatalf("sucesso zera o histórico: %+v", h)
	}
	// Autenticação e financeiro: o tempo não resolve; só Clear.
	for _, class := range []string{availability.Auth, availability.Billing} {
		h, _ = e.h.Record(ctx, e.b, availability.Failure{Class: class, Official: true, Detail: "401/402"})
		if h.State != class || h.Until != nil || h.Source != availability.SourceOfficial {
			t.Fatalf("%+v", h)
		}
		e.now = e.now.Add(48 * time.Hour)
		if h, _ = e.h.Get(ctx, e.b); h.Usable() {
			t.Fatalf("%s não some com o tempo", class)
		}
		if err := e.h.Clear(ctx, e.b, "credencial renovada"); err != nil {
			t.Fatal(err)
		}
		if h, _ = e.h.Get(ctx, e.b); !h.Usable() {
			t.Fatalf("%+v", h)
		}
	}
	// Todas as contas da cadeia dependem de intervenção: sem horário de nova tentativa.
	_, _ = e.h.Record(ctx, e.a, availability.Failure{Class: availability.Auth})
	_, _ = e.h.Record(ctx, e.b, availability.Failure{Class: availability.Billing})
	d, _ := e.r.Choose(ctx, e.req())
	if d.RetryAt != nil || d.Action != availability.ActionHold {
		t.Fatalf("%+v", d)
	}
	if _, err := e.h.Record(ctx, e.a, availability.Failure{Class: "outra"}); err == nil {
		t.Fatal("classe inválida")
	}
	var n int
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM provider_events WHERE connection_id = $1`, e.b).Scan(&n)
	if n < 4 {
		t.Fatalf("histórico de eventos: %d", n)
	}
}

// T09-04 — "combinar sinais oficiais e histórico": a cota observada pelo adaptador vale.
func TestObservedSubscriptionQuotaIsAnOfficialSignal(t *testing.T) {
	e := newAvEnv(t)
	ctx := context.Background()
	reset := e.now.Add(2 * time.Hour)
	if err := e.bud.ObserveQuota(ctx, budget.QuotaObservation{ConnectionID: e.a, Used: 100, Limit: 100, Unit: "mensagens", ResetsAt: &reset, Source: "cli:/usage", ObservedAt: e.now}); err != nil {
		t.Fatal(err)
	}
	h, err := e.h.Get(ctx, e.a)
	if err != nil || h.State != availability.Quota || h.Source != availability.SourceOfficial || h.Until == nil || !h.Until.Equal(reset) {
		t.Fatalf("%+v %v", h, err)
	}
	e.now = reset.Add(time.Second) // reset passou
	if h, _ = e.h.Get(ctx, e.a); !h.Usable() {
		t.Fatalf("%+v", h)
	}
	// Falha por cota registrada e depois uma observação com folga libera a conta.
	e.now = reset.Add(time.Hour)
	_, _ = e.h.Record(ctx, e.b, availability.Failure{Class: availability.Quota})
	if h, _ = e.h.Get(ctx, e.b); h.State != availability.Quota {
		t.Fatalf("%+v", h)
	}
	e.now = e.now.Add(time.Minute)
	_ = e.bud.ObserveQuota(ctx, budget.QuotaObservation{ConnectionID: e.b, Used: 10, Limit: 100, Unit: "mensagens", Source: "cli:/usage", ObservedAt: e.now})
	if h, _ = e.h.Get(ctx, e.b); !h.Usable() {
		t.Fatalf("observação posterior com folga libera: %+v", h)
	}
}

func TestFallbackSkipsIncompatibleAndOverBudgetAlternatives(t *testing.T) {
	e := newAvEnv(t)
	ctx := context.Background()
	cp := e.checkpoint(t, "x", "package main")
	// Alternativa com modelo não cadastrado → incompatível; terceira dentro do orçamento.
	r := e.req()
	r.Chain = []availability.Step{{ConnectionID: e.a, Model: "m-a"}, {ConnectionID: e.b, Model: "nao-existe"}, {ConnectionID: e.c, Model: "m-x"}}
	d, err := e.r.Choose(ctx, e.fail(r, availability.Failure{Class: availability.Quota, Official: true}, true, cp.ID))
	if err != nil || d.Step == nil || d.Step.ConnectionID != e.c || len(d.Skipped) != 2 {
		t.Fatalf("%+v %v", d, err)
	}
	// Orçamento: sem orçamento global a alternativa é recusada; com folga, passa; sem folga, recusa.
	e2 := newAvEnv(t)
	cp2 := e2.checkpoint(t, "y", "package main")
	r2 := e2.req()
	r2.EstimateMicros = 5_000
	d, _ = e2.r.Choose(ctx, e2.fail(r2, availability.Failure{Class: availability.Quota, Official: true}, true, cp2.ID))
	if d.Step != nil || d.Action != availability.ActionWait {
		t.Fatalf("sem orçamento global nada é gasto: %+v", d)
	}
	if _, err := e2.bud.Set(ctx, budget.SetInput{Scope: "global", Period: "month", LimitMicros: 1_000}); err != nil {
		t.Fatal(err)
	}
	e2.now = e2.now.Add(2 * availability.DefaultQuotaWait) // quota liberada, mas o orçamento recusa
	d, _ = e2.r.Choose(ctx, r2)
	if d.Step != nil {
		t.Fatalf("estimativa acima do teto: %+v", d)
	}
	if _, err := e2.bud.Set(ctx, budget.SetInput{Scope: "global", Period: "month", LimitMicros: 1_000_000}); err != nil {
		t.Fatal(err)
	}
	d, _ = e2.r.Choose(ctx, r2)
	if d.Step == nil || d.Step.ConnectionID != e2.a {
		t.Fatalf("%+v", d)
	}
	var active int
	_ = e2.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM usage_reservations WHERE state = 'active'`).Scan(&active)
	if active != 0 {
		t.Fatalf("a reserva de teste deve ser liberada: %d", active)
	}
}

func TestChooseRejectsAnEmptyChainAndAnUnidentifiedFailure(t *testing.T) {
	e := newAvEnv(t)
	r := e.req()
	r.Chain = nil
	if _, err := e.r.Choose(context.Background(), r); !errors.Is(err, availability.ErrNoChain) {
		t.Fatal(err)
	}
	r = e.req()
	r.Failure = &availability.Failure{Class: availability.Quota}
	if _, err := e.r.Choose(context.Background(), r); err == nil {
		t.Fatal("falha sem passo de origem")
	}
}
