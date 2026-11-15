package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type logger interface {
	Info(string, ...any)
	Error(string, ...any)
}

// connectGitHub guarda no cofre o token do GitHub que o usuário já tem na máquina
// (gh auth login). O token vem por variável de ambiente (`make connect-github`),
// nunca por argumento; a conta conectada é a dona das permissões no repositório.
func connectGitHub(ctx context.Context, log logger) int {
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		log.Error("sem token do GitHub (use `make connect-github`)")
		return 2
	}
	cfg, err := config.LoadAPI(os.LookupEnv)
	if err != nil {
		log.Error("configuração inválida: " + err.Error())
		return 2
	}
	login, scopes, err := githubWhoAmI(ctx, token)
	if err != nil {
		log.Error(err.Error())
		return 1
	}
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
	name := strings.TrimSpace(os.Getenv("CONNECTION_NAME"))
	if name == "" {
		name = "GitHub (" + login + ")"
	}
	secret := security.NewSecret(token)
	items, err := vault.List(ctx)
	if err != nil {
		log.Error("falha ao ler o cofre", "error", err.Error())
		return 1
	}
	for _, c := range items {
		if c.Kind == "github" && c.Name == name && c.State == "active" {
			if _, err := vault.Rotate(ctx, c.ID, secret, c.Version); err != nil {
				log.Error("falha ao atualizar a conexão", "error", err.Error())
				return 1
			}
			log.Info("conexão GitHub atualizada", "conta", login, "nome", name, "escopos", scopes)
			return 0
		}
	}
	if _, err := vault.Create(ctx, security.NewConnection{Kind: "github", Name: name, Target: login, Scopes: scopes, Secret: secret}); err != nil {
		if errors.Is(err, persistence.ErrDuplicate) {
			log.Error("já existe outra conexão com o nome " + name + " (revogada?); defina CONNECTION_NAME")
		} else {
			log.Error("falha ao guardar a conexão", "error", err.Error())
		}
		return 1
	}
	log.Info("conexão GitHub criada", "conta", login, "nome", name, "escopos", scopes)
	return 0
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
