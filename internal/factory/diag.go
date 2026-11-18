package factory

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
)

func (e *Executor) redact(s string) string {
	if e.Cfg.Redact != nil {
		return e.Cfg.Redact(s)
	}
	return s
}

func (e *Executor) logger() *slog.Logger {
	if e.Cfg.Log != nil {
		return e.Cfg.Log
	}
	return slog.Default()
}

// diag registra um evento de diagnóstico técnico da execução (nunca segredos: as strings passam pela redação).
// Vai para três lugares: o evento `run_diag` da execução (painel e relatório), o log estruturado do worker e o
// arquivo <workspaces>/.factory/<execução>/run.log (uma linha JSON por registro).
func (e *Executor) diag(info runner.Info, node, step string, fields map[string]any) {
	payload := map[string]any{"node": node, "step": step, "at": time.Now().UTC().Format(time.RFC3339Nano)}
	for k, v := range fields {
		if s, ok := v.(string); ok {
			v = e.redact(s)
		}
		payload[k] = v
	}
	info.Emit(runner.EventDiag, payload)
	args := []any{"execution", info.Run.ExecutionID, "node", node, "step", step}
	for k, v := range fields {
		if s, ok := v.(string); ok {
			v = e.redact(s)
		}
		args = append(args, k, v)
	}
	e.logger().Info("diagnóstico da fábrica", args...)
	e.appendRunLog(info.Run.ExecutionID, payload)
}

func (e *Executor) appendRunLog(executionID string, payload map[string]any) {
	if e.Cfg.Workspaces == "" {
		return
	}
	dir := filepath.Join(e.Cfg.Workspaces, ".factory", executionID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "run.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return
	}
	defer f.Close()
	b, _ := json.Marshal(payload)
	_, _ = f.Write(append(b, '\n'))
}

// timed devolve a função que fecha a medição: registra o passo com a duração e, se houver, o erro.
func (e *Executor) timed(info runner.Info, node, step string, fields map[string]any) func(extra map[string]any, err error) {
	start := time.Now()
	return func(extra map[string]any, err error) {
		f := map[string]any{"ms": time.Since(start).Milliseconds()}
		for k, v := range fields {
			f[k] = v
		}
		for k, v := range extra {
			f[k] = v
		}
		if err != nil {
			f["erro"] = err.Error()
			f["ok"] = false
		} else if _, set := f["ok"]; !set {
			f["ok"] = true
		}
		e.diag(info, node, step, f)
	}
}

// Preflight confere o ambiente antes do primeiro passo de uma execução e registra cada verificação como diagnóstico:
// assim, uma falha de ambiente (Docker, imagem, conta, repositório) aparece com a causa, não só como "nó falhou".
func (e *Executor) Preflight(ctx context.Context, run runner.Run) {
	var emit func(string, map[string]any)
	if info, ok := runner.InfoFrom(ctx); ok {
		emit = func(step string, f map[string]any) { e.diag(info, "", step, f) }
	} else {
		emit = func(step string, f map[string]any) {
			e.logger().Info("pré-voo da fábrica", "execution", run.ExecutionID, "step", step, "campos", f)
		}
	}
	check := func(name string, ok bool, detail string, hint string) {
		f := map[string]any{"check": name, "ok": ok, "detail": detail}
		if !ok && hint != "" {
			f["dica"] = hint
		}
		emit("preflight", f)
	}
	dockerOK := e.Cfg.Docker != nil && e.Cfg.Docker.Available(ctx)
	check("docker", dockerOK, boolText(dockerOK, "o Docker responde", "o Docker não responde"),
		"Monte /var/run/docker.sock no worker (só em desenvolvimento) e confirme que o Docker está rodando.")
	if dockerOK {
		images := map[string]string{}
		if e.Cfg.CLIImage != "" {
			images[e.Cfg.CLIImage] = "terminal do agente (make cli-image)"
		}
		for _, name := range []string{"base", "go", "node"} {
			if p, ok := e.profile(name); ok {
				images[p.Image] = "perfil " + name + " (scripts e checks)"
			}
		}
		for img, why := range images {
			out, err := exec.CommandContext(ctx, "docker", "image", "inspect", "--format", "{{.Id}}", img).Output()
			ok := err == nil && strings.TrimSpace(string(out)) != ""
			check("imagem "+img, ok, why+": "+boolText(ok, "presente", "ausente"),
				"Construa ou baixe a imagem (para o terminal do agente: make cli-image).")
		}
	}
	wsOK := e.Cfg.Workspaces != "" && writable(e.Cfg.Workspaces)
	check("pasta de workspaces", wsOK, e.Cfg.Workspaces, "Defina WORKSPACES_DIR no worker e monte a pasta com permissão de escrita.")
	_, gitErr := exec.LookPath("git")
	check("git no worker", gitErr == nil, boolText(gitErr == nil, "git encontrado", "git não encontrado"), "A imagem do worker precisa do git.")
	for _, prov := range []string{"claude", "codex"} {
		var n int
		_ = e.DB.QueryRow(ctx, `SELECT count(*) FROM connections WHERE kind = 'cli' AND target = $1 AND state = 'active'`, prov).Scan(&n)
		check("conta "+prov, n >= 1, plural(n, "conta ativa", "contas ativas"),
			"Conecte a conta em Conexões (botão de login oficial); sem conta o agente "+prov+" não roda.")
	}
	emit("preflight", map[string]any{"check": "configuração", "ok": true, "detail": "push_ao_github=" + boolText(e.Cfg.AllowPush, "ligado", "desligado") +
		", imagem_dos_checks=" + boolText(e.Cfg.ToolImage != "", e.Cfg.ToolImage, "padrão dos perfis") + ", modo=" + run.Kind})
}

func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".lf-write-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	return os.Remove(name) == nil
}

func boolText(b bool, yes, no string) string {
	if b {
		return yes
	}
	return no
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strings.TrimSpace(strings.Join([]string{itoa(n), many}, " "))
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

var _ = execution.NetNone
