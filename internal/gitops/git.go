package gitops

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Erros de domínio.
var (
	ErrInvalidRepo     = errors.New("repositório inválido")
	ErrAuth            = errors.New("credencial recusada pelo repositório")
	ErrNotFound        = errors.New("repositório ou ramo não encontrado")
	ErrRejected        = errors.New("o remoto recusou a operação")
	ErrTimeout         = errors.New("tempo limite excedido")
	ErrGit             = errors.New("falha do git")
	ErrNoGit           = errors.New("git indisponível no worker")
	ErrBranchForbidden = errors.New("push restrito: só a branch da feature pode ser publicada")
)

var (
	ownerPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`)
	namePattern  = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
	shaPattern   = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)
)

// Repo é o repositório do projeto. BaseURL vazio = https://github.com.
type Repo struct {
	Owner      string
	Name       string
	BaseBranch string
	Token      security.Secret
	BaseURL    string
	// AllowFile permite file:// (SÓ em teste: repositório local).
	AllowFile bool
}

func (r Repo) baseURL() string {
	if r.BaseURL == "" {
		return "https://github.com"
	}
	return strings.TrimRight(r.BaseURL, "/")
}

// URL é o endereço do remoto, montado a partir de dono e nome já validados.
func (r Repo) URL() string { return r.baseURL() + "/" + r.Owner + "/" + r.Name + ".git" }

// Validate confere dono, nome, ramo base e esquema do endereço.
func (r Repo) Validate() error {
	switch {
	case !ownerPattern.MatchString(r.Owner), !namePattern.MatchString(r.Name), r.Name == "." || r.Name == "..":
		return fmt.Errorf("%w: dono ou nome", ErrInvalidRepo)
	case validRefName(r.BaseBranch) != nil:
		return fmt.Errorf("%w: ramo base", ErrInvalidRepo)
	}
	u := r.baseURL()
	switch {
	case strings.HasPrefix(u, "https://"):
		if strings.Contains(u, "@") || strings.ContainsAny(u, "?# \t\n") {
			return fmt.Errorf("%w: endereço com credencial ou parâmetros", ErrInvalidRepo)
		}
	case strings.HasPrefix(u, "file://") && r.AllowFile:
	default:
		return fmt.Errorf("%w: só https é aceito", ErrInvalidRepo)
	}
	return nil
}

var refNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,200}$`)

func validRefName(n string) error {
	if !refNamePattern.MatchString(n) || strings.Contains(n, "..") || strings.Contains(n, "//") || strings.HasSuffix(n, "/") || strings.HasSuffix(n, ".") || strings.HasSuffix(n, ".lock") || strings.Contains(n, "/.") {
		return errors.New("nome de ramo inválido")
	}
	return nil
}

// Git executa o binário git do host com ambiente saneado.
type Git struct {
	Bin     string        // padrão "git"
	Timeout time.Duration // padrão 2 min por comando
	// Intercept é uma emenda de teste: envolve a execução de cada subcomando
	// (ex.: executar o push de verdade e devolver um timeout simulado).
	Intercept func(sub string, run func() error) error
	// OnRun observa cada subcomando antes de executar (teste: provar o que NÃO rodou).
	OnRun func(sub string, args []string)

	home string
}

func (g *Git) bin() string {
	if g.Bin != "" {
		return g.Bin
	}
	return "git"
}

func (g *Git) timeout() time.Duration {
	if g.Timeout > 0 {
		return g.Timeout
	}
	return 2 * time.Minute
}

// Available diz se o git existe no host.
func (g *Git) Available() bool {
	_, err := exec.LookPath(g.bin())
	return err == nil
}

func (g *Git) env(repo *Repo) ([]string, error) {
	if g.home == "" {
		d, err := os.MkdirTemp("", "lf-git-home-")
		if err != nil {
			return nil, err
		}
		g.home = d
	}
	allow := "https"
	if repo != nil && repo.AllowFile {
		allow = "https:file"
	}
	env := []string{
		"PATH=" + os.Getenv("PATH"), "HOME=" + g.home, "LC_ALL=C", "LANG=C",
		"GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=true", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_ALLOW_PROTOCOL=" + allow, "GIT_NO_REPLACE_OBJECTS=1",
	}
	if repo != nil && !repo.Token.IsZero() && strings.HasPrefix(repo.baseURL(), "https://") {
		// Autenticação por cabeçalho no ambiente do processo: não vai para argv, URL nem .git/config.
		cred := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + repo.Token.Reveal()))
		env = append(env, "GIT_CONFIG_COUNT=1",
			"GIT_CONFIG_KEY_0=http."+repo.baseURL()+"/.extraheader",
			"GIT_CONFIG_VALUE_0=AUTHORIZATION: basic "+cred)
	}
	return env, nil
}

// Cleanup remove o HOME temporário do git.
func (g *Git) Cleanup() {
	if g.home != "" {
		_ = os.RemoveAll(g.home)
		g.home = ""
	}
}

// Result é a saída de um comando.
type Result struct {
	Stdout string
	Stderr string
}

const maxGitOut = 8 << 20

type capBuf struct {
	bytes.Buffer
	over bool
}

func (c *capBuf) Write(p []byte) (int, error) {
	if c.Len()+len(p) > maxGitOut {
		c.over = true
		p = p[:max(0, maxGitOut-c.Len())]
	}
	return c.Buffer.Write(p)
}

// run executa `git <args>` em dir (vazio = cwd atual do processo), com hooks
// desligados. repo (opcional) fornece a credencial e o protocolo permitido.
func (g *Git) run(ctx context.Context, repo *Repo, dir string, stdin []byte, args ...string) (Result, error) {
	sub := ""
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			sub = a
			break
		}
	}
	if g.OnRun != nil {
		g.OnRun(sub, args)
	}
	exec1 := func() (Result, error) {
		ctx, cancel := context.WithTimeout(ctx, g.timeout())
		defer cancel()
		env, err := g.env(repo)
		if err != nil {
			return Result{}, err
		}
		full := append([]string{"-c", "core.hooksPath=/dev/null", "-c", "protocol.version=2", "-c", "credential.helper=", "-c", "core.fsmonitor=false", "-c", "gc.auto=0"}, args...)
		cmd := exec.CommandContext(ctx, g.bin(), full...)
		cmd.Dir, cmd.Env = dir, env
		if stdin != nil {
			cmd.Stdin = bytes.NewReader(stdin)
		}
		var out, errb capBuf
		cmd.Stdout, cmd.Stderr = &out, &errb
		runErr := cmd.Run()
		res := Result{Stdout: out.String(), Stderr: errb.String()}
		if runErr == nil {
			return res, nil
		}
		var ee *exec.ExitError
		switch {
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return res, fmt.Errorf("%w: git %s", ErrTimeout, sub)
		case errors.Is(runErr, exec.ErrNotFound):
			return res, ErrNoGit
		case errors.As(runErr, &ee):
			return res, g.classify(repo, sub, ee.ExitCode(), res.Stderr)
		}
		return res, fmt.Errorf("%w: git %s", ErrGit, sub)
	}
	if g.Intercept != nil {
		var res Result
		err := g.Intercept(sub, func() error {
			var e error
			res, e = exec1()
			return e
		})
		return res, err
	}
	return exec1()
}

// classify traduz a falha do git em erro de domínio, sem repassar o texto do stderr
// (pode conter URL ou cabeçalho): só uma linha redigida e curta, para diagnóstico.
func (g *Git) classify(repo *Repo, sub string, code int, stderr string) error {
	low := strings.ToLower(stderr)
	why := redact(repo, firstLine(stderr))
	switch {
	case strings.Contains(low, "authentication failed"), strings.Contains(low, "could not read username"), strings.Contains(low, "403"), strings.Contains(low, "401"), strings.Contains(low, "permission denied"):
		return fmt.Errorf("%w: git %s", ErrAuth, sub)
	case strings.Contains(low, "does not appear to be a git repository"), strings.Contains(low, "repository not found"), strings.Contains(low, "couldn't find remote ref"), strings.Contains(low, "remote branch") && strings.Contains(low, "not found"):
		return fmt.Errorf("%w: git %s", ErrNotFound, sub)
	case strings.Contains(low, "[rejected]"), strings.Contains(low, "[remote rejected]"), strings.Contains(low, "non-fast-forward"), strings.Contains(low, "protected branch"), strings.Contains(low, "pre-receive hook declined"):
		return fmt.Errorf("%w: git %s: %s", ErrRejected, sub, why)
	}
	return fmt.Errorf("%w: git %s (código %d): %s", ErrGit, sub, code, why)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// redact remove a credencial e o cabeçalho de um texto.
func redact(repo *Repo, s string) string {
	if repo != nil && !repo.Token.IsZero() {
		s = strings.ReplaceAll(s, repo.Token.Reveal(), "[REDACTED]")
		s = strings.ReplaceAll(s, base64.StdEncoding.EncodeToString([]byte("x-access-token:"+repo.Token.Reveal())), "[REDACTED]")
	}
	return s
}
