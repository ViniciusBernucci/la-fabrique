package integration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/budget"
	"github.com/ViniciusBernucci/le-fabrique/internal/gateways"
	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type budEnv struct {
	e     *authEnv
	c     *client
	store *budget.Store
	now   *time.Time
	mu    sync.Mutex
}

func newBudEnv(t *testing.T) *budEnv {
	t.Helper()
	e, c, _ := newProjectEnv(t)
	b := &budEnv{e: e, c: c}
	start := time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	b.now = &start
	b.store = &budget.Store{DB: e.db.Pool, Now: func() time.Time { b.mu.Lock(); defer b.mu.Unlock(); return *b.now }}
	return b
}

func (b *budEnv) advance(d time.Duration) { b.mu.Lock(); *b.now = b.now.Add(d); b.mu.Unlock() }

func (b *budEnv) agent(t *testing.T, name string) string {
	t.Helper()
	var a agents.Agent
	b.c.expect(b.c.post("/api/agents", agentBody(name, nil)), 201).JSON(&a)
	return a.ID
}

func (b *budEnv) setGlobal(t *testing.T, limit int64) {
	t.Helper()
	if _, err := b.store.Set(context.Background(), budget.SetInput{Scope: "global", Period: "month", LimitMicros: limit}); err != nil {
		t.Fatal(err)
	}
}

func isExceeded(err error) bool { var ex *budget.ExceededError; return errors.As(err, &ex) }

// T09-03 — "Chamadas concorrentes disputam reserva sem ultrapassar teto autorizado."
func TestConcurrentReservationsNeverExceedTheAuthorizedCeiling(t *testing.T) {
	b := newBudEnv(t)
	ctx := context.Background()
	b.setGlobal(t, 1_000_000)
	var ok, exceeded, other atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := b.store.Reserve(ctx, budget.ReserveRequest{AmountMicros: 100_000})
			switch {
			case err == nil:
				ok.Add(1)
			case isExceeded(err):
				exceeded.Add(1)
			default:
				other.Add(1)
				t.Errorf("erro inesperado: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if ok.Load() != 10 || exceeded.Load() != 30 || other.Load() != 0 {
		t.Fatalf("exatamente 10 reservas cabem: ok=%d excedido=%d", ok.Load(), exceeded.Load())
	}
	st, _ := b.store.List(ctx)
	if len(st) != 1 || st[0].ReservedMicros != 1_000_000 || st[0].RemainingMicros != 0 {
		t.Fatalf("%+v", st)
	}

	// Com um teto por agente: o agente nunca passa do dele e o total nunca passa do global.
	b2 := newBudEnv(t)
	b2.setGlobal(t, 1_500_000)
	a, other2 := b2.agent(t, "a"), b2.agent(t, "b")
	if _, err := b2.store.Set(ctx, budget.SetInput{Scope: "agent", AgentID: a, Period: "month", LimitMicros: 300_000}); err != nil {
		t.Fatal(err)
	}
	var okA, okB atomic.Int64
	start = make(chan struct{})
	for i := 0; i < 40; i++ {
		agent, counter := a, &okA
		if i%2 == 1 {
			agent, counter = other2, &okB
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := b2.store.Reserve(ctx, budget.ReserveRequest{AgentID: agent, AmountMicros: 100_000}); err == nil {
				counter.Add(1)
			} else if !isExceeded(err) {
				t.Errorf("%v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	// Quem sobra é decidido pela disputa, mas as invariantes são fixas.
	if okA.Load() > 3 || okA.Load()+okB.Load() != 15 {
		t.Fatalf("agente A=%d (teto 3), total=%d (teto 15)", okA.Load(), okA.Load()+okB.Load())
	}
}

func TestReservationSettlementReleaseAndExpiry(t *testing.T) {
	b := newBudEnv(t)
	ctx := context.Background()
	b.setGlobal(t, 1_000_000)
	ag := b.agent(t, "dev")
	res, err := b.store.Reserve(ctx, budget.ReserveRequest{AgentID: ag, AmountMicros: 600_000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.store.Reserve(ctx, budget.ReserveRequest{AgentID: ag, AmountMicros: 500_000}); !isExceeded(err) {
		t.Fatalf("a reserva ativa conta contra o teto: %v", err)
	}
	// Liquidar abaixo do reservado devolve a folga.
	st, err := b.store.Settle(ctx, res.ID, budget.Actual{Model: "m", InputTokens: 100, OutputTokens: 50, CostMicros: 100_000, Source: budget.SourceProvider})
	if err != nil || st.CostMicros != 100_000 || st.OverrunMicros != 0 || st.AlreadySettled {
		t.Fatalf("%+v %v", st, err)
	}
	if again, err := b.store.Settle(ctx, res.ID, budget.Actual{CostMicros: 999, Source: budget.SourceProvider}); err != nil || !again.AlreadySettled || again.UsageID != st.UsageID || again.CostMicros != 100_000 {
		t.Fatalf("liquidar de novo é idempotente: %+v %v", again, err)
	}
	if r2, err := b.store.Reserve(ctx, budget.ReserveRequest{AgentID: ag, AmountMicros: 900_000}); err != nil {
		t.Fatalf("a folga voltou: %v", err)
	} else {
		// Gasto acima do reservado é registrado e vira overrun; passa a contar.
		over, err := b.store.Settle(ctx, r2.ID, budget.Actual{CostMicros: 950_000, Source: budget.SourceProvider})
		if err != nil || over.OverrunMicros != 50_000 {
			t.Fatalf("%+v %v", over, err)
		}
	}
	if _, err := b.store.Reserve(ctx, budget.ReserveRequest{AgentID: ag, AmountMicros: 1}); !isExceeded(err) {
		t.Fatalf("o gasto real esgotou o teto (100k + 950k > 1M): %v", err)
	}
	// Liberar devolve; liberar de novo é inofensivo; liquidar depois de liberar é recusado.
	b2 := newBudEnv(t)
	b2.setGlobal(t, 100_000)
	r3, _ := b2.store.Reserve(ctx, budget.ReserveRequest{AmountMicros: 100_000})
	if err := b2.store.Release(ctx, r3.ID); err != nil || b2.store.Release(ctx, r3.ID) != nil {
		t.Fatal("liberar é idempotente")
	}
	if _, err := b2.store.Settle(ctx, r3.ID, budget.Actual{CostMicros: 1, Source: budget.SourceProvider}); !errors.Is(err, budget.ErrReservationClosed) {
		t.Fatalf("%v", err)
	}
	r4, err := b2.store.Reserve(ctx, budget.ReserveRequest{AmountMicros: 100_000})
	if err != nil {
		t.Fatalf("a liberação devolveu o saldo: %v", err)
	}
	if _, err := b2.store.Settle(ctx, r4.ID, budget.Actual{CostMicros: 10, Source: budget.SourceProvider}); err != nil {
		t.Fatal(err)
	}
	if err := b2.store.Release(ctx, r4.ID); !errors.Is(err, budget.ErrSettled) {
		t.Fatalf("liberar o que já foi liquidado: %v", err)
	}
	// Reserva abandonada expira sozinha e devolve o saldo.
	b3 := newBudEnv(t)
	b3.setGlobal(t, 100_000)
	if _, err := b3.store.Reserve(ctx, budget.ReserveRequest{AmountMicros: 100_000, TTL: time.Minute}); err != nil {
		t.Fatal(err)
	}
	if _, err := b3.store.Reserve(ctx, budget.ReserveRequest{AmountMicros: 1}); !isExceeded(err) {
		t.Fatalf("%v", err)
	}
	b3.advance(2 * time.Minute)
	if _, err := b3.store.Reserve(ctx, budget.ReserveRequest{AmountMicros: 100_000}); err != nil {
		t.Fatalf("a reserva vencida não conta mais: %v", err)
	}
}

func TestBudgetFailsClosedAndWindowsRoll(t *testing.T) {
	b := newBudEnv(t)
	ctx := context.Background()
	if _, err := b.store.Reserve(ctx, budget.ReserveRequest{AmountMicros: 1}); !errors.Is(err, budget.ErrNoBudget) {
		t.Fatalf("sem orçamento global: %v", err)
	}
	ag := b.agent(t, "dev")
	if _, err := b.store.Set(ctx, budget.SetInput{Scope: "agent", AgentID: ag, Period: "month", LimitMicros: 2_000_000}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.store.Reserve(ctx, budget.ReserveRequest{AgentID: ag, AmountMicros: 1}); !errors.Is(err, budget.ErrNoBudget) {
		t.Fatalf("orçamento só de agente não substitui o global: %v", err)
	}
	if _, err := b.store.Set(ctx, budget.SetInput{Scope: "global", Period: "day", LimitMicros: 500_000}); err != nil {
		t.Fatal(err)
	}
	r, err := b.store.Reserve(ctx, budget.ReserveRequest{AgentID: ag, AmountMicros: 500_000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.store.Settle(ctx, r.ID, budget.Actual{CostMicros: 500_000, Source: budget.SourceProvider}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.store.Reserve(ctx, budget.ReserveRequest{AgentID: ag, AmountMicros: 1}); !isExceeded(err) {
		t.Fatalf("dia esgotado: %v", err)
	}
	b.advance(24 * time.Hour) // outro dia, mesmo mês: o teto diário zera, o do agente (mensal) não
	if _, err := b.store.Reserve(ctx, budget.ReserveRequest{AgentID: ag, AmountMicros: 500_000}); err != nil {
		t.Fatalf("o dia seguinte tem teto novo: %v", err)
	}
	var ex *budget.ExceededError
	if _, err := b.store.Reserve(ctx, budget.ReserveRequest{AgentID: ag, AmountMicros: 1}); !errors.As(err, &ex) || ex.Scope != "global" {
		t.Fatalf("%v", err)
	}
	// Desativar o orçamento global volta a falhar fechado.
	if _, err := b.store.Set(ctx, budget.SetInput{Scope: "global", Period: "day", LimitMicros: 500_000, State: "disabled"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.store.Reserve(ctx, budget.ReserveRequest{AmountMicros: 1}); !errors.Is(err, budget.ErrNoBudget) {
		t.Fatalf("%v", err)
	}
	for name, in := range map[string]budget.SetInput{
		"escopo":            {Scope: "x", Period: "day"},
		"global com agente": {Scope: "global", AgentID: ag, Period: "day"},
		"agente sem agente": {Scope: "agent", Period: "day"},
		"período":           {Scope: "global", Period: "ano"},
		"limite negativo":   {Scope: "global", Period: "day", LimitMicros: -1},
		"estado":            {Scope: "global", Period: "day", State: "x"},
	} {
		if _, err := b.store.Set(ctx, in); err == nil {
			t.Errorf("%s deveria ser recusado", name)
		}
	}
}

// T09-03 — "Cota de assinatura é registrada separadamente de gasto em API."
func TestSubscriptionQuotaIsRecordedSeparatelyFromAPISpend(t *testing.T) {
	b := newBudEnv(t)
	ctx := context.Background()
	b.setGlobal(t, 1_000_000)
	sub := connWith(t, b.c, "cli", "claude-assinatura", "claude")
	api := connWith(t, b.c, "api", "gateway", "openrouter")
	ag := b.agent(t, "dev")
	for i := 0; i < 5; i++ {
		if _, err := b.store.RecordSubscription(ctx, budget.SubscriptionUse{AgentID: ag, ConnectionID: sub.ID, Model: "claude-sonnet-5-5", Units: 1, Unit: "messages", InputTokens: 1000, OutputTokens: 400, Source: budget.SourceEstimated}); err != nil {
			t.Fatal(err)
		}
	}
	reset := b.now.Add(3 * time.Hour)
	if err := b.store.ObserveQuota(ctx, budget.QuotaObservation{ConnectionID: sub.ID, Used: 5, Limit: 45, Unit: "messages", ResetsAt: &reset, Source: "cli:/usage", ObservedAt: *b.now}); err != nil {
		t.Fatal(err)
	}
	if st, _ := b.store.List(ctx); st[0].SpentMicros != 0 {
		t.Fatalf("assinatura não consome o orçamento de API: %+v", st[0])
	}
	r, _ := b.store.Reserve(ctx, budget.ReserveRequest{AgentID: ag, ConnectionID: api.ID, AmountMicros: 200_000})
	if _, err := b.store.Settle(ctx, r.ID, budget.Actual{Model: "vendor/m", InputTokens: 10, OutputTokens: 10, CostMicros: 150_000, Source: budget.SourceProvider}); err != nil {
		t.Fatal(err)
	}
	rep, err := b.store.Report(ctx, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if rep.API.SpentMicros != 150_000 || rep.API.BySource[budget.SourceProvider] != 150_000 || len(rep.Subscription) != 1 || rep.Subscription[0].Units != 5 || rep.Subscription[0].Calls != 5 || rep.Subscription[0].ConnectionID != sub.ID {
		t.Fatalf("%+v", rep)
	}
	q, err := b.store.LatestQuota(ctx, sub.ID)
	if err != nil || q.Used != 5 || q.Limit != 45 || q.Source != "cli:/usage" || q.ResetsAt == nil {
		t.Fatalf("a cota fica com fonte e horário observados: %+v %v", q, err)
	}
	// O banco impede misturar: assinatura com custo e API com cota.
	if _, err := b.e.db.Pool.Exec(ctx, `INSERT INTO usage_records (kind, cost_micros, source) VALUES ('subscription', 10, 'estimated')`); err == nil {
		t.Fatal("assinatura com custo deveria ser recusada pelo banco")
	}
	if _, err := b.e.db.Pool.Exec(ctx, `INSERT INTO usage_records (kind, quota_units, quota_unit, source) VALUES ('api_cost', 3, 'x', 'estimated')`); err == nil {
		t.Fatal("API com cota deveria ser recusada pelo banco")
	}
	for name, u := range map[string]budget.SubscriptionUse{
		"sem conta":   {Units: 1, Unit: "messages", Source: budget.SourceEstimated},
		"sem unidade": {ConnectionID: sub.ID, Units: 1, Source: budget.SourceEstimated},
		"fonte":       {ConnectionID: sub.ID, Units: 1, Unit: "messages", Source: "chute"},
	} {
		if _, err := b.store.RecordSubscription(ctx, u); err == nil {
			t.Errorf("%s deveria ser recusado", name)
		}
	}
	if err := b.store.ObserveQuota(ctx, budget.QuotaObservation{ConnectionID: sub.ID, Used: 1, Limit: 2, Unit: "messages"}); err == nil {
		t.Fatal("cota sem fonte e horário não é uma observação")
	}
}

func TestReconciliationAdjustsMissingSpendAndKeepsTheHigherNumber(t *testing.T) {
	b := newBudEnv(t)
	ctx := context.Background()
	b.setGlobal(t, 10_000_000)
	api := connWith(t, b.c, "api", "gateway", "openrouter")
	r, _ := b.store.Reserve(ctx, budget.ReserveRequest{ConnectionID: api.ID, AmountMicros: 300_000})
	if _, err := b.store.Settle(ctx, r.ID, budget.Actual{CostMicros: 300_000, Source: budget.SourceEstimated}); err != nil {
		t.Fatal(err)
	}
	from, to := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	rec, err := b.store.Reconcile(ctx, api.ID, from, to, 450_000, "openrouter:/credits")
	if err != nil || rec.RecordedMicros != 300_000 || rec.DriftMicros != 150_000 || rec.AdjustmentID == 0 {
		t.Fatalf("%+v %v", rec, err)
	}
	if st, _ := b.store.List(ctx); st[0].SpentMicros != 450_000 {
		t.Fatalf("o total bate com o provedor: %+v", st[0])
	}
	rec2, err := b.store.Reconcile(ctx, api.ID, from, to, 450_000, "openrouter:/credits")
	if err != nil || rec2.DriftMicros != 0 || rec2.AdjustmentID != 0 {
		t.Fatalf("conferir de novo não lança ajuste: %+v %v", rec2, err)
	}
	// O provedor informa menos do que registramos: guarda a diferença e mantém o maior.
	rec3, err := b.store.Reconcile(ctx, api.ID, from, to, 400_000, "openrouter:/credits")
	if err != nil || rec3.DriftMicros != -50_000 || rec3.AdjustmentID != 0 {
		t.Fatalf("%+v %v", rec3, err)
	}
	if st, _ := b.store.List(ctx); st[0].SpentMicros != 450_000 {
		t.Fatalf("mantém o valor conservador: %+v", st[0])
	}
	var n int
	_ = b.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM usage_reconciliations WHERE connection_id = $1`, api.ID).Scan(&n)
	if n != 3 {
		t.Fatalf("toda conferência fica registrada: %d", n)
	}
}

func TestPriceEstimateIsConservativeAndRequiresASource(t *testing.T) {
	p := budget.Price{InputPerMTok: 3_000_000, OutputPerMTok: 15_000_000}
	if got := budget.CostOf(p, 1000, 500); got != 3_000+7_500 {
		t.Fatalf("%d", got)
	}
	if got := budget.CostOf(p, 1, 1); got != 3+15 { // arredonda para cima
		t.Fatalf("%d", got)
	}
	// Sem métrica oficial: ceil(bytes/3) por texto.
	if got := budget.EstimateTokens("abcdef", "g"); got != 3 {
		t.Fatalf("%d", got)
	}
	m := models.Model{Name: "x", Provider: "api"}
	if _, err := budget.PriceOf(m); !errors.Is(err, budget.ErrNoPrice) {
		t.Fatalf("%v", err)
	}
	in, out := int64(3_000_000), int64(15_000_000)
	m.InputMicrosPerMTok, m.OutputMicrosPerMTok = &in, &out
	if got, err := budget.PriceOf(m); err != nil || got != p {
		t.Fatalf("%+v %v", got, err)
	}
	// O catálogo exige os dois preços e a fonte.
	_, c, _ := newProjectEnv(t)
	base := map[string]any{"provider": "api", "gateway": "openrouter", "name": "vendor/priced"}
	withPrice := func(extra map[string]any) apiResp {
		body := map[string]any{}
		for k, v := range base {
			body[k] = v
		}
		for k, v := range extra {
			body[k] = v
		}
		return c.post("/api/models", body)
	}
	if r := withPrice(map[string]any{"input_micros_per_mtok": 3000000}); r.StatusCode != 400 {
		t.Fatalf("só um preço: %d", r.StatusCode)
	}
	if r := withPrice(map[string]any{"input_micros_per_mtok": 3000000, "output_micros_per_mtok": 15000000}); r.StatusCode != 400 {
		t.Fatalf("sem fonte: %d", r.StatusCode)
	}
	if r := withPrice(map[string]any{"price_source": "x"}); r.StatusCode != 400 {
		t.Fatalf("fonte sem preço: %d", r.StatusCode)
	}
	var created models.Model
	c.expect(withPrice(map[string]any{"input_micros_per_mtok": 3000000, "output_micros_per_mtok": 15000000, "price_source": "openrouter.ai/models, 2026-11-12"}), 201).JSON(&created)
	if created.InputMicrosPerMTok == nil || *created.OutputMicrosPerMTok != 15000000 || created.PriceSource == "" {
		t.Fatalf("%+v", created)
	}
}

// T09-03 — orçamento imposto no laço do gateway: sem reserva, nada é enviado.
func TestGatewayLoopEnforcesTheBudgetBeforeEveryCall(t *testing.T) {
	b := newBudEnv(t)
	ctx := context.Background()
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("X-Fail") == "1" {
			w.WriteHeader(500)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "ok"}, "finish_reason": "stop"}},
			"usage": map[string]any{"prompt_tokens": 1000, "completion_tokens": 500}})
	}))
	defer srv.Close()
	gw, err := gateways.New(models.GatewayOpenRouter, security.NewSecret("tok"), srv.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	in, out := int64(3_000_000), int64(15_000_000) // US$ 3 / US$ 15 por milhão
	maxTok := 1000
	model := models.Model{Provider: "api", Gateway: "openrouter", Name: "vendor/m", State: "active", SupportsTools: true, MaxOutputTokens: 2000,
		Params: models.Params{MaxTokens: &maxTok}, InputMicrosPerMTok: &in, OutputMicrosPerMTok: &out}
	api := connWith(t, b.c, "api", "gw", "openrouter")
	ag := b.agent(t, "dev")
	meter := budget.Meter{Store: b.store, AgentID: ag, ConnectionID: api.ID}
	run := func(m models.Model) (gateways.Result, error) {
		return gateways.Run(ctx, gw, gateways.RunConfig{Model: m, Prompt: "faça", Meter: meter})
	}

	// Sem orçamento global: nada é enviado.
	if _, err := run(model); !errors.Is(err, budget.ErrNoBudget) || hits.Load() != 0 {
		t.Fatalf("%v hits=%d", err, hits.Load())
	}
	// Modelo sem preço: nada é enviado.
	b.setGlobal(t, 100_000)
	noPrice := model
	noPrice.InputMicrosPerMTok, noPrice.OutputMicrosPerMTok = nil, nil
	if _, err := run(noPrice); !errors.Is(err, budget.ErrNoPrice) || hits.Load() != 0 {
		t.Fatalf("%v hits=%d", err, hits.Load())
	}
	// Chamada orçada: reserva (≥ teto de saída 1000 tokens = 15.000 micro-USD) e liquida com o consumo real.
	res, err := run(model)
	if err != nil || res.Text != "ok" || hits.Load() != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	rep, _ := b.store.Report(ctx, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	if want := budget.CostOf(budget.Price{InputPerMTok: in, OutputPerMTok: out}, 1000, 500); rep.API.SpentMicros != want || rep.API.BySource[budget.SourceProvider] != want || rep.API.ReservedMicros != 0 {
		t.Fatalf("gasto real = %d (esperado %d): %+v", rep.API.SpentMicros, want, rep.API)
	}
	// Falha do gateway devolve a reserva.
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(500) }))
	defer failing.Close()
	gwFail, _ := gateways.New(models.GatewayOpenRouter, security.NewSecret("tok"), failing.URL, true)
	before, _ := b.store.List(ctx)
	if _, err := gateways.Run(ctx, gwFail, gateways.RunConfig{Model: model, Prompt: "faça", Meter: meter}); err == nil {
		t.Fatal("o gateway falhou")
	}
	if after, _ := b.store.List(ctx); after[0].ReservedMicros != 0 || after[0].SpentMicros != before[0].SpentMicros {
		t.Fatalf("a falha não gasta nem segura saldo: %+v", after[0])
	}
	// Teto esgotado: a próxima chamada nem sai.
	b.setGlobal(t, 20_000) // gasto 10.500 já feito; a reserva (≥ 15.000) não cabe
	hitsBefore := hits.Load()
	if _, err := run(model); !isExceeded(err) || hits.Load() != hitsBefore {
		t.Fatalf("%v: nada deve ser enviado", err)
	}
	// Assinatura: sem custo nem teto; registra cota em separado.
	sub := connWith(t, b.c, "cli", "assinatura", "claude")
	subMeter := budget.Meter{Store: b.store, AgentID: ag, ConnectionID: sub.ID, Subscription: true}
	if _, err := gateways.Run(ctx, gw, gateways.RunConfig{Model: model, Prompt: "faça", Meter: subMeter}); err != nil {
		t.Fatalf("assinatura não depende do teto de API: %v", err)
	}
	rep2, _ := b.store.Report(ctx, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	if rep2.API.SpentMicros != rep.API.SpentMicros || len(rep2.Subscription) != 1 || rep2.Subscription[0].Calls != 1 {
		t.Fatalf("%+v", rep2)
	}
}

func TestBudgetAPISetsLimitsAndReportsUsage(t *testing.T) {
	b := newBudEnv(t)
	c := b.c
	ag := b.agent(t, "dev")
	var set budget.Budget
	c.expect(c.post("/api/budgets", map[string]any{"scope": "global", "period": "month", "limit_micros": 5_000_000}), 200).JSON(&set)
	c.expect(c.post("/api/budgets", map[string]any{"scope": "agent", "agent_id": ag, "period": "day", "limit_micros": 1_000_000}), 200)
	var again budget.Budget
	c.expect(c.post("/api/budgets", map[string]any{"scope": "global", "period": "month", "limit_micros": 6_000_000}), 200).JSON(&again)
	if again.ID != set.ID || again.Version != set.Version+1 || again.LimitMicros != 6_000_000 {
		t.Fatalf("upsert: %+v %+v", set, again)
	}
	var list struct{ Items []budget.Status }
	c.expect(c.get("/api/budgets"), 200).JSON(&list)
	if len(list.Items) != 2 || list.Items[0].RemainingMicros == 0 {
		t.Fatalf("%+v", list.Items)
	}
	var rep budget.Report
	c.expect(c.get("/api/usage"), 200).JSON(&rep)
	if rep.API.BySource == nil || rep.Subscription == nil {
		t.Fatalf("%+v", rep)
	}
	for name, body := range map[string]map[string]any{
		"escopo":            {"scope": "x", "period": "month", "limit_micros": 1},
		"global com agente": {"scope": "global", "agent_id": ag, "period": "month", "limit_micros": 1},
		"limite":            {"scope": "global", "period": "month", "limit_micros": -5},
		"campo novo":        {"scope": "global", "period": "month", "limit_micros": 1, "admin": true},
	} {
		if r := c.post("/api/budgets", body); r.StatusCode != 400 {
			t.Errorf("%s: %d %s", name, r.StatusCode, r.Body)
		}
	}
	for _, q := range []string{"?from=ontem", "?from=2026-11-10T00:00:00Z&to=2026-11-01T00:00:00Z", "?from=2020-01-01T00:00:00Z&to=2026-01-01T00:00:00Z"} {
		if r := c.get("/api/usage" + q); r.StatusCode != 400 {
			t.Errorf("%s: %d", q, r.StatusCode)
		}
	}
	anon := b.e.newClient()
	if r := anon.get("/api/budgets"); r.StatusCode != 401 {
		t.Fatalf("%d", r.StatusCode)
	}
	if r := c.do("POST", "/api/budgets", map[string]any{"scope": "global", "period": "day", "limit_micros": 1}, map[string]string{"X-CSRF-Token": ""}); r.StatusCode != 403 {
		t.Fatalf("%d", r.StatusCode)
	}
}
