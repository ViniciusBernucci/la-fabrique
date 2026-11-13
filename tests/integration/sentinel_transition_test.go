package integration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

func prdEvidence() statusflow.Evidence {
	return statusflow.Evidence{PRDComplete: true, InitialContract: true}
}

// AJ03-02 — só a Sentinela efetiva; reentrega, evento atrasado e corrida não duplicam nem regridem.
func TestSentinelIsTheOnlyAuthorityForTransitions(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	if err := (&statusflow.Store{DB: db.Pool}).EnsureCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	p := mustProject(t, db.Pool, "sent-trans")
	c := &sentinel.Coordinator{DB: db.Pool}
	iss, err := c.Track(ctx, sentinel.NewIssue{ProjectID: p.ID, Key: "PRJ-1", Kind: "feature", Status: statusflow.PRDDoc})
	if err != nil || iss.Status != statusflow.PRDDoc || iss.Version != 1 {
		t.Fatalf("%+v %v", iss, err)
	}
	if again, _ := c.Track(ctx, sentinel.NewIssue{ProjectID: p.ID, Key: "PRJ-1", Kind: "feature", Status: statusflow.Concluido}); again.ID != iss.ID || again.Status != statusflow.PRDDoc {
		t.Fatalf("track não é idempotente: %+v", again)
	}
	t0 := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	req := func(actor, key string, ev statusflow.Evidence) sentinel.Request {
		return sentinel.Request{ProjectID: p.ID, IssueKey: "PRJ-1", To: statusflow.PRDDocConcluido, Actor: actor, Observer: "agente-x",
			ObservationKey: key, ObservedAt: t0, ExpectedVersion: 1, Evidence: ev}
	}
	// Agente escrevendo direto: recusado, estado intacto.
	if o, err := c.Transition(ctx, req(sentinel.ActorAgent, "obs-agent", prdEvidence())); !errors.Is(err, sentinel.ErrNotAuthority) || o.Decision != sentinel.DecisionRefused {
		t.Fatalf("%+v %v", o, err)
	}
	// Sentinela com PRD incompleto: recusado com motivo.
	o, err := c.Transition(ctx, req(sentinel.ActorSentinel, "obs-1", statusflow.Evidence{}))
	if err != nil || o.Decision != sentinel.DecisionRejected || len(o.Reasons) != 2 {
		t.Fatalf("%+v %v", o, err)
	}
	if got, _ := c.Get(ctx, p.ID, "PRJ-1"); got.Status != statusflow.PRDDoc || got.Version != 1 {
		t.Fatalf("%+v", got)
	}
	// Válida: aplica uma vez; outbox e evento saem juntos. Duas goroutines com a mesma observação.
	var wg sync.WaitGroup
	var mu sync.Mutex
	applied := 0
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o, err := c.Transition(ctx, req(sentinel.ActorSentinel, "obs-2", prdEvidence()))
			if err != nil {
				t.Error(err)
				return
			}
			if o.Decision == sentinel.DecisionApplied && !o.Duplicate {
				mu.Lock()
				applied++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	got, _ := c.Get(ctx, p.ID, "PRJ-1")
	if applied != 1 || got.Status != statusflow.PRDDocConcluido || got.Version != 2 {
		t.Fatalf("applied=%d %+v", applied, got)
	}
	var outbox, events int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE destination=$1`, sentinel.DestProjection).Scan(&outbox)
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM sentinel_events WHERE kind=$1`, sentinel.EventStateChanged).Scan(&events)
	if outbox != 1 || events != 1 {
		t.Fatalf("outbox=%d events=%d", outbox, events)
	}
	// Evento atrasado (versão antiga / observado antes) não cria avanço.
	late := sentinel.Request{ProjectID: p.ID, IssueKey: "PRJ-1", To: statusflow.Planejamento, Actor: sentinel.ActorSentinel, ObservationKey: "obs-late",
		ObservedAt: t0.Add(-time.Hour), ExpectedVersion: 2,
		Evidence: statusflow.Evidence{ProjectActive: true, WorkflowPublished: true, InputsValid: true, PlanningSlot: true}}
	if o, err := c.Transition(ctx, late); !errors.Is(err, sentinel.ErrStale) || o.Decision != sentinel.DecisionStale {
		t.Fatalf("%+v %v", o, err)
	}
	// Reentrega da recusa do agente não regrava.
	if o, _ := c.Transition(ctx, req(sentinel.ActorAgent, "obs-agent", prdEvidence())); !o.Duplicate {
		t.Fatalf("%+v", o)
	}
	hist, _ := c.History(ctx, iss.ID)
	var dec []string
	for _, h := range hist {
		dec = append(dec, h.Decision)
	}
	if len(dec) != 4 { // refused, rejected, applied, stale
		t.Fatalf("%v", dec)
	}
	// Alias legado converge ao canônico na Issue.
	rp, _ := c.Track(ctx, sentinel.NewIssue{ProjectID: p.ID, Key: "PRJ-9", Kind: "correction", Status: "reprovED_pr", ParentID: iss.ID})
	if rp.Status != statusflow.ReprovadoPR || rp.RawStatus != "reprovED_pr" {
		t.Fatalf("%+v", rp)
	}
}
