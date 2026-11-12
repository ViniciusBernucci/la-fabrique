package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
)

type noRow struct{}

func (noRow) Scan(...any) error { return errors.New("sem banco") }

type noDB struct{}

func (noDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("sem banco")
}
func (noDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("sem banco")
}
func (noDB) QueryRow(context.Context, string, ...any) pgx.Row { return noRow{} }
func (noDB) Ping(context.Context) error                       { return nil }
func (noDB) Begin(context.Context) (pgx.Tx, error)            { return nil, errors.New("sem banco") }

func newTestServer(t *testing.T, mutate func(*config.API)) *Server {
	t.Helper()
	cfg := config.API{
		Common: config.Common{Env: config.Development}, SessionCookieName: "lf_session",
		VaultKey: []byte("0123456789abcdef0123456789abcdef"), VaultKeyID: "k1",
	}
	if mutate != nil {
		mutate(&cfg)
	}
	s, err := NewServer(cfg, noDB{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewServerRejectsBadVaultKey(t *testing.T) {
	_, err := NewServer(config.API{VaultKey: []byte("curta")}, noDB{}, slog.Default())
	if err == nil {
		t.Fatal("chave curta deve impedir a API de montar")
	}
}

func TestResponsesAreRedactedBeforeLeavingTheServer(t *testing.T) {
	s := newTestServer(t, nil)
	s.redactor.Register("valor-secreto-xyz-123")
	h := s.withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			writeJSON(w, http.StatusTeapot, map[string]string{"detail": "falhou com valor-secreto-xyz-123 e " + "ghp" + "_abcdefghijklmnopqrstuvwxyz0123456789"})
		case "/err":
			writeError(w, r, 400, "x", "mensagem com valor-secreto-xyz-123", "")
		case "/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: passa direto\n\n")
		}
	}))
	for _, p := range []string{"/json", "/err"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		body := rec.Body.String()
		if strings.Contains(body, "valor-secreto-xyz-123") || strings.Contains(body, "ghp_") || !strings.Contains(body, "[REDACTED]") {
			t.Fatalf("%s: segredo na resposta: %s", p, body)
		}
		if rec.Header().Get("Content-Length") == "" {
			t.Fatalf("%s: Content-Length deve refletir o corpo redigido", p)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/json", nil))
	if rec.Code != http.StatusTeapot {
		t.Fatalf("status preservado: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/stream", nil))
	if rec.Body.String() != "data: passa direto\n\n" {
		t.Fatalf("fluxo não-JSON passa sem buffer: %q", rec.Body.String())
	}
}

func TestPanicBecomesInternalErrorWithoutLeaking(t *testing.T) {
	s := newTestServer(t, nil)
	h := s.withMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom com detalhe interno") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "boom") || !strings.Contains(rec.Body.String(), "internal_error") {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}

func TestSameOrigin(t *testing.T) {
	s := newTestServer(t, nil)
	sp := newTestServer(t, func(c *config.API) { c.TrustProxy = true })
	req := func(method, host string, hdr map[string]string) *http.Request {
		r := httptest.NewRequest(method, "http://"+host+"/api/x", nil)
		for k, v := range hdr {
			r.Header.Set(k, v)
		}
		return r
	}
	cases := []struct {
		name string
		s    *Server
		r    *http.Request
		want bool
	}{
		{"GET nunca checa", s, req("GET", "app.local", map[string]string{"Origin": "https://evil"}), true},
		{"origem igual", s, req("POST", "app.local:5180", map[string]string{"Origin": "http://app.local:5180"}), true},
		{"porta diferente", s, req("POST", "app.local:5180", map[string]string{"Origin": "http://app.local:9999"}), false},
		{"origem estrangeira", s, req("POST", "app.local", map[string]string{"Origin": "https://evil.example"}), false},
		{"null", s, req("POST", "app.local", map[string]string{"Origin": "null"}), false},
		{"sem Origin, same-origin", s, req("POST", "app.local", map[string]string{"Sec-Fetch-Site": "same-origin"}), true},
		{"sem Origin, cross-site", s, req("POST", "app.local", map[string]string{"Sec-Fetch-Site": "cross-site"}), false},
		{"sem Origin nem Fetch-Site (curl)", s, req("POST", "app.local", nil), true},
		{"atrás do proxy usa X-Forwarded-Host", sp, req("POST", "api:8480", map[string]string{"Origin": "https://painel.exemplo", "X-Forwarded-Host": "painel.exemplo"}), true},
		{"sem proxy confiável ignora X-Forwarded-Host", s, req("POST", "api:8480", map[string]string{"Origin": "https://painel.exemplo", "X-Forwarded-Host": "painel.exemplo"}), false},
	}
	for _, tc := range cases {
		if got := tc.s.sameOrigin(tc.r); got != tc.want {
			t.Errorf("%s: %v, esperado %v", tc.name, got, tc.want)
		}
	}
}

func TestClientIPOnlyTrustsProxyWhenConfigured(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.5:4444"
	r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
	if got := newTestServer(t, nil).clientIP(r); got != "10.0.0.5" {
		t.Fatalf("sem proxy confiável: %s", got)
	}
	if got := newTestServer(t, func(c *config.API) { c.TrustProxy = true }).clientIP(r); got != "203.0.113.7" {
		t.Fatalf("com proxy confiável: %s", got)
	}
	r.Header.Set("X-Forwarded-For", "isso-nao-e-ip")
	if got := newTestServer(t, func(c *config.API) { c.TrustProxy = true }).clientIP(r); got != "10.0.0.5" {
		t.Fatalf("XFF inválido deve ser ignorado: %s", got)
	}
}
