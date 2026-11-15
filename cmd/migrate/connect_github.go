package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/ghcli"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type logger interface {
	Info(string, ...any)
	Error(string, ...any)
}

// connectGitHub importa para o cofre as contas logadas no GitHub CLI da máquina
// (modelo do Orca): lista as contas, obtém o token de cada uma com
// `gh auth token --user U --hostname H` num ambiente limpo, confere a identidade
// e os escopos e grava uma conexão por conta ("GitHub · usuário"). Rodar de novo
// atualiza os tokens (equivale ao "Re-check"). Sem conta, abre `gh auth login`.
// A conta de cada projeto é escolhida na configuração do projeto.
func connectGitHub(ctx context.Context, log logger) int {
	const host = "github.com" // GitHub Enterprise ainda não é suportado pela fábrica
	if h := strings.TrimSpace(os.Getenv("GH_HOSTNAME")); h != "" && h != host {
		log.Error("só github.com é suportado por enquanto (GitHub Enterprise não)")
		return 2
	}
	cfg, err := config.LoadAPI(os.LookupEnv)
	if err != nil {
		log.Error("configuração inválida: " + err.Error())
		return 2
	}
	gh := ghcli.New()
	accounts, err := gh.Accounts(ctx, host)
	if errors.Is(err, ghcli.ErrNoAccount) {
		fmt.Println("Nenhuma conta no GitHub CLI. Abrindo o login (siga as instruções no navegador)…")
		login := exec.CommandContext(ctx, "gh", "auth", "login", "--hostname", host, "--git-protocol", "https", "--web",
			"--scopes", strings.Join(ghcli.RequiredScopes, ","))
		login.Stdin, login.Stdout, login.Stderr = os.Stdin, os.Stdout, os.Stderr
		if lerr := login.Run(); lerr != nil {
			log.Error("o login do gh não foi concluído")
			return 1
		}
		accounts, err = gh.Accounts(ctx, host)
	}
	if err != nil {
		log.Error(err.Error())
		return 1
	}
	only := strings.TrimSpace(os.Getenv("GH_USER"))

	pool, err := persistence.Connect(ctx, cfg.DatabaseURL, 10*time.Second)
	if err != nil {
		log.Error("falha ao conectar ao banco", "error", err.Error())
		return 1
	}
	defer pool.Close()
	sealer, err := security.NewSealer(cfg.VaultKey, cfg.VaultKeyID)
	if err != nil {
		log.Error("VAULT_KEY inválida")
		return 2
	}
	vault := security.NewVault(pool, sealer, security.NewRedactor())

	done := 0
	for _, a := range accounts {
		if only != "" && a.User != only {
			continue
		}
		token, terr := gh.Token(ctx, host, a.User)
		if terr != nil {
			log.Error("sem token para a conta " + a.User + ": " + terr.Error())
			continue
		}
		login, scopes, werr := githubWhoAmI(ctx, token)
		if werr != nil || !strings.EqualFold(login, a.User) {
			log.Error("a conta " + a.User + " não passou na verificação de identidade; rode `gh auth login` de novo")
			continue
		}
		name := "GitHub · " + login
		if err := upsertGitHub(ctx, vault, name, login, scopes, token); err != nil {
			log.Error("falha ao guardar "+name, "error", err.Error())
			return 1
		}
		msg := "conta conectada"
		if miss := ghcli.MissingScopes(strings.Fields(scopes)); len(miss) > 0 {
			msg += " (faltam escopos: " + strings.Join(miss, ", ") + "; rode `gh auth refresh -s " + strings.Join(miss, ",") + "`)"
		}
		log.Info(msg, "conta", login, "nome", name, "ativa_no_gh", a.Active, "escopos", scopes)
		done++
	}
	if done == 0 {
		log.Error("nenhuma conta foi conectada")
		return 1
	}
	return 0
}

func upsertGitHub(ctx context.Context, vault *security.Vault, name, login, scopes, token string) error {
	secret := security.NewSecret(token)
	items, err := vault.List(ctx)
	if err != nil {
		return err
	}
	for _, c := range items {
		if c.Kind == "github" && c.Name == name && c.State == "active" {
			_, err := vault.Rotate(ctx, c.ID, secret, c.Version)
			return err
		}
	}
	_, err = vault.Create(ctx, security.NewConnection{Kind: "github", Name: name, Target: login, Scopes: scopes, Secret: secret})
	return err
}

func githubWhoAmI(ctx context.Context, token string) (login, scopes string, err error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, derr := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if derr != nil {
		return "", "", errors.New("não consegui falar com o GitHub")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", errors.New("o GitHub recusou o token; rode `gh auth login` de novo")
	}
	var u struct {
		Login string `json:"login"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&u)
	return u.Login, strings.ReplaceAll(resp.Header.Get("X-OAuth-Scopes"), ", ", " "), nil
}
