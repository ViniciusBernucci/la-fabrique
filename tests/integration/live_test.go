package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/httpapi"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
)

type sseEvent struct {
	ID, Name string
	Data     map[string]any
}

type sseConn struct {
	resp   *http.Response
	events chan sseEvent
	lines  chan string
	cancel context.CancelFunc
}

func (c *client) sse(t *testing.T, path string, headers map[string]string) *sseConn {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", c.e.srv.URL+path, nil)
	req.Header.Set("User-Agent", c.ua)
	req.Header.Set("Origin", c.e.srv.URL)
	req.Header.Set("Accept", "text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	conn := &sseConn{resp: resp, events: make(chan sseEvent, 64), lines: make(chan string, 256), cancel: cancel}
	go func() {
		defer close(conn.events)
		defer close(conn.lines)
		sc := bufio.NewScanner(resp.Body)
		var cur sseEvent
		var data []string
		for sc.Scan() {
			line := sc.Text()
			conn.lines <- line
			switch {
			case line == "":
				if cur.Name != "" || len(data) > 0 {
					if len(data) > 0 {
						_ = json.Unmarshal([]byte(strings.Join(data, "\n")), &cur.Data)
					}
					conn.events <- cur
				}
				cur, data = sseEvent{}, nil
			case strings.HasPrefix(line, "id: "):
				cur.ID = strings.TrimPrefix(line, "id: ")
			case strings.HasPrefix(line, "event: "):
				cur.Name = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				data = append(data, strings.TrimPrefix(line, "data: "))
			}
		}
	}()
	t.Cleanup(conn.close)
	return conn
}

func (c *sseConn) close() { c.cancel(); _ = c.resp.Body.Close() }

func (c *sseConn) next(t *testing.T, timeout time.Duration) (sseEvent, bool) {
	t.Helper()
	select {
	case e, ok := <-c.events:
		return e, ok
	case <-time.After(timeout):
		return sseEvent{}, false
	}
}

type liveEnv struct {
	e    *authEnv
	c    *client
	proj string
	exec string
}

func newLiveEnv(t *testing.T, slug string) *liveEnv {
	t.Helper()
	e, c, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, slug)
	ex := waitingExecution(t, e, p.ID, "LV-1")
	return &liveEnv{e: e, c: c, proj: p.ID, exec: ex.ID}
}

func (l *liveEnv) emit(t *testing.T, typ string, payload any) {
	t.Helper()
	if err := persistence.WithTx(context.Background(), l.e.db.Pool, func(tx pgx.Tx) error {
		_, err := persistence.AppendEvent(context.Background(), tx, l.proj, l.exec, typ, payload)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func fastStream(t *testing.T) {
	t.Helper()
	p, h, m := httpapi.StreamPoll, httpapi.StreamHeartbeat, httpapi.StreamMaxAge
	httpapi.StreamPoll, httpapi.StreamHeartbeat, httpapi.StreamMaxAge = 30*time.Millisecond, 120*time.Millisecond, 3*time.Second
	t.Cleanup(func() { httpapi.StreamPoll, httpapi.StreamHeartbeat, httpapi.StreamMaxAge = p, h, m })
}

// T16-01 — "Desconexão/reconexão recupera eventos pelo cursor."
func TestSSEReconnectionRecoversEventsByTheCursor(t *testing.T) {
	fastStream(t)
	l := newLiveEnv(t, "live-sse")
	base := "/api/projects/" + l.proj + "/executions/" + l.exec
	for i := 1; i <= 3; i++ {
		l.emit(t, "step", map[string]any{"n": i})
	}
	conn := l.c.sse(t, base+"/stream", nil)
	if conn.resp.StatusCode != 200 || !strings.HasPrefix(conn.resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("%d %s", conn.resp.StatusCode, conn.resp.Header.Get("Content-Type"))
	}
	var lastID string
	for i := 1; i <= 3; i++ {
		e, ok := conn.next(t, 3*time.Second)
		if !ok || e.ID != string(rune('0'+i)) || e.Data["type"] != "step" {
			t.Fatalf("evento %d: %+v ok=%v", i, e, ok)
		}
		lastID = e.ID
	}
	// Ao vivo: um evento novo chega sem reconectar.
	l.emit(t, "step", map[string]any{"n": 4})
	if e, ok := conn.next(t, 3*time.Second); !ok || e.ID != "4" {
		t.Fatalf("ao vivo: %+v ok=%v", e, ok)
	}
	lastID = "4"
	// Desconecta; acontecem eventos enquanto o painel está fora.
	conn.close()
	l.emit(t, "step", map[string]any{"n": 5})
	l.emit(t, "checkpoint", map[string]any{"n": 6})
	// Reconecta com o cursor (Last-Event-ID): só chegam os que faltavam, sem repetir nenhum.
	re := l.c.sse(t, base+"/stream", map[string]string{"Last-Event-ID": lastID})
	for _, want := range []string{"5", "6"} {
		e, ok := re.next(t, 3*time.Second)
		if !ok || e.ID != want {
			t.Fatalf("esperava %s: %+v ok=%v", want, e, ok)
		}
	}
	if e, ok := re.next(t, 300*time.Millisecond); ok {
		t.Fatalf("nada pode ser repetido: %+v", e)
	}
	// Também pelo parâmetro ?after= (o painel usa quando reabre a aba).
	via := l.c.sse(t, base+"/stream?after=5", nil)
	if e, ok := via.next(t, 3*time.Second); !ok || e.ID != "6" || e.Data["type"] != "checkpoint" {
		t.Fatalf("%+v ok=%v", e, ok)
	}
	// Cursor inválido e execução de outro projeto.
	if r := l.c.get(base + "/stream?after=abc"); r.StatusCode != 400 {
		t.Fatalf("%d", r.StatusCode)
	}
	other := mustProject(t, l.e.db.Pool, "live-outro")
	if r := l.c.get("/api/projects/" + other.ID + "/executions/" + l.exec + "/stream"); r.StatusCode != 404 {
		t.Fatalf("escopo por projeto: %d", r.StatusCode)
	}
}

func TestSSEHeartbeatsMaxAgeAndSecretRedaction(t *testing.T) {
	fastStream(t)
	httpapi.StreamMaxAge = 700 * time.Millisecond
	l := newLiveEnv(t, "live-hb")
	secret := connWith(t, l.c, "api", "gw-live", "openrouter")
	_ = secret
	l.emit(t, "log", map[string]any{"line": "token marcador-gw-live no log"})
	conn := l.c.sse(t, "/api/projects/"+l.proj+"/executions/"+l.exec+"/stream", nil)
	e, ok := conn.next(t, 3*time.Second)
	if !ok || strings.Contains(jsonOf(e.Data), "marcador-gw-live") || !strings.Contains(jsonOf(e.Data), "[REDACTED]") {
		t.Fatalf("o segredo conhecido não pode sair no fluxo: %+v ok=%v", e, ok)
	}
	sawPing, sawReconnect := false, false
	deadline := time.After(4 * time.Second)
loop:
	for {
		select {
		case line, ok := <-conn.lines:
			if !ok {
				break loop
			}
			sawPing = sawPing || line == ": ping"
			sawReconnect = sawReconnect || line == "event: reconnect"
		case <-deadline:
			t.Fatal("o fluxo deveria ter sido encerrado pelo limite de duração")
		}
	}
	if !sawPing || !sawReconnect {
		t.Fatalf("heartbeat=%v reconnect=%v", sawPing, sawReconnect)
	}
}

func jsonOf(v any) string { b, _ := json.Marshal(v); return string(b) }

func TestEventHistoryIsSearchableAndPaged(t *testing.T) {
	l := newLiveEnv(t, "live-hist")
	for i := 1; i <= 6; i++ {
		typ := "step"
		if i%3 == 0 {
			typ = "error"
		}
		l.emit(t, typ, map[string]any{"msg": "linha " + string(rune('a'+i))})
	}
	type page struct {
		Items []struct {
			Seq  int64  `json:"seq"`
			Type string `json:"type"`
		} `json:"items"`
		Next int64 `json:"next_cursor"`
	}
	base := "/api/projects/" + l.proj + "/executions/" + l.exec + "/events"
	var all, errs, paged, tail page
	l.c.expect(l.c.get(base), 200).JSON(&all)
	l.c.expect(l.c.get(base+"?q=error"), 200).JSON(&errs)
	l.c.expect(l.c.get(base+"?after=2&limit=2"), 200).JSON(&paged)
	l.c.expect(l.c.get(base+"?q=linha%20e"), 200).JSON(&tail) // procura também no conteúdo
	if len(all.Items) != 6 || all.Next != 6 || len(errs.Items) != 2 || errs.Items[0].Seq != 3 || errs.Next != 6 ||
		len(paged.Items) != 2 || paged.Items[0].Seq != 3 || paged.Next != 4 || len(tail.Items) != 1 || tail.Items[0].Seq != 4 {
		t.Fatalf("%+v %+v %+v %+v", all, errs, paged, tail)
	}
	if r := l.c.get(base + "?after=-1"); r.StatusCode != 400 {
		t.Fatalf("%d", r.StatusCode)
	}
	if r := l.c.get(base + "?q=" + strings.Repeat("x", 101)); r.StatusCode != 400 {
		t.Fatalf("%d", r.StatusCode)
	}
}

// T16-01 — "Estado bloqueado explica causa e painel exibe controles persistidos."
func TestOverviewExplainsBlockedStatesAndShowsThePersistedControls(t *testing.T) {
	s := newSchedEnv(t)
	ctx := context.Background()
	admin := s.admin(t)
	p := s.project(t, "ov", planOf("schema", "api"))
	p.item(t, "schema", 0)
	p.item(t, "api", 1)
	exec := p.execution(t)
	got, err := s.q.Claim(ctx, "w", exec, time.Minute)
	if err != nil || got == nil || got.TaskKey != "schema" {
		t.Fatalf("%+v %v", got, err)
	}
	// Controles persistidos: pausa global e suspensão do projeto (o item de A continua rodando; o painel mostra).
	if _, err := s.q.SetMode(ctx, queue.Scope{Global: true}, queue.ModeDraining, admin, "fechar antes do deploy"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.q.SetMode(ctx, queue.Scope{ProjectID: p.proj}, queue.ModeSuspended, admin, "incidente"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.q.Block(queue.WithFence(ctx, queue.Fence{Owner: "w", Generation: got.Generation}), got.ID, queue.WaitDecision, "dec-1", p.checkpoint(t, exec), true); err != nil {
		t.Fatal(err)
	}
	// O painel lê o mesmo estado pela API.
	type ov struct {
		Controls struct {
			Global  struct{ Mode, Reason string } `json:"global"`
			Project struct{ Mode, Reason string } `json:"project"`
		} `json:"controls"`
		Counts map[string]int `json:"counts"`
		Items  []struct {
			TaskKey string `json:"task_key"`
			State   string `json:"state"`
			Blocked string `json:"blocked"`
			Seconds int64  `json:"seconds"`
		} `json:"items"`
		Executions []struct {
			ID      string `json:"id"`
			LastSeq int64  `json:"last_seq"`
		} `json:"executions"`
	}
	api := s.apiClient(t)
	var out ov
	api.expect(api.get("/api/projects/"+p.proj+"/overview"), 200).JSON(&out)
	if out.Controls.Global.Mode != "draining" || out.Controls.Global.Reason != "fechar antes do deploy" || out.Controls.Project.Mode != "suspended" || out.Controls.Project.Reason != "incidente" {
		t.Fatalf("%+v", out.Controls)
	}
	by := map[string]string{}
	state := map[string]string{}
	for _, it := range out.Items {
		by[it.TaskKey], state[it.TaskKey] = it.Blocked, it.State
	}
	if state["schema"] != "waiting" || !strings.Contains(by["schema"], "decisão humana") {
		t.Fatalf("a causa do bloqueio aparece: %+v", out.Items)
	}
	if state["api"] != "queued" || !strings.Contains(by["api"], "pré-requisito") || !strings.Contains(by["api"], "suspenso") {
		t.Fatalf("a fila explica por que não inicia: %+v", out.Items)
	}
	if out.Counts["waiting"] != 1 || out.Counts["queued"] != 1 {
		t.Fatalf("%v", out.Counts)
	}
	// Retomar o projeto: o painel passa a mostrar o controle ativo.
	if _, _, err := s.q.ResumeScope(ctx, queue.Scope{ProjectID: p.proj}, admin, "ok"); err != nil {
		t.Fatal(err)
	}
	api.expect(api.get("/api/projects/"+p.proj+"/overview"), 200).JSON(&out)
	if out.Controls.Project.Mode != "active" {
		t.Fatalf("%+v", out.Controls)
	}
	if r := api.get("/api/projects/00000000-0000-0000-0000-000000000000/overview"); r.StatusCode != 404 {
		t.Fatalf("%d", r.StatusCode)
	}
}
