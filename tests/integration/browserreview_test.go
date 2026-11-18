package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/factory"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
	"github.com/ViniciusBernucci/le-fabrique/internal/skills"
)

// CLI de mentira do revisor: confere que o MCP do navegador chegou nos argumentos, que o app sobe no terminal e responde em
// localhost, e deixa uma "captura" em .fabrica/evidencias (a fábrica a copia para o host).
const browserFakeCLI = `#!/bin/sh
cat > PROMPT.txt
case "$*" in *"--mcp-config"*"chrome-devtools"*"--strict-mcp-config"*) mcp=sim ;; *) mcp=nao ;; esac
case "$*" in *"mcp__chrome-devtools"*) tools=sim ;; *) tools=nao ;; esac
status=$(node -e 'fetch("http://localhost:4173/").then(r=>console.log(r.status)).catch(e=>console.log("erro "+e.message))')
mkdir -p .fabrica/evidencias
printf 'mcp=%s tools=%s app=%s\n' "$mcp" "$tools" "$status" > .fabrica/evidencias/resultado.txt
echo '{"type":"system","subtype":"init","session_id":"sess-nav"}'
echo '{"type":"assistant","message":{"content":[{"type":"text","text":"naveguei"}]}}'
echo '{"type":"result","is_error":false,"result":"revisão visual feita","usage":{"input_tokens":1,"output_tokens":1}}'
`

// ADR-0069 — a revisão visual: o agente com a skill de revisão do Frontend recebe o MCP do navegador, o app sobe no terminal
// e responde em localhost, e as evidências voltam para o host.
func TestBrowserReviewStartsTheAppGivesTheMCPAndCollectsEvidence(t *testing.T) {
	dockerReady(t)
	if out, _ := exec.Command("docker", "images", "-q", "le-fabrique/tools:dev").Output(); strings.TrimSpace(string(out)) == "" {
		t.Skip("imagem le-fabrique/tools:dev ausente (make tools-image)")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fakecli"), []byte(browserFakeCLI), 0o755); err != nil {
		t.Fatal(err)
	}
	df := "FROM le-fabrique/tools:dev\nUSER root\nCOPY fakecli /usr/local/bin/fakecli\nRUN chmod 0755 /usr/local/bin/fakecli\n"
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(df), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("docker", "build", "-q", "-t", "lf-test-fakecli-node:latest", dir).CombinedOutput(); err != nil {
		t.Skipf("não consegui montar a imagem de ensaio: %s", out)
	}

	r := newRelEnv(t)
	db := r.e.db.Pool
	ctx := context.Background()
	if _, err := skills.InstallDefaults(ctx, db, policy.DefaultPolicy(policy.Options{})); err != nil {
		t.Fatal(err)
	}
	if _, err := agents.InstallDefaults(ctx, db); err != nil {
		t.Fatal(err)
	}
	connWith(t, r.c, "cli", "Claude de ensaio", "claude")
	g := linear(wfNode("rev", "agent", 0, 0, map[string]any{"agent": "revisor-frontend", "prompt": "Revisar visualmente a interface."}))
	g.Nodes = append(g.Nodes, wfNode("f", "end", 0, 0, map[string]any{"outcome": "failure"}))
	g.Edges = append(g.Edges, wfEdge("x", "rev", "error", "f", "in"))
	w := r.draft(t, "fluxo-revisao", "feature", g)
	var run runner.Run
	r.c.expect(r.c.post("/api/projects/"+r.proj+"/runs/test", map[string]any{"workflow_id": w.ID}), 201).JSON(&run)

	var acct string
	_ = db.QueryRow(ctx, `SELECT id::text FROM connections WHERE kind = 'cli' AND target = 'claude' AND state = 'active'`).Scan(&acct)
	d := &execution.Docker{}
	if err := d.EnsureVolume(ctx, "lf-acct-"+acct); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.RemoveVolume(ctx, "lf-acct-"+acct) }()

	prof := execution.Profile{Name: "agent", Image: "lf-test-fakecli-node:latest", CPUs: 1, MemoryMB: 512, PidsLimit: 256, TmpMB: 64, WorkspaceMB: 256, Network: execution.NetNone}
	fx := &factory.Executor{DB: db, Cfg: factory.Config{Workspaces: t.TempDir(), Docker: d, Binary: map[string]string{"claude": "/usr/local/bin/fakecli"},
		Profiles: map[string]execution.Profile{"agent": prof}}}
	done := driveRun(t, r, fx, run)
	if done.Status != "completed" || done.Outcome != "success" {
		t.Fatalf("a revisão visual deveria concluir: %+v", done)
	}
	// as evidências voltaram para o host
	b, err := os.ReadFile(filepath.Join(fx.Cfg.Workspaces, ".factory", run.ExecutionID, "evidencias", "rev", "resultado.txt"))
	if err != nil {
		t.Fatalf("as evidências deveriam estar no host: %v", err)
	}
	if got := strings.TrimSpace(string(b)); got != "mcp=sim tools=sim app=404" && got != "mcp=sim tools=sim app=200" {
		t.Errorf("o MCP e as ferramentas deveriam chegar ao CLI e o app responder em localhost: %q", got)
	}
	var prompt, started string
	_ = db.QueryRow(ctx, `SELECT payload->>'text' FROM execution_events WHERE execution_id = $1 AND payload->>'kind' = 'prompt'`, run.ExecutionID).Scan(&prompt)
	if !strings.Contains(prompt, "Navegador isolado (MCP)") || !strings.Contains(prompt, "http://localhost:4173/") {
		t.Errorf("o comando deveria ensinar o uso do navegador:\n%s", prompt)
	}
	_ = db.QueryRow(ctx, `SELECT payload->>'ok' FROM execution_events WHERE execution_id = $1 AND payload->>'step' = 'app.start'`, run.ExecutionID).Scan(&started)
	if started != "true" {
		t.Errorf("o diagnóstico deveria registrar o app no ar: %q", started)
	}
}

// Sem ambiente de teste aprovado (execução real), a revisão visual NÃO derruba o fluxo: segue sem navegador, avisa o
// agente e deixa os cenários visuais PENDENTES.
func TestBrowserReviewWithoutAnApprovedEnvironmentIsPendingNotFailing(t *testing.T) {
	dockerReady(t)
	if out, _ := exec.Command("docker", "images", "-q", "lf-test-fakecli-node:latest").Output(); strings.TrimSpace(string(out)) == "" {
		t.Skip("imagem de ensaio ausente (o teste da revisão visual a constrói)")
	}
	r := newRelEnv(t)
	db := r.e.db.Pool
	ctx := context.Background()
	if _, err := skills.InstallDefaults(ctx, db, policy.DefaultPolicy(policy.Options{})); err != nil {
		t.Fatal(err)
	}
	if _, err := agents.InstallDefaults(ctx, db); err != nil {
		t.Fatal(err)
	}
	connWith(t, r.c, "cli", "Claude de ensaio", "claude")
	root := t.TempDir()
	src := filepath.Join(root, "proj")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, src, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(src, "README.md"), []byte("projeto\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, src, "add", "-A")
	gitIn(t, src, "commit", "-q", "-m", "inicial")
	repo := projects.Repository{Mode: "local", Path: "proj", Owner: "acme", Name: "app", BaseBranch: "main"}
	if _, err := projects.Update(ctx, db, r.proj, 1, projects.Patch{Repository: &repo}); err != nil {
		t.Fatal(err)
	}
	g := linear(wfNode("rev", "agent", 0, 0, map[string]any{"agent": "revisor-frontend", "prompt": "Revisar visualmente a interface."}))
	g.Nodes = append(g.Nodes, wfNode("f", "end", 0, 0, map[string]any{"outcome": "failure"}))
	g.Edges = append(g.Edges, wfEdge("x", "rev", "error", "f", "in"))
	w := r.draft(t, "fluxo-revisao-sem-ambiente", "feature", g)
	r.c.expect(r.publish(w), 201)
	run, err := runner.Start(ctx, db, runner.StartInput{ProjectID: r.proj, WorkflowID: w.ID, Title: "Revisão sem ambiente", Task: "x", Workspace: "proj"})
	if err != nil {
		t.Fatal(err)
	}

	var acct string
	_ = db.QueryRow(ctx, `SELECT id::text FROM connections WHERE kind = 'cli' AND target = 'claude' AND state = 'active'`).Scan(&acct)
	d := &execution.Docker{}
	if err := d.EnsureVolume(ctx, "lf-acct-"+acct); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.RemoveVolume(ctx, "lf-acct-"+acct) }()
	prof := execution.Profile{Name: "agent", Image: "lf-test-fakecli-node:latest", CPUs: 1, MemoryMB: 512, PidsLimit: 256, TmpMB: 64, WorkspaceMB: 256, Network: execution.NetNone}
	fx := &factory.Executor{DB: db, Cfg: factory.Config{Workspaces: root, Docker: d, Binary: map[string]string{"claude": "/usr/local/bin/fakecli"},
		Profiles: map[string]execution.Profile{"agent": prof}}}
	done := driveRun(t, r, fx, run)
	if done.Status != "completed" || done.Outcome != "success" {
		t.Fatalf("sem ambiente aprovado a revisão deveria seguir (pendente), não falhar: %+v", done)
	}
	var skipped int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM execution_events WHERE execution_id = $1 AND payload->>'step' = 'app.skipped'`, run.ExecutionID).Scan(&skipped)
	var prompt string
	_ = db.QueryRow(ctx, `SELECT payload->>'text' FROM execution_events WHERE execution_id = $1 AND payload->>'kind' = 'prompt'`, run.ExecutionID).Scan(&prompt)
	if skipped != 1 || !strings.Contains(prompt, "Sem navegador nesta revisão") || strings.Contains(prompt, "Navegador isolado (MCP)") {
		t.Errorf("o aviso de revisão sem navegador deveria ir ao agente (skipped=%d):\n%s", skipped, prompt)
	}
	if got := done.State.Vars["rev"]; got == nil {
		t.Fatalf("saídas do nó ausentes: %v", done.State.Vars)
	} else if m, _ := got.(map[string]any); m["revisao_visual"] != "pendente" {
		t.Errorf("a saída deveria marcar revisao_visual=pendente: %v", m)
	}
}
