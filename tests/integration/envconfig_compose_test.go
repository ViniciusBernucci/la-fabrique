package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/envconfig"
)

const composeFixture = `services:
  web:
    image: node:24-alpine
    command: ["node","-e","require('http').createServer((q,r)=>{r.setHeader('content-type','text/html; charset=utf-8');r.end('<html><meta charset=utf-8><title>Painel</title><h1>Painel pronto</h1></html>')}).listen(5173)"]
    ports: ["5180:5173"]
    depends_on: [api]
    healthcheck: {test: ["CMD","wget","-q","-O","-","http://localhost:5173"], interval: 2s, timeout: 3s, retries: 20}
  api:
    image: node:24-alpine
    command: ["node","-e","require('http').createServer((q,r)=>r.end('ok')).listen(8080)"]
    ports: ["8480:8080"]
    healthcheck: {test: ["CMD","wget","-q","-O","-","http://localhost:8080"], interval: 2s, timeout: 3s, retries: 20}
  db:
    image: postgres:17-alpine
    environment:
      POSTGRES_HOST_AUTH_METHOD: trust
    healthcheck: {test: ["CMD-SHELL","pg_isready -U postgres"], interval: 2s, timeout: 3s, retries: 20}
`

func composeRepo(t *testing.T, body string) (repo, sha string) {
	t.Helper()
	repo = t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "docker-compose.yml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "base")
	return repo, gitIn(t, repo, "rev-parse", "HEAD")
}

func realEnvConfig() envconfig.Config {
	c := goodEnv()
	c.DataSetup = ""
	return c
}

func dockerOrSkip(t *testing.T) {
	t.Helper()
	if out, err := exec.Command("docker", "compose", "version").CombinedOutput(); err != nil {
		t.Skipf("docker compose indisponível: %s", out)
	}
	for _, img := range []string{"node:24-alpine", "postgres:17-alpine", "mcr.microsoft.com/playwright/mcp:latest"} {
		if out, _ := exec.Command("docker", "images", "-q", img).Output(); strings.TrimSpace(string(out)) == "" {
			t.Skip("imagem ausente: " + img)
		}
	}
}

// AJ08-04 / CT08, CT09 — validação real com Docker Compose: sobe a composição do repositório na revisão em rede
// interna sem portas no host, espera os healthchecks, abre a aplicação no Chromium MCP real e comprova a limpeza.
// Uma versão com serviço que nunca fica saudável é invalidada e a composição ainda é derrubada.
func TestEnvConfigValidatesWithRealDockerCompose(t *testing.T) {
	dockerOrSkip(t)
	db := newMigratedDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	p := mustProject(t, db.Pool, "envcompose")
	st := &envconfig.Store{DB: db.Pool}
	repo, sha := composeRepo(t, composeFixture)
	var published string
	runner := &envconfig.ComposeRunner{Checkout: envconfig.GitCheckout(repo, t.TempDir()), BrowserImage: "mcr.microsoft.com/playwright/mcp:latest", Timeout: 90 * time.Second, ExecutionID: "tcompose1",
		AfterUp: func(project string) {
			out, _ := exec.Command("docker", "ps", "--filter", "label=com.docker.compose.project="+project, "--format", "{{.Ports}}").Output()
			published = strings.TrimSpace(string(out))
		}}
	t.Cleanup(func() { _ = exec.Command("docker", "compose", "-p", "lfenv-tcompose1", "down", "-v", "--remove-orphans", "--timeout", "2").Run() })

	v1, err := st.Save(ctx, p.ID, realEnvConfig(), sha)
	if err != nil {
		t.Fatal(err)
	}
	v1, err = st.Validate(ctx, p.ID, v1.Version, sha, runner)
	if err != nil || v1.State != envconfig.StateValidated || v1.Evidence == nil || !v1.Evidence.BrowserOK || !v1.Evidence.CleanupOK {
		t.Fatalf("%+v %v", v1, err)
	}
	for _, s := range []string{"web", "api", "db"} {
		if !v1.Evidence.Ready[s] {
			t.Fatalf("%s deveria ficar pronto: %+v", s, v1.Evidence)
		}
	}
	if strings.Contains(published, "->") {
		t.Fatalf("nenhuma porta pode ser publicada no host: %q", published)
	}
	if _, err := st.Approve(ctx, p.ID, v1.Version, "admin"); err != nil {
		t.Fatal(err)
	}
	if out, _ := exec.Command("docker", "ps", "-aq", "--filter", "label=lf.exec=tcompose1").Output(); strings.TrimSpace(string(out)) != "" {
		t.Fatalf("sobrou recurso: %s", out)
	}

	// Serviço que nunca fica saudável: inválida, não promove, e a composição é derrubada.
	broken := strings.Replace(composeFixture, `"wget","-q","-O","-","http://localhost:8080"`, `"CMD","false"`, 1)
	broken = strings.Replace(broken, `["CMD","CMD","false"]`, `["CMD","false"]`, 1)
	repo2, sha2 := composeRepo(t, broken)
	runner2 := *runner
	runner2.Checkout = envconfig.GitCheckout(repo2, t.TempDir())
	runner2.Timeout = 15 * time.Second
	runner2.ExecutionID = "tcompose2"
	t.Cleanup(func() { _ = exec.Command("docker", "compose", "-p", "lfenv-tcompose2", "down", "-v", "--remove-orphans", "--timeout", "2").Run() })
	c2 := realEnvConfig()
	c2.Start = "docker compose up -d --build"
	v2, err := st.Save(ctx, p.ID, c2, sha2)
	if err != nil {
		t.Fatal(err)
	}
	v2, err = st.Validate(ctx, p.ID, v2.Version, sha2, &runner2)
	if err != nil || v2.State != envconfig.StateInvalid || !strings.Contains(v2.Diagnostic, "api não ficou pronto") {
		t.Fatalf("%+v %v", v2, err)
	}
	if _, err := st.Approve(ctx, p.ID, v2.Version, "admin"); err == nil {
		t.Fatal("falha de readiness impede promover")
	}
	if out, _ := exec.Command("docker", "ps", "-aq", "--filter", "label=com.docker.compose.project=lfenv-tcompose2").Output(); strings.TrimSpace(string(out)) != "" {
		t.Fatalf("a composição com falha também deve ser derrubada: %s", out)
	}
	// Composição fora da política não chega a subir.
	bad := strings.Replace(composeFixture, "  db:", "  db:\n    privileged: true", 1)
	repo3, sha3 := composeRepo(t, bad)
	runner3 := *runner
	runner3.Checkout = envconfig.GitCheckout(repo3, t.TempDir())
	runner3.ExecutionID = "tcompose3"
	v3, _ := st.Save(ctx, p.ID, c2, sha3)
	v3, _ = st.Validate(ctx, p.ID, v3.Version, sha3, &runner3)
	if v3.State != envconfig.StateInvalid || !strings.Contains(v3.Diagnostic, "privileged") {
		t.Fatalf("%+v", v3)
	}
}
