package factory

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	Docker     *execution.Docker
	Vault      *security.Vault // push e Draft PR (token do GitHub)
	// AllowPush libera enviar a branch da execução ao GitHub e abrir a Draft PR (FACTORY_PUSH=true). Desligado,
	// os nós Git push/draft_pr saem pela porta de erro explicando como ligar: efeito externo é decisão da pessoa.
	AllowPush bool

	// Ajustes para ensaios; vazios em produção.
	GitHubAPI string                          // base da API do GitHub (padrão https://api.github.com)
	RemoteURL func(owner, name string) string // URL do remoto (padrão https://github.com/<dono>/<nome>.git)
	Binary    map[string]string               // provedor → executável (ensaios com um CLI de mentira)
	Profiles  map[string]execution.Profile    // sobrescreve perfis (ensaios)
	AgentWait time.Duration                   // tempo máximo de um agente quando o cadastro não diz
}

// Executor implementa workflows.Executor para os nós de trabalho.
type Executor struct {
	DB  runner.DB
	Cfg Config
}

// Run implementa workflows.Executor. Falhas esperadas (CLI recusou, comando falhou) saem pela porta
// "error" com o motivo; só erros de infraestrutura voltam como erro.
func (e *Executor) Run(ctx context.Context, n workflows.Node, vars map[string]any) (workflows.ExecResult, error) {
	info, ok := runner.InfoFrom(ctx)
	if !ok {
		return workflows.ExecResult{}, errors.New("o executor só roda dentro de uma execução")
	}
	info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "start", "text": "iniciando " + n.Type})
	var res workflows.ExecResult
	var err error
	switch n.Type {
	case workflows.TAgent:
		res, err = e.agent(ctx, info, n)
	case workflows.TDocs:
		res, err = e.docs(ctx, info, n)
	case workflows.TScript:
		res, err = e.commands(ctx, info, n, splitLines(propStr(n, "command")), true)
	case workflows.TChecks:
		res, err = e.commands(ctx, info, n, splitLines(propStr(n, "commands")), false)
	case workflows.TGit:
		res, err = e.git(ctx, info, n)
	default:
		return fail(info, n, "o tipo de nó %q ainda não tem executor na fábrica", n.Type), nil
	}
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
	if rel == "" {
		return "", p, errors.New("o projeto não tem pasta de repositório: prepare o repositório em Setup")
	}
	src, err := workspace.Root{Dir: e.Cfg.Workspaces}.Resolve(rel)
	if err != nil {
		return "", p, fmt.Errorf("pasta do repositório inválida: %w", err)
	}
	if !workspace.IsRepo(src) {
		return "", p, fmt.Errorf("a pasta %q não é um repositório git", rel)
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
	return p, ok
}

// withSandbox provisiona um sandbox, semeia o repositório de trabalho, roda fn e o destrói.
func (e *Executor) withSandbox(ctx context.Context, run runner.Run, profile string, mounts []execution.Mount, repo string,
	fn func(*execution.Sandbox) error) error {
	if e.Cfg.Docker == nil {
		return errors.New("o executor não tem acesso ao Docker (monte o docker.sock no worker)")
	}
	p, ok := e.profile(profile)
	if !ok {
		return fmt.Errorf("perfil de sandbox desconhecido: %q", profile)
	}
	sb, err := e.Cfg.Docker.Provision(ctx, execution.Spec{ProjectID: run.ProjectID, FeatureID: strings.ReplaceAll(run.ExecutionID, "-", ""), Profile: p, Mounts: mounts})
	if err != nil {
		return fmt.Errorf("criar o terminal isolado: %w", err)
	}
	defer func() { _ = sb.Destroy(context.WithoutCancel(ctx)) }()
	seed, err := e.tarWorktree(ctx, repo)
	if err != nil {
		return err
	}
	if err := sb.Seed(ctx, bytes.NewReader(seed)); err != nil {
		return fmt.Errorf("semear o código no terminal: %w", err)
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

// exportPatch lê do terminal as alterações feitas (patch git binário) e as aplica na cópia de trabalho.
func (e *Executor) exportPatch(ctx context.Context, sb *execution.Sandbox, repo string) (changed bool, err error) {
	res, err := sb.Exec(ctx, execution.ExecSpec{Timeout: 2 * time.Minute, Env: map[string]string{"HOME": "/tmp"}, Cmd: []string{"sh", "-c",
		"git -c safe.directory=/workspace -c user.name=x -c user.email=x@x add -A >/dev/null 2>&1; " +
			"git -c safe.directory=/workspace diff --cached --binary HEAD"}})
	if err != nil {
		return false, err
	}
	if res.ExitCode != 0 {
		return false, fmt.Errorf("recolher as alterações: %s", tail(res.Stderr, 300))
	}
	if len(res.Stdout) >= 4<<20 {
		return false, errors.New("as alterações do agente passam do limite de tamanho recolhível")
	}
	if strings.TrimSpace(res.Stdout) == "" {
		return false, nil
	}
	if _, err := e.git1(ctx, repo, nil, nil, "checkout", "--quiet", "--", "."); err != nil {
		return false, err
	}
	if _, err := e.git1(ctx, repo, nil, nil, "clean", "-fdq"); err != nil {
		return false, err
	}
	if _, err := e.git1(ctx, repo, []byte(res.Stdout), nil, "apply", "--whitespace=nowarn", "-"); err != nil {
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

func (e *Executor) agent(ctx context.Context, info runner.Info, n workflows.Node) (workflows.ExecResult, error) {
	eff, err := agents.EffectiveForNode(ctx, e.DB, n)
	if err != nil {
		return fail(info, n, "configuração do agente: %v", err), nil
	}
	return e.runAgent(ctx, info, n, eff)
}

// docs usa um agente genérico (sem skill) com o pedido de documentação do nó.
func (e *Executor) docs(ctx context.Context, info runner.Info, n workflows.Node) (workflows.ExecResult, error) {
	eff := agents.Effective{AgentName: "documentacao", Provider: providers.Claude.String(),
		Instructions: "Você atualiza a documentação do projeto com precisão, sem inventar o que não está no código.",
		Prompt:       fmt.Sprintf("Atualize %s.\n%s", propStr(n, "target"), propStr(n, "instruction")), TimeoutMinutes: 30}
	return e.runAgent(ctx, info, n, eff)
}

func (e *Executor) runAgent(ctx context.Context, info runner.Info, n workflows.Node, eff agents.Effective) (workflows.ExecResult, error) {
	acct, err := e.resolveAccount(ctx, eff)
	if err != nil {
		return fail(info, n, "%v", err), nil
	}
	repo, proj, err := e.ensureRepo(ctx, info.Run)
	if err != nil {
		return fail(info, n, "%v", err), nil
	}
	sdir := skillDirFor(acct.CLI)
	prompt := Prompt(eff, proj.Name, info.Run.Title, info.Run.Task, sdir)
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
	var res providers.Result
	err = e.withSandbox(runCtx, info.Run, "agent", []execution.Mount{{Volume: acct.VolumeName(), Target: providers.AccountMount}}, repo, func(sb *execution.Sandbox) error {
		if len(skillTarBytes) > 0 {
			if err := sb.Seed(runCtx, bytes.NewReader(skillTarBytes)); err != nil {
				return fmt.Errorf("gravar as skills do agente: %w", err)
			}
			// as skills são da fábrica: ficam fora do patch
			ex := ""
			for _, s := range eff.Skills {
				ex += sdir + "/" + s + "/\n"
			}
			if r, err := sb.Exec(runCtx, execution.ExecSpec{Cmd: []string{"sh", "-c", "mkdir -p .git/info && printf '%s' \"$1\" >> .git/info/exclude", "x", ex}}); err != nil || r.ExitCode != 0 {
				return errors.New("preparar a exclusão das skills do patch")
			}
		}
		req := providers.Request{Account: acct, Prompt: prompt, Mode: providers.ModeWorkspaceWrite, Model: eff.Model, Binary: e.Cfg.Binary[eff.Provider]}
		res, err = providers.Run(runCtx, providers.SandboxRunner{S: sb}, req, func(ev providers.Event) {
			if ev.Kind == providers.EvUsage || ev.Text == "" && ev.Tool == "" {
				return
			}
			info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": ev.Kind, "text": tail(ev.Text, 1500), "tool": ev.Tool})
		})
		if err != nil {
			return fmt.Errorf("terminal do agente: %w", err)
		}
		changed, perr := e.exportPatch(context.WithoutCancel(runCtx), sb, repo)
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
	repo, _, err := e.ensureRepo(ctx, info.Run)
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
	err = e.withSandbox(ctx, info.Run, profile, nil, repo, func(sb *execution.Sandbox) error {
		for _, c := range cmds {
			info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "command", "text": "$ " + c})
			r, err := sb.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", c}, Timeout: timeout, Env: map[string]string{"HOME": "/tmp", "CI": "true"}})
			if err != nil {
				return err
			}
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
				}
				return nil
			}
		}
		if keepChanges {
			if _, err := e.exportPatch(context.WithoutCancel(ctx), sb, repo); err != nil {
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
