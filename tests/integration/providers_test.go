package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/availability"
	"github.com/ViniciusBernucci/le-fabrique/internal/budget"
)

type providerRow struct {
	ConnectionID string `json:"connection_id"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	State        string `json:"state"`
	Health       struct {
		State  string `json:"state"`
		Source string `json:"source"`
		Reason string `json:"reason"`
	} `json:"health"`
	Quota *struct {
		Used   int64  `json:"used"`
		Limit  int64  `json:"limit"`
		Unit   string `json:"unit"`
		Source string `json:"source"`
	} `json:"quota"`
}

// T16-03 — "Valor não observado aparece explicitamente."
func TestProvidersShowAvailabilityAndOnlyObservedQuota(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	ctx := context.Background()
	cli := connWith(t, c, "cli", "claude-assinatura", "claude")
	api := connWith(t, c, "api", "gateway", "openrouter")
	connWith(t, c, "github", "gh", "org") // não é conta de IA: não aparece

	var list struct{ Items []providerRow }
	c.expect(c.get("/api/providers"), 200).JSON(&list)
	if len(list.Items) != 2 {
		t.Fatalf("só contas de IA: %+v", list.Items)
	}
	for _, p := range list.Items {
		if p.Quota != nil || p.Health.State != "available" || p.State != "active" {
			t.Fatalf("sem observação a cota é nula: %+v", p)
		}
	}
	// Cota observada e falha oficial aparecem; nada é inventado para a outra conta.
	bs := &budget.Store{DB: e.db.Pool}
	if err := bs.ObserveQuota(ctx, budget.QuotaObservation{ConnectionID: cli.ID, Used: 40, Limit: 100, Unit: "mensagens", Source: "cli:/usage", ObservedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	tr := &availability.Tracker{DB: e.db.Pool}
	if _, err := tr.Record(ctx, api.ID, availability.Failure{Class: availability.Billing, Official: true, Detail: "saldo insuficiente"}); err != nil {
		t.Fatal(err)
	}
	c.expect(c.get("/api/providers"), 200).JSON(&list)
	by := map[string]providerRow{}
	for _, p := range list.Items {
		by[p.Name] = p
	}
	if q := by["claude-assinatura"].Quota; q == nil || q.Used != 40 || q.Limit != 100 || q.Unit != "mensagens" || q.Source != "cli:/usage" {
		t.Fatalf("%+v", by["claude-assinatura"])
	}
	g := by["gateway"]
	if g.Quota != nil || g.Health.State != "billing" || g.Health.Source != "official" || !strings.Contains(g.Health.Reason, "saldo") {
		t.Fatalf("%+v", g)
	}
	if body := string(c.get("/api/providers").Body); strings.Contains(body, "marcador-") {
		t.Fatal("sem segredos")
	}
}
