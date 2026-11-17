package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/factory"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/skills"
)

// CLI de mentira: confere que a skill do agente chegou ao terminal, registra o comando recebido e edita um arquivo.
const factoryFakeCLI = `#!/bin/sh
cat > PROMPT.txt
if [ -f .claude/skills/lf-tech-lead/SKILL.md ]; then echo "skill_ok" >> PROMPT.txt; fi
if [ -f .claude/skills/lf-tech-lead/references/proposta-de-plano.md ]; then echo "references_ok" >> PROMPT.txt; fi
echo "feito pelo agente" >> README.md
echo '{"type":"system","subtype":"init","session_id":"sess-0001"}'
echo '{"type":"assistant","message":{"content":[{"type":"text","text":"editei o README"},{"type":"tool_use","name":"Edit"}]}}'
echo '{"type":"result","is_error":false,"result":"README atualizado","usage":{"input_tokens":12,"output_tokens":7}}'
`

// ADR-0069 — ponta a ponta com Docker real e um CLI de mentira: o agente roda num terminal isolado com a skill
// do agente e o comando da fábrica, as alterações voltam, os checks rodam, a fábrica commita, envia a branch
// e abre a Draft PR.
func TestFactoryRunsAWorkflowEndToEnd(t *testing.T) {
	d := &execution.Docker{}
	if !d.Available(context.Background()) {
		t.Skip("Docker indisponível")
	}
	if out, _ := exec.Command("docker", "images", "-q", "le-fabrique/worker:dev").Output(); strings.TrimSpace(string(out)) == "" {
		t.Skip("imagem le-fabrique/worker:dev ausente")
	}
	r := newRelEnv(t)
	db := r.e.db.Pool
	ctx := context.Background()

	// repositório do projeto + remoto "bare" que recebe o push
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
	bare := filepath.Join(root, "remote.git")
	gitIn(t, root, "init", "-q", "--bare", bare)

	ghConn := connWith(t, r.c, "github", "GitHub de ensaio", "acme")
	repo := projects.Repository{Mode: "local", Path: "proj", Owner: "acme", Name: "app", BaseBranch: "main", ConnectionID: &ghConn.ID}
	if _, err := projects.Update(ctx, db, r.proj, 1, projects.Patch{Repository: &repo}); err != nil {
		t.Fatalf("repositório do projeto: %v", err)
	}

	// GitHub de mentira: só a criação da PR
	var mu sync.Mutex
	var prBody map[string]any
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		if rq.Method == "POST" && strings.HasSuffix(rq.URL.Path, "/repos/acme/app/pulls") {
			mu.Lock()
			_ = json.NewDecoder(rq.Body).Decode(&prBody)
			mu.Unlock()
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"html_url":"https://github.example/acme/app/pull/7","number":7}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer gh.Close()

	// conta de IA ativa (o executor a escolhe sozinho) e o agente com a skill da fábrica
	acct := connWith(t, r.c, "cli", "Claude de ensaio", "claude")
	if err := d.EnsureVolume(ctx, "lf-acct-"+acct.ID); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.RemoveVolume(ctx, "lf-acct-"+acct.ID) }()
	b := agentBody("tech-lead", nil)
	b["model"], b["tools"], b["skills"] = "", []string{"read_file"}, []string{"lf-tech-lead"}
	b["instructions"] = "Planeje com rigor."
	r.c.expect(r.c.post("/api/agents", b), 201)
	if _, err := skills.InstallDefaults(ctx, db, policy.DefaultPolicy(policy.Options{})); err != nil {
		t.Fatal(err)
	}

	g := linear(
		wfNode("dev", "agent", 0, 0, map[string]any{"agent": "tech-lead", "prompt": "Planeje a tela de pedidos."}),
		wfNode("chk", "checks", 0, 0, map[string]any{"commands": "grep -q 'feito pelo agente' README.md\ntest -f PROMPT.txt", "profile": "go"}),
		wfNode("git1", "git", 0, 0, map[string]any{"operation": "commit", "message": "Planejamento da tela"}),
		wfNode("pr", "git", 0, 0, map[string]any{"operation": "draft_pr"}),
	)
	for _, id := range []string{"dev", "chk", "git1", "pr"} {
		g.Nodes = append(g.Nodes, wfNode("f-"+id, "end", 0, 0, map[string]any{"outcome": "failure"}))
		g.Edges = append(g.Edges, wfEdge("x-"+id, id, "error", "f-"+id, "in"))
	}
	w := r.draft(t, "fluxo-fabrica", "feature", g)
	r.c.expect(r.publish(w), 201)

	// imagem com sh, git e tar (a do worker): o terminal de ensaio e o perfil "go" dos checks
	sandboxProfile := execution.Profile{Name: "agent", Image: "le-fabrique/worker:dev", CPUs: 1, MemoryMB: 512, PidsLimit: 256, TmpMB: 64, WorkspaceMB: 256, Network: execution.NetNone}
	sealer, err := security.NewSealer(r.e.cfg.VaultKey, r.e.cfg.VaultKeyID)
	if err != nil {
		t.Fatal(err)
	}
	fx := &factory.Executor{DB: db, Cfg: factory.Config{
		Workspaces: root, Docker: d, AllowPush: true, Vault: security.NewVault(db, sealer, security.NewRedactor()), GitHubAPI: gh.URL,
		RemoteURL: func(string, string) string { return bare },
		Binary:    map[string]string{"claude": "/workspace/fakecli.sh"},
		Profiles:  map[string]execution.Profile{"agent": sandboxProfile, "go": {Name: "go", Image: "le-fabrique/worker:dev", CPUs: 1, MemoryMB: 512, PidsLimit: 256, TmpMB: 64, WorkspaceMB: 256, Network: execution.NetNone}},
	}}

	// o CLI de mentira e a skill precisam existir no workspace: o script entra no repositório do projeto
	if err := os.WriteFile(filepath.Join(src, "fakecli.sh"), []byte(factoryFakeCLI), 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, src, "add", "-A")
	gitIn(t, src, "commit", "-q", "-m", "cli de ensaio")

	run, err := runner.Start(ctx, db, runner.StartInput{ProjectID: r.proj, WorkflowID: w.ID, Title: "Tela de pedidos", Task: "Criar a tela de pedidos.", PRDDone: true})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := runner.Claim(ctx, db, "w", 5*time.Minute)
	if err != nil || claimed == nil {
		t.Fatalf("claim: %v %v", claimed, err)
	}
	run, err = runner.Advance(ctx, db, *claimed, runner.Options{Exec: fx, Lease: 5 * time.Minute})
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if run.Status != "completed" || run.Outcome != "success" {
		var evs []string
		rows, _ := db.Query(ctx, `SELECT payload::text FROM execution_events WHERE execution_id = $1 ORDER BY seq`, run.ExecutionID)
		for rows.Next() {
			var p string
			_ = rows.Scan(&p)
			evs = append(evs, p)
		}
		t.Fatalf("o fluxo deveria concluir com sucesso: status=%s outcome=%s falha=%q\n%s", run.Status, run.Outcome, run.Failure, strings.Join(evs, "\n"))
	}

	// o agente rodou no terminal com a skill e o comando da fábrica; as alterações voltaram ao repositório de trabalho
	work := filepath.Join(root, ".factory", run.ExecutionID, "repo")
	prompt, _ := os.ReadFile(filepath.Join(work, "PROMPT.txt"))
	for _, want := range []string{"agente \"tech-lead\"", "Use a skill `lf-tech-lead`", "Planeje a tela de pedidos.", "Criar a tela de pedidos.", "Planeje com rigor.", "skill_ok", "references_ok", "Não faça commit"} {
		if !strings.Contains(string(prompt), want) {
			t.Errorf("o comando do agente deveria conter %q:\n%s", want, prompt)
		}
	}
	// o commit e a branch da execução foram enviados ao remoto; as skills da fábrica não entraram no commit
	branch := "lf/" + strings.ReplaceAll(run.ExecutionID, "-", "")[:12]
	if got := gitIn(t, bare, "rev-parse", "--verify", branch); got == "" {
		t.Fatal("a branch não chegou ao remoto")
	}
	files := gitIn(t, bare, "ls-tree", "-r", "--name-only", branch)
	if strings.Contains(files, ".claude/skills") {
		t.Errorf("as skills da fábrica não podem entrar no commit:\n%s", files)
	}
	if !strings.Contains(gitIn(t, bare, "show", branch+":README.md"), "feito pelo agente") {
		t.Error("o commit deveria conter a edição do agente")
	}
	mu.Lock()
	defer mu.Unlock()
	if prBody["draft"] != true || prBody["head"] != branch || prBody["base"] != "main" || prBody["title"] != "Tela de pedidos" {
		t.Errorf("Draft PR: %v", prBody)
	}
	// o terminal ficou registrado como eventos da execução
	var lines int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM execution_events WHERE execution_id = $1 AND type = $2 AND payload->>'text' LIKE '%README%'`, run.ExecutionID, runner.EventNode).Scan(&lines)
	if lines == 0 {
		t.Error("a saída do terminal do agente deveria virar evento da execução")
	}
}
