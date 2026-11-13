package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/httpapi"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/logging"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

const (
	sentinelA = "SENTINELA-A-5e8a1f0c-nao-pode-vazar"
	sentinelB = "SENTINELA-B-91d3b7e2-nao-pode-vazar"
)

// fakeGitHub só aceita o segredo "valid". Em caso de recusa devolve um corpo que
// ECOA o cabeçalho recebido, para provar que nada disso chega ao cliente da API.
type fakeGitHub struct {
	*httptest.Server
	valid atomic.Value // string
	calls atomic.Int32
}

func newFakeGitHub(t *testing.T) *fakeGitHub {
	f := &fakeGitHub{}
	f.valid.Store("")
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		got := r.Header.Get("Authorization")
		if got != "Bearer "+f.valid.Load().(string) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, `{"message":"Bad credentials for %s"}`, got)
			return
		}
		w.Header().Set("X-OAuth-Scopes", "repo")
		fmt.Fprint(w, `{"login":"octocat"}`)
	}))
	t.Cleanup(f.Close)
	return f
}

func newVaultEnv(t *testing.T) (*authEnv, *client, *fakeGitHub) {
	t.Helper()
	gh := newFakeGitHub(t)
	tester := integrations.NewTester()
	tester.GitHubBase = gh.URL
	e := newAuthEnv(t, nil, httpapi.WithTester(tester))
	c := e.newClient()
	c.login(nil)
	return e, c, gh
}

func createConn(t *testing.T, c *client, name, secret string) security.Connection {
	t.Helper()
	var conn security.Connection
	c.expect(c.post("/api/connections", map[string]string{
		"kind": "github", "name": name, "target": "org-exemplo", "scopes": "repo", "secret": secret,
	}), 201).JSON(&conn)
	return conn
}

// T04-03 — "Busca com segredo sentinela não acha texto em logs, respostas ou prompts."
func TestSentinelSecretNeverAppearsAnywhere(t *testing.T) {
	e, c, gh := newVaultEnv(t)
	var bodies strings.Builder
	collect := func(r apiResp) apiResp {
		bodies.Write(r.Body)
		bodies.WriteString(r.Header.Get("Set-Cookie"))
		return r
	}

	gh.valid.Store(sentinelA)
	conn := createConn(t, c, "github-principal", sentinelA)
	bodies.WriteString(fmt.Sprint(conn))
	collect(c.get("/api/connections"))
	collect(c.get("/api/connections/" + conn.ID))

	// Teste com credencial válida.
	var tested security.Connection
	collect(c.expect(c.post("/api/connections/"+conn.ID+"/test", nil), 200)).JSON(&tested)
	if tested.LastTestStatus != "ok" || !strings.Contains(tested.LastTestDetail, "octocat") {
		t.Fatalf("teste válido: %+v", tested)
	}

	// Teste com credencial recusada: o corpo remoto ecoa o segredo, mas ele não pode voltar.
	gh.valid.Store("outra-credencial-qualquer")
	collect(c.expect(c.post("/api/connections/"+conn.ID+"/test", nil), 200)).JSON(&tested)
	if tested.LastTestStatus != "failed" || !strings.Contains(tested.LastTestDetail, "401") {
		t.Fatalf("teste recusado: %+v", tested)
	}

	// Rotação: o valor anterior também some de tudo.
	var rotated security.Connection
	collect(c.expect(c.post("/api/connections/"+conn.ID+"/rotate", map[string]any{"secret": sentinelB, "expected_version": tested.Version}), 200)).JSON(&rotated)
	gh.valid.Store(sentinelB)
	collect(c.expect(c.post("/api/connections/"+conn.ID+"/test", nil), 200))
	collect(c.post("/api/connections/"+conn.ID+"/rotate", map[string]any{"secret": "x" + sentinelA, "expected_version": 1}))  // versão obsoleta
	collect(c.post("/api/connections", map[string]string{"kind": "github", "name": "github-principal", "secret": sentinelA})) // nome duplicado
	collect(c.post("/api/connections", map[string]string{"kind": "inexistente", "name": "z", "secret": sentinelA}))           // tipo inválido
	collect(c.post("/api/connections", map[string]any{"kind": "api", "name": "z", "secret": sentinelA, "x": 1}))              // campo extra
	collect(c.expect(c.post("/api/connections/"+conn.ID+"/revoke", nil), 200))
	collect(c.post("/api/connections/"+conn.ID+"/test", nil)) // conexão revogada
	collect(c.get("/api/auth/events"))

	// Logs reais da API, incluindo erros que carregam o valor (a redação cobre o texto livre).
	log := logging.New(e.logs, "debug", "api")
	log.Error("falha ao chamar " + sentinelA)
	log.Error("falha", "error", errors.New("resposta: "+sentinelB), "detalhe", "usa "+sentinelA)
	log.Info("seguro", "token", sentinelA, "password", sentinelB)

	haystacks := map[string]string{
		"respostas HTTP": bodies.String(),
		"logs":           e.logs.String(),
	}
	for _, q := range []string{
		`SELECT string_agg(connections::text, ' ') FROM connections`,
		`SELECT string_agg(auth_events::text, ' ') FROM auth_events`,
		`SELECT string_agg(admin_sessions::text, ' ') FROM admin_sessions`,
		`SELECT string_agg(admins::text, ' ') FROM admins`,
		`SELECT string_agg(outbox::text, ' ') FROM outbox`,
		`SELECT string_agg(execution_events::text, ' ') FROM execution_events`,
	} {
		var out *string
		if err := e.db.Pool.QueryRow(ctx, q).Scan(&out); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		if out != nil {
			haystacks[q] = *out
		}
	}
	for where, text := range haystacks {
		for _, s := range []string{sentinelA, sentinelB} {
			if strings.Contains(text, s) {
				t.Errorf("segredo sentinela encontrado em %s", where)
			}
		}
	}
	if len(e.logs.String()) < 100 || !strings.Contains(e.logs.String(), "[REDACTED]") {
		t.Fatal("o log de teste deveria conter entradas e marcas de redação")
	}
}

func TestSecretsAreEncryptedWithKeyOutsideDatabase(t *testing.T) {
	e, c, _ := newVaultEnv(t)
	conn := createConn(t, c, "cifrada", sentinelA)
	var ct []byte
	var keyID string
	if err := e.db.Pool.QueryRow(ctx, `SELECT secret_ciphertext, key_id FROM connections WHERE id = $1`, conn.ID).Scan(&ct, &keyID); err != nil {
		t.Fatal(err)
	}
	if keyID != "k1" || len(ct) < 28 || strings.Contains(string(ct), sentinelA) {
		t.Fatalf("cifrado inválido (key=%s, %d bytes)", keyID, len(ct))
	}
	// Com a chave do ambiente o serviço decifra; com OUTRA chave não decifra.
	sealer, _ := security.NewSealer(e.cfg.VaultKey, "k1")
	vault := security.NewVault(e.db.Pool, sealer, security.NewRedactor())
	got, _, err := vault.Resolve(ctx, conn.ID)
	if err != nil || got.Reveal() != sentinelA {
		t.Fatalf("resolver: %v", err)
	}
	wrong, _ := security.NewSealer([]byte("fedcba9876543210fedcba9876543210"), "k2")
	wrongVault := security.NewVault(e.db.Pool, wrong, security.NewRedactor())
	if _, _, err := wrongVault.Resolve(ctx, conn.ID); err == nil {
		t.Fatal("chave diferente não pode decifrar")
	}
	// Na subida, chave errada é detectada e contada (aviso), sem derrubar a API.
	if n, err := wrongVault.LoadRedactor(ctx); err != nil || n != 1 {
		t.Fatalf("LoadRedactor com chave errada: n=%d err=%v", n, err)
	}
	if n, err := vault.LoadRedactor(ctx); err != nil || n != 0 {
		t.Fatalf("LoadRedactor com a chave certa: n=%d err=%v", n, err)
	}
	// O cifrado de uma conexão copiado para outra linha não abre (AAD = id).
	other := createConn(t, c, "outra", "valor-da-outra-conexao")
	if _, err := e.db.Pool.Exec(ctx, `UPDATE connections SET secret_ciphertext = $1 WHERE id = $2`, ct, other.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := vault.Resolve(ctx, other.ID); err == nil || !errors.Is(err, security.ErrSealed) {
		t.Fatalf("cifrado transplantado deveria falhar: %v", err)
	}
}

// T04-03 — "Rotação mantém vínculos sem expor valor anterior."
func TestRotationKeepsLinksAndDropsPreviousValue(t *testing.T) {
	e, c, gh := newVaultEnv(t)
	conn := createConn(t, c, "rotacionavel", sentinelA)
	// "Vínculo": uma referência por id (connection:<id>) guardada por quem usa a conexão.
	ref := security.ConnectionRef(conn.ID)

	sealer, _ := security.NewSealer(e.cfg.VaultKey, "k1")
	vault := security.NewVault(e.db.Pool, sealer, security.NewRedactor())
	before, _, err := vault.ResolveRef(ctx, ref)
	if err != nil || before.Reveal() != sentinelA {
		t.Fatalf("antes da rotação: %v", err)
	}

	var rotated security.Connection
	c.expect(c.post("/api/connections/"+conn.ID+"/rotate", map[string]any{"secret": sentinelB, "expected_version": conn.Version}), 200).JSON(&rotated)
	if rotated.ID != conn.ID || rotated.SecretVersion != 2 || rotated.Version != 2 || rotated.RotatedAt == nil || rotated.State != "active" {
		t.Fatalf("rotação deve manter o id e subir as versões: %+v", rotated)
	}
	if rotated.LastTestStatus != "not_run" {
		t.Fatal("rotação invalida o resultado do teste anterior")
	}
	// A MESMA referência agora resolve para o valor novo; o antigo não existe mais.
	after, _, err := vault.ResolveRef(ctx, ref)
	if err != nil || after.Reveal() != sentinelB {
		t.Fatalf("referência após rotação: %v", err)
	}
	var n int
	_ = e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM connections`).Scan(&n)
	if n != 1 {
		t.Fatalf("rotação não pode criar conexão: %d", n)
	}
	// Teste usa o valor novo.
	gh.valid.Store(sentinelB)
	var tested security.Connection
	c.expect(c.post("/api/connections/"+conn.ID+"/test", nil), 200).JSON(&tested)
	if tested.LastTestStatus != "ok" {
		t.Fatalf("teste com o valor novo: %+v", tested)
	}
	// Versão obsoleta é recusada com 409.
	if r := c.post("/api/connections/"+conn.ID+"/rotate", map[string]any{"secret": "terceiro-valor-xyz", "expected_version": 1}); r.StatusCode != 409 || r.ErrCode() != "version_conflict" {
		t.Fatalf("versão obsoleta: %d %s", r.StatusCode, r.Body)
	}
	// Segredo vazio é recusado.
	if r := c.post("/api/connections/"+conn.ID+"/rotate", map[string]any{"secret": "  ", "expected_version": rotated.Version}); r.StatusCode != 400 {
		t.Fatalf("segredo vazio: %d", r.StatusCode)
	}
	// Referência inválida / conexão inexistente.
	if _, _, err := vault.ResolveRef(ctx, "connection:nao-e-uuid"); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("ref inválida: %v", err)
	}
	if _, _, err := vault.ResolveRef(ctx, security.ConnectionRef("00000000-0000-4000-8000-000000000000")); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("ref inexistente: %v", err)
	}
}

func TestConcurrentRotationsAreSerializedByVersion(t *testing.T) {
	_, c, _ := newVaultEnv(t)
	conn := createConn(t, c, "disputada", sentinelA)
	var ok, conflict atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			switch r := c.post("/api/connections/"+conn.ID+"/rotate", map[string]any{"secret": fmt.Sprintf("valor-concorrente-%d", i), "expected_version": conn.Version}); r.StatusCode {
			case 200:
				ok.Add(1)
			case 409:
				conflict.Add(1)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if ok.Load() != 1 || conflict.Load() != 5 {
		t.Fatalf("esperado 1 sucesso e 5 conflitos, veio %d/%d", ok.Load(), conflict.Load())
	}
}

// T04-03 — "Revogação é observável antes do próximo uso."
func TestRevocationIsObservableBeforeNextUse(t *testing.T) {
	e, c, gh := newVaultEnv(t)
	conn := createConn(t, c, "revogavel", sentinelA)
	gh.valid.Store(sentinelA)

	sealer, _ := security.NewSealer(e.cfg.VaultKey, "k1")
	vault := security.NewVault(e.db.Pool, sealer, security.NewRedactor())
	if _, _, err := vault.Resolve(ctx, conn.ID); err != nil {
		t.Fatal(err)
	}
	callsBefore := gh.calls.Load()

	var revoked security.Connection
	c.expect(c.post("/api/connections/"+conn.ID+"/revoke", nil), 200).JSON(&revoked)
	if revoked.State != "revoked" || revoked.RevokedAt == nil {
		t.Fatalf("estado após revogar: %+v", revoked)
	}
	// O próximo uso (mesmo por quem já tinha resolvido antes) falha SEM chamar o destino.
	if _, _, err := vault.Resolve(ctx, conn.ID); !errors.Is(err, security.ErrConnectionRevoked) {
		t.Fatalf("resolver após revogar: %v", err)
	}
	if _, _, err := vault.ResolveRef(ctx, security.ConnectionRef(conn.ID)); !errors.Is(err, security.ErrConnectionRevoked) {
		t.Fatalf("resolver por ref após revogar: %v", err)
	}
	if r := c.post("/api/connections/"+conn.ID+"/test", nil); r.StatusCode != 409 || r.ErrCode() != "connection_revoked" {
		t.Fatalf("teste após revogar: %d %s", r.StatusCode, r.Body)
	}
	if r := c.post("/api/connections/"+conn.ID+"/rotate", map[string]any{"secret": "novo-valor-xyz", "expected_version": revoked.Version}); r.StatusCode != 409 || r.ErrCode() != "connection_revoked" {
		t.Fatalf("rotacionar revogada: %d %s", r.StatusCode, r.Body)
	}
	if gh.calls.Load() != callsBefore {
		t.Fatal("destino não pode ser chamado depois da revogação")
	}
	// O segredo é apagado do banco, não apenas marcado.
	var ct []byte
	var keyID *string
	_ = e.db.Pool.QueryRow(ctx, `SELECT secret_ciphertext, key_id FROM connections WHERE id = $1`, conn.ID).Scan(&ct, &keyID)
	if ct != nil || keyID != nil {
		t.Fatal("segredo revogado deve ser apagado")
	}
	// Idempotente.
	c.expect(c.post("/api/connections/"+conn.ID+"/revoke", nil), 200)
	// Aparece na listagem como revogada, sem segredo.
	var list struct{ Items []security.Connection }
	c.expect(c.get("/api/connections"), 200).JSON(&list)
	if len(list.Items) != 1 || list.Items[0].State != "revoked" {
		t.Fatalf("listagem: %+v", list.Items)
	}
}

func TestConnectionValidationAndNotFound(t *testing.T) {
	_, c, _ := newVaultEnv(t)
	good := map[string]string{"kind": "api", "name": "ok", "secret": "valor-valido-123"}
	c.expect(c.post("/api/connections", good), 201)
	if r := c.post("/api/connections", good); r.StatusCode != 409 || r.ErrCode() != "duplicate" {
		t.Fatalf("nome duplicado: %d %s", r.StatusCode, r.Body)
	}
	for name, body := range map[string]map[string]string{
		"tipo":        {"kind": "ftp", "name": "a", "secret": "valor-valido-123"},
		"nome vazio":  {"kind": "api", "name": " ", "secret": "valor-valido-123"},
		"nome enorme": {"kind": "api", "name": strings.Repeat("n", 81), "secret": "valor-valido-123"},
		"sem segredo": {"kind": "api", "name": "b"},
	} {
		if r := c.post("/api/connections", body); r.StatusCode != 400 {
			t.Errorf("%s: esperado 400, veio %d %s", name, r.StatusCode, r.Body)
		}
	}
	for _, p := range []string{"/api/connections/00000000-0000-4000-8000-000000000000", "/api/connections/xyz"} {
		if r := c.get(p); r.StatusCode != 404 {
			t.Errorf("%s: %d", p, r.StatusCode)
		}
	}
	if r := c.post("/api/connections/00000000-0000-4000-8000-000000000000/test", nil); r.StatusCode != 404 {
		t.Fatalf("teste de inexistente: %d", r.StatusCode)
	}
	// Sem sessão: tudo é recusado.
	anon := newAuthAnon(t, c)
	for _, p := range []string{"/api/connections"} {
		if r := anon.get(p); r.StatusCode != 401 {
			t.Fatalf("%s sem sessão: %d", p, r.StatusCode)
		}
	}
}

func newAuthAnon(t *testing.T, like *client) *client { return like.e.newClient() }

// Valores gravados ANTES de um restart também são redigidos depois dele (Prime).
func TestRedactorIsRestoredAfterRestart(t *testing.T) {
	e, c, _ := newVaultEnv(t)
	createConn(t, c, "persistente", sentinelA)

	logging.SetRedactor(func(s string) string { return s }) // processo "novo": nada registrado
	var buf syncBuffer
	logging.New(&buf, "info", "api").Info("antes do Prime: " + sentinelA)
	if !strings.Contains(buf.String(), sentinelA) {
		t.Fatal("pré-condição: sem Prime o valor ainda não é conhecido")
	}

	api2, err := httpapi.NewServer(e.cfg, e.db.Pool, logging.New(&buf, "info", "api"))
	if err != nil {
		t.Fatal(err)
	}
	if err := api2.Prime(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { logging.SetRedactor(func(s string) string { return s }) })
	buf = syncBuffer{}
	logging.New(&buf, "info", "api").Info("depois do Prime: "+sentinelA, "campo", sentinelA)
	if strings.Contains(buf.String(), sentinelA) {
		t.Fatalf("valor persistido deveria ser redigido após o restart: %s", buf.String())
	}
}

func TestTesterBehaviors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/myself":
			if u, p, ok := r.BasicAuth(); ok && u == "eu@exemplo.com" && p == "tok" {
				fmt.Fprint(w, `{}`)
				return
			}
			w.WriteHeader(401)
		case "/user":
			if r.Header.Get("Authorization") == "tok-clickup" {
				fmt.Fprint(w, `{}`)
				return
			}
			w.WriteHeader(403)
		case "/redirect/user":
			http.Redirect(w, r, "/capture", http.StatusFound)
		case "/capture":
			t.Error("o testador não pode seguir redirecionamentos com credencial")
		}
	}))
	defer ts.Close()
	tester := integrations.NewTester()
	tester.GitHubBase, tester.ClickUpBase, tester.AllowAnyTarget = ts.URL+"/redirect-not-used", ts.URL, true

	run := func(kind, target, secret string) integrations.TestResult {
		return tester.Test(ctx, kind, target, security.NewSecret(secret))
	}
	if r := run("jira", ts.URL, "eu@exemplo.com:tok"); r.Status != integrations.TestOK {
		t.Fatalf("jira ok: %+v", r)
	}
	if r := run("jira", ts.URL, "eu@exemplo.com:errado"); r.Status != integrations.TestFailed || !strings.Contains(r.Detail, "401") {
		t.Fatalf("jira recusado: %+v", r)
	}
	if r := run("jira", ts.URL, "sem-formato"); r.Status != integrations.TestFailed || strings.Contains(r.Detail, "sem-formato") {
		t.Fatalf("jira formato: %+v", r)
	}
	if r := run("clickup", "", "tok-clickup"); r.Status != integrations.TestOK {
		t.Fatalf("clickup ok: %+v", r)
	}
	if r := run("clickup", "", "x"); r.Status != integrations.TestFailed || !strings.Contains(r.Detail, "403") {
		t.Fatalf("clickup 403: %+v", r)
	}
	tester.GitHubBase = ts.URL + "/redirect"
	if r := run("github", "", "tok"); r.Status != integrations.TestFailed {
		t.Fatalf("redirecionamento deve contar como falha sem ser seguido: %+v", r)
	}
	for _, kind := range []string{"cli", "api", "mcp"} {
		if r := run(kind, "", "valor"); r.Status != integrations.TestNotRun || strings.Contains(r.Detail, "valor") {
			t.Fatalf("%s: %+v", kind, r)
		}
	}
	// Restrição de destino do Jira (SSRF) sem AllowAnyTarget.
	strict := integrations.NewTester()
	for _, target := range []string{"http://x.atlassian.net", "https://evil.example", "https://169.254.169.254", "", "https://atlassian.net.evil.example"} {
		if r := strict.Test(ctx, "jira", target, security.NewSecret("a@b.c:t")); r.Status != integrations.TestFailed {
			t.Errorf("destino %q deveria ser recusado: %+v", target, r)
		}
	}
	// Falha de rede não ecoa URL nem erro do cliente.
	dead := integrations.NewTester()
	dead.GitHubBase = "http://127.0.0.1:1"
	if r := dead.Test(ctx, "github", "", security.NewSecret("segredo-x")); r.Status != integrations.TestFailed || strings.Contains(r.Detail, "127.0.0.1") {
		t.Fatalf("rede: %+v", r)
	}
}
