package integration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/worker"
)

type stepClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *stepClock) now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *stepClock) add(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

// AJ03-01 — rotinas com relógios e cursores independentes; evento durável recuperado após reinício.
func TestSentinelRoutinesAreIndependentAndEventsSurviveRestart(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	loc := time.FixedZone("BRT", -3*3600)
	clk := &stepClock{t: time.Date(2026, 11, 13, 6, 0, 0, 0, loc)} // antes das 07:00 BRT
	s := &sentinel.Scheduler{DB: db.Pool, Now: clk.now}
	var syncRuns, ghRuns, sweepRuns int
	ghFail, wantCursor := true, false
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(s.Register(sentinel.Routine{Name: sentinel.RoutineIssueSync, Daily: &sentinel.DailyAt{Hour: 7, Loc: loc},
		Run: func(context.Context, string, string) (string, error) { syncRuns++; return "p1", nil }}))
	must(s.Register(sentinel.Routine{Name: sentinel.RoutinePRs, Every: 10 * time.Minute, Backoff: time.Minute,
		Run: func(_ context.Context, _ string, cur string) (string, error) {
			ghRuns++
			if ghFail {
				return "pg2", errors.New("rate limit")
			}
			if wantCursor && cur != "pg2" {
				t.Errorf("cursor parcial perdido: %q", cur)
			}
			return "", nil
		}}))
	must(s.Register(sentinel.Routine{Name: sentinel.RoutineMaintenance, Every: 5 * time.Minute,
		Run: func(context.Context, string, string) (string, error) { sweepRuns++; return "", nil }}))

	res, err := s.Tick(ctx, nil) // 06:00: primeira execução recupera a janela matinal perdida
	must(err)
	if syncRuns != 1 || ghRuns != 1 || sweepRuns != 1 || len(res) != 3 {
		t.Fatalf("sync=%d gh=%d sweep=%d", syncRuns, ghRuns, sweepRuns)
	}
	// Falha em GitHub não suprime a manutenção nem o sync matinal.
	clk.add(61 * time.Minute) // 07:01
	_, err = s.Tick(ctx, nil)
	must(err)
	if syncRuns != 2 || sweepRuns != 2 { // 07:01 abre nova janela matinal
		t.Fatalf("sync=%d sweep=%d", syncRuns, sweepRuns)
	}
	if ghRuns != 2 { // venceu o backoff e falhou de novo
		t.Fatalf("gh=%d", ghRuns)
	}
	st, _ := s.State(ctx, sentinel.RoutinePRs, "")
	if st.FailCount != 2 || st.Cursor != "pg2" || st.NextRunAt == nil || !st.NextRunAt.After(clk.now()) {
		t.Fatalf("%+v", st)
	}
	// Mesma janela matinal: não roda de novo. Manual força a mesma rotina.
	_, _ = s.Tick(ctx, nil)
	if syncRuns != 2 {
		t.Fatalf("sync repetido: %d", syncRuns)
	}
	if r, err := s.RunOne(ctx, sentinel.RoutineIssueSync, "", true); err != nil || !r.Ran || syncRuns != 3 {
		t.Fatalf("manual %+v %v", r, err)
	}
	// GitHub volta: retoma do cursor e limpa a falha.
	ghFail, wantCursor = false, true
	clk.add(10 * time.Minute)
	_, err = s.Tick(ctx, nil)
	must(err)
	if st, _ = s.State(ctx, sentinel.RoutinePRs, ""); st.FailCount != 0 || st.LastError != "" || st.Cursor != "" {
		t.Fatalf("%+v", st)
	}
	wantCursor = false
	// Dia seguinte: nova janela.
	clk.add(24 * time.Hour)
	_, _ = s.Tick(ctx, nil)
	if syncRuns != 4 {
		t.Fatalf("sync=%d", syncRuns)
	}

	// Sobreposição: execução em andamento impede outra (reserva atômica).
	started, release := make(chan struct{}), make(chan struct{})
	s2 := &sentinel.Scheduler{DB: db.Pool, Now: clk.now}
	must(s2.Register(sentinel.Routine{Name: "slow", Every: time.Minute, Run: func(context.Context, string, string) (string, error) {
		close(started)
		<-release
		return "", nil
	}}))
	done := make(chan struct{})
	go func() { _, _ = s2.RunOne(ctx, "slow", "", false); close(done) }()
	<-started
	if r, _ := s2.RunOne(ctx, "slow", "", true); r.Ran {
		t.Fatal("execução sobreposta")
	}
	close(release)
	<-done

	// Eventos: dedupe, falha não bloqueia os demais, e reinício recupera os pendentes.
	for _, k := range []string{"a", "b", "a"} {
		_, err := sentinel.Publish(ctx, db.Pool, "", sentinel.EventCompleted, k, map[string]string{"k": k})
		must(err)
	}
	h, f, err := sentinel.Drain(ctx, db.Pool, 10, func(_ context.Context, e sentinel.Event) error {
		if e.Key == "a" {
			return errors.New("indisponível")
		}
		return nil
	})
	must(err)
	if h != 1 || f != 1 {
		t.Fatalf("handled=%d failed=%d", h, f)
	}
	// "reinício": novo processo drena o que ficou pendente.
	var seen []string
	h, f, err = sentinel.Drain(ctx, db.Pool, 10, func(_ context.Context, e sentinel.Event) error { seen = append(seen, e.Key); return nil })
	must(err)
	if h != 1 || f != 0 || len(seen) != 1 || seen[0] != "a" {
		t.Fatalf("%d %d %v", h, f, seen)
	}
	if h, _, _ = sentinel.Drain(ctx, db.Pool, 10, func(context.Context, sentinel.Event) error { return nil }); h != 0 {
		t.Fatal("evento tratado de novo")
	}
}

// AJ03-01 — o worker existente executa eventos e rotinas da Sentinela.
func TestTheWorkerRunsSentinelEventsAndRoutines(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "sent-worker")
	clk := &stepClock{t: time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)}
	s := &sentinel.Scheduler{DB: db.Pool, Now: clk.now}
	var scopes []string
	if err := s.Register(sentinel.Routine{Name: "r", Every: time.Hour, Run: func(_ context.Context, sc, _ string) (string, error) {
		scopes = append(scopes, sc)
		return "", nil
	}}); err != nil {
		t.Fatal(err)
	}
	var evs []string
	w := jobWorker(t)
	w.AddJob(worker.SentinelRoutinesJob(s, func(c context.Context) ([]string, error) { return sentinel.ProjectScopes(c, db.Pool) }, time.Hour))
	w.AddJob(worker.SentinelEventsJob(db.Pool, func(_ context.Context, e sentinel.Event) error { evs = append(evs, e.Kind); return nil }, time.Hour))
	_, _ = sentinel.Publish(ctx, db.Pool, p.ID, sentinel.EventSlotReleased, "s1", nil)
	if _, err := w.RunJobOnce(ctx, "sentinela-eventos"); err != nil {
		t.Fatal(err)
	}
	if _, err := w.RunJobOnce(ctx, "sentinela-rotinas"); err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 || evs[0] != sentinel.EventSlotReleased || len(scopes) != 1 || scopes[0] != p.ID {
		t.Fatalf("%v %v", evs, scopes)
	}
}
