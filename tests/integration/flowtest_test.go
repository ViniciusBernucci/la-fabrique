package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/factory"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
	"github.com/ViniciusBernucci/le-fabrique/internal/skills"
	"github.com/ViniciusBernucci/le-fabrique/internal/stdflow"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// CLI de mentira do teste do fluxo: cada trilha cria só o arquivo que o comando da fábrica pede.
const flowTestFakeCLI = `#!/bin/sh
p=$(cat)
case "$p" in *"FORMATO DE SAÍDA OBRIGATÓRIO DO PLANO"*)
  mkdir -p .fabrica
  cat > .fabrica/plano.json <<'JSON'
{"summary":"Selo de saúde com uma trilha backend e uma frontend","tasks":[
 {"key":"backend-saude","title":"Endpoint de saúde","description":"Descrever GET /saude","track":"BACKEND","risk":"low","order":1,"depends_on":[],
  "paths":["fabrica-teste/backend/saude.md"],"criteria":[{"text":"o arquivo do endpoint existe","check":"file","path":"fabrica-teste/backend/saude.md"}]},
 {"key":"frontend-selo","title":"Selo de saúde","description":"Página com o selo","track":"FRONTEND","risk":"low","order":2,"depends_on":["backend-saude"],
  "paths":["fabrica-teste/frontend/selo-saude.html"],"criteria":[{"text":"a página do selo existe","check":"file","path":"fabrica-teste/frontend/selo-saude.html"}]}]}
JSON
  ;;
esac
case "$p" in *"subtarefas BACKEND"*)
  mkdir -p fabrica-teste/backend
  printf '# Endpoint de saúde\nGET /saude\n{"status":"ok"}\n' > fabrica-teste/backend/saude.md
  case "$p" in *"MODO TESTE DA FÁBRICA"*) echo modo_teste_ok >> fabrica-teste/backend/saude.md ;; esac ;;
esac
case "$p" in *"subtarefas FRONTEND"*)
  mkdir -p fabrica-teste/frontend
  printf '<p class="selo">Sistema saudável</p><!-- /saude -->\n' > fabrica-teste/frontend/selo-saude.html ;;
esac
echo '{"type":"system","subtype":"init","session_id":"sess-teste"}'
echo '{"type":"assistant","message":{"content":[{"type":"text","text":"feito"}]}}'
echo '{"type":"result","is_error":false,"result":"etapa concluída","usage":{"input_tokens":3,"output_tokens":2}}'
`

// flowTestImage constrói (uma vez) a imagem de ensaio: a do worker (sh, git, tar) com o CLI de mentira.
func flowTestImage(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fakecli"), []byte(flowTestFakeCLI), 0o755); err != nil {
		t.Fatal(err)
	}
	base := "le-fabrique/worker:dev"
	if out, _ := exec.Command("docker", "images", "-q", "le-fabrique/tools:dev").Output(); strings.TrimSpace(string(out)) != "" {
		base = "le-fabrique/tools:dev" // tem Node: o revisor visual sobe o app no terminal
	}
	df := "FROM " + base + "\nUSER root\nCOPY fakecli /usr/local/bin/fakecli\nRUN chmod 0755 /usr/local/bin/fakecli\n"
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(df), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("docker", "build", "-q", "-t", "lf-test-fakecli:latest", dir).CombinedOutput(); err != nil {
		t.Skipf("não consegui montar a imagem de ensaio: %s", out)
	}
	return "lf-test-fakecli:latest"
}

func flowTestExecutor(t *testing.T, r *relEnv, image string) *factory.Executor {
	prof := func(name string) execution.Profile {
		return execution.Profile{Name: name, Image: image, CPUs: 1, MemoryMB: 512, PidsLimit: 256, TmpMB: 64, WorkspaceMB: 256, Network: execution.NetNone}
	}
	return &factory.Executor{DB: r.e.db.Pool, Cfg: factory.Config{
		Workspaces: t.TempDir(), Docker: &execution.Docker{}, Binary: map[string]string{"claude": "/usr/local/bin/fakecli"},
		Profiles: map[string]execution.Profile{"agent": prof("agent"), "base": prof("base"), "go": prof("go"), "node": prof("node")},
	}}
}

func driveRun(t *testing.T, r *relEnv, fx *factory.Executor, run runner.Run) runner.Run {
	t.Helper()
	ctx := context.Background()
	claimed, err := runner.Claim(ctx, r.e.db.Pool, "w", 5*time.Minute)
	if err != nil || claimed == nil {
		t.Fatalf("claim: %v %v", claimed, err)
	}
	done, err := runner.Advance(ctx, r.e.db.Pool, *claimed, runner.Options{Exec: fx, Lease: 5 * time.Minute, Before: fx.Preflight})
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	return done
}

func dockerReady(t *testing.T) {
	t.Helper()
	if !(&execution.Docker{}).Available(context.Background()) {
		t.Skip("Docker indisponível")
	}
	if out, _ := exec.Command("docker", "images", "-q", "le-fabrique/worker:dev").Output(); strings.TrimSpace(string(out)) == "" {
		t.Skip("imagem le-fabrique/worker:dev ausente")
	}
}

// ADR-0069 — o botão Testar o fluxo roda a feature fictícia (backend + frontend) pelo fluxo padrão REAL, num projeto
// SEM repositório, e termina com sucesso: checks da feature, push e Draft PR só simulados.
func TestFlowTestRunsTheStandardFlowInAProjectWithoutRepository(t *testing.T) {
	dockerReady(t)
	r := newRelEnv(t)
	db := r.e.db.Pool
	ctx := context.Background()
	if _, err := skills.InstallDefaults(ctx, db, policy.DefaultPolicy(policy.Options{})); err != nil {
		t.Fatal(err)
	}
	if _, err := agents.InstallDefaults(ctx, db); err != nil {
		t.Fatal(err)
	}
	connWith(t, r.c, "cli", "Claude de ensaio", "claude") // a única conta ativa do provedor
	acctVol := ""
	_ = acctVol
	if _, _, err := stdflow.Provision(ctx, db, r.proj, "Projeto sem repositório"); err != nil {
		t.Fatal(err)
	}
	var run runner.Run
	r.c.expect(r.c.post("/api/projects/"+r.proj+"/runs/test", map[string]any{}), 201).JSON(&run)
	if run.Kind != "test" || run.Title != runner.TestFeatureTitle || !strings.Contains(run.Task, "FEATURE FICTÍCIA") {
		t.Fatalf("execução de teste: %+v", run)
	}
	// a conta precisa do volume (o executor o monta no terminal)
	var acct string
	_ = db.QueryRow(ctx, `SELECT id::text FROM connections WHERE kind = 'cli' AND target = 'claude' AND state = 'active'`).Scan(&acct)
	d := &execution.Docker{}
	if err := d.EnsureVolume(ctx, "lf-acct-"+acct); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.RemoveVolume(ctx, "lf-acct-"+acct) }()

	fx := flowTestExecutor(t, r, flowTestImage(t))
	done := driveRun(t, r, fx, run)
	if done.Status != "completed" || done.Outcome != "success" {
		var evs []string
		rows, _ := db.Query(ctx, `SELECT type || ' ' || payload::text FROM execution_events WHERE execution_id = $1 ORDER BY seq`, run.ExecutionID)
		for rows.Next() {
			var p string
			_ = rows.Scan(&p)
			evs = append(evs, p)
		}
		t.Fatalf("o teste do fluxo deveria concluir com sucesso: status=%s outcome=%s falha=%q\n%s", done.Status, done.Outcome, done.Failure, strings.Join(evs, "\n"))
	}
	// as duas trilhas criaram seus arquivos; o push e a PR foram simulados
	work := filepath.Join(fx.Cfg.Workspaces, ".factory", run.ExecutionID, "repo")
	for _, f := range []string{runner.TestBackendFile, runner.TestFrontendFile} {
		if _, err := os.Stat(filepath.Join(work, f)); err != nil {
			t.Errorf("a trilha deveria ter criado %s: %v", f, err)
		}
	}
	// diagnóstico: relatório com pré-voo, sandbox, CLI, patch e tempos; run.log com as mesmas linhas
	var diag struct {
		Report string `json:"report"`
		Hints  []any  `json:"hints"`
	}
	r.c.expect(r.c.get("/api/projects/"+r.proj+"/runs/"+run.ExecutionID+"/diagnostics"), 200).JSON(&diag)
	for _, want := range []string{"## Pré-voo do ambiente", "✔ docker", "sandbox.provision", "sandbox.seed", "cli.command", "cli.run", "patch.export", "repo.ready", "Nenhuma falha registrada."} {
		if !strings.Contains(diag.Report, want) {
			t.Errorf("o relatório deveria conter %q:\n%s", want, diag.Report)
		}
	}
	if logb, err := os.ReadFile(filepath.Join(fx.Cfg.Workspaces, ".factory", run.ExecutionID, "run.log")); err != nil || !strings.Contains(string(logb), `"step":"cli.run"`) {
		t.Errorf("run.log deveria ter as linhas de diagnóstico: %v", err)
	}
	// só as etapas que de fato editaram arquivos contam como "alterou": o patch é cumulativo e as revisões não mudam nada
	changedBy := map[string]string{}
	crows, _ := db.Query(ctx, `SELECT payload->>'node', payload->>'changed' FROM execution_events
		WHERE execution_id = $1 AND type = 'run_diag' AND payload->>'step' = 'patch.export'`, run.ExecutionID)
	for crows.Next() {
		var n, c string
		_ = crows.Scan(&n, &c)
		changedBy[n] = c
	}
	crows.Close()
	for node, want := range map[string]string{"guard_planning": "false", "guard_impl_front": "true", "guard_impl_back": "true", "guard_review_front": "false", "guard_review_back": "false"} {
		if changedBy[node] != want {
			t.Errorf("patch.export de %s: changed=%q, esperado %q (todos: %v)", node, changedBy[node], want, changedBy)
		}
	}
	// plano estruturado: aceito, materializado e entregue por trilha
	var planVer, subtasks int
	_ = db.QueryRow(ctx, `SELECT COALESCE(max(version),0) FROM plans WHERE feature_id = (SELECT feature_id FROM executions WHERE id = $1) AND status = 'current'`, run.ExecutionID).Scan(&planVer)
	_ = db.QueryRow(ctx, `SELECT count(*) FROM subtasks WHERE feature_id = (SELECT feature_id FROM executions WHERE id = $1)`, run.ExecutionID).Scan(&subtasks)
	if planVer != 1 || subtasks != 2 {
		t.Errorf("o plano deveria estar aceito (v1) com 2 subtarefas: versão=%d subtarefas=%d", planVer, subtasks)
	}
	promptOf := func(node string) string {
		var txt string
		_ = db.QueryRow(ctx, `SELECT payload->>'text' FROM execution_events WHERE execution_id = $1 AND type = 'node_event' AND payload->>'node' = $2 AND payload->>'kind' = 'prompt'`, run.ExecutionID, node).Scan(&txt)
		return txt
	}
	if p := promptOf("guard_planning"); !strings.Contains(p, "FORMATO DE SAÍDA OBRIGATÓRIO DO PLANO") {
		t.Error("o Tech Lead deveria receber o formato do plano")
	}
	back, front := promptOf("guard_impl_back"), promptOf("guard_impl_front")
	if !strings.Contains(back, "Subtarefa `backend-saude`") || strings.Contains(back, "Subtarefa `frontend-selo`") {
		t.Errorf("o implementador backend deveria receber só a subtarefa BACKEND:\n%s", back)
	}
	if !strings.Contains(front, "Subtarefa `frontend-selo`") || strings.Contains(front, "Subtarefa `backend-saude`") {
		t.Errorf("o implementador frontend deveria receber só a subtarefa FRONTEND:\n%s", front)
	}
	if !strings.Contains(promptOf("guard_review_back"), "O que as etapas anteriores relataram") {
		t.Error("o revisor deveria receber o que as etapas anteriores relataram")
	}
	if files, _ := os.ReadFile(filepath.Join(work, ".fabrica", "plano.json")); len(files) != 0 {
		t.Error("o canal da fábrica não deveria ir para a cópia de trabalho")
	}
	var simulated, prompts, configs int
	_ = db.QueryRow(ctx, `SELECT count(*) FILTER (WHERE payload->>'text' LIKE '%SIMULADO%'),
		count(*) FILTER (WHERE payload->>'kind' = 'prompt' AND payload->>'text' LIKE '%MODO TESTE DA FÁBRICA%'),
		count(*) FILTER (WHERE payload->>'kind' = 'agent')
		FROM execution_events WHERE execution_id = $1 AND type = 'node_event'`, run.ExecutionID).Scan(&simulated, &prompts, &configs)
	if simulated != 1 || prompts != 5 || configs != 5 {
		t.Errorf("eventos: simulado=%d prompts=%d agentes=%d (esperado 1, 5, 5)", simulated, prompts, configs)
	}
}

// ADR-0069 — num fluxo QUALQUER (um workflow próprio, fora do projeto) a falha aparece com o nó e o motivo.
func TestFlowTestReportsWhereAnyFlowFails(t *testing.T) {
	dockerReady(t)
	r := newRelEnv(t)
	db := r.e.db.Pool
	ctx := context.Background()
	connWith(t, r.c, "cli", "Claude de ensaio", "claude")
	ab := agentBody("dev", nil)
	ab["model"], ab["tools"], ab["skills"] = "", []string{"read_file"}, []string{}
	r.c.expect(r.c.post("/api/agents", ab), 201)
	g := linear(
		wfNode("dev", "agent", 0, 0, map[string]any{"agent": "dev", "prompt": "Faça o que o papel pede."}),
		wfNode("chk", "checks", 0, 0, map[string]any{"commands": "true", "profile": "base"}),
	)
	g.Nodes = append(g.Nodes, wfNode("f-chk", "end", 0, 0, map[string]any{"outcome": "failure"}), wfNode("f-dev", "end", 0, 0, map[string]any{"outcome": "failure"}))
	g.Edges = append(g.Edges, wfEdge("x1", "chk", "error", "f-chk", "in"), wfEdge("x2", "dev", "error", "f-dev", "in"))
	w := r.draft(t, "fluxo-qualquer", "feature", g)

	var run runner.Run
	r.c.expect(r.c.post("/api/projects/"+r.proj+"/runs/test", map[string]any{"workflow_id": w.ID}), 201).JSON(&run)

	var acct string
	_ = db.QueryRow(ctx, `SELECT id::text FROM connections WHERE kind = 'cli' AND target = 'claude' AND state = 'active'`).Scan(&acct)
	d := &execution.Docker{}
	if err := d.EnsureVolume(ctx, "lf-acct-"+acct); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.RemoveVolume(ctx, "lf-acct-"+acct) }()

	fx := flowTestExecutor(t, r, flowTestImage(t))
	done := driveRun(t, r, fx, run)
	// o agente "dev" não é de backend nem de frontend: não cria arquivos; os checks do teste, então, reprovam
	if done.Status != "completed" || done.Outcome != "failure" {
		t.Fatalf("deveria terminar em falha: %+v", done)
	}
	var where, why string
	_ = db.QueryRow(ctx, `SELECT payload->>'node', payload->>'text' FROM execution_events
		WHERE execution_id = $1 AND type = 'node_event' AND payload->>'kind' = 'error' ORDER BY seq DESC LIMIT 1`, run.ExecutionID).Scan(&where, &why)
	if where != "chk" || !strings.Contains(why, "falhou") {
		t.Fatalf("a falha deveria apontar o nó chk e o comando: nó=%q motivo=%q", where, why)
	}
	var diag struct {
		Report  string                        `json:"report"`
		Hints   []struct{ Cause, Fix string } `json:"hints"`
		Failure struct{ Node, Reason string } `json:"failure"`
	}
	r.c.expect(r.c.get("/api/projects/"+r.proj+"/runs/"+run.ExecutionID+"/diagnostics"), 200).JSON(&diag)
	if diag.Failure.Node != "chk" || len(diag.Hints) == 0 || !strings.Contains(diag.Report, "## Causas prováveis e o que fazer") || !strings.Contains(diag.Report, "command.run") {
		t.Errorf("diagnóstico da falha: %+v\n%s", diag.Failure, diag.Report)
	}
	// um subfluxo não tem execução própria
	sub := r.draft(t, "sub", "subflow", workflows.Starter())
	r.c.expect(r.c.post("/api/projects/"+r.proj+"/runs/test", map[string]any{"workflow_id": sub.ID}), 409)
}
