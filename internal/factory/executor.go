package factory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/providers"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
	"github.com/ViniciusBernucci/le-fabrique/internal/workspace"
)

// Config reúne o que o executor precisa do ambiente.
type Config struct {
	Workspaces string // raiz de workspaces (WORKSPACES_DIR)
	CLIImage   string // imagem com claude e codex (make cli-image)
	// ToolImage, se informada, troca a imagem dos perfis de scripts e checks (base, go, node): as imagens
	// base são mínimas (sem make nem git) e o projeto precisa da própria toolchain para rodar make lint/test.
	ToolImage string
	// ChecksNetwork: "egress" dá internet aos scripts e checks (baixar dependências); vazio ou "none" os isola.
	ChecksNetwork string
	// TestModel é o modelo (Claude) dos agentes no botão Testar o fluxo quando o agente não fixa um (padrão claude-haiku-5-5).
	TestModel string
	Docker    *execution.Docker
	Vault     *security.Vault // push e Draft PR (token do GitHub)
	// AllowPush libera enviar a branch da execução ao GitHub e abrir a Draft PR (FACTORY_PUSH=true). Desligado,
	// os nós Git push/draft_pr saem pela porta de erro explicando como ligar: efeito externo é decisão da pessoa.
	AllowPush bool

	// Ajustes para ensaios; vazios em produção.
	GitHubAPI string                          // base da API do GitHub (padrão https://api.github.com)
	RemoteURL func(owner, name string) string // URL do remoto (padrão https://github.com/<dono>/<nome>.git)
	Binary    map[string]string               // provedor → executável (ensaios com um CLI de mentira)
	Profiles  map[string]execution.Profile    // sobrescreve perfis (ensaios)
	AgentWait time.Duration
	Log       *slog.Logger        // log estruturado (padrão: slog.Default)
	Redact    func(string) string // redação de segredos nos diagnósticos (padrão: nenhuma)                   // tempo máximo de um agente quando o cadastro não diz
}

// Executor implementa workflows.Executor para os nós de trabalho.
type Executor struct {
	DB  runner.DB
	Cfg Config

	// baselines guarda, por sandbox, o hash do patch cumulativo logo depois de semear o código: o que já veio das
	// etapas anteriores não conta como alteração da etapa atual.
	baselines sync.Map
}

// Run implementa workflows.Executor. Falhas esperadas (CLI recusou, comando falhou) saem pela porta
// "error" com o motivo; só erros de infraestrutura voltam como erro.
func (e *Executor) Run(ctx context.Context, n workflows.Node, vars map[string]any) (workflows.ExecResult, error) {
	info, ok := runner.InfoFrom(ctx)
	if !ok {
		return workflows.ExecResult{}, errors.New("o executor só roda dentro de uma execução")
	}
	info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "start", "text": "iniciando " + n.Type})
	e.diag(info, n.ID, "node.start", map[string]any{"type": n.Type, "agent": propStr(n, "agent"), "skill": propStr(n, "skill"),
		"operation": propStr(n, "operation"), "profile": propStr(n, "profile")})
	began := time.Now()
	var res workflows.ExecResult
	var err error
	switch n.Type {
	case workflows.TAgent:
		res, err = e.agent(ctx, info, n, vars)
	case workflows.TDocs:
		res, err = e.docs(ctx, info, n, vars)
	case workflows.TSkill:
		res, err = e.skill(ctx, info, n, vars)
	case workflows.TScript:
		res, err = e.commands(ctx, info, n, splitLines(propStr(n, "command")), true)
	case workflows.TChecks:
		cmds := splitLines(propStr(n, "commands"))
		if info.Run.Kind == runner.KindTest {
			// no teste os checks conferem a feature fictícia, qualquer que seja o projeto
			cmds = runner.TestChecks
			info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "note", "text": "modo teste: os checks conferem os arquivos da feature fictícia (em vez dos comandos do nó)"})
		}
		res, err = e.commands(ctx, info, n, cmds, false)
	case workflows.TGit:
		res, err = e.git(ctx, info, n)
	default:
		return fail(info, n, "o tipo de nó %q ainda não tem executor na fábrica", n.Type), nil
	}
	port := res.Port
	if port == "" {
		port = workflows.PortOut
	}
	f := map[string]any{"type": n.Type, "port": port, "ms": time.Since(began).Milliseconds()}
	if err != nil {
		f["erro_infra"] = err.Error()
	}
	if port == workflows.PortError {
		f["erro"] = fmt.Sprint(res.Outputs["erro"])
	}
	e.diag(info, n.ID, "node.end", f)
	return res, err
}

func propStr(n workflows.Node, k string) string {
	s, _ := n.Props[k].(string)
	return strings.TrimSpace(s)
}

func splitLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// fail devolve o nó pela porta de erro, com o motivo no terminal da execução e nas saídas.
func fail(info runner.Info, n workflows.Node, format string, a ...any) workflows.ExecResult {
	msg := fmt.Sprintf(format, a...)
	info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "error", "text": msg})
	return workflows.ExecResult{Port: workflows.PortError, Outputs: map[string]any{"ok": false, "erro": msg}}
}

func tail(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return "…" + s[len(s)-max:]
}

// repoFor prepara (ou reaproveita) a cópia de trabalho e registra o diagnóstico.
func (e *Executor) repoFor(ctx context.Context, info runner.Info, n workflows.Node) (string, projects.Project, error) {
	done := e.timed(info, n.ID, "repo.ready", map[string]any{"workspace": info.Run.Workspace, "kind": info.Run.Kind})
	repo, p, err := e.ensureRepo(ctx, info.Run)
	done(map[string]any{"path": repo, "branch": branchOf(info.Run.ExecutionID)}, err)
	return repo, p, err
}

// ---- repositório de trabalho da execução ----

func (e *Executor) repoDir(executionID string) string {
	return filepath.Join(e.Cfg.Workspaces, ".factory", executionID, "repo")
}

func branchOf(executionID string) string {
	return "lf/" + strings.ReplaceAll(executionID, "-", "")[:12]
}

func (e *Executor) git1(ctx context.Context, dir string, stdin []byte, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append([]string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.TempDir(), "GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_NAME=La fabrique", "GIT_AUTHOR_EMAIL=fabrica@lafabrique.local",
		"GIT_COMMITTER_NAME=La fabrique", "GIT_COMMITTER_EMAIL=fabrica@lafabrique.local"}, env...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("git %s: %s", args[0], tail(errb.String(), 600))
	}
	return out.String(), nil
}

// ensureRepo cria (uma vez por execução) a cópia de trabalho do repositório do projeto, numa branch própria.
func (e *Executor) ensureRepo(ctx context.Context, run runner.Run) (string, projects.Project, error) {
	p, err := projects.Get(ctx, e.DB, run.ProjectID)
	if err != nil {
		return "", p, err
	}
	dest := e.repoDir(run.ExecutionID)
	if workspace.IsRepo(dest) {
		return dest, p, nil
	}
	rel := strings.TrimSpace(run.Workspace)
	if rel == "" {
		rel = p.Repository.Path
	}
	var src string
	if rel != "" {
		if s, rerr := (workspace.Root{Dir: e.Cfg.Workspaces}).Resolve(rel); rerr == nil && workspace.IsRepo(s) {
			src = s
		} else if run.Kind != runner.KindTest {
			if rerr != nil {
				return "", p, fmt.Errorf("pasta do repositório inválida: %w", rerr)
			}
			return "", p, fmt.Errorf("a pasta %q não é um repositório git", rel)
		}
	}
	if src == "" {
		if run.Kind != runner.KindTest {
			return "", p, errors.New("o projeto não tem pasta de repositório: prepare o repositório em Setup")
		}
		return e.throwawayRepo(ctx, run, dest)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return "", p, err
	}
	if _, err := e.git1(ctx, filepath.Dir(dest), nil, nil, "clone", "--quiet", "--no-hardlinks", src, "repo"); err != nil {
		_ = os.RemoveAll(filepath.Dir(dest))
		return "", p, err
	}
	base := strings.TrimSpace(p.Repository.BaseBranch)
	if base != "" {
		_, _ = e.git1(ctx, dest, nil, nil, "checkout", "--quiet", base)
	}
	if _, err := e.git1(ctx, dest, nil, nil, "checkout", "--quiet", "-b", branchOf(run.ExecutionID)); err != nil {
		return "", p, err
	}
	return dest, p, nil
}

// throwawayRepo cria, só para o teste do fluxo, um repositório vazio com um commit inicial: o projeto pode não ter
// repositório preparado ainda e o teste precisa funcionar em qualquer projeto.
func (e *Executor) throwawayRepo(ctx context.Context, run runner.Run, dest string) (string, projects.Project, error) {
	p, err := projects.Get(ctx, e.DB, run.ProjectID)
	if err != nil {
		return "", p, err
	}
	if err := os.MkdirAll(dest, 0o750); err != nil {
		return "", p, err
	}
	if err := os.WriteFile(filepath.Join(dest, "README.md"), []byte("# Repositório descartável do teste da fábrica\n"), 0o640); err != nil {
		return "", p, err
	}
	for _, args := range [][]string{{"init", "--quiet", "-b", "main"}, {"add", "-A"}, {"commit", "--quiet", "-m", "Repositório descartável do teste"},
		{"checkout", "--quiet", "-b", branchOf(run.ExecutionID)}} {
		if _, err := e.git1(ctx, dest, nil, nil, args...); err != nil {
			_ = os.RemoveAll(filepath.Dir(dest))
			return "", p, err
		}
	}
	return dest, p, nil
}

// ---- sandbox, patch ----

func (e *Executor) profile(name string) (execution.Profile, bool) {
	if p, ok := e.Cfg.Profiles[name]; ok {
		return p, true
	}
	if name == "agent" {
		return execution.Profile{Name: "agent", UID: "10002", Image: e.Cfg.CLIImage, CPUs: 2, MemoryMB: 2048, PidsLimit: 512,
			TmpMB: 512, WorkspaceMB: 2048, Network: execution.NetEgress}, e.Cfg.CLIImage != ""
	}
	p, ok := execution.Profiles[name]
	if ok && e.Cfg.ToolImage != "" {
		p.Image = e.Cfg.ToolImage
	}
	if ok && e.Cfg.ChecksNetwork == "egress" {
		p.Network = execution.NetEgress
	}
	return p, ok
}

// withSandbox provisiona um sandbox, semeia o repositório de trabalho, roda fn e o destrói.
func (e *Executor) withSandbox(ctx context.Context, info runner.Info, node, profile string, mounts []execution.Mount, repo string,
	fn func(*execution.Sandbox) error) error {
	run := info.Run
	if e.Cfg.Docker == nil {
		return errors.New("o executor não tem acesso ao Docker (monte o docker.sock no worker)")
	}
	p, ok := e.profile(profile)
	if !ok {
		return fmt.Errorf("perfil de sandbox desconhecido: %q", profile)
	}
	provisioned := e.timed(info, node, "sandbox.provision", map[string]any{"profile": p.Name, "image": p.Image, "network": string(p.Network),
		"uid": p.UID, "cpus": p.CPUs, "memory_mb": p.MemoryMB, "mounts": len(mounts)})
	sb, err := e.Cfg.Docker.Provision(ctx, execution.Spec{ProjectID: run.ProjectID, FeatureID: strings.ReplaceAll(run.ExecutionID, "-", ""), Profile: p, Mounts: mounts})
	if err != nil {
		provisioned(nil, err)
		return fmt.Errorf("criar o terminal isolado: %w", err)
	}
	provisioned(map[string]any{"container": sb.Container}, nil)
	defer func() {
		destroyed := e.timed(info, node, "sandbox.destroy", map[string]any{"container": sb.Container})
		derr := sb.Destroy(context.WithoutCancel(ctx))
		destroyed(nil, derr)
	}()
	seeded := e.timed(info, node, "sandbox.seed", nil)
	seed, err := e.tarWorktree(ctx, repo)
	if err != nil {
		seeded(nil, err)
		return err
	}
	if err := sb.Seed(ctx, bytes.NewReader(seed)); err != nil {
		seeded(map[string]any{"bytes": len(seed)}, err)
		return fmt.Errorf("semear o código no terminal: %w", err)
	}
	seeded(map[string]any{"bytes": len(seed)}, nil)
	if base, err := e.readPatch(ctx, sb); err == nil {
		e.baselines.Store(sb.ID, patchHash(base))
		defer e.baselines.Delete(sb.ID)
	}
	return fn(sb)
}

// tarWorktree empacota a árvore de trabalho atual (inclusive o que ainda não foi commitado) mais o .git.
func (e *Executor) tarWorktree(ctx context.Context, repo string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "tar", "-c", "-C", repo, ".")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("empacotar o repositório: %s", tail(errb.String(), 300))
	}
	return out.Bytes(), nil
}

const maxPatch = 3 << 20

// readPatch lê do terminal o patch git binário (cumulativo desde o último commit) da árvore de trabalho.
func (e *Executor) readPatch(ctx context.Context, sb *execution.Sandbox) (string, error) {
	res, err := sb.Exec(ctx, execution.ExecSpec{Timeout: 2 * time.Minute, Env: map[string]string{"HOME": "/tmp"}, Cmd: []string{"sh", "-c",
		"git -c safe.directory=/workspace -c user.name=x -c user.email=x@x add -A >/dev/null 2>&1; " +
			"git -c safe.directory=/workspace diff --cached --binary HEAD"}})
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("recolher as alterações: %s", tail(res.Stderr, 300))
	}
	if len(res.Stdout) >= 4<<20 {
		return "", errors.New("as alterações do agente passam do limite de tamanho recolhível")
	}
	return res.Stdout, nil
}

func patchHash(p string) string {
	h := sha256.Sum256([]byte(p))
	return hex.EncodeToString(h[:])
}

// exportPatch lê do terminal as alterações feitas (patch git binário) e as aplica na cópia de trabalho.
func (e *Executor) exportPatch(ctx context.Context, info runner.Info, node string, sb *execution.Sandbox, repo string) (changed bool, err error) {
	done := e.timed(info, node, "patch.export", nil)
	patchBytes := 0
	defer func() { done(map[string]any{"patch_bytes": patchBytes, "changed": changed}, err) }()
	patch, err := e.readPatch(ctx, sb)
	if err != nil {
		return false, err
	}
	patchBytes = len(patch)
	if strings.TrimSpace(patch) == "" {
		return false, nil
	}
	// o patch é cumulativo (tudo desde o último commit): só conta como alteração desta etapa o que difere do que já
	// veio semeado. Sem diferença, a cópia de trabalho não é tocada.
	if base, ok := e.baselines.Load(sb.ID); ok && base == patchHash(patch) {
		return false, nil
	}
	if _, err := e.git1(ctx, repo, nil, nil, "checkout", "--quiet", "--", "."); err != nil {
		return false, err
	}
	if _, err := e.git1(ctx, repo, nil, nil, "clean", "-fdq"); err != nil {
		return false, err
	}
	if _, err := e.git1(ctx, repo, []byte(patch), nil, "apply", "--whitespace=nowarn", "-"); err != nil {
		return false, fmt.Errorf("aplicar as alterações do agente: %w", err)
	}
	return true, nil
}

// ---- agente ----

// resolveAccount escolhe a conta de IA: a do agente/nó ou, se nenhuma foi fixada, a única conta ativa do provedor.
func (e *Executor) resolveAccount(ctx context.Context, eff agents.Effective) (providers.Account, error) {
	cli := providers.CLI(eff.Provider)
	if cli != providers.Claude && cli != providers.Codex {
		return providers.Account{}, fmt.Errorf("o provedor %q ainda não é executado pela fábrica (só Claude e Codex em terminal)", eff.Provider)
	}
	if eff.ConnectionID != nil {
		var name string
		if err := e.DB.QueryRow(ctx, `SELECT name FROM connections WHERE id = $1 AND state = 'active'`, *eff.ConnectionID).Scan(&name); err != nil {
			return providers.Account{}, errors.New("a conta do agente não existe ou foi revogada")
		}
		return providers.Account{ID: *eff.ConnectionID, Name: name, CLI: cli}, nil
	}
	rows, err := e.DB.Query(ctx, `SELECT id::text, name FROM connections WHERE kind = 'cli' AND target = $1 AND state = 'active' ORDER BY created_at`, eff.Provider)
	if err != nil {
		return providers.Account{}, persistence.MapError(err)
	}
	defer rows.Close()
	var found []providers.Account
	for rows.Next() {
		a := providers.Account{CLI: cli}
		if err := rows.Scan(&a.ID, &a.Name); err != nil {
			return providers.Account{}, err
		}
		found = append(found, a)
	}
	switch len(found) {
	case 0:
		return providers.Account{}, fmt.Errorf("não há conta %s conectada: conecte em Conexões", eff.Provider)
	case 1:
		return found[0], nil
	}
	return providers.Account{}, fmt.Errorf("há %d contas %s ativas: escolha a conta no cadastro do agente", len(found), eff.Provider)
}

func skillDirFor(cli providers.CLI) string {
	if cli == providers.Codex {
		return ".agents/skills"
	}
	return ".claude/skills"
}

func (e *Executor) agent(ctx context.Context, info runner.Info, n workflows.Node, vars map[string]any) (workflows.ExecResult, error) {
	eff, err := agents.EffectiveForNode(ctx, e.DB, n)
	if err != nil {
		return fail(info, n, "configuração do agente: %v", err), nil
	}
	return e.runAgent(ctx, info, n, eff, vars)
}

// docs usa um agente genérico (sem skill) com o pedido de documentação do nó.
func (e *Executor) docs(ctx context.Context, info runner.Info, n workflows.Node, vars map[string]any) (workflows.ExecResult, error) {
	eff := agents.Effective{AgentName: "documentacao", Provider: providers.Claude.String(),
		Instructions: "Você atualiza a documentação do projeto com precisão, sem inventar o que não está no código.",
		Prompt:       fmt.Sprintf("Atualize %s.\n%s", propStr(n, "target"), propStr(n, "instruction")), TimeoutMinutes: 30}
	return e.runAgent(ctx, info, n, eff, vars)
}

func (e *Executor) runAgent(ctx context.Context, info runner.Info, n workflows.Node, eff agents.Effective, vars map[string]any) (workflows.ExecResult, error) {
	if info.Run.Kind == runner.KindTest && eff.Provider == "claude" && eff.Model == "" {
		// teste do fluxo não gasta a cota com o modelo mais caro da conta
		eff.Model = e.Cfg.TestModel
		if eff.Model == "" {
			eff.Model = "claude-haiku-5-5"
		}
	}
	acct, err := e.resolveAccount(ctx, eff)
	if err != nil {
		e.diag(info, n.ID, "account.error", map[string]any{"provider": eff.Provider, "erro": err.Error()})
		return fail(info, n, "%v", err), nil
	}
	e.diag(info, n.ID, "account.resolved", map[string]any{"provider": eff.Provider, "account": acct.Name, "account_id": acct.ID, "fixed": eff.ConnectionID != nil, "volume": acct.VolumeName()})
	repo, proj, err := e.repoFor(ctx, info, n)
	if err != nil {
		return fail(info, n, "%v", err), nil
	}
	sdir := skillDirFor(acct.CLI)
	planner := isPlanner(eff)
	extra := ""
	if planner {
		extra = plannerFormat
	} else {
		extra = e.planContext(ctx, info, n, eff, vars)
	}
	prompt := Prompt(eff, proj.Name, info.Run.Title, info.Run.Task, sdir, info.Run.Kind == runner.KindTest, extra)
	info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "agent", "agent": eff.AgentName, "provider": eff.Provider,
		"model": eff.Model, "account": acct.Name, "skills": eff.Skills, "timeout_minutes": eff.TimeoutMinutes})
	info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "prompt", "text": tail(prompt, 12000)})
	var skillTarBytes []byte
	if len(eff.Skills) > 0 {
		if skillTarBytes, err = skillTar(ctx, e.DB, append([]string{}, eff.Skills...), sdir); err != nil {
			return fail(info, n, "%v", err), nil
		}
	}
	timeout := time.Duration(eff.TimeoutMinutes) * time.Minute
	if timeout <= 0 {
		timeout = e.Cfg.AgentWait
	}
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	out := map[string]any{"ok": false}
	var planFail error
	var res providers.Result
	err = e.withSandbox(runCtx, info, n.ID, "agent", []execution.Mount{{Volume: acct.VolumeName(), Target: providers.AccountMount}}, repo, func(sb *execution.Sandbox) error {
		if len(skillTarBytes) > 0 {
			if err := sb.Seed(runCtx, bytes.NewReader(skillTarBytes)); err != nil {
				return fmt.Errorf("gravar as skills do agente: %w", err)
			}
		}
		// as skills e o canal da fábrica (.fabrica/) são da fábrica: ficam fora do patch
		ex := factoryDir + "/\n"
		for _, s := range eff.Skills {
			ex += sdir + "/" + s + "/\n"
		}
		if r, err := sb.Exec(runCtx, execution.ExecSpec{Cmd: []string{"sh", "-c", "mkdir -p .git/info && printf '%s' \"$1\" >> .git/info/exclude", "x", ex}}); err != nil || r.ExitCode != 0 {
			return errors.New("preparar a exclusão das skills e do canal da fábrica do patch")
		}
		req := providers.Request{Account: acct, Prompt: prompt, Mode: providers.ModeWorkspaceWrite, Model: eff.Model, Binary: e.Cfg.Binary[eff.Provider]}
		cliDone := e.timed(info, n.ID, "cli.run", nil)
		if cmd, cerr := providers.BuildCommand(req); cerr == nil {
			e.diag(info, n.ID, "cli.command", map[string]any{"argv": strings.Join(cmd.Args, " "), "env": envKeys(cmd.Env), "prompt_bytes": len(prompt), "timeout": timeout.String()})
		}
		res, err = providers.Run(runCtx, providers.SandboxRunner{S: sb}, req, func(ev providers.Event) {
			if ev.Kind == providers.EvUsage || ev.Text == "" && ev.Tool == "" {
				return
			}
			info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": ev.Kind, "text": tail(ev.Text, 1500), "tool": ev.Tool})
		})
		cliDone(map[string]any{"exit_code": res.ExitCode, "failed": res.Failed, "failure": res.Failure, "stderr": res.Stderr, "events": res.Events,
			"tokens_in": res.InputTokens, "tokens_out": res.OutputTokens, "session": res.Session, "cancelled": res.Cancelled,
			"final": tail(res.Final, 1500)}, err)
		if err != nil {
			return fmt.Errorf("terminal do agente: %w", err)
		}
		if planner && !res.Failed {
			// o plano estruturado do Tech Lead vira subtarefas da fábrica (ou a etapa falha dizendo por quê)
			if plan, perr := e.harvestPlan(context.WithoutCancel(runCtx), info, n, sb); perr != nil {
				planFail = perr
			} else {
				for k, v := range plan {
					out[k] = v
				}
			}
		}
		changed, perr := e.exportPatch(context.WithoutCancel(runCtx), info, n.ID, sb, repo)
		if perr != nil {
			return perr
		}
		out["alterou_arquivos"] = changed
		return nil
	})
	if err != nil {
		if runCtx.Err() != nil && ctx.Err() == nil {
			return fail(info, n, "o agente passou do tempo máximo (%s)", timeout), nil
		}
		if ctx.Err() != nil {
			return workflows.ExecResult{}, ctx.Err()
		}
		return fail(info, n, "%v", err), nil
	}
	if planFail != nil {
		return fail(info, n, "%v", planFail), nil
	}
	out["resumo"], out["sessao"] = tail(res.Final, 4000), res.Session
	out["tokens_entrada"], out["tokens_saida"] = res.InputTokens, res.OutputTokens
	if res.Failed {
		msg := res.Failure
		if msg == "" {
			msg = "o cliente terminou com falha"
		}
		out["ok"], out["erro"] = false, msg
		info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "error", "text": msg})
		return workflows.ExecResult{Port: workflows.PortError, Outputs: out}, nil
	}
	out["ok"] = true
	return workflows.ExecResult{Outputs: out}, nil
}

// ---- scripts e checks ----

func (e *Executor) commands(ctx context.Context, info runner.Info, n workflows.Node, cmds []string, keepChanges bool) (workflows.ExecResult, error) {
	if len(cmds) == 0 {
		return fail(info, n, "nenhum comando informado"), nil
	}
	repo, _, err := e.repoFor(ctx, info, n)
	if err != nil {
		return fail(info, n, "%v", err), nil
	}
	profile := propStr(n, "profile")
	timeout := 10 * time.Minute
	if v, ok := n.Props["timeout_minutes"].(float64); ok && v > 0 {
		timeout = time.Duration(v) * time.Minute
	}
	out := map[string]any{"ok": true, "codigo": 0, "saida": ""}
	var log strings.Builder
	err = e.withSandbox(ctx, info, n.ID, profile, nil, repo, func(sb *execution.Sandbox) error {
		for _, c := range cmds {
			info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "command", "text": "$ " + c})
			cmdDone := e.timed(info, n.ID, "command.run", map[string]any{"cmd": c})
			r, err := sb.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", c}, Timeout: timeout, Env: map[string]string{"HOME": "/tmp", "CI": "true"}})
			if err != nil {
				cmdDone(nil, err)
				return err
			}
			cmdDone(map[string]any{"exit_code": r.ExitCode, "timed_out": r.TimedOut, "stderr": tail(r.Stderr, 600), "ok": r.ExitCode == 0 && !r.TimedOut}, nil)
			text := strings.TrimSpace(r.Stdout + "\n" + r.Stderr)
			log.WriteString("$ " + c + "\n" + text + "\n")
			if text != "" {
				info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "output", "text": tail(text, 3000)})
			}
			if r.TimedOut || r.ExitCode != 0 {
				out["ok"], out["codigo"] = false, r.ExitCode
				if r.TimedOut {
					out["erro"] = "tempo máximo excedido em: " + c
				} else {
					out["erro"] = fmt.Sprintf("falhou (código %d): %s", r.ExitCode, c)
					if last := lastLine(r.Stderr + "\n" + r.Stdout); last != "" {
						out["erro"] = fmt.Sprint(out["erro"]) + " — " + last
					}
				}
				return nil
			}
		}
		if keepChanges {
			if _, err := e.exportPatch(context.WithoutCancel(ctx), info, n.ID, sb, repo); err != nil {
				return err
			}
		}
		return nil
	})
	out["saida"] = tail(log.String(), 4000)
	if err != nil {
		if ctx.Err() != nil {
			return workflows.ExecResult{}, ctx.Err()
		}
		return fail(info, n, "%v", err), nil
	}
	if ok, _ := out["ok"].(bool); !ok {
		info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "error", "text": fmt.Sprint(out["erro"])})
		return workflows.ExecResult{Port: workflows.PortError, Outputs: out}, nil
	}
	return workflows.ExecResult{Outputs: out}, nil
}

func envKeys(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

// lastLine devolve a última linha não vazia (a mensagem de erro costuma ser a última do comando).
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return tail(l, 240)
		}
	}
	return ""
}
