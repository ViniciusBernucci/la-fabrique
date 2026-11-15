package workspace

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Git executa o git do host em ambiente limpo (sem config global, sem prompt, sem
// hooks do repositório) e autentica por cabeçalho no ambiente do processo: o token
// não vai para argv, URL nem .git/config.
type Git struct {
	Bin     string
	Timeout time.Duration
	// AllowFile permite file:// (SÓ em teste).
	AllowFile bool
}

// ErrGit é uma falha do git com a primeira linha (já sem segredos) da saída.
type ErrGit struct{ Msg string }

func (e *ErrGit) Error() string { return e.Msg }

func (g Git) bin() string {
	if g.Bin != "" {
		return g.Bin
	}
	return "git"
}

func (g Git) timeout() time.Duration {
	if g.Timeout > 0 {
		return g.Timeout
	}
	return 3 * time.Minute
}

// Available diz se o git existe no host.
func (g Git) Available() bool {
	_, err := exec.LookPath(g.bin())
	return err == nil
}

func (g Git) run(ctx context.Context, dir string, token security.Secret, args ...string) (string, error) {
	home, err := os.MkdirTemp("", "lf-ws-home-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(home)
	allow := "https"
	if g.AllowFile {
		allow = "https:file"
	}
	env := []string{
		"PATH=" + os.Getenv("PATH"), "HOME=" + home, "LC_ALL=C", "LANG=C",
		"GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=true", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_ALLOW_PROTOCOL=" + allow,
	}
	if !token.IsZero() {
		cred := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token.Reveal()))
		env = append(env, "GIT_CONFIG_COUNT=1",
			"GIT_CONFIG_KEY_0=http.https://github.com/.extraheader",
			"GIT_CONFIG_VALUE_0=AUTHORIZATION: basic "+cred)
	}
	// Pasta de outro usuário (volume do host) e repositório de terceiros: confia só nesta pasta
	// e desliga o que executaria código do repositório.
	full := []string{"-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "-c", "protocol.ext.allow=never"}
	if dir != "" {
		full = append(full, "-c", "safe.directory="+dir)
	}
	full = append(full, args...)
	ctx, cancel := context.WithTimeout(ctx, g.timeout())
	defer cancel()
	cmd := exec.CommandContext(ctx, g.bin(), full...)
	cmd.Dir = dir
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", &ErrGit{"o git demorou demais e foi interrompido"}
		}
		return "", &ErrGit{scrub(out.String(), token)}
	}
	return out.String(), nil
}

func scrub(s string, token security.Secret) string {
	if !token.IsZero() {
		s = strings.ReplaceAll(s, token.Reveal(), "***")
	}
	for _, line := range strings.Split(s, "\n") {
		if l := strings.TrimSpace(line); l != "" {
			if len(l) > 300 {
				l = l[:300]
			}
			return l
		}
	}
	return "falha do git"
}

// Clone clona url (uma branch) em dest, que não pode existir com conteúdo.
func (g Git) Clone(ctx context.Context, url, dest, branch string, token security.Secret) error {
	if !IsEmptyOrMissing(dest) {
		return ErrNotEmpty
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	args := []string{"clone", "--single-branch"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, "--", url, dest)
	if _, err := g.run(ctx, "", token, args...); err != nil {
		_ = removeIfEmptyish(dest)
		return err
	}
	return nil
}

// removeIfEmptyish limpa o que um clone falho possa ter deixado (só se estava vazio antes).
func removeIfEmptyish(dest string) error { return os.RemoveAll(dest) }

// Info descreve um repositório existente.
type Info struct {
	Branch     string
	HasCommits bool
	Owner      string
	Name       string
	RemoteURL  string
}

var originPattern = regexp.MustCompile(`(?i)github\.com[:/]([A-Za-z0-9-]+)/([A-Za-z0-9._-]+?)(?:\.git)?/?$`)

// ParseOrigin extrai dono e nome de uma URL do GitHub (https ou ssh); ignora credenciais.
func ParseOrigin(u string) (owner, name string, ok bool) {
	m := originPattern.FindStringSubmatch(strings.TrimSpace(u))
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// Inspect lê branch atual, remoto origin e se há commits.
func (g Git) Inspect(ctx context.Context, dir string) (Info, error) {
	if !isRepo(dir) {
		return Info{}, ErrNotRepo
	}
	var in Info
	br, err := g.run(ctx, dir, security.Secret{}, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		br, err = g.run(ctx, dir, security.Secret{}, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return Info{}, ErrNotRepo
		}
	}
	in.Branch = strings.TrimSpace(br)
	if in.Branch == "HEAD" {
		in.Branch = ""
	}
	_, herr := g.run(ctx, dir, security.Secret{}, "rev-parse", "--verify", "-q", "HEAD")
	in.HasCommits = herr == nil
	if url, err := g.run(ctx, dir, security.Secret{}, "remote", "get-url", "origin"); err == nil {
		in.RemoteURL = strings.TrimSpace(url)
		in.Owner, in.Name, _ = ParseOrigin(in.RemoteURL)
		// Nunca devolva credenciais embutidas na URL.
		if strings.Contains(in.RemoteURL, "@") && strings.HasPrefix(in.RemoteURL, "http") {
			in.RemoteURL = ""
		}
	}
	return in, nil
}

// Identity é o autor do commit inicial.
type Identity struct{ Name, Email string }

// InitRepo cria o repositório em dir (vazio ou inexistente) com um README e o commit inicial.
func (g Git) InitRepo(ctx context.Context, dir, branch, title string, id Identity) error {
	if !IsEmptyOrMissing(dir) {
		return ErrNotEmpty
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if _, err := g.run(ctx, dir, security.Secret{}, "init", "-b", branch); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# "+title+"\n"), 0o644); err != nil {
		return err
	}
	if _, err := g.run(ctx, dir, security.Secret{}, "add", "README.md"); err != nil {
		return err
	}
	if id.Name == "" {
		id.Name = "La fabrique"
	}
	if id.Email == "" {
		id.Email = "fabrique@localhost"
	}
	_, err := g.run(ctx, dir, security.Secret{}, "-c", "user.name="+id.Name, "-c", "user.email="+id.Email,
		"commit", "-q", "-m", "Commit inicial")
	return err
}

// SetOriginAndPush liga o remoto origin e envia a branch.
func (g Git) SetOriginAndPush(ctx context.Context, dir, url, branch string, token security.Secret) error {
	if _, err := g.run(ctx, dir, security.Secret{}, "remote", "add", "origin", url); err != nil {
		return err
	}
	if _, err := g.run(ctx, dir, token, "push", "-u", "origin", branch); err != nil {
		return err
	}
	return nil
}

// ErrGitMissing é devolvido quando o git não está instalado.
var ErrGitMissing = errors.New("o git não está instalado no servidor")

// Check devolve ErrGitMissing se o git não existir.
func (g Git) Check() error {
	if !g.Available() {
		return ErrGitMissing
	}
	return nil
}

// Describe monta uma frase curta para mensagens.
func (i Info) Describe() string {
	if i.Owner != "" {
		return fmt.Sprintf("%s/%s (%s)", i.Owner, i.Name, i.Branch)
	}
	return i.Branch
}
