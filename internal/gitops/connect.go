package gitops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Connector é a interface de LEITURA: valida a conexão e traz o código. Nada aqui
// altera o remoto.
type Connector interface {
	Check(ctx context.Context) (Report, error)
	Clone(ctx context.Context, root string) (*Workdir, error)
	RemoteRef(ctx context.Context, branch string) (sha string, exists bool, err error)
}

// RepoInfo é o que a API do GitHub informa (opcional; sem API, o que depende dela fica "não verificado").
type RepoInfo struct {
	DefaultBranch string
	CanPush       bool
	Archived      bool
	Disabled      bool
	BaseProtected bool
}

// RepoAPI consulta metadados do repositório.
type RepoAPI interface {
	Info(ctx context.Context, owner, name, base string) (RepoInfo, error)
}

// Report é o resultado do preflight da conexão.
type Report struct {
	OK         bool     `json:"ok"`
	BaseSHA    string   `json:"base_sha,omitempty"`
	CloneOK    bool     `json:"clone_ok"`
	Problems   []string `json:"problems"`
	Unverified []string `json:"unverified"` // o que não pôde ser conferido sem a API do GitHub
}

// GitConnector implementa Connector com o git do host.
type GitConnector struct {
	Repo  Repo
	Git   *Git
	API   RepoAPI // opcional
	Depth int     // 0 = clone completo
}

// RemoteRef lê a ref do ramo no remoto (ls-remote), sem baixar objetos.
func (c *GitConnector) RemoteRef(ctx context.Context, branch string) (string, bool, error) {
	if err := c.Repo.Validate(); err != nil {
		return "", false, err
	}
	if err := validRefName(branch); err != nil {
		return "", false, fmt.Errorf("%w: ramo", ErrInvalidRepo)
	}
	ref := "refs/heads/" + branch
	res, err := c.Git.run(ctx, &c.Repo, "", nil, "ls-remote", "--refs", c.Repo.URL(), ref)
	if err != nil {
		return "", false, err
	}
	for _, l := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
		f := strings.Fields(l)
		if len(f) == 2 && f[1] == ref && shaPattern.MatchString(f[0]) {
			return f[0], true, nil
		}
	}
	return "", false, nil
}

// Check valida a conexão DE VERDADE: o ramo base existe no remoto e um clone dele
// funciona e confere com a ref remota. A API (quando há) informa permissão de push,
// arquivamento e proteção do ramo base.
func (c *GitConnector) Check(ctx context.Context) (Report, error) {
	rep := Report{Problems: []string{}, Unverified: []string{}}
	if err := c.Repo.Validate(); err != nil {
		return rep, err
	}
	if !c.Git.Available() {
		return rep, ErrNoGit
	}
	sha, exists, err := c.RemoteRef(ctx, c.Repo.BaseBranch)
	switch {
	case errors.Is(err, ErrAuth):
		rep.Problems = append(rep.Problems, "A credencial foi recusada pelo repositório.")
		return rep, nil
	case errors.Is(err, ErrNotFound):
		rep.Problems = append(rep.Problems, "O repositório não foi encontrado (ou a credencial não o enxerga).")
		return rep, nil
	case err != nil:
		return rep, err
	case !exists:
		rep.Problems = append(rep.Problems, fmt.Sprintf("O ramo base %q não existe no repositório.", c.Repo.BaseBranch))
		return rep, nil
	}
	rep.BaseSHA = sha
	root, err := os.MkdirTemp("", "lf-check-")
	if err != nil {
		return rep, err
	}
	defer os.RemoveAll(root)
	wd, err := c.Clone(ctx, root)
	if err != nil {
		rep.Problems = append(rep.Problems, "O clone falhou: "+firstLine(redact(&c.Repo, err.Error())))
		return rep, nil
	}
	if got, gerr := wd.rev(ctx, "refs/heads/"+c.Repo.BaseBranch); gerr != nil || got != sha {
		rep.Problems = append(rep.Problems, "O clone não confere com a ref remota do ramo base.")
	} else {
		rep.CloneOK = true
	}
	if c.API == nil {
		rep.Unverified = append(rep.Unverified, "permissao_push", "base_protegida", "repositorio_arquivado")
	} else {
		info, ierr := c.API.Info(ctx, c.Repo.Owner, c.Repo.Name, c.Repo.BaseBranch)
		switch {
		case ierr != nil:
			rep.Unverified = append(rep.Unverified, "permissao_push", "base_protegida", "repositorio_arquivado")
		default:
			if info.Archived || info.Disabled {
				rep.Problems = append(rep.Problems, "O repositório está arquivado ou desativado.")
			}
			if !info.CanPush {
				rep.Problems = append(rep.Problems, "A credencial não tem permissão de escrita (push) no repositório.")
			}
			if !info.BaseProtected {
				rep.Problems = append(rep.Problems, fmt.Sprintf("O ramo base %q não está protegido: a fábrica só pode publicar branches de feature e a base precisa recusar push direto.", c.Repo.BaseBranch))
			}
		}
	}
	rep.OK = len(rep.Problems) == 0
	return rep, nil
}

// Workdir é um clone bare (só o banco de objetos) numa pasta do worker. Nunca há
// árvore de trabalho: o conteúdo entra por git fast-import.
type Workdir struct {
	Path string
	Repo Repo
	Git  *Git
}

// Clone cria o clone bare do ramo base dentro de root.
func (c *GitConnector) Clone(ctx context.Context, root string) (*Workdir, error) {
	if err := c.Repo.Validate(); err != nil {
		return nil, err
	}
	dest, err := os.MkdirTemp(root, "clone-")
	if err != nil {
		return nil, err
	}
	args := []string{"clone", "--bare", "--single-branch", "--no-tags", "--branch", c.Repo.BaseBranch}
	if c.Depth > 0 {
		args = append(args, "--depth", fmt.Sprint(c.Depth))
	}
	args = append(args, "--", c.Repo.URL(), dest)
	if _, err := c.Git.run(ctx, &c.Repo, "", nil, args...); err != nil {
		_ = os.RemoveAll(dest)
		return nil, err
	}
	// O repositório clonado não pode influenciar o git do worker.
	_ = os.Remove(dest + "/hooks")
	return &Workdir{Path: dest, Repo: c.Repo, Git: c.Git}, nil
}

// Cleanup apaga o clone.
func (w *Workdir) Cleanup() { _ = os.RemoveAll(w.Path) }

func (w *Workdir) git(ctx context.Context, stdin []byte, args ...string) (Result, error) {
	return w.Git.run(ctx, &w.Repo, w.Path, stdin, args...)
}

// rev resolve uma ref/objeto para o SHA completo.
func (w *Workdir) rev(ctx context.Context, name string) (string, error) {
	res, err := w.git(ctx, nil, "rev-parse", "--verify", "--quiet", name+"^{commit}")
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(res.Stdout)
	if !shaPattern.MatchString(sha) {
		return "", fmt.Errorf("%w: SHA inesperado", ErrGit)
	}
	return sha, nil
}

// BaseTip é o commit do ramo base no clone.
func (w *Workdir) BaseTip(ctx context.Context) (string, error) {
	return w.rev(ctx, "refs/heads/"+w.Repo.BaseBranch)
}

// ---- API do GitHub (REST) -------------------------------------------------

// GitHubAPI consulta repositório e proteção do ramo. Não segue redirecionamento;
// erros têm mensagem fixa.
type GitHubAPI struct {
	Base  string
	Token security.Secret
	HTTP  *http.Client
}

// NewGitHubAPI valida o endereço (https; http/privado só em teste).
func NewGitHubAPI(token security.Secret, base string, allowPrivate bool) (*GitHubAPI, error) {
	if base == "" {
		base = "https://api.github.com"
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, ErrInvalidRepo
	}
	if u.Scheme != "https" && !(allowPrivate && u.Scheme == "http") {
		return nil, ErrInvalidRepo
	}
	if !allowPrivate && policy.IsInternalHost(u.Hostname()) {
		return nil, ErrInvalidRepo
	}
	if token.IsZero() {
		return nil, ErrAuth
	}
	return &GitHubAPI{Base: strings.TrimRight(u.String(), "/"), Token: token,
		HTTP: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (a *GitHubAPI) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.Base+path, nil)
	if err != nil {
		return ErrInvalidRepo
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+a.Token.Reveal())
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%w: API do GitHub indisponível", ErrGit)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return ErrAuth
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("%w: API do GitHub respondeu %d", ErrGit, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil || json.Unmarshal(data, out) != nil {
		return fmt.Errorf("%w: resposta da API fora do formato", ErrGit)
	}
	return nil
}

// Info implementa RepoAPI.
func (a *GitHubAPI) Info(ctx context.Context, owner, name, base string) (RepoInfo, error) {
	if !ownerPattern.MatchString(owner) || !namePattern.MatchString(name) || validRefName(base) != nil {
		return RepoInfo{}, ErrInvalidRepo
	}
	var repo struct {
		DefaultBranch string `json:"default_branch"`
		Archived      bool   `json:"archived"`
		Disabled      bool   `json:"disabled"`
		Permissions   struct {
			Push bool `json:"push"`
		} `json:"permissions"`
	}
	if err := a.get(ctx, "/repos/"+owner+"/"+name, &repo); err != nil {
		return RepoInfo{}, err
	}
	var br struct {
		Protected bool `json:"protected"`
	}
	if err := a.get(ctx, "/repos/"+owner+"/"+name+"/branches/"+escapePath(base), &br); err != nil {
		return RepoInfo{}, err
	}
	return RepoInfo{DefaultBranch: repo.DefaultBranch, CanPush: repo.Permissions.Push, Archived: repo.Archived, Disabled: repo.Disabled, BaseProtected: br.Protected}, nil
}

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}
