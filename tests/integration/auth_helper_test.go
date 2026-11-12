package integration

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/httpapi"
	"github.com/ViniciusBernucci/le-fabrique/internal/logging"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

const (
	testUsername = "admin"
	testPassword = "senha-de-teste-123"
)

// syncBuffer captura logs de forma segura entre goroutines.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}
func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// fakeClock avança o tempo dos códigos TOTP sem esperar 30 s reais.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type authEnv struct {
	t     *testing.T
	srv   *httptest.Server
	db    testDB
	clock *fakeClock
	logs  *syncBuffer
	api   *httpapi.Server
	cfg   config.API
}

// newAuthEnv sobe a API real sobre um PostgreSQL descartável, com o administrador criado.
func newAuthEnv(t *testing.T, mutate func(*config.API), opts ...httpapi.Option) *authEnv {
	t.Helper()
	db := newMigratedDB(t)
	cfg := apiConfig()
	if mutate != nil {
		mutate(&cfg)
	}
	e := &authEnv{t: t, db: db, logs: &syncBuffer{}, cfg: cfg, clock: &fakeClock{t: time.Now()}}
	log := logging.New(e.logs, "debug", "api")
	api, err := httpapi.NewServer(cfg, db.Pool, log, append([]httpapi.Option{httpapi.WithClock(e.clock.Now)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	if err := api.Prime(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { logging.SetRedactor(func(s string) string { return s }) })
	e.api = api
	e.srv = httptest.NewServer(api.Handler())
	t.Cleanup(e.srv.Close)
	if _, err := security.NewAuth(db.Pool, nil, security.AuthConfig{}, nil).CreateAdmin(context.Background(), testUsername, testPassword); err != nil {
		t.Fatal(err)
	}
	return e
}

type client struct {
	e    *authEnv
	http *http.Client
	csrf string
	ua   string
	// secret é o segredo TOTP da última autenticação, para novos logins do mesmo admin.
	secret []byte
}

func (e *authEnv) newClient() *client {
	jar, _ := cookiejar.New(nil)
	return &client{e: e, http: &http.Client{Jar: jar}, ua: "teste-integracao"}
}

type apiResp struct {
	*http.Response
	Body []byte
}

func (r apiResp) JSON(v any) {
	r.Response.Request.Context() // mantém o linter calmo; só decodifica o corpo
	if err := json.Unmarshal(r.Body, v); err != nil {
		panic("JSON inválido: " + string(r.Body))
	}
}

func (r apiResp) ErrCode() string {
	var e struct{ Error struct{ Code string } }
	_ = json.Unmarshal(r.Body, &e)
	return e.Error.Code
}

// do envia a requisição como um navegador na mesma origem. headers sobrescreve os padrões ("" remove).
func (c *client) do(method, path string, body any, headers map[string]string) apiResp {
	c.e.t.Helper()
	var rdr io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rdr = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, c.e.srv.URL+path, rdr)
	req.Header.Set("User-Agent", c.ua)
	req.Header.Set("Origin", c.e.srv.URL)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.csrf != "" && method != "GET" {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	for k, v := range headers {
		if v == "" {
			req.Header.Del(k)
		} else {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return apiResp{Response: resp, Body: raw}
}

func (c *client) get(path string) apiResp            { return c.do("GET", path, nil, nil) }
func (c *client) post(path string, body any) apiResp { return c.do("POST", path, body, nil) }
func (c *client) delete(path string) apiResp         { return c.do("DELETE", path, nil, nil) }
func (c *client) expect(r apiResp, status int) apiResp {
	c.e.t.Helper()
	if r.StatusCode != status {
		c.e.t.Fatalf("esperado %d, veio %d: %s", status, r.StatusCode, r.Body)
	}
	return r
}

func decodeSecret(t *testing.T, b32 string) []byte {
	t.Helper()
	b, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(b32)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// code gera um código TOTP válido e novo (avança o relógio para um passo inédito).
func (e *authEnv) code(secret []byte) string {
	e.clock.Advance(31 * time.Second)
	return security.TOTPCodeAt(secret, e.clock.Now())
}

type loginResult struct {
	Secret        []byte
	RecoveryCodes []string
	Session       security.Session
}

// login faz o fluxo completo (senha → 2º fator). Se o admin ainda não tem 2FA,
// cadastra e devolve segredo e códigos; senão usa `secret`.
func (c *client) login(secret []byte) loginResult {
	c.e.t.Helper()
	var res loginResult
	var stage struct{ Status string }
	c.expect(c.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200).JSON(&stage)
	switch stage.Status {
	case security.StageEnroll:
		var enr struct{ Secret string }
		c.expect(c.post("/api/auth/totp/setup", nil), 200).JSON(&enr)
		res.Secret = decodeSecret(c.e.t, enr.Secret)
		var out struct {
			security.Session
			RecoveryCodes []string `json:"recovery_codes"`
		}
		c.expect(c.post("/api/auth/totp/enable", map[string]string{"code": c.e.code(res.Secret)}), 201).JSON(&out)
		res.Session, res.RecoveryCodes = out.Session, out.RecoveryCodes
	case security.StageTOTP:
		res.Secret = secret
		c.expect(c.post("/api/auth/totp", map[string]string{"code": c.e.code(secret)}), 200).JSON(&res.Session)
	default:
		c.e.t.Fatalf("estágio inesperado: %q", stage.Status)
	}
	c.csrf = res.Session.CSRFToken
	c.secret = res.Secret
	return res
}

func mustURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func newReq(t *testing.T, rawURL, cookieName, cookieValue string) *http.Request {
	t.Helper()
	req, _ := http.NewRequest("GET", rawURL, nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: cookieValue})
	return req
}
