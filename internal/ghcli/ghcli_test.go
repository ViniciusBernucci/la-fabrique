package ghcli

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const multi = `github.com
  ✓ Logged in to github.com account octocat (keyring)
  - Active account: true
  - Git operations protocol: https
  - Token: gho_************************************
  - Token scopes: 'gist', 'read:org', 'repo', 'workflow'

  ✓ Logged in to github.com account trabalho (keyring)
  - Active account: false
  - Git operations protocol: https
  - Token: gho_************************************
  - Token scopes: 'repo'
`

func TestParseStatusMultiAccount(t *testing.T) {
	a := ParseStatus(multi)
	if len(a) != 2 {
		t.Fatalf("contas: %+v", a)
	}
	if a[0].User != "octocat" || !a[0].Active || a[0].Source != "keyring" || a[0].Host != "github.com" || len(a[0].Scopes) != 4 {
		t.Fatalf("conta 1: %+v", a[0])
	}
	if a[1].User != "trabalho" || a[1].Active || strings.Join(a[1].Scopes, ",") != "repo" {
		t.Fatalf("conta 2: %+v", a[1])
	}
	if m := MissingScopes(a[1].Scopes); strings.Join(m, ",") != "read:org,workflow" {
		t.Fatalf("faltam: %v", m)
	}
	if m := MissingScopes(a[0].Scopes); len(m) != 0 {
		t.Fatalf("faltam: %v", m)
	}
}

func TestParseStatusLegacyFormat(t *testing.T) {
	a := ParseStatus("github.com\n  ✓ Logged in to github.com as velho (/home/u/.config/gh/hosts.yml)\n  ✓ Token scopes: repo, read:org\n")
	if len(a) != 1 || a[0].User != "velho" || !a[0].Active || len(a[0].Scopes) != 2 {
		t.Fatalf("%+v", a)
	}
}

func TestCleanEnvDropsTokenVariables(t *testing.T) {
	env := CleanEnv([]string{"PATH=/bin", "GH_TOKEN=x", "GITHUB_TOKEN=y", "GH_ENTERPRISE_TOKEN=z", "GH_HOST=h", "GH_REPO=r", "HOME=/h"})
	joined := strings.Join(env, " ")
	for _, bad := range []string{"GH_TOKEN=x", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GH_HOST=h", "GH_REPO"} {
		if strings.Contains(joined, bad) {
			t.Fatalf("%s não deveria passar: %s", bad, joined)
		}
	}
	if !strings.Contains(joined, "PATH=/bin") || !strings.Contains(joined, "HOME=/h") || !strings.Contains(joined, "GH_PROMPT_DISABLED=1") {
		t.Fatalf("env: %s", joined)
	}
}

func TestAccountsAndTokenArgv(t *testing.T) {
	var calls [][]string
	c := &Client{Timeout: 1e9, Run: func(_ context.Context, env []string, args ...string) (string, error) {
		calls = append(calls, args)
		for _, kv := range env {
			if strings.HasPrefix(kv, "GH_TOKEN=") {
				t.Fatal("GH_TOKEN vazou para o gh")
			}
		}
		if args[1] == "status" {
			return multi, nil
		}
		return "gho_token-da-conta\n", nil
	}}
	acc, err := c.Accounts(context.Background(), "github.com")
	if err != nil || len(acc) != 2 {
		t.Fatalf("%v %+v", err, acc)
	}
	tok, err := c.Token(context.Background(), "github.com", "trabalho")
	if err != nil || tok != "gho_token-da-conta" {
		t.Fatalf("%q %v", tok, err)
	}
	want := "auth token --hostname github.com --user trabalho"
	if got := strings.Join(calls[1], " "); got != want {
		t.Fatalf("argv: %s", got)
	}
}

func TestDiagnosticStates(t *testing.T) {
	missing := &Client{Timeout: 1e9, Run: func(context.Context, []string, ...string) (string, error) { return "", ErrNotInstalled }}
	if _, err := missing.Accounts(context.Background(), "github.com"); !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("%v", err)
	}
	loggedOut := &Client{Timeout: 1e9, Run: func(context.Context, []string, ...string) (string, error) {
		return "You are not logged into any GitHub hosts. To log in, run: gh auth login", errors.New("exit 1")
	}}
	if _, err := loggedOut.Accounts(context.Background(), "github.com"); !errors.Is(err, ErrNoAccount) {
		t.Fatalf("%v", err)
	}
}
