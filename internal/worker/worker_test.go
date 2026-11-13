package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type fakeDB struct{ fail atomic.Bool }

func (f *fakeDB) Ping(context.Context) error {
	if f.fail.Load() {
		return errors.New("down")
	}
	return nil
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestRunBeatsAndStopsOnCancel(t *testing.T) {
	w := New(&fakeDB{}, quiet(), 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { w.Run(ctx); close(done) }()
	deadline := time.After(2 * time.Second)
	for w.Beats() < 3 {
		select {
		case <-deadline:
			t.Fatalf("heartbeats insuficientes: %d", w.Beats())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run deveria encerrar após cancelamento")
	}
}

func TestReadinessFollowsDatabase(t *testing.T) {
	db := &fakeDB{}
	w := New(db, quiet(), time.Hour)
	h := w.HealthHandler()
	get := func(path string) int {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec.Code
	}
	w.beat(context.Background())
	if get("/healthz") != 200 || get("/readyz") != 200 {
		t.Fatal("deveria estar saudável")
	}
	db.fail.Store(true)
	w.beat(context.Background())
	if get("/healthz") != 200 {
		t.Fatal("liveness não depende do banco")
	}
	if get("/readyz") != 503 {
		t.Fatal("readiness deveria falhar com banco indisponível")
	}
	db.fail.Store(false)
	w.beat(context.Background())
	if get("/readyz") != 200 {
		t.Fatal("readiness deveria se recuperar")
	}
}

func TestPeriodicJobsRunAndTheirStatusIsVisible(t *testing.T) {
	w := New(&fakeDB{}, quiet(), time.Hour)
	var runs atomic.Int64
	w.AddJob(Job{Name: "contar", Every: 10 * time.Millisecond, Run: func(context.Context) (string, error) {
		if runs.Add(1) == 2 {
			return "", errors.New("falhou uma vez")
		}
		return "feito", nil
	}})
	w.AddJob(Job{Name: "", Every: time.Second, Run: nil}) // inválida: ignorada
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)
	deadline := time.Now().Add(3 * time.Second)
	for runs.Load() < 4 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if runs.Load() < 4 {
		t.Fatalf("a tarefa não rodou periodicamente: %d", runs.Load())
	}
	st := w.Jobs()
	if len(st) != 1 || st[0].Name != "contar" || st[0].Runs < 4 || st[0].LastRun.IsZero() {
		t.Fatalf("%+v", st)
	}
	rec := httptest.NewRecorder()
	w.HealthHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/jobs", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"contar"`) {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if _, err := w.RunJobOnce(ctx, "inexistente"); err == nil {
		t.Fatal("tarefa desconhecida")
	}
}
