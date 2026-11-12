package integration

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// T04-01 — "Senha sem 2FA não abre sessão."
func TestPasswordAloneNeverOpensSession(t *testing.T) {
	e := newAuthEnv(t, nil)
	c := e.newClient()

	var stage struct{ Status string }
	c.expect(c.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200).JSON(&stage)
	if stage.Status != security.StageEnroll {
		t.Fatalf("primeiro acesso deve exigir cadastro do 2FA: %q", stage.Status)
	}
	// Só existe o cookie do desafio: nenhuma rota de dados abre.
	for _, p := range []string{"/api/session", "/api/projects", "/api/connections", "/api/sessions"} {
		if r := c.get(p); r.StatusCode != 401 {
			t.Fatalf("%s só com senha: %d", p, r.StatusCode)
		}
	}
	var n int
	_ = e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM admin_sessions`).Scan(&n)
	if n != 0 {
		t.Fatalf("senha sozinha criou %d sessão(ões)", n)
	}
	// Usar o token do desafio como cookie de sessão também não funciona.
	for _, ck := range c.http.Jar.Cookies(mustURL(e.srv.URL + "/api/auth/")) {
		if ck.Name == "lf_challenge" {
			bad := e.newClient()
			r, _ := bad.http.Do(newReq(t, e.srv.URL+"/api/session", "lf_session", ck.Value))
			if r.StatusCode != 401 {
				t.Fatalf("token de desafio aceito como sessão: %d", r.StatusCode)
			}
		}
	}
	// Já com 2FA ativo, senha correta continua sem abrir sessão até o código.
	c.login(nil)
	c2 := e.newClient()
	c.expect(c2.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200).JSON(&stage)
	if stage.Status != security.StageTOTP || c2.get("/api/session").StatusCode != 401 {
		t.Fatalf("com 2FA ativo a senha não pode bastar: %q", stage.Status)
	}
	// Sem desafio (ou com código errado) o segundo fator é recusado.
	fresh := e.newClient()
	if r := fresh.post("/api/auth/totp", map[string]string{"code": "123456"}); r.StatusCode != 401 || r.ErrCode() != "challenge_expired" {
		t.Fatalf("2º fator sem desafio: %d %s", r.StatusCode, r.Body)
	}
	if r := c2.post("/api/auth/totp", map[string]string{"code": "000000"}); r.StatusCode != 401 || r.ErrCode() != "invalid_code" {
		t.Fatalf("código errado: %d %s", r.StatusCode, r.Body)
	}
}

func TestEnrollmentAndTOTPLoginWithReplayProtection(t *testing.T) {
	e := newAuthEnv(t, nil)
	c := e.newClient()
	var stage struct{ Status string }
	c.expect(c.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200).JSON(&stage)
	var enr struct {
		Secret     string `json:"secret"`
		OTPAuthURI string `json:"otpauth_uri"`
	}
	c.expect(c.post("/api/auth/totp/setup", nil), 200).JSON(&enr)
	if !strings.HasPrefix(enr.OTPAuthURI, "otpauth://totp/") {
		t.Fatalf("uri: %s", enr.OTPAuthURI)
	}
	secret := decodeSecret(t, enr.Secret)

	// O segredo pendente é cifrado no banco (nunca em claro nem em base32).
	var sealed []byte
	_ = e.db.Pool.QueryRow(ctx, `SELECT totp_pending_secret FROM admins`).Scan(&sealed)
	if len(sealed) == 0 || strings.Contains(string(sealed), string(secret)) || strings.Contains(string(sealed), enr.Secret) {
		t.Fatal("segredo TOTP deve estar cifrado em repouso")
	}

	if r := c.post("/api/auth/totp/enable", map[string]string{"code": "000000"}); r.StatusCode != 401 {
		t.Fatalf("código errado no cadastro: %d", r.StatusCode)
	}
	var out struct {
		security.Session
		RecoveryCodes []string `json:"recovery_codes"`
	}
	c.expect(c.post("/api/auth/totp/enable", map[string]string{"code": e.code(secret)}), 201).JSON(&out)
	if len(out.RecoveryCodes) != 10 || out.CSRFToken == "" || out.Username != testUsername {
		t.Fatalf("resposta do cadastro: %+v", out)
	}
	// Sessão do cadastro vale; cookie HttpOnly/Strict.
	c.csrf = out.CSRFToken
	c.expect(c.get("/api/session"), 200)
	var cookieOK bool
	for _, ck := range c.http.Jar.Cookies(mustURL(e.srv.URL)) {
		cookieOK = cookieOK || ck.Name == "lf_session"
	}
	if !cookieOK {
		t.Fatal("cookie de sessão ausente")
	}
	// Nem os códigos de recuperação nem o segredo ficam em claro no banco.
	var dump string
	_ = e.db.Pool.QueryRow(ctx, `SELECT (SELECT string_agg(code_hash::text, ',') FROM recovery_codes) || (SELECT totp_secret::text FROM admins)`).Scan(&dump)
	for _, rc := range out.RecoveryCodes {
		if strings.Contains(dump, rc) || strings.Contains(dump, strings.ReplaceAll(rc, "-", "")) {
			t.Fatal("código de recuperação em claro")
		}
	}

	// Segundo login com TOTP. Reaproveitar o MESMO código (replay) falha.
	c2 := e.newClient()
	c2.expect(c2.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200)
	code := e.code(secret)
	c2.expect(c2.post("/api/auth/totp", map[string]string{"code": code}), 200)
	c3 := e.newClient()
	c3.expect(c3.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200)
	if r := c3.post("/api/auth/totp", map[string]string{"code": code}); r.StatusCode != 401 || r.ErrCode() != "invalid_code" {
		t.Fatalf("replay do mesmo código deve falhar: %d %s", r.StatusCode, r.Body)
	}
	// O desafio expira ao ser usado: o mesmo cookie não serve de novo.
	c2.expect(c2.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200)
	c4 := e.newClient()
	c4.login(secret)
}

// T04-01 — "Recuperação funciona uma vez mesmo com requisições concorrentes."
func TestRecoveryCodeWorksExactlyOnceUnderConcurrency(t *testing.T) {
	e := newAuthEnv(t, nil)
	first := e.newClient()
	res := first.login(nil)
	code := res.RecoveryCodes[0]

	// (a) Mesmo código em desafios diferentes, ao mesmo tempo.
	const n = 6
	clients := make([]*client, n)
	for i := range clients {
		clients[i] = e.newClient()
		clients[i].expect(clients[i].post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200)
	}
	var ok atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, cl := range clients {
		wg.Add(1)
		go func(cl *client) {
			defer wg.Done()
			<-start
			if r := cl.post("/api/auth/recover", map[string]string{"recovery_code": code}); r.StatusCode == 200 {
				ok.Add(1)
			}
		}(cl)
	}
	close(start)
	wg.Wait()
	if ok.Load() != 1 {
		t.Fatalf("o código deveria funcionar exatamente uma vez, funcionou %d", ok.Load())
	}

	// Os perdedores da disputa contaram como falhas e bloquearam a conta (correto);
	// limpa o bloqueio para exercitar a fase seguinte.
	clearThrottle(t, e)

	// (b) Mesmo código e MESMO desafio, em paralelo.
	other := res.RecoveryCodes[1]
	same := e.newClient()
	same.expect(same.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200)
	ok.Store(0)
	start = make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if r := same.post("/api/auth/recover", map[string]string{"recovery_code": other}); r.StatusCode == 200 {
				ok.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if ok.Load() != 1 {
		t.Fatalf("mesmo desafio: esperado 1 sucesso, veio %d", ok.Load())
	}

	clearThrottle(t, e)

	// (c) Depois de usado, o código não volta a funcionar; o 2FA fica marcado para substituição.
	late := e.newClient()
	late.expect(late.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200)
	if r := late.post("/api/auth/recover", map[string]string{"recovery_code": code}); r.StatusCode != 401 || r.ErrCode() != "invalid_code" {
		t.Fatalf("código reutilizado: %d %s", r.StatusCode, r.Body)
	}
	var remaining int
	_ = e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM recovery_codes WHERE used_at IS NULL`).Scan(&remaining)
	if remaining != 8 {
		t.Fatalf("restam %d códigos, esperado 8", remaining)
	}
	// Digitação com minúsculas/sem hífen vale como o mesmo código.
	var s struct {
		TOTPResetRequired bool `json:"totp_reset_required"`
	}
	third := e.newClient()
	third.expect(third.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200)
	third.expect(third.post("/api/auth/recover", map[string]string{"recovery_code": strings.ToLower(strings.ReplaceAll(res.RecoveryCodes[2], "-", ""))}), 200).JSON(&s)
	if !s.TOTPResetRequired {
		t.Fatal("após recuperação o 2FA deve ser marcado para substituição")
	}
}

func TestThrottleLocksAccountAfterRepeatedFailures(t *testing.T) {
	e := newAuthEnv(t, nil)
	c := e.newClient()
	wrong := func(user string) apiResp {
		return c.post("/api/auth/login", map[string]string{"username": user, "password": "senha-errada-qualquer"})
	}
	// Usuário inexistente e senha errada são indistinguíveis.
	a, b := wrong("admin"), wrong("ninguem")
	if a.StatusCode != 401 || b.StatusCode != 401 || a.ErrCode() != b.ErrCode() || a.ErrCode() != "invalid_credentials" {
		t.Fatalf("respostas distinguíveis: %d/%s vs %d/%s", a.StatusCode, a.Body, b.StatusCode, b.Body)
	}
	for i := 0; i < 4; i++ { // com a 1ª acima, são 5 falhas na conta "admin"
		wrong("admin")
	}
	// 5 falhas na conta: a próxima — mesmo com a senha CERTA — é bloqueada.
	r := c.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword})
	if r.StatusCode != 429 || r.ErrCode() != "too_many_attempts" || r.Header.Get("Retry-After") == "" {
		t.Fatalf("esperado bloqueio 429 com Retry-After: %d %s", r.StatusCode, r.Body)
	}
	var types string
	_ = e.db.Pool.QueryRow(ctx, `SELECT string_agg(DISTINCT type, ',') FROM auth_events`).Scan(&types)
	if !strings.Contains(types, "login_failed") || !strings.Contains(types, "login_blocked") {
		t.Fatalf("trilha deve registrar falha e bloqueio: %s", types)
	}
	// Expirado o bloqueio, o login volta (simulado no banco).
	if _, err := e.db.Pool.Exec(ctx, `UPDATE auth_throttle SET locked_until = now() - interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	c.expect(c.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200)
}

func TestSecondFactorAttemptsAreLimitedPerChallenge(t *testing.T) {
	e := newAuthEnv(t, func(c *config.API) {})
	secret := e.newClient().login(nil).Secret
	c := e.newClient()
	c.expect(c.post("/api/auth/login", map[string]string{"username": testUsername, "password": testPassword}), 200)
	for i := 0; i < 5; i++ {
		c.post("/api/auth/totp", map[string]string{"code": "000000"})
	}
	// Esgotadas as tentativas, nem o código correto passa: é preciso recomeçar.
	if r := c.post("/api/auth/totp", map[string]string{"code": e.code(secret)}); r.StatusCode == 200 {
		t.Fatalf("desafio esgotado não pode abrir sessão: %s", r.Body)
	}
}

func TestLoginRequestValidation(t *testing.T) {
	e := newAuthEnv(t, nil)
	c := e.newClient()
	if r := c.do("POST", "/api/auth/login", nil, map[string]string{"Content-Type": "text/plain"}); r.StatusCode != 415 {
		t.Fatalf("content-type: %d", r.StatusCode)
	}
	if r := c.post("/api/auth/login", map[string]any{"username": "a", "password": "b", "extra": 1}); r.StatusCode != 400 {
		t.Fatalf("campo desconhecido: %d", r.StatusCode)
	}
	if r := c.post("/api/auth/login", map[string]string{"username": testUsername, "password": strings.Repeat("x", 5000)}); r.StatusCode != 401 {
		t.Fatalf("senha enorme: %d", r.StatusCode)
	}
}

func clearThrottle(t *testing.T, e *authEnv) {
	t.Helper()
	if _, err := e.db.Pool.Exec(ctx, `DELETE FROM auth_throttle`); err != nil {
		t.Fatal(err)
	}
}
