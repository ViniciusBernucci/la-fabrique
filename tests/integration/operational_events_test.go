package integration

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/opslog"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/worker"
)

type opEvEnv struct {
	e *authEnv
	c *client
}

func newOpEvEnv(t *testing.T) *opEvEnv {
	t.Helper()
	e := newAuthEnv(t, nil)
	c := e.newClient()
	c.login(nil)
	return &opEvEnv{e: e, c: c}
}

func (o *opEvEnv) emit(t *testing.T, level, source, message string, payload map[string]any) {
	t.Helper()
	if err := opslog.Emit(context.Background(), o.e.db.Pool, level, source, message, payload); err != nil {
		t.Fatal(err)
	}
}

// T16-05, CA-04/CA-05/CA-06 — histórico com cursor, filtro por nível/origem e busca.
func TestOperationalEventsHistoryIsFilterableSearchableAndPaged(t *testing.T) {
	o := newOpEvEnv(t)
	o.emit(t, opslog.LevelInfo, opslog.SourceSentinela, "rotina concluída", nil)
	o.emit(t, opslog.LevelError, opslog.SourceRetencao, "falhou ao apagar", nil)
	o.emit(t, opslog.LevelInfo, opslog.SourceCLILogin, "login concluído: linha de busca", nil)

	type page struct {
		Items []struct {
			Seq     int64  `json:"seq"`
			Level   string `json:"level"`
			Source  string `json:"source"`
			Message string `json:"message"`
		} `json:"items"`
		Next int64 `json:"next_cursor"`
	}
	var all, lvl, src, q, paged page
	o.c.expect(o.c.get("/api/operations/events"), 200).JSON(&all)
	o.c.expect(o.c.get("/api/operations/events?level=error"), 200).JSON(&lvl)
	o.c.expect(o.c.get("/api/operations/events?source=clilogin"), 200).JSON(&src)
	o.c.expect(o.c.get("/api/operations/events?q=busca"), 200).JSON(&q)
	o.c.expect(o.c.get("/api/operations/events?after=1&limit=1"), 200).JSON(&paged)

	if len(all.Items) != 3 || all.Next != 3 {
		t.Fatalf("all: %+v", all)
	}
	if len(lvl.Items) != 1 || lvl.Items[0].Level != "error" {
		t.Fatalf("level: %+v", lvl)
	}
	if len(src.Items) != 1 || src.Items[0].Source != "clilogin" {
		t.Fatalf("source: %+v", src)
	}
	if len(q.Items) != 1 || !strings.Contains(q.Items[0].Message, "busca") {
		t.Fatalf("q: %+v", q)
	}
	if len(paged.Items) != 1 || paged.Items[0].Seq != 2 || paged.Next != 2 {
		t.Fatalf("paged: %+v", paged)
	}
	if r := o.c.get("/api/operations/events?level=critico"); r.StatusCode != 400 {
		t.Fatalf("nível inválido: %d", r.StatusCode)
	}
	if r := o.c.get("/api/operations/events?source=desconhecida"); r.StatusCode != 400 {
		t.Fatalf("origem inválida: %d", r.StatusCode)
	}
	if r := o.c.get("/api/operations/events?after=-1"); r.StatusCode != 400 {
		t.Fatalf("cursor inválido: %d", r.StatusCode)
	}
}

// T16-05, CA-07 — reconexão por cursor sem lacuna nem duplicata.
func TestOperationalEventsStreamReconnectsByTheCursor(t *testing.T) {
	fastStream(t)
	o := newOpEvEnv(t)
	o.emit(t, opslog.LevelInfo, opslog.SourceFabrica, "evento 1", nil)
	conn := o.c.sse(t, "/api/operations/stream", nil)
	if conn.resp.StatusCode != 200 {
		t.Fatalf("%d", conn.resp.StatusCode)
	}
	e, ok := conn.next(t, 3*time.Second)
	if !ok || e.ID != "1" {
		t.Fatalf("%+v ok=%v", e, ok)
	}
	conn.close()
	o.emit(t, opslog.LevelError, opslog.SourceFabrica, "evento 2", nil)
	re := o.c.sse(t, "/api/operations/stream", map[string]string{"Last-Event-ID": "1"})
	e2, ok := re.next(t, 3*time.Second)
	if !ok || e2.ID != "2" {
		t.Fatalf("%+v ok=%v", e2, ok)
	}
	if e3, ok := re.next(t, 300*time.Millisecond); ok {
		t.Fatalf("nada pode repetir: %+v", e3)
	}
}

// T16-05, CA-08 — segredo conhecido sai redigido no histórico e no stream.
func TestOperationalEventsRedactSecrets(t *testing.T) {
	o := newOpEvEnv(t)
	secret := connWith(t, o.c, "api", "gw-ops", "openrouter")
	_ = secret
	o.emit(t, opslog.LevelError, opslog.SourceFabrica, "token marcador-gw-ops no log", nil)
	type page struct {
		Items []struct{ Message string } `json:"items"`
	}
	var out page
	o.c.expect(o.c.get("/api/operations/events"), 200).JSON(&out)
	if len(out.Items) != 1 || strings.Contains(out.Items[0].Message, "marcador-gw-ops") {
		t.Fatalf("o segredo conhecido não pode sair: %+v", out)
	}
}

// T16-05, CA-12 — sem sessão válida, 401.
func TestOperationalEventsRequireSession(t *testing.T) {
	o := newOpEvEnv(t)
	anon := o.e.newClient()
	if r := anon.get("/api/operations/events"); r.StatusCode != 401 {
		t.Fatalf("%d", r.StatusCode)
	}
	if r := anon.get("/api/operations/stream"); r.StatusCode != 401 {
		t.Fatalf("%d", r.StatusCode)
	}
}

// T16-05, CA-01/CA-02 — job periódico do worker grava evento de erro (sempre) e de nota (quando houver).
func TestWorkerJobsRecordOperationalEventsOnFailureAndNote(t *testing.T) {
	o := newOpEvEnv(t)
	log := slog.New(slog.NewTextHandler(nopWriter{}, nil))
	w := worker.New(o.e.db.Pool, log, time.Hour)
	calls := 0
	w.AddJob(worker.Job{Name: "retencao", Every: time.Hour, Run: func(context.Context) (string, error) {
		calls++
		if calls == 1 {
			return "", errFalhaDeTeste
		}
		return "3 removido(s)", nil
	}})
	ctx := context.Background()
	if errMsg, err := w.RunJobOnce(ctx, "retencao"); err != nil || errMsg == "" {
		t.Fatalf("esperava falha na primeira chamada: %q %v", errMsg, err)
	}
	if errMsg, err := w.RunJobOnce(ctx, "retencao"); err != nil || errMsg != "" {
		t.Fatalf("%q %v", errMsg, err)
	}
	type page struct {
		Items []struct {
			Level, Source, Message string
		} `json:"items"`
	}
	var out page
	o.c.expect(o.c.get("/api/operations/events"), 200).JSON(&out)
	if len(out.Items) != 2 {
		t.Fatalf("%+v", out)
	}
	if out.Items[0].Level != "error" || out.Items[0].Source != "retencao" {
		t.Fatalf("falha: %+v", out.Items[0])
	}
	if out.Items[1].Level != "info" || out.Items[1].Source != "retencao" || out.Items[1].Message != "3 removido(s)" {
		t.Fatalf("nota: %+v", out.Items[1])
	}
}

// T16-05 — rotina da sentinela (RunOne) também grava evento operacional ao rodar.
func TestSentinelRoutineRecordsAnOperationalEvent(t *testing.T) {
	o := newOpEvEnv(t)
	sch := &sentinel.Scheduler{DB: o.e.db.Pool, Log: slog.New(slog.NewTextHandler(nopWriter{}, nil))}
	if err := sch.Register(sentinel.Routine{Name: sentinel.RoutineMaintenance, Every: time.Hour,
		Run: func(context.Context, string, string) (string, error) { return "", nil }}); err != nil {
		t.Fatal(err)
	}
	if _, err := sch.RunOne(context.Background(), sentinel.RoutineMaintenance, "", true); err != nil {
		t.Fatal(err)
	}
	type page struct {
		Items []struct{ Source string } `json:"items"`
	}
	var out page
	o.c.expect(o.c.get("/api/operations/events?source=sentinela"), 200).JSON(&out)
	if len(out.Items) != 1 {
		t.Fatalf("%+v", out)
	}
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

var errFalhaDeTeste = &testErr{"falhou"}

type testErr struct{ s string }

func (e *testErr) Error() string { return e.s }
