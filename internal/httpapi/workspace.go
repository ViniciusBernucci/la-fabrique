package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/workspace"
)

// WithWorkspaceSetup ajusta o serviço de workspaces (raiz, GitHub falso e git em testes).
func WithWorkspaceSetup(f func(*workspace.Service)) Option { return func(s *Server) { f(&s.ws) } }

// workspaceBrowse lista as subpastas da área de workspaces (só caminhos relativos).
func (s *Server) workspaceBrowse(w http.ResponseWriter, r *http.Request) {
	l, err := s.ws.Root.Browse(r.URL.Query().Get("path"))
	switch {
	case errors.Is(err, workspace.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "Pasta não encontrada.", "Volte e escolha outra pasta.")
		return
	case err != nil:
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Caminho inválido.", "Escolha uma pasta dentro da área de workspaces.")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// githubConnection resolve uma conexão GitHub ativa do cofre.
func (s *Server) githubConnection(w http.ResponseWriter, r *http.Request, id string) (security.Secret, security.Connection, bool) {
	if !uuidPattern.MatchString(id) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Conexão inválida.", "Escolha uma conexão GitHub.")
		return security.Secret{}, security.Connection{}, false
	}
	secret, conn, err := s.vault.Resolve(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return security.Secret{}, security.Connection{}, false
	}
	if conn.Kind != "github" || conn.State != "active" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "A conexão escolhida não é uma conta GitHub ativa.", "Escolha uma conexão GitHub ativa em Conexões.")
		return security.Secret{}, security.Connection{}, false
	}
	return secret, conn, true
}

// githubRepos lista os repositórios que a conta da conexão enxerga (para escolher o que clonar).
func (s *Server) githubRepos(w http.ResponseWriter, r *http.Request) {
	secret, _, ok := s.githubConnection(w, r, r.URL.Query().Get("connection_id"))
	if !ok {
		return
	}
	ctx, cancel := ctxTimeout(r, 40*time.Second)
	defer cancel()
	repos, err := s.ws.GH.ListRepos(ctx, secret)
	if err != nil {
		writeWorkspaceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": repos})
}

func writeWorkspaceError(w http.ResponseWriter, r *http.Request, err error) {
	var ue *workspace.UserError
	switch {
	case errors.As(err, &ue):
		writeError(w, r, ue.Status, ue.Code, ue.Msg, ue.Action)
	case errors.Is(err, workspace.ErrGitHubAuth):
		writeError(w, r, http.StatusBadGateway, "github_auth", err.Error(), "Reconecte a conta em Conexões.")
	case errors.Is(err, workspace.ErrGitHubForbidden):
		writeError(w, r, http.StatusBadGateway, "github_forbidden", err.Error(), "Confira o escopo repo da conexão.")
	default:
		writeError(w, r, http.StatusBadGateway, "github_error", err.Error(), "Tente de novo em instantes.")
	}
}

type prepareBody struct {
	ExpectedVersion int     `json:"expected_version"`
	Mode            string  `json:"mode"`
	Path            string  `json:"path"`
	Owner           string  `json:"owner"`
	Name            string  `json:"name"`
	BaseBranch      string  `json:"base_branch"`
	ConnectionID    *string `json:"connection_id"`
	CreateRemote    bool    `json:"create_remote"`
	Private         bool    `json:"private"`
}

// prepareRepository executa o modo escolhido (clonar, pasta existente ou criar do zero) e grava o
// resultado na configuração do projeto. Só rascunho ou suspenso, como o restante da configuração.
func (s *Server) prepareRepository(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req prepareBody
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ExpectedVersion < 1 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "expected_version é obrigatório.", "Envie a versão do projeto que você carregou.")
		return
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if p.Version != req.ExpectedVersion {
		s.writeDomainError(w, r, persistence.ErrVersionConflict)
		return
	}
	if p.State == "active" {
		s.writeDomainError(w, r, projects.ErrActiveLocked)
		return
	}
	repo, err := projects.NormalizeRepository(projects.Repository{
		Mode: req.Mode, Path: req.Path, Owner: req.Owner, Name: req.Name, BaseBranch: req.BaseBranch, ConnectionID: req.ConnectionID,
	})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if !s.pathFree(w, r, p, repo) {
		return
	}
	wr := workspace.Request{
		Mode: repo.Mode, Slug: p.Slug, Title: p.Name, Path: repo.Path, Owner: repo.Owner, Name: repo.Name, Base: repo.BaseBranch,
		CreateRemote: req.CreateRemote, Private: req.Private, Description: p.Description,
	}
	if repo.ConnectionID != nil {
		secret, conn, ok := s.githubConnection(w, r, *repo.ConnectionID)
		if !ok {
			return
		}
		wr.Token, wr.Login = secret, conn.Target
	}
	ctx, cancel := ctxTimeout(r, 4*time.Minute)
	defer cancel()
	res, err := s.ws.Prepare(ctx, wr)
	if err != nil {
		writeWorkspaceError(w, r, err)
		return
	}
	repo.Path, repo.Owner, repo.Name, repo.BaseBranch = res.Path, res.Owner, res.Name, res.Base
	updated, err := projects.Configure(r.Context(), s.db, s.adminID(r), id, req.ExpectedVersion, projects.Patch{Repository: &repo})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": updated, "notes": res.Notes})
}

func ctxTimeout(r *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), d)
}

// pathFree recusa a pasta que outro projeto já usa, comparando o caminho real (links simbólicos
// e diferença de maiúsculas não escapam da regra).
func (s *Server) pathFree(w http.ResponseWriter, r *http.Request, p projects.Project, repo projects.Repository) bool {
	rel := repo.Path
	if rel == "" && repo.Mode != projects.RepoLocal {
		rel = p.Slug
	}
	if rel == "" || !s.ws.Root.Enabled() {
		return true
	}
	abs, err := s.ws.Root.Resolve(rel)
	if err != nil {
		return true // o Prepare devolve o erro de caminho com a mensagem certa
	}
	others, err := projects.OtherRepoPaths(r.Context(), s.db, p.ID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return false
	}
	for _, o := range others {
		if other, err := s.ws.Root.Resolve(o.Path); err == nil && strings.EqualFold(other, abs) {
			writeError(w, r, http.StatusConflict, "workspace_taken", "A pasta "+rel+" já é usada pelo projeto \""+o.Name+"\".",
				"Escolha outra pasta ou apague/ajuste o outro projeto.")
			return false
		}
	}
	return true
}

// checkRepository prova o repositório configurado (pasta, origin, branch base, conta e push) sem escrever
// e grava o resultado vinculado à configuração verificada; a ativação o exige.
func (s *Server) checkRepository(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	repo := p.Repository
	if repo.Owner == "" || repo.Name == "" || repo.BaseBranch == "" {
		writeError(w, r, http.StatusConflict, "repository_incomplete", "Repositório incompleto: faltam dono, nome ou branch base.",
			"Preencha a etapa Repositório antes de verificar.")
		return
	}
	req := workspace.CheckRequest{Path: repo.Path, Owner: repo.Owner, Name: repo.Name, Base: repo.BaseBranch}
	secretVersion := 0
	if repo.ConnectionID != nil {
		secret, conn, err := s.vault.Resolve(r.Context(), *repo.ConnectionID)
		if err == nil && conn.Kind == "github" && conn.State == "active" {
			req.Token, secretVersion = secret, conn.SecretVersion
		}
	}
	ctx, cancel := ctxTimeout(r, 60*time.Second)
	defer cancel()
	good, steps := s.ws.Check(ctx, req)
	res := projects.RepoCheck{OK: good, Steps: make([]projects.SourceStep, 0, len(steps)), CheckedAt: time.Now().UTC()}
	for _, st := range steps {
		res.Steps = append(res.Steps, projects.SourceStep{ID: st.ID, Label: st.Label, Status: st.Status, Detail: st.Detail})
	}
	if err := projects.RecordRepoCheck(r.Context(), s.db, s.adminID(r), id, projects.RepoFingerprint(p, secretVersion), res); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
