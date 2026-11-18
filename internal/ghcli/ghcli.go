// Package ghcli usa o GitHub CLI (`gh`) da máquina como fonte de identidade, no
// mesmo modelo do Orca: lista as contas logadas, obtém o token de uma conta
// específica na hora do uso e nunca deixa variáveis de token do ambiente
// sobrescreverem a conta escolhida. O executável é chamado com vetor de
// argumentos (sem shell) e timeout curto.
package ghcli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Estados do diagnóstico (iguais aos do cartão de integração do Orca).
const (
	StateConnected        = "connected"
	StateNotInstalled     = "not-installed"
	StateNotAuthenticated = "not-authenticated"
)

// RequiredScopes são os escopos que a fábrica usa (commits, push, PRs, Actions).
// `read:org` e `project` aparecem como aviso: o Orca também os confere.
var RequiredScopes = []string{"repo", "read:org", "workflow"}

// Erros.
var (
	ErrNotInstalled = errors.New("GitHub CLI (gh) não instalado; instale com `brew install gh`")
	ErrNoAccount    = errors.New("nenhuma conta logada no gh; rode `gh auth login`")
	ErrToken        = errors.New("o gh não devolveu token para esta conta")
)

// Account é uma conta conhecida pelo gh.
type Account struct {
	Host   string
	User   string
	Active bool
	Source string // keyring, GH_TOKEN, hosts.yml…
	Scopes []string
}

// Runner executa o gh. Substituível em testes.
type Runner func(ctx context.Context, env []string, args ...string) (string, error)

// Client fala com o gh.
type Client struct {
	Run     Runner
	Timeout time.Duration
}

// New usa o `gh` do PATH.
func New() *Client { return &Client{Run: execRunner, Timeout: 10 * time.Second} }

func execRunner(ctx context.Context, env []string, args ...string) (string, error) {
	path, err := exec.LookPath("gh")
	if err != nil {
		return "", ErrNotInstalled
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out // versões antigas do gh escrevem o status em stderr
	err = cmd.Run()
	return out.String(), err
}

// CleanEnv remove as variáveis que fariam o gh ignorar a conta escolhida e
// preserva o resto (PATH, HOME, GH_CONFIG_DIR…).
func CleanEnv(environ []string) []string {
	drop := map[string]bool{"GH_TOKEN": true, "GITHUB_TOKEN": true, "GH_ENTERPRISE_TOKEN": true,
		"GITHUB_ENTERPRISE_TOKEN": true, "GH_HOST": true, "GH_REPO": true, "GH_PROMPT_DISABLED": true}
	out := make([]string, 0, len(environ)+1)
	for _, kv := range environ {
		k, _, _ := strings.Cut(kv, "=")
		if !drop[k] {
			out = append(out, kv)
		}
	}
	return append(out, "GH_PROMPT_DISABLED=1")
}

func (c *Client) run(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	return c.Run(ctx, CleanEnv(os.Environ()), args...)
}

var (
	reLogged = regexp.MustCompile(`Logged in to (\S+) (?:account (\S+)|as (\S+))(?: \(([^)]*)\))?`)
	reActive = regexp.MustCompile(`Active account:\s*(true|false)`)
	reScopes = regexp.MustCompile(`Token scopes:\s*(.*)`)
	reHost   = regexp.MustCompile(`^(\S+\.\S+)\s*$`)
)

// ParseStatus interpreta a saída de `gh auth status` (texto, como o Orca faz).
func ParseStatus(out string) []Account {
	var accounts []Account
	var cur *Account
	host := ""
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if m := reHost.FindStringSubmatch(trimmed); m != nil && !strings.Contains(trimmed, " ") {
			host = m[1]
		}
		if m := reLogged.FindStringSubmatch(trimmed); m != nil {
			user := m[2]
			if user == "" {
				user = m[3]
			}
			accounts = append(accounts, Account{Host: m[1], User: strings.TrimSpace(user), Source: m[4]})
			cur = &accounts[len(accounts)-1]
			if cur.Host == "" {
				cur.Host = host
			}
			continue
		}
		if cur == nil {
			continue
		}
		if m := reActive.FindStringSubmatch(trimmed); m != nil {
			cur.Active = m[1] == "true"
		}
		if m := reScopes.FindStringSubmatch(trimmed); m != nil {
			for _, s := range strings.Split(m[1], ",") {
				if s = strings.Trim(strings.TrimSpace(s), `'"`); s != "" && s != "none" {
					cur.Scopes = append(cur.Scopes, s)
				}
			}
		}
	}
	if len(accounts) == 1 {
		accounts[0].Active = true // formato antigo não informa "Active account"
	}
	return accounts
}

// Accounts lista as contas logadas no host. Erro: ErrNotInstalled ou ErrNoAccount.
func (c *Client) Accounts(ctx context.Context, host string) ([]Account, error) {
	out, err := c.run(ctx, "auth", "status", "--hostname", host)
	if errors.Is(err, ErrNotInstalled) {
		return nil, err
	}
	accounts := ParseStatus(out)
	if len(accounts) == 0 {
		return nil, ErrNoAccount
	}
	return accounts, nil
}

// Token devolve o token de uma conta (`gh auth token --user U --hostname H`).
// Exige gh ≥ 2.40; sem --user (gh antigo) só a conta ativa é possível.
func (c *Client) Token(ctx context.Context, host, user string) (string, error) {
	out, err := c.run(ctx, "auth", "token", "--hostname", host, "--user", user)
	if err != nil {
		if strings.Contains(out, "unknown flag") {
			out, err = c.run(ctx, "auth", "token", "--hostname", host)
		}
		if err != nil {
			return "", ErrToken
		}
	}
	tok := strings.TrimSpace(out)
	if tok == "" || strings.ContainsAny(tok, " \n\t") {
		return "", ErrToken
	}
	return tok, nil
}

// MissingScopes diz quais escopos exigidos faltam (para o aviso do diagnóstico).
func MissingScopes(have []string) []string {
	set := map[string]bool{}
	for _, s := range have {
		set[s] = true
	}
	var miss []string
	for _, r := range RequiredScopes {
		if !set[r] {
			miss = append(miss, r)
		}
	}
	return miss
}
