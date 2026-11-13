package envconfig

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/browserenv"
)

// ComposeRunner valida a configuração subindo a composição do repositório, na revisão analisada, num
// ambiente isolado: a política recusa o que escapa (privileged, rede/pid do host, docker.sock, bind de
// caminho absoluto ou fora do repositório, container_name), um override tira as portas publicadas, põe a rede
// do projeto como interna e rotula tudo por execução; a prontidão vem dos healthchecks (ou do comando de
// prontidão da configuração); o navegador MCP real acessa o serviço-alvo; e o fim sempre derruba a composição
// com volumes e recolhe o navegador, comprovando a limpeza por rótulo.
type ComposeRunner struct {
	// Checkout materializa a revisão (diretório isolado + limpeza). Use GitCheckout.
	Checkout func(ctx context.Context, sha string) (dir string, cleanup func(), err error)
	// ComposeFile é o arquivo de composição no repositório (padrão docker-compose.yml).
	ComposeFile  string
	BrowserImage string // imagem do Chromium com servidor MCP
	DockerBin    string
	Timeout      time.Duration // espera pela prontidão (padrão 120 s)
	ExecutionID  string        // identifica os recursos (padrão derivado do SHA)
	// AfterUp é chamado com a composição de pé (inspeção em testes).
	AfterUp func(project string)
}

var composeBad = []struct {
	re  *regexp.Regexp
	msg string
}{
	{regexp.MustCompile(`(?m)^\s*privileged:\s*true`), "privileged"},
	{regexp.MustCompile(`(?m)^\s*network_mode:\s*["']?(host|container:)`), "network_mode do host/de outro container"},
	{regexp.MustCompile(`(?m)^\s*(pid|ipc|userns_mode|uts):\s*["']?host`), "namespace do host"},
	{regexp.MustCompile(`docker\.sock`), "docker.sock"},
	{regexp.MustCompile(`(?m)^\s*container_name:`), "container_name (nomes fixos colidem entre execuções)"},
	{regexp.MustCompile(`(?m)^\s*cap_add:`), "cap_add"},
	{regexp.MustCompile(`(?m)^\s*-\s*["']?(/|~|\.\./)[^:]*:`), "bind de caminho absoluto ou fora do repositório"},
	{regexp.MustCompile(`(?m)^\s*(devices|extra_hosts):`), "devices/extra_hosts"},
}

// CheckCompose aplica a política ao texto da composição (conteúdo do repositório é dado até a validação).
func CheckCompose(content string) []string {
	var out []string
	for _, b := range composeBad {
		if b.re.MatchString(content) {
			out = append(out, "composição fora da política: "+b.msg)
		}
	}
	return out
}

// Override gera o arquivo que isola a composição: sem portas no host, rede interna e rótulos por execução.
func Override(services []string, exec, project string) string {
	var b strings.Builder
	b.WriteString("services:\n")
	for _, s := range services {
		fmt.Fprintf(&b, "  %s:\n    ports: !reset []\n    labels:\n      lf.managed: \"true\"\n      lf.kind: env\n      lf.exec: %q\n      lf.project: %q\n", s, exec, project)
	}
	b.WriteString("networks:\n  default:\n    internal: true\n")
	return b.String()
}

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,40}$`)

func (r *ComposeRunner) bin() string {
	if r.DockerBin != "" {
		return r.DockerBin
	}
	return "docker"
}

func (r *ComposeRunner) docker(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, r.bin(), args...)
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		m := strings.TrimSpace(errb.String())
		if len(m) > 400 {
			m = m[len(m)-400:]
		}
		return out.String(), fmt.Errorf("docker %s: %v: %s", args[0], err, m)
	}
	return strings.TrimSpace(out.String()), nil
}

// Run implementa Runner. Devolve evidência mesmo em falha parcial (o que não ficou pronto aparece como falso).
func (r *ComposeRunner) Run(ctx context.Context, c Config, sha string) (ev Evidence, rerr error) {
	ev = Evidence{SHA: sha, Ready: map[string]bool{}}
	id := r.ExecutionID
	if id == "" {
		id = "env-" + strings.ToLower(sha[:min(8, len(sha))])
	}
	project := "lfenv-" + id
	if !nameRe.MatchString(project) {
		return ev, errors.New("identificação da execução inválida")
	}
	dir, cleanup, err := r.Checkout(ctx, sha)
	if err != nil {
		return ev, fmt.Errorf("materializar a revisão %s: %w", sha, err)
	}
	defer cleanup()
	file := r.ComposeFile
	if file == "" {
		file = "docker-compose.yml"
	}
	raw, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return ev, fmt.Errorf("a revisão não tem %s: %w", file, err)
	}
	if bad := CheckCompose(string(raw)); len(bad) > 0 {
		return ev, errors.New(strings.Join(bad, "; "))
	}
	var names []string
	for _, s := range c.Services {
		names = append(names, s.Name)
	}
	overridePath := filepath.Join(dir, ".lf-override.yml")
	if err := os.WriteFile(overridePath, []byte(Override(names, id, project)), 0o600); err != nil {
		return ev, err
	}
	base := []string{"compose", "-p", project, "-f", file, "-f", overridePath}
	rt := browserenv.DockerCLI{Bin: r.bin()}
	// A limpeza é incondicional e comprovada por rótulo/projeto.
	defer func() {
		cctx := context.WithoutCancel(ctx)
		_, _ = r.docker(cctx, dir, append(base, "down", "-v", "--remove-orphans", "--timeout", "5")...)
		_ = browserenv.CleanupExecution(cctx, rt, id)
		left1, _ := r.docker(cctx, dir, "ps", "-aq", "--filter", "label=com.docker.compose.project="+project)
		left2, _ := r.docker(cctx, dir, "ps", "-aq", "--filter", "label=lf.exec="+id)
		left3, _ := r.docker(cctx, dir, "network", "ls", "-q", "--filter", "label=com.docker.compose.project="+project)
		ev.CleanupOK = strings.TrimSpace(left1+left2+left3) == ""
	}()
	// Serviços da configuração precisam existir na composição.
	got, err := r.docker(ctx, dir, append(base, "config", "--services")...)
	if err != nil {
		return ev, fmt.Errorf("composição inválida: %w", err)
	}
	have := map[string]bool{}
	for _, s := range strings.Fields(got) {
		have[s] = true
	}
	for _, n := range names {
		if !have[n] {
			return ev, fmt.Errorf("o serviço %q da configuração não existe na composição", n)
		}
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	upCtx, cancel := context.WithTimeout(ctx, timeout+3*time.Minute)
	defer cancel()
	if _, err := r.docker(upCtx, dir, append(base, "up", "-d", "--no-build", "--pull", "never")...); err != nil {
		// serviços com `build` precisam da imagem construída antes: tenta com build.
		if _, err2 := r.docker(upCtx, dir, append(base, "up", "-d", "--build")...); err2 != nil {
			return ev, fmt.Errorf("subir a composição: %w", err2)
		}
	}
	if r.AfterUp != nil {
		r.AfterUp(project)
	}
	// Prontidão: healthcheck da composição; sem healthcheck, o comando de prontidão da configuração.
	deadline := time.Now().Add(timeout)
	for _, s := range c.Services {
		ev.Ready[s.Name] = r.waitReady(ctx, dir, base, project, s, deadline)
	}
	if c.DataSetup != "" {
		target := ""
		for _, s := range c.Services {
			if s.Kind == "api" {
				target = s.Name
				break
			}
		}
		if target == "" && len(names) > 0 {
			target = names[0]
		}
		if _, err := r.docker(ctx, dir, append(base, "exec", "-T", target, "sh", "-c", c.DataSetup)...); err != nil {
			ev.Diagnostics = append(ev.Diagnostics, "preparo de dados falhou: "+err.Error())
			return ev, fmt.Errorf("preparo de dados: %w", err)
		}
	}
	// Navegador MCP real no container isolado, ligado ao serviço-alvo.
	origin, err := browserenv.ParseOrigin(c.BrowserTarget)
	if err != nil {
		return ev, err
	}
	targetSvc := strings.Split(origin.Host, ":")[0]
	spec := browserenv.Spec{ProjectID: "env", ExecutionID: id, Image: r.BrowserImage, Entrypoint: "sleep", Command: []string{"3600"},
		Allowed: []string{c.BrowserTarget}, MemoryMB: 1536, CPUs: 2}
	env, err := browserenv.Provision(ctx, rt, spec)
	if err != nil {
		return ev, fmt.Errorf("navegador isolado: %w", err)
	}
	if err := env.Ready(ctx, 30*time.Second); err != nil {
		return ev, err
	}
	cid, err := r.docker(ctx, dir, append(base, "ps", "-q", targetSvc)...)
	if err != nil || strings.TrimSpace(cid) == "" {
		return ev, fmt.Errorf("o serviço-alvo %s não está em execução", targetSvc)
	}
	if err := env.Attach(ctx, strings.Fields(cid)[0], targetSvc); err != nil {
		return ev, fmt.Errorf("ligar o alvo à rede do navegador: %w", err)
	}
	drv, err := browserenv.StartMCP(ctx, r.bin(), env)
	if err != nil {
		return ev, fmt.Errorf("MCP do navegador: %w", err)
	}
	defer drv.Close()
	if err := drv.Navigate(ctx, c.BrowserTarget); err != nil {
		ev.Diagnostics = append(ev.Diagnostics, "o navegador não abriu "+c.BrowserTarget+": "+err.Error())
		return ev, nil
	}
	snap, err := drv.Interact(ctx, "snapshot", nil)
	if err != nil || strings.TrimSpace(snap) == "" {
		ev.Diagnostics = append(ev.Diagnostics, "o navegador abriu a página, mas o snapshot falhou")
		return ev, nil
	}
	ev.BrowserOK = true
	return ev, nil
}

func (r *ComposeRunner) waitReady(ctx context.Context, dir string, base []string, project string, s Service, deadline time.Time) bool {
	for {
		cid, _ := r.docker(ctx, dir, append(base, "ps", "-q", s.Name)...)
		if id := strings.Fields(cid); len(id) > 0 {
			health, err := r.docker(ctx, dir, "inspect", "--format", "{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}|{{.State.Running}}", id[0])
			if err == nil {
				parts := strings.SplitN(strings.TrimSpace(health), "|", 2)
				switch {
				case parts[0] == "healthy":
					return true
				case parts[0] == "none" && len(parts) == 2 && parts[1] == "true":
					// sem healthcheck: usa o comando de prontidão da configuração, se houver.
					if strings.TrimSpace(s.Readiness) == "" {
						return true
					}
					if _, err := r.docker(ctx, dir, append(base, "exec", "-T", s.Name, "sh", "-c", s.Readiness)...); err == nil {
						return true
					}
				case parts[0] == "unhealthy":
					return false
				}
			}
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return false
		}
		time.Sleep(time.Second)
	}
}

// GitCheckout materializa uma revisão numa worktree destacada e temporária do repositório.
func GitCheckout(repo, root string) func(ctx context.Context, sha string) (string, func(), error) {
	return func(ctx context.Context, sha string) (string, func(), error) {
		if !regexp.MustCompile(`^[0-9a-f]{7,64}$`).MatchString(sha) {
			return "", nil, errors.New("SHA inválida")
		}
		if err := os.MkdirAll(root, 0o750); err != nil {
			return "", nil, err
		}
		dir, err := os.MkdirTemp(root, "env-")
		if err != nil {
			return "", nil, err
		}
		cmd := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", dir, sha)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			_ = os.RemoveAll(dir)
			return "", nil, fmt.Errorf("git worktree: %v: %s", err, strings.TrimSpace(string(out)))
		}
		return dir, func() {
			rm := exec.Command("git", "worktree", "remove", "--force", dir)
			rm.Dir = repo
			_ = rm.Run()
			_ = os.RemoveAll(dir)
		}, nil
	}
}
