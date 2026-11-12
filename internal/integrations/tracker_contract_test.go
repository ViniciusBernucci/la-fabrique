package integrations

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const secretSentinel = "SENTINELA-TRACKER-7c1e9a-nao-pode-vazar"

type call struct{ Method, Path, Query, Body, Auth string }

// recorder guarda o que a plataforma falsa recebeu.
type recorder struct {
	mu    sync.Mutex
	calls []call
}

func (r *recorder) add(req *http.Request) call {
	raw, _ := io.ReadAll(req.Body)
	c := call{req.Method, req.URL.Path, req.URL.RawQuery, string(raw), req.Header.Get("Authorization")}
	r.mu.Lock()
	r.calls = append(r.calls, c)
	r.mu.Unlock()
	return c
}

func (r *recorder) writes() []call {
	r.mu.Lock()
	defer r.mu.Unlock()
	var w []call
	for _, c := range r.calls {
		if c.Method != "GET" {
			w = append(w, c)
		}
	}
	return w
}

func (r *recorder) reset() {
	r.mu.Lock()
	r.calls = nil
	r.mu.Unlock()
}

func fixture(t testing.TB, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// fake é uma plataforma falsa com modos de falha comuns.
type fake struct {
	*httptest.Server
	rec  *recorder
	mode atomic.Value // "", "unauth", "forbidden", "ratelimit", "error"
}

func newFake(t *testing.T, route func(w http.ResponseWriter, c call)) *fake {
	f := &fake{rec: &recorder{}}
	f.mode.Store("")
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := f.rec.add(r)
		// Os corpos de erro ecoam a credencial: nada disso pode chegar ao chamador.
		echo := `{"errorMessages":["negado para ` + c.Auth + `"],"err":"` + c.Auth + `"}`
		switch f.mode.Load().(string) {
		case "unauth":
			w.WriteHeader(401)
			io.WriteString(w, echo)
		case "forbidden":
			w.WriteHeader(403)
			io.WriteString(w, echo)
		case "ratelimit":
			w.Header().Set("Retry-After", "30")
			w.Header().Set("X-RateLimit-Reset", itoa(time.Now().Add(30*time.Second).Unix()))
			w.WriteHeader(429)
			io.WriteString(w, echo)
		case "error":
			w.WriteHeader(503)
			io.WriteString(w, echo)
		default:
			route(w, c)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// harness descreve o que a suíte espera de cada plataforma.
type harness struct {
	name      string
	tracker   Tracker
	fake      *fake
	container string
	story     string // item que não é subtarefa
	subtask   string // subtarefa de story
	dependent string // item que depende de story
	missing   string // ID inexistente
	okTrans   string // transição válida para story
	badTrans  string
	// Esperado do payload das escritas.
	wantSubtaskBody []string
	wantDepBody     []string
	wantLinkBody    []string
	linkPath        string
	depPath         string
	// Jira monta JQL, então o rótulo é validado; ClickUp só o envia codificado na URL.
	validatesLabel bool
}

func runContract(t *testing.T, h harness) {
	ctx := t.Context()
	secretIn := func(err error) {
		t.Helper()
		if err != nil && (strings.Contains(err.Error(), secretSentinel) || strings.Contains(err.Error(), "negado para") || strings.Contains(err.Error(), h.fake.URL)) {
			t.Fatalf("mensagem de erro vaza segredo, corpo ou URL: %q", err.Error())
		}
	}
	kindOf := func(err error) ErrKind {
		t.Helper()
		var e *Error
		if !errors.As(err, &e) {
			t.Fatalf("esperado *Error, veio %T %v", err, err)
		}
		return e.Kind
	}

	t.Run("identidade", func(t *testing.T) {
		id, err := h.tracker.Identity(ctx)
		if err != nil || id.Account == "" || id.Name == "" {
			t.Fatalf("identidade: %+v %v", id, err)
		}
	})

	t.Run("lista pagina e distingue item de subtarefa", func(t *testing.T) {
		p1, err := h.tracker.ListItems(ctx, h.container, ListOptions{Label: "lf-ready", Limit: 2})
		if err != nil || len(p1.Items) != 2 || p1.NextCursor == "" {
			t.Fatalf("página 1: %+v %v", p1, err)
		}
		p2, err := h.tracker.ListItems(ctx, h.container, ListOptions{Label: "lf-ready", Limit: 2, Cursor: p1.NextCursor})
		if err != nil || len(p2.Items) != 1 || p2.NextCursor != "" {
			t.Fatalf("página 2 deve ser a última: %+v %v", p2, err)
		}
		story, sub, dep := p1.Items[0], p1.Items[1], p2.Items[0]
		if story.ID != h.story || story.Subtask || story.ParentID != "" || story.Title != "Cadastro de clientes" ||
			story.Priority != "High" && story.Priority != "high" || story.Category != CategoryTodo ||
			len(story.Labels) != 2 || story.Version == "" || story.URL == "" || story.Provider != h.name {
			t.Fatalf("item normalizado: %+v", story)
		}
		if !strings.Contains(story.Description, "cadastrar clientes") {
			t.Fatalf("descrição: %q", story.Description)
		}
		if sub.ID != h.subtask || !sub.Subtask || sub.ParentID != h.story || sub.Category != CategoryInProgress {
			t.Fatalf("subtarefa deve apontar o pai: %+v", sub)
		}
		if dep.ID != h.dependent || dep.Category != CategoryDone || dep.Priority != "" ||
			len(dep.Dependencies) != 1 || dep.Dependencies[0] != h.story {
			t.Fatalf("dependência deve apontar só o que bloqueia este item: %+v", dep)
		}
	})

	t.Run("leitura de item, campos, estados, prioridades", func(t *testing.T) {
		it, err := h.tracker.GetItem(ctx, h.story)
		if err != nil || it.ID != h.story || it.Subtask {
			t.Fatalf("GetItem: %+v %v", it, err)
		}
		if st, err := h.tracker.Statuses(ctx, h.container); err != nil || len(st) < 3 {
			t.Fatalf("estados: %+v %v", st, err)
		}
		if f, err := h.tracker.Fields(ctx, h.container); err != nil || len(f) == 0 {
			t.Fatalf("campos: %+v %v", f, err)
		}
		if p, err := h.tracker.Priorities(ctx); err != nil || len(p) < 3 {
			t.Fatalf("prioridades: %+v %v", p, err)
		}
		if _, err := h.tracker.ItemTypes(ctx, h.container); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("só transições válidas são enviadas", func(t *testing.T) {
		ts, err := h.tracker.Transitions(ctx, h.story)
		if err != nil || len(ts) == 0 {
			t.Fatalf("transições: %+v %v", ts, err)
		}
		found := false
		for _, tr := range ts {
			found = found || tr.ID == h.okTrans
		}
		if !found {
			t.Fatalf("transição válida ausente: %+v", ts)
		}
		h.fake.rec.reset()
		err = h.tracker.Transition(ctx, h.story, h.badTrans)
		if kindOf(err) != ErrInvalid || len(h.fake.rec.writes()) != 0 {
			t.Fatalf("transição inválida deve ser recusada sem escrita: %v %+v", err, h.fake.rec.writes())
		}
		if err := h.tracker.Transition(ctx, h.story, h.okTrans); err != nil {
			t.Fatal(err)
		}
		if w := h.fake.rec.writes(); len(w) != 1 || !strings.Contains(w[0].Body, h.okTrans) {
			t.Fatalf("escrita de transição: %+v", w)
		}
	})

	t.Run("cria subtarefa, dependência e link", func(t *testing.T) {
		h.fake.rec.reset()
		sub, err := h.tracker.CreateSubtask(ctx, h.story, SubtaskInput{Title: "Nova subtarefa", Description: "Detalhe"})
		if err != nil || !sub.Subtask || sub.ParentID != h.story {
			t.Fatalf("CreateSubtask: %+v %v", sub, err)
		}
		w := h.fake.rec.writes()
		if len(w) != 1 {
			t.Fatalf("esperada 1 escrita: %+v", w)
		}
		for _, want := range h.wantSubtaskBody {
			if !strings.Contains(w[0].Body, want) {
				t.Errorf("corpo da subtarefa sem %q: %s", want, w[0].Body)
			}
		}
		// Subtarefa não tem subtarefa; título vazio é recusado sem escrita.
		h.fake.rec.reset()
		if _, err := h.tracker.CreateSubtask(ctx, h.subtask, SubtaskInput{Title: "x"}); kindOf(err) != ErrInvalid {
			t.Fatalf("subtarefa de subtarefa: %v", err)
		}
		if _, err := h.tracker.CreateSubtask(ctx, h.story, SubtaskInput{Title: "  "}); kindOf(err) != ErrInvalid {
			t.Fatalf("título vazio: %v", err)
		}
		if len(h.fake.rec.writes()) != 0 {
			t.Fatalf("entrada inválida não pode escrever: %+v", h.fake.rec.writes())
		}

		h.fake.rec.reset()
		if err := h.tracker.AddDependency(ctx, h.dependent, h.story); err != nil {
			t.Fatal(err)
		}
		w = h.fake.rec.writes()
		if len(w) != 1 || w[0].Path != h.depPath {
			t.Fatalf("dependência: %+v", w)
		}
		for _, want := range h.wantDepBody {
			if !strings.Contains(w[0].Body, want) {
				t.Errorf("corpo da dependência sem %q: %s", want, w[0].Body)
			}
		}

		h.fake.rec.reset()
		if err := h.tracker.AddLink(ctx, h.dependent, "http://inseguro.example/pr/1", "PR"); kindOf(err) != ErrInvalid {
			t.Fatalf("link http deve ser recusado: %v", err)
		}
		if err := h.tracker.AddLink(ctx, h.dependent, "https://github.com/o/r/pull/7", "Draft PR #7"); err != nil {
			t.Fatal(err)
		}
		w = h.fake.rec.writes()
		if len(w) != 1 || w[0].Path != h.linkPath {
			t.Fatalf("link: %+v", w)
		}
		for _, want := range h.wantLinkBody {
			if !strings.Contains(w[0].Body, want) {
				t.Errorf("corpo do link sem %q: %s", want, w[0].Body)
			}
		}
	})

	t.Run("item inexistente", func(t *testing.T) {
		if _, err := h.tracker.GetItem(ctx, h.missing); kindOf(err) != ErrNotFound {
			t.Fatalf("esperado not_found: %v", err)
		}
	})

	t.Run("entradas malformadas nunca chegam à plataforma", func(t *testing.T) {
		h.fake.rec.reset()
		for _, bad := range []string{"", "../etc", "a b", "x?y=1", "a/b"} {
			if _, err := h.tracker.GetItem(ctx, bad); kindOf(err) != ErrInvalid {
				t.Errorf("GetItem(%q): %v", bad, err)
			}
			if _, err := h.tracker.ListItems(ctx, bad, ListOptions{}); kindOf(err) != ErrInvalid {
				t.Errorf("ListItems(%q): %v", bad, err)
			}
		}
		if h.validatesLabel {
			if _, err := h.tracker.ListItems(ctx, h.container, ListOptions{Label: `x" OR project = Z`}); kindOf(err) != ErrInvalid {
				t.Errorf("rótulo com aspas: %v", err)
			}
		}
		if n := len(h.fake.rec.calls); n != 0 {
			t.Fatalf("nenhuma chamada era esperada: %d", n)
		}
	})

	for mode, want := range map[string]ErrKind{"unauth": ErrAuth, "forbidden": ErrPermission, "ratelimit": ErrRateLimited, "error": ErrUnavailable} {
		t.Run("falha "+mode, func(t *testing.T) {
			h.fake.mode.Store(mode)
			defer h.fake.mode.Store("")
			_, err1 := h.tracker.ListItems(ctx, h.container, ListOptions{})
			_, err2 := h.tracker.GetItem(ctx, h.story)
			err3 := h.tracker.AddLink(ctx, h.story, "https://github.com/o/r/pull/1", "PR")
			for _, err := range []error{err1, err2, err3} {
				if kindOf(err) != want {
					t.Fatalf("esperado %s, veio %v", want, err)
				}
				secretIn(err)
			}
			var e *Error
			errors.As(err1, &e)
			if mode == "ratelimit" {
				if e.RetryAfter < 20*time.Second || e.RetryAfter > 31*time.Second || !e.Retryable() {
					t.Fatalf("RetryAfter/Retryable: %+v", e)
				}
			} else if mode == "error" && !e.Retryable() || (mode == "unauth" || mode == "forbidden") && e.Retryable() {
				t.Fatalf("Retryable incorreto para %s", mode)
			}
		})
	}

	t.Run("autenticação usa a credencial e sem redirecionamento", func(t *testing.T) {
		h.fake.rec.reset()
		if _, err := h.tracker.Identity(ctx); err != nil {
			t.Fatal(err)
		}
		if h.fake.rec.calls[0].Auth == "" {
			t.Fatal("a credencial deveria ser enviada ao host configurado")
		}
	})
}
