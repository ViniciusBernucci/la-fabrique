package integration

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/httpapi"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

func apiConfig() config.API {
	return config.API{
		Common:            config.Common{Env: config.Development, LogLevel: "error"},
		Addr:              "127.0.0.1:0",
		SessionTTL:        time.Hour,
		SessionCookieName: "lf_session",
		VaultKey:          []byte("0123456789abcdef0123456789abcdef"),
		VaultKeyID:        "k1",
	}
}

func newAPI(t *testing.T) (*httptest.Server, testDB) {
	t.Helper()
	db := newMigratedDB(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h, err := httpapi.New(apiConfig(), db.Pool, log)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, db
}

func do(t *testing.T, srv *httptest.Server, method, path, cookie string) (*http.Response, []byte) {
	t.Helper()
	req, _ := http.NewRequest(method, srv.URL+path, nil)
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: "lf_session", Value: cookie})
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp, body
}

func decodeError(t *testing.T, body []byte) httpapi.ErrorDetail {
	t.Helper()
	var e httpapi.ErrorBody
	if err := json.Unmarshal(body, &e); err != nil || e.Error.Code == "" {
		t.Fatalf("corpo de erro inválido: %s", body)
	}
	return e.Error
}

// loginCookie cria admin e sessão diretamente no banco (o login real é do T04-01).
func loginCookie(t *testing.T, db testDB) (token, sessionID string) {
	t.Helper()
	var adminID string
	if err := db.Pool.QueryRow(ctx, `INSERT INTO admins (username, password_hash) VALUES ('admin', 'x') RETURNING id`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	token, sess, err := security.CreateSession(ctx, db.Pool, adminID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return token, sess.ID
}

func TestHealthEndpointsAreOpen(t *testing.T) {
	srv, _ := newAPI(t)
	for _, p := range []string{"/api/healthz", "/api/readyz"} {
		if resp, body := do(t, srv, "GET", p, ""); resp.StatusCode != 200 {
			t.Fatalf("%s: %d %s", p, resp.StatusCode, body)
		}
	}
}

func TestServerRefusesRequestsWithoutSession(t *testing.T) {
	srv, _ := newAPI(t)
	for _, p := range []string{"/api/session", "/api/projects", "/api/projects/00000000-0000-4000-8000-000000000000"} {
		resp, body := do(t, srv, "GET", p, "")
		if resp.StatusCode != 401 {
			t.Fatalf("%s sem sessão deveria ser 401: %d", p, resp.StatusCode)
		}
		e := decodeError(t, body)
		if e.Code != "unauthorized" || e.CorrelationID == "" || e.CorrelationID != resp.Header.Get("X-Request-ID") {
			t.Fatalf("erro padronizado esperado com correlação: %+v", e)
		}
	}
	if resp, _ := do(t, srv, "GET", "/api/projects", "token-forjado"); resp.StatusCode != 401 {
		t.Fatalf("token forjado: %d", resp.StatusCode)
	}
}

func TestAuthenticatedFlow(t *testing.T) {
	srv, db := newAPI(t)
	token, sessionID := loginCookie(t, db)
	p := mustProject(t, db.Pool, "meu-projeto")

	resp, body := do(t, srv, "GET", "/api/session", token)
	if resp.StatusCode != 200 || !strings.Contains(string(body), `"username":"admin"`) {
		t.Fatalf("session: %d %s", resp.StatusCode, body)
	}
	resp, body = do(t, srv, "GET", "/api/projects", token)
	var page projects.Page
	if resp.StatusCode != 200 || json.Unmarshal(body, &page) != nil || len(page.Items) != 1 || page.Items[0].ID != p.ID {
		t.Fatalf("lista: %d %s", resp.StatusCode, body)
	}
	resp, body = do(t, srv, "GET", "/api/projects/"+p.ID, token)
	var got projects.Project
	if resp.StatusCode != 200 || json.Unmarshal(body, &got) != nil || got.Slug != "meu-projeto" {
		t.Fatalf("detalhe: %d %s", resp.StatusCode, body)
	}

	// Erros: inexistente, id malformado, parâmetros inválidos, rota desconhecida.
	for path, want := range map[string]int{
		"/api/projects/00000000-0000-4000-8000-000000000000": 404,
		"/api/projects/nao-e-uuid":                           404,
		"/api/projects?limit=0":                              400,
		"/api/projects?limit=abc":                            400,
		"/api/projects?limit=999":                            400,
		"/api/projects?state=desconhecido":                   400,
		"/api/projects?cursor=lixo":                          400,
		"/api/rota-inexistente":                              404,
	} {
		resp, body := do(t, srv, "GET", path, token)
		if resp.StatusCode != want {
			t.Errorf("%s: esperado %d, veio %d (%s)", path, want, resp.StatusCode, body)
			continue
		}
		if e := decodeError(t, body); e.CorrelationID == "" || e.Message == "" {
			t.Errorf("%s: erro sem mensagem/correlação: %+v", path, e)
		}
	}

	// Revogação tem efeito imediato no servidor.
	if err := security.Revoke(ctx, db.Pool, sessionID); err != nil {
		t.Fatal(err)
	}
	if resp, _ := do(t, srv, "GET", "/api/projects", token); resp.StatusCode != 401 {
		t.Fatalf("sessão revogada deveria perder acesso: %d", resp.StatusCode)
	}
}

func TestCorrelationIDIsPropagatedAndSanitized(t *testing.T) {
	srv, _ := newAPI(t)
	req, _ := http.NewRequest("GET", srv.URL+"/api/projects", nil)
	req.Header.Set("X-Request-ID", "meu-id-12345")
	resp, _ := srv.Client().Do(req)
	if resp.Header.Get("X-Request-ID") != "meu-id-12345" {
		t.Fatal("ID válido deveria ser reaproveitado")
	}
	req.Header.Set("X-Request-ID", "<script>alert(1)</script>")
	resp, _ = srv.Client().Do(req)
	if got := resp.Header.Get("X-Request-ID"); strings.ContainsAny(got, "<>") || len(got) < 8 {
		t.Fatalf("ID inválido deveria ser substituído: %q", got)
	}
}

// O acesso de dev provisório (ADR-0004) foi removido no T04-01: nenhuma rota
// emite sessão sem senha + segundo fator.
func TestDevSessionEndpointIsGone(t *testing.T) {
	srv, _ := newAPI(t)
	if resp, _ := do(t, srv, "POST", "/api/dev/session", ""); resp.StatusCode != 404 {
		t.Fatalf("o endpoint de dev não pode existir: %d", resp.StatusCode)
	}
}

func TestSecurityHeaders(t *testing.T) {
	srv, _ := newAPI(t)
	resp, _ := do(t, srv, "GET", "/api/healthz", "")
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" || resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("cabeçalhos de segurança ausentes: %v", resp.Header)
	}
}
