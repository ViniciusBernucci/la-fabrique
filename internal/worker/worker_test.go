package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
