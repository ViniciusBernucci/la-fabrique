package workspace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// GitHub fala com a API do GitHub com o token da conexão (permissões da conta conectada).
type GitHub struct {
	BaseURL string // padrão https://api.github.com
	HTTP    *http.Client
}

// RepoRef é um repositório visível para a conta.
type RepoRef struct {
	FullName      string `json:"full_name"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	CanPush       bool   `json:"can_push"`
	Description   string `json:"description"`
}

// Erros da API.
var (
	ErrGitHubAuth      = errors.New("o GitHub recusou a conexão: reconecte a conta em Conexões")
	ErrGitHubForbidden = errors.New("a conta conectada não tem permissão para esta operação no GitHub")
	ErrGitHubExists    = errors.New("já existe um repositório com esse nome nessa conta")
	ErrGitHubDown      = errors.New("não consegui falar com o GitHub")
)

func (g GitHub) base() string {
	if g.BaseURL == "" {
		return "https://api.github.com"
	}
	return strings.TrimRight(g.BaseURL, "/")
}

func (g GitHub) client() *http.Client {
	if g.HTTP != nil {
		return g.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func (g GitHub) do(ctx context.Context, token security.Secret, method, path string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, g.base()+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token.Reveal())
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := g.client().Do(req)
	if err != nil {
		return ErrGitHubDown
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return ErrGitHubAuth
	case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound:
		return ErrGitHubForbidden
	case resp.StatusCode == http.StatusUnprocessableEntity:
		return ErrGitHubExists
	case resp.StatusCode >= 300:
		return fmt.Errorf("o GitHub respondeu %d", resp.StatusCode)
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// ListRepos devolve até 300 repositórios da conta (mais recentes primeiro).
func (g GitHub) ListRepos(ctx context.Context, token security.Secret) ([]RepoRef, error) {
	out := []RepoRef{}
	for page := 1; page <= 3; page++ {
		var rows []struct {
			FullName      string `json:"full_name"`
			Name          string `json:"name"`
			Private       bool   `json:"private"`
			DefaultBranch string `json:"default_branch"`
			Description   string `json:"description"`
			Archived      bool   `json:"archived"`
			Owner         struct {
				Login string `json:"login"`
			} `json:"owner"`
			Permissions struct {
				Push bool `json:"push"`
			} `json:"permissions"`
		}
		q := url.Values{"per_page": {"100"}, "page": {fmt.Sprint(page)}, "sort": {"pushed"},
			"affiliation": {"owner,collaborator,organization_member"}}
		if err := g.do(ctx, token, http.MethodGet, "/user/repos?"+q.Encode(), nil, &rows); err != nil {
			return nil, err
		}
		for _, r := range rows {
			if r.Archived {
				continue
			}
			out = append(out, RepoRef{FullName: r.FullName, Owner: r.Owner.Login, Name: r.Name, Private: r.Private,
				DefaultBranch: r.DefaultBranch, CanPush: r.Permissions.Push, Description: r.Description})
		}
		if len(rows) < 100 {
			break
		}
	}
	return out, nil
}

// Login devolve o usuário dono do token.
func (g GitHub) Login(ctx context.Context, token security.Secret) (string, error) {
	var u struct {
		Login string `json:"login"`
	}
	if err := g.do(ctx, token, http.MethodGet, "/user", nil, &u); err != nil {
		return "", err
	}
	return u.Login, nil
}

// CreateRepo cria o repositório na conta do token (owner == login) ou numa organização.
func (g GitHub) CreateRepo(ctx context.Context, token security.Secret, owner, login, name, description string, private bool) (RepoRef, error) {
	path := "/user/repos"
	if owner != "" && !strings.EqualFold(owner, login) {
		path = "/orgs/" + url.PathEscape(owner) + "/repos"
	}
	var r struct {
		FullName      string `json:"full_name"`
		Name          string `json:"name"`
		Private       bool   `json:"private"`
		DefaultBranch string `json:"default_branch"`
		Owner         struct {
			Login string `json:"login"`
		} `json:"owner"`
	}
	body := map[string]any{"name": name, "private": private, "description": description, "auto_init": false}
	if err := g.do(ctx, token, http.MethodPost, path, body, &r); err != nil {
		return RepoRef{}, err
	}
	return RepoRef{FullName: r.FullName, Owner: r.Owner.Login, Name: r.Name, Private: r.Private, DefaultBranch: r.DefaultBranch, CanPush: true}, nil
}

// RepoInfo é o que a API informa sobre um repositório para a verificação.
type RepoInfo struct {
	FullName      string
	DefaultBranch string
	CanPush       bool
	Archived      bool
}

func (g GitHub) status(ctx context.Context, token security.Secret, path string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.base()+path, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.Reveal())
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := g.client().Do(req)
	if err != nil {
		return 0, nil, ErrGitHubDown
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return resp.StatusCode, raw, nil
}

// Repo consulta o repositório com as permissões da conta do token.
func (g GitHub) Repo(ctx context.Context, token security.Secret, owner, name string) (RepoInfo, error) {
	code, raw, err := g.status(ctx, token, "/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name))
	if err != nil {
		return RepoInfo{}, err
	}
	switch {
	case code == http.StatusUnauthorized:
		return RepoInfo{}, ErrGitHubAuth
	case code == http.StatusNotFound || code == http.StatusForbidden:
		return RepoInfo{}, ErrGitHubForbidden
	case code >= 300:
		return RepoInfo{}, fmt.Errorf("o GitHub respondeu %d", code)
	}
	var r struct {
		FullName      string `json:"full_name"`
		DefaultBranch string `json:"default_branch"`
		Archived      bool   `json:"archived"`
		Permissions   struct {
			Push bool `json:"push"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return RepoInfo{}, err
	}
	return RepoInfo{FullName: r.FullName, DefaultBranch: r.DefaultBranch, Archived: r.Archived, CanPush: r.Permissions.Push}, nil
}

// BranchExists diz se a branch existe no remoto.
func (g GitHub) BranchExists(ctx context.Context, token security.Secret, owner, name, branch string) (bool, error) {
	code, _, err := g.status(ctx, token, "/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/branches/"+url.PathEscape(branch))
	if err != nil {
		return false, err
	}
	switch {
	case code == http.StatusOK:
		return true, nil
	case code == http.StatusNotFound:
		return false, nil
	case code == http.StatusUnauthorized:
		return false, ErrGitHubAuth
	}
	return false, fmt.Errorf("o GitHub respondeu %d", code)
}
