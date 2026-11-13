package integration

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// T04-02 — "Sessão revogada perde acesso."
func TestRevokedSessionsLoseAccess(t *testing.T) {
	e := newAuthEnv(t, nil)
	a := e.newClient()
	res := a.login(nil)
	b := e.newClient()
	b.login(res.Secret)
	other := e.newClient()
	other.login(res.Secret)

	var list struct{ Items []security.SessionInfo }
	a.expect(a.get("/api/sessions"), 200).JSON(&list)
	if len(list.Items) != 3 {
		t.Fatalf("esperava 3 sessões ativas, veio %d", len(list.Items))
	}
	current := 0
	for _, s := range list.Items {
		if s.Current {
			current++
		}
	}
	if current != 1 {
		t.Fatalf("exatamente uma sessão é a atual: %d", current)
	}

	// Revogar uma sessão específica tira o acesso dela imediatamente.
	var bs security.Session
	b.expect(b.get("/api/session"), 200).JSON(&bs)
	a.expect(a.delete("/api/sessions/"+bs.ID), 200)
	if r := b.get("/api/projects"); r.StatusCode != 401 {
		t.Fatalf("sessão revogada deveria perder acesso: %d", r.StatusCode)
	}
	a.expect(a.get("/api/projects"), 200)
	if r := a.delete("/api/sessions/" + bs.ID); r.StatusCode != 404 {
		t.Fatalf("revogar de novo: %d", r.StatusCode)
	}
	if r := a.delete("/api/sessions/nao-e-uuid"); r.StatusCode != 404 {
		t.Fatalf("id inválido: %d", r.StatusCode)
	}

	// Revogar todas as outras preserva a atual.
	var out struct{ Revoked int64 }
	a.expect(a.post("/api/sessions/revoke-others", nil), 200).JSON(&out)
	if out.Revoked != 1 {
		t.Fatalf("revogadas: %d", out.Revoked)
	}
	if r := other.get("/api/session"); r.StatusCode != 401 {
		t.Fatalf("outra sessão deveria ter caído: %d", r.StatusCode)
	}
	a.expect(a.get("/api/session"), 200)

	// Logout encerra a própria sessão no servidor (o cookie antigo não volta a valer).
	var cookie string
	for _, ck := range a.http.Jar.Cookies(mustURL(e.srv.URL)) {
		if ck.Name == "lf_session" {
			cookie = ck.Value
		}
	}
	a.expect(a.post("/api/auth/logout", nil), 200)
	if r, body := do(t, e.srv, "GET", "/api/session", cookie); r.StatusCode != 401 {
		t.Fatalf("cookie após logout: %d %s", r.StatusCode, body)
	}
}

// T04-02 — "Requisição forjada ... é recusada nas ações sensíveis."
func TestCSRFAndOriginAreEnforced(t *testing.T) {
	e := newAuthEnv(t, nil)
	c := e.newClient()
	c.login(nil)
	body := map[string]string{"kind": "api", "name": "x", "secret": "valor-qualquer-123"}

	cases := []struct {
		name    string
		headers map[string]string
		want    int
	}{
		{"sem token CSRF", map[string]string{"X-CSRF-Token": ""}, 403},
		{"token CSRF errado", map[string]string{"X-CSRF-Token": "forjado"}, 403},
		{"origem estrangeira", map[string]string{"Origin": "https://evil.example"}, 403},
		{"origem null (iframe sandbox)", map[string]string{"Origin": "null"}, 403},
		{"Sec-Fetch-Site cross-site", map[string]string{"Origin": "", "Sec-Fetch-Site": "cross-site"}, 403},
	}
	for _, tc := range cases {
		r := c.do("POST", "/api/connections", body, tc.headers)
		if r.StatusCode != tc.want || r.ErrCode() != "csrf_rejected" {
			t.Errorf("%s: %d %s", tc.name, r.StatusCode, r.Body)
		}
	}
	var n int
	_ = e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM connections`).Scan(&n)
	if n != 0 {
		t.Fatalf("requisição forjada criou %d conexão(ões)", n)
	}
	// Mesma origem com o token certo passa; o cookie sozinho (outra aba) não basta.
	c.expect(c.post("/api/connections", body), 201)
	// Logout também exige o token: um site externo não consegue deslogar a vítima.
	if r := c.do("POST", "/api/auth/logout", nil, map[string]string{"X-CSRF-Token": ""}); r.StatusCode != 403 {
		t.Fatalf("logout forjado: %d", r.StatusCode)
	}
	// Login (sem sessão) também recusa origem estrangeira.
	fresh := e.newClient()
	r := fresh.do("POST", "/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}, map[string]string{"Origin": "https://evil.example"})
	if r.StatusCode != 403 {
		t.Fatalf("login cross-origin: %d", r.StatusCode)
	}
}

// T04-02 — "sessão sem reautenticação é recusada nas ações sensíveis."
func TestSensitiveActionsRequireRecentReauthentication(t *testing.T) {
	e := newAuthEnv(t, nil)
	c := e.newClient()
	res := c.login(nil)
	body := map[string]string{"kind": "api", "name": "primeira", "secret": "valor-qualquer-123"}

	// Sessão recém-criada vale como reautenticação.
	if res.Session.ReauthUntil == nil {
		t.Fatal("sessão nova deveria trazer reauth_until")
	}
	c.expect(c.post("/api/connections", body), 201)

	// Passada a janela, a ação é recusada com código próprio.
	if _, err := e.db.Pool.Exec(ctx, `UPDATE admin_sessions SET reauth_at = now() - interval '10 minutes'`); err != nil {
		t.Fatal(err)
	}
	body["name"] = "segunda"
	if r := c.post("/api/connections", body); r.StatusCode != 403 || r.ErrCode() != "reauth_required" {
		t.Fatalf("sem reauth: %d %s", r.StatusCode, r.Body)
	}
	if r := c.post("/api/sessions/revoke-others", nil); r.StatusCode != 403 || r.ErrCode() != "reauth_required" {
		t.Fatalf("revoke-others sem reauth: %d %s", r.StatusCode, r.Body)
	}
	var s security.Session
	c.expect(c.get("/api/session"), 200).JSON(&s)
	if s.ReauthUntil != nil {
		t.Fatal("reauth_until não deve existir com a janela vencida")
	}
	// Leitura continua liberada; encerrar a PRÓPRIA sessão também.
	c.expect(c.get("/api/connections"), 200)

	// Reautenticação com senha errada ou código errado falha sem derrubar a sessão.
	for _, bad := range []map[string]string{
		{"password": "senha-errada-123456", "code": e.code(res.Secret)},
		{"password": testPassword, "code": "000000"},
	} {
		if r := c.post("/api/auth/reauth", bad); r.StatusCode != 403 || r.ErrCode() != "reauth_failed" {
			t.Fatalf("reauth inválida: %d %s", r.StatusCode, r.Body)
		}
	}
	c.expect(c.get("/api/session"), 200)
	// Com senha + código corretos a ação volta a ser aceita.
	c.expect(c.post("/api/auth/reauth", map[string]string{"password": testPassword, "code": e.code(res.Secret)}), 200).JSON(&s)
	if s.ReauthUntil == nil {
		t.Fatal("reauth deveria abrir a janela")
	}
	c.expect(c.post("/api/connections", body), 201)

	// O mesmo código não serve duas vezes (replay) na reautenticação.
	if _, err := e.db.Pool.Exec(ctx, `UPDATE admin_sessions SET reauth_at = now() - interval '10 minutes'`); err != nil {
		t.Fatal(err)
	}
	code := e.code(res.Secret)
	c.expect(c.post("/api/auth/reauth", map[string]string{"password": testPassword, "code": code}), 200)
	if r := c.post("/api/auth/reauth", map[string]string{"password": testPassword, "code": code}); r.StatusCode != 403 {
		t.Fatalf("replay na reauth: %d", r.StatusCode)
	}
	// Revogar OUTRA sessão exige reauth; a própria, não.
	other := e.newClient()
	other.login(res.Secret)
	var os security.Session
	other.expect(other.get("/api/session"), 200).JSON(&os)
	if _, err := e.db.Pool.Exec(ctx, `UPDATE admin_sessions SET reauth_at = now() - interval '10 minutes'`); err != nil {
		t.Fatal(err)
	}
	if r := c.delete("/api/sessions/" + os.ID); r.StatusCode != 403 || r.ErrCode() != "reauth_required" {
		t.Fatalf("revogar outra sem reauth: %d %s", r.StatusCode, r.Body)
	}
	other.expect(other.get("/api/session"), 200)
	c.expect(c.post("/api/auth/logout", nil), 200)
}

func TestReplaceSecondFactor(t *testing.T) {
	e := newAuthEnv(t, nil)
	c := e.newClient()
	res := c.login(nil)
	other := e.newClient()
	other.login(res.Secret)

	var enr struct{ Secret string }
	c.expect(c.post("/api/auth/totp/replace/begin", nil), 200).JSON(&enr)
	newSecret := decodeSecret(t, enr.Secret)
	// Antes de confirmar, o segredo antigo continua valendo.
	probe := e.newClient()
	probe.login(res.Secret)
	if r := c.post("/api/auth/totp/replace/confirm", map[string]string{"code": "000000"}); r.StatusCode != 401 {
		t.Fatalf("código errado na substituição: %d", r.StatusCode)
	}
	var out struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	c.expect(c.post("/api/auth/totp/replace/confirm", map[string]string{"code": e.code(newSecret)}), 200).JSON(&out)
	if len(out.RecoveryCodes) != 10 {
		t.Fatalf("novos códigos: %d", len(out.RecoveryCodes))
	}
	// Demais sessões caem; a atual permanece.
	if r := other.get("/api/session"); r.StatusCode != 401 {
		t.Fatalf("sessões antigas deveriam cair: %d", r.StatusCode)
	}
	c.expect(c.get("/api/session"), 200)

	// Códigos de recuperação antigos deixam de valer; os novos funcionam.
	old := e.newClient()
	old.expect(old.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200)
	if r := old.post("/api/auth/recover", map[string]string{"recovery_code": res.RecoveryCodes[0]}); r.StatusCode != 401 {
		t.Fatalf("código antigo após substituição: %d", r.StatusCode)
	}
	// O segredo antigo não abre mais sessão; o novo abre.
	oldSecret := e.newClient()
	oldSecret.expect(oldSecret.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200)
	if r := oldSecret.post("/api/auth/totp", map[string]string{"code": e.code(res.Secret)}); r.StatusCode != 401 {
		t.Fatalf("segredo antigo ainda vale: %d", r.StatusCode)
	}
	fresh := e.newClient()
	fresh.login(newSecret)
	// Sem 2FA pendente, confirmar falha com estado incompatível.
	if r := c.post("/api/auth/totp/replace/confirm", map[string]string{"code": e.code(newSecret)}); r.StatusCode != 409 || r.ErrCode() != "totp_state" {
		t.Fatalf("confirmar sem pendência: %d %s", r.StatusCode, r.Body)
	}
}

func TestRecoveryLoginAllowsReplacingSecondFactor(t *testing.T) {
	e := newAuthEnv(t, nil)
	first := e.newClient()
	res := first.login(nil)

	// Perdeu o app autenticador: entra com código de recuperação e substitui o 2FA.
	c := e.newClient()
	c.expect(c.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200)
	var s security.Session
	c.expect(c.post("/api/auth/recover", map[string]string{"recovery_code": res.RecoveryCodes[0]}), 200).JSON(&s)
	c.csrf = s.CSRFToken
	if !s.TOTPResetRequired || s.ReauthUntil == nil {
		t.Fatalf("sessão de recuperação: %+v", s)
	}
	var enr struct{ Secret string }
	c.expect(c.post("/api/auth/totp/replace/begin", nil), 200).JSON(&enr)
	ns := decodeSecret(t, enr.Secret)
	c.expect(c.post("/api/auth/totp/replace/confirm", map[string]string{"code": e.code(ns)}), 200)
	c.expect(c.get("/api/session"), 200).JSON(&s)
	if s.TOTPResetRequired {
		t.Fatal("marca de substituição deve sumir")
	}
	if r := first.get("/api/session"); r.StatusCode != 401 {
		t.Fatalf("sessão antiga após recuperação: %d", r.StatusCode)
	}
}

func TestAuthTrailRecordsEventsWithoutSecrets(t *testing.T) {
	e := newAuthEnv(t, nil)
	c := e.newClient()
	c.post("/api/auth/login", map[string]string{"username": testUsername, "password": "senha-errada-123456"})
	res := c.login(nil)
	code := res.RecoveryCodes[0]
	c.expect(c.post("/api/auth/logout", nil), 200)

	again := e.newClient()
	again.login(res.Secret)
	var ev struct {
		Items []security.Event `json:"items"`
	}
	again.expect(again.get("/api/auth/events"), 200).JSON(&ev)
	got := map[string]bool{}
	var all strings.Builder
	for _, it := range ev.Items {
		got[it.Type] = true
		all.WriteString(it.Type + string(it.Detail) + it.IP + it.UserAgent)
		// admin_created vem da linha de comando, sem origem HTTP.
		if it.Type != "admin_created" && (it.IP == "" || it.UserAgent != "teste-integracao") {
			t.Errorf("evento sem origem: %+v", it)
		}
	}
	for _, want := range []string{"admin_created", "login_failed", "password_ok", "totp_enrolled", "login_success", "logout"} {
		if !got[want] {
			t.Errorf("trilha sem %q: %v", want, got)
		}
	}
	// A trilha jamais guarda senha, códigos, segredo TOTP nem o nome tentado.
	var raw string
	_ = e.db.Pool.QueryRow(ctx, `SELECT string_agg(auth_events::text, ' ') FROM auth_events`).Scan(&raw)
	for _, bad := range []string{testPassword, "senha-errada-123456", code, strings.ReplaceAll(code, "-", ""), security.EncodeTOTPSecret(res.Secret)} {
		if strings.Contains(raw, bad) || strings.Contains(all.String(), bad) || strings.Contains(e.logs.String(), bad) {
			t.Fatalf("segredo apareceu na trilha/logs: %q", bad)
		}
	}
}

func TestHTTPSIsEnforcedWhenRequired(t *testing.T) {
	e := newAuthEnv(t, func(c *config.API) {
		c.Env = config.Production
		c.RequireHTTPS, c.TrustProxy, c.SessionCookieSecure = true, true, true
	})
	c := e.newClient()
	login := map[string]string{"username": testUsername, "password": testPassword}
	if r := c.post("/api/auth/login", login); r.StatusCode != 400 || r.ErrCode() != "https_required" {
		t.Fatalf("HTTP puro deve ser recusado: %d %s", r.StatusCode, r.Body)
	}
	r := c.do("POST", "/api/auth/login", login, map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-For": "203.0.113.9"})
	if r.StatusCode != 200 {
		t.Fatalf("atrás do proxy TLS deve passar: %d %s", r.StatusCode, r.Body)
	}
	if !strings.Contains(r.Header.Get("Strict-Transport-Security"), "max-age=") {
		t.Fatal("HSTS ausente")
	}
	// Cookies saem com Secure + HttpOnly + SameSite=Strict.
	var seen int
	for _, ck := range r.Cookies() {
		seen++
		if !ck.Secure || !ck.HttpOnly || ck.SameSite != http.SameSiteStrictMode {
			t.Fatalf("cookie inseguro: %+v", ck)
		}
	}
	if seen == 0 {
		t.Fatal("nenhum cookie emitido")
	}
	// IP do cliente vem do proxy e vai para a trilha.
	var ip string
	_ = e.db.Pool.QueryRow(ctx, `SELECT ip FROM auth_events WHERE type = 'password_ok'`).Scan(&ip)
	if ip != "203.0.113.9" {
		t.Fatalf("IP da trilha: %q", ip)
	}
	// Sondas de saúde ficam isentas (o orquestrador não fala TLS).
	if r := c.get("/api/healthz"); r.StatusCode != 200 {
		t.Fatalf("healthz: %d", r.StatusCode)
	}
}
