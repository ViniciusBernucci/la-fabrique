package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/ViniciusBernucci/le-fabrique/internal/clilogin"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// fakeDocker imita o `docker` com a saída REAL dos clientes (observada nas imagens, v0.164.0 e 2.1.325).
// Controle por arquivos em FAKE_DIR: "finish" libera o login do Codex; "killed" é criado por `docker kill`.
const fakeDocker = `#!/bin/bash
echo "$*" >> "$FAKE_DIR/log"
args=" $* "
case "$1" in
  version) [ -f "$FAKE_DIR/no-docker" ] && exit 1; echo 29.0.0; exit 0;;
  image|volume|rm) exit 0;;
  kill) touch "$FAKE_DIR/killed"; exit 0;;
  run)
    case "$args" in
      *" chown "*) exit 0;;
      *"login --device-auth"*)
        printf 'Welcome to Codex [v0.164.0]\n1. Open this link\n   \033[94mhttps://auth.openai.com/codex/device\033[0m\n2. Enter this one-time code\n   \033[94mABCD-EFGHI\033[0m\n'
        for i in $(seq 1 200); do
          [ -f "$FAKE_DIR/finish" ] && exit ${FAKE_EXIT:-0}
          [ -f "$FAKE_DIR/killed" ] && exit 137
          sleep 0.05
        done; exit 1;;
      *"auth login"*)
        printf "Opening browser to sign in…\nIf the browser didn't open, visit: https://claude.com/cai/oauth/authorize?code=true&state=zzz\nPaste code here if prompted > "
        read -r line; echo "$line" > "$FAKE_DIR/pasted"
        [ "$line" = "good-code-123" ] && exit 0
        echo "Invalid code https://claude.com/cai/oauth/authorize?state=zzz"; exit 1;;
      *"auth status"*) echo '{ "loggedIn": true }'; exit 0;;
      *"login status"*) echo "Logged in using ChatGPT"; exit 0;;
    esac;;
esac
exit 0
`

type loginRig struct {
	t      *testing.T
	e      *authEnv
	c      *client
	runner *clilogin.Runner
	dir    string
}

func newLoginRig(t *testing.T) *loginRig {
	t.Helper()
	e := newAuthEnv(t, nil)
	c := e.newClient()
	c.login(nil)
	dir := t.TempDir()
	bin := filepath.Join(dir, "docker")
	if err := os.WriteFile(bin, []byte(fakeDocker), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_DIR", dir)
	sealer, _ := security.NewSealer(e.cfg.VaultKey, "k1")
	r := &clilogin.Runner{
		Store: &clilogin.Store{DB: e.db.Pool}, Vault: security.NewVault(e.db.Pool, sealer, security.NewRedactor()),
		Log: slog.New(slog.NewTextHandler(os.Stderr, nil)), Bin: bin, Image: "img:test", PollEvery: 30 * time.Millisecond,
	}
	return &loginRig{t: t, e: e, c: c, runner: r, dir: dir}
}

type loginView struct {
	ID, Provider, Name, State string
	VerificationURI           string `json:"verification_uri"`
	UserCode                  string `json:"user_code"`
	NeedsCode                 bool   `json:"needs_code"`
	Error                     string
	Connection                *security.Connection
}

func (g *loginRig) start(provider, name string) loginView {
	g.t.Helper()
	var v loginView
	g.c.expect(g.c.post("/api/cli-logins", map[string]string{"provider": provider, "name": name}), 201).JSON(&v)
	if _, err := g.runner.Tick(ctx); err != nil {
		g.t.Fatal(err)
	}
	return v
}

func (g *loginRig) waitState(id string, states ...string) loginView {
	g.t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		var v loginView
		g.c.expect(g.c.get("/api/cli-logins/"+id), 200).JSON(&v)
		for _, s := range states {
			if v.State == s {
				return v
			}
		}
		time.Sleep(40 * time.Millisecond)
	}
	g.t.Fatalf("login %s não chegou a %v", id, states)
	return loginView{}
}

func (g *loginRig) dockerLog() string {
	b, _ := os.ReadFile(filepath.Join(g.dir, "log"))
	return string(b)
}

func (g *loginRig) challengeCleared(id string) {
	g.t.Helper()
	var url, code, pasted *string
	if err := g.e.db.Pool.QueryRow(ctx, `SELECT challenge_url, challenge_code, submitted_code FROM cli_logins WHERE id = $1`, id).Scan(&url, &code, &pasted); err != nil {
		g.t.Fatal(err)
	}
	if url != nil || code != nil || pasted != nil {
		g.t.Fatalf("o desafio e o código colado devem ser apagados ao terminar: %v %v %v", url, code, pasted)
	}
}

// Codex: link + código na tela, a pessoa autoriza, o cliente termina e a conexão nasce com o id do volume.
func TestCLILoginCodexDeviceFlowCreatesTheConnection(t *testing.T) {
	g := newLoginRig(t)
	v := g.start("codex", "Codex pessoal")
	aw := g.waitState(v.ID, "awaiting_user")
	if aw.VerificationURI != "https://auth.openai.com/codex/device" || aw.UserCode != "ABCD-EFGHI" || aw.NeedsCode {
		t.Fatalf("desafio: %+v", aw)
	}
	if err := os.WriteFile(filepath.Join(g.dir, "finish"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	done := g.waitState(v.ID, "done")
	if done.Connection == nil || done.Connection.Kind != "cli" || done.Connection.Target != "codex" || done.Connection.State != "active" || done.Connection.Name != "Codex pessoal" {
		t.Fatalf("conexão criada: %+v", done.Connection)
	}
	// o id da conexão é o do volume em que o login foi feito
	if !strings.Contains(g.dockerLog(), "lf-acct-"+done.Connection.ID+":/accounts/current") {
		t.Fatalf("o login deve usar o volume lf-acct-<id da conexão>:\n%s", g.dockerLog())
	}
	// nada do login (link, código) permanece, e o segredo no cofre é só um marcador
	g.challengeCleared(v.ID)
	var list struct{ Items []security.Connection }
	g.c.expect(g.c.get("/api/connections"), 200).JSON(&list)
	if len(list.Items) != 1 || list.Items[0].ID != done.Connection.ID {
		t.Fatalf("conexões: %+v", list.Items)
	}
	if body := g.c.get("/api/cli-logins/" + v.ID).Body; strings.Contains(string(body), "ABCD-EFGHI") || strings.Contains(string(body), "auth.openai.com") {
		t.Fatalf("o desafio não pode aparecer depois de concluído: %s", body)
	}
	// o container roda sem privilégios
	for _, flag := range []string{"--cap-drop ALL", "no-new-privileges", "--pids-limit", "--memory"} {
		if !strings.Contains(g.dockerLog(), flag) {
			t.Errorf("container do login sem %q", flag)
		}
	}
}

// Claude: a pessoa abre o link, autoriza e cola de volta o código que o site mostrou.
func TestCLILoginClaudePastedCodeIsDeliveredOnceAndErased(t *testing.T) {
	g := newLoginRig(t)
	v := g.start("claude", "Claude Max")
	aw := g.waitState(v.ID, "awaiting_user")
	if !aw.NeedsCode || aw.UserCode != "" || !strings.HasPrefix(aw.VerificationURI, "https://claude.com/cai/oauth/authorize") {
		t.Fatalf("desafio: %+v", aw)
	}
	// código com formato impossível é recusado antes de chegar ao cliente
	if r := g.c.post("/api/cli-logins/"+v.ID+"/code", map[string]string{"code": "a b\nc"}); r.StatusCode != 400 {
		t.Fatalf("código inválido: %d %s", r.StatusCode, r.Body)
	}
	g.c.expect(g.c.post("/api/cli-logins/"+v.ID+"/code", map[string]string{"code": "good-code-123"}), 200)
	done := g.waitState(v.ID, "done")
	if done.Connection == nil || done.Connection.Target != "claude" {
		t.Fatalf("conexão: %+v", done.Connection)
	}
	pasted, _ := os.ReadFile(filepath.Join(g.dir, "pasted"))
	if strings.TrimSpace(string(pasted)) != "good-code-123" {
		t.Fatalf("o cliente deveria receber o código colado: %q", pasted)
	}
	g.challengeCleared(v.ID)
}

func TestCLILoginWrongCodeFailsWithoutLeakingTheLink(t *testing.T) {
	g := newLoginRig(t)
	v := g.start("claude", "Claude errado")
	g.waitState(v.ID, "awaiting_user")
	g.c.expect(g.c.post("/api/cli-logins/"+v.ID+"/code", map[string]string{"code": "wrong-code-999"}), 200)
	f := g.waitState(v.ID, "failed")
	if f.Error == "" || strings.Contains(f.Error, "claude.com") || strings.Contains(f.Error, "zzz") {
		t.Fatalf("a mensagem deve explicar sem vazar o link: %q", f.Error)
	}
	if !strings.Contains(g.dockerLog(), "volume rm -f lf-acct-") {
		t.Fatal("login que falhou não deve deixar o volume da conta")
	}
	g.challengeCleared(v.ID)
	var n int
	_ = g.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM connections`).Scan(&n)
	if n != 0 {
		t.Fatalf("falha não pode criar conexão: %d", n)
	}
}

func TestCLILoginCancelStopsTheContainerAndCleansUp(t *testing.T) {
	g := newLoginRig(t)
	v := g.start("codex", "Cancelável")
	g.waitState(v.ID, "awaiting_user")
	var c loginView
	g.c.expect(g.c.post("/api/cli-logins/"+v.ID+"/cancel", nil), 200).JSON(&c)
	if c.State != "cancelled" || c.VerificationURI != "" || c.UserCode != "" {
		t.Fatalf("cancelado: %+v", c)
	}
	deadline := time.Now().Add(10 * time.Second)
	for !strings.Contains(g.dockerLog(), "kill lf-login-"+v.ID) || !strings.Contains(g.dockerLog(), "volume rm -f") {
		if time.Now().After(deadline) {
			t.Fatalf("cancelar deve parar o container e apagar o volume:\n%s", g.dockerLog())
		}
		time.Sleep(40 * time.Millisecond)
	}
	// idempotente, e a conta nunca foi criada
	g.c.expect(g.c.post("/api/cli-logins/"+v.ID+"/cancel", nil), 200)
	var n int
	_ = g.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM connections`).Scan(&n)
	if n != 0 {
		t.Fatalf("cancelar não pode criar conexão: %d", n)
	}
}

func TestCLILoginRules(t *testing.T) {
	g := newLoginRig(t)
	// um login aberto por provedor
	v := g.start("codex", "Primeiro")
	if r := g.c.post("/api/cli-logins", map[string]string{"provider": "codex", "name": "Segundo"}); r.StatusCode != 409 || r.ErrCode() != "login_in_progress" {
		t.Fatalf("segundo login do mesmo provedor: %d %s", r.StatusCode, r.Body)
	}
	g.c.expect(g.c.post("/api/cli-logins/"+v.ID+"/cancel", nil), 200)
	// provedor desconhecido e nome vazio
	if r := g.c.post("/api/cli-logins", map[string]string{"provider": "antigravity", "name": "x"}); r.StatusCode != 400 {
		t.Fatalf("provedor sem suporte: %d", r.StatusCode)
	}
	if r := g.c.post("/api/cli-logins", map[string]string{"provider": "claude", "name": "  "}); r.StatusCode != 400 {
		t.Fatalf("nome vazio: %d", r.StatusCode)
	}
	// nome de uma conexão ATIVA continua ocupado
	connWith(t, g.c, "cli", "Já existe", "claude")
	if r := g.c.post("/api/cli-logins", map[string]string{"provider": "claude", "name": "Já existe"}); r.StatusCode != 409 || r.ErrCode() != "duplicate" {
		t.Fatalf("nome ocupado: %d %s", r.StatusCode, r.Body)
	}
	// código em login que não espera código
	cv := g.start("claude", "Claude regras")
	g.c.expect(g.c.post("/api/cli-logins/"+cv.ID+"/cancel", nil), 200)
	if r := g.c.post("/api/cli-logins/"+cv.ID+"/code", map[string]string{"code": "good-code-123"}); r.StatusCode != 409 {
		t.Fatalf("código em login encerrado: %d", r.StatusCode)
	}
	// id inexistente e id inválido
	if r := g.c.get("/api/cli-logins/00000000-0000-4000-8000-000000000000"); r.StatusCode != 404 {
		t.Fatalf("login inexistente: %d", r.StatusCode)
	}
	if r := g.c.get("/api/cli-logins/nao-e-uuid"); r.StatusCode != 400 {
		t.Fatalf("id inválido: %d", r.StatusCode)
	}
	// como as demais rotas que mudam estado, exige CSRF e sessão
	if r := g.c.do("POST", "/api/cli-logins", map[string]string{"provider": "codex", "name": "x"}, map[string]string{"X-CSRF-Token": ""}); r.StatusCode != 403 {
		t.Fatalf("sem CSRF: %d", r.StatusCode)
	}
	if r := g.e.newClient().post("/api/cli-logins", map[string]string{"provider": "codex", "name": "x"}); r.StatusCode != 401 {
		t.Fatalf("sem sessão: %d", r.StatusCode)
	}
}

func TestCLILoginExplainsWhenDockerIsUnavailable(t *testing.T) {
	g := newLoginRig(t)
	if err := os.WriteFile(filepath.Join(g.dir, "no-docker"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	v := g.start("codex", "Sem docker")
	f := g.waitState(v.ID, "failed")
	if !strings.Contains(f.Error, "docker.sock") {
		t.Fatalf("a mensagem deve dizer o que montar: %q", f.Error)
	}
}

func TestCLILoginExpiredAndLeftoverLoginsAreSwept(t *testing.T) {
	g := newLoginRig(t)
	// um login que ficou aberto quando o worker caiu
	var id string
	if err := g.e.db.Pool.QueryRow(ctx, `INSERT INTO cli_logins (provider, name, state, challenge_url, challenge_code, expires_at)
		VALUES ('codex', 'Sobra', 'awaiting_user', 'https://auth.openai.com/codex/device', 'ABCD-EFGHI', now() + interval '10 minutes') RETURNING id::text`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := g.runner.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	var v loginView
	g.c.expect(g.c.get("/api/cli-logins/"+id), 200).JSON(&v)
	if v.State != "failed" || !strings.Contains(v.Error, "reiniciou") {
		t.Fatalf("sobra: %+v", v)
	}
	g.challengeCleared(id)
	if !strings.Contains(g.dockerLog(), "rm -f lf-login-"+id) {
		t.Fatalf("o container da sobra deve ser removido:\n%s", g.dockerLog())
	}
}

// Passado o prazo, o worker derruba o container e o login falha com uma mensagem clara.
func TestCLILoginExpiresWhenTheUserTakesTooLong(t *testing.T) {
	g := newLoginRig(t)
	v := g.start("codex", "Demorado")
	g.waitState(v.ID, "awaiting_user")
	if _, err := g.e.db.Pool.Exec(ctx, `UPDATE cli_logins SET expires_at = now() - interval '1 second' WHERE id = $1`, v.ID); err != nil {
		t.Fatal(err)
	}
	f := g.waitState(v.ID, "failed")
	if !strings.Contains(f.Error, "tempo") {
		t.Fatalf("mensagem de prazo: %q", f.Error)
	}
	g.challengeCleared(v.ID)
	if !strings.Contains(g.dockerLog(), "kill lf-login-"+v.ID) {
		t.Fatalf("o container deve ser derrubado:\n%s", g.dockerLog())
	}
}
