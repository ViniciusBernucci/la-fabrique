package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Modos (iguais aos de projects.Repository.Mode).
const (
	ModeGitHub = "github"
	ModeLocal  = "local"
	ModeNew    = "new"
)

// Service prepara o repositório de um projeto.
type Service struct {
	Root   Root
	Git    Git
	GH     GitHub
	WebURL string // padrão https://github.com (testes: file://…)
}

// Request descreve o pedido. Token/Login vêm da conexão GitHub escolhida (podem faltar nos modos local e new sem remoto).
type Request struct {
	Mode         string
	Slug         string // identificador do projeto (pasta padrão)
	Title        string // nome do projeto (README)
	Path         string // relativo à raiz; vazio = Slug
	Owner, Name  string // repositório no GitHub
	Base         string // branch base
	CreateRemote bool   // modo new: cria também no GitHub
	Private      bool
	Description  string
	Token        security.Secret
	Login        string
}

// Result é o que o projeto deve guardar.
type Result struct {
	Path, Owner, Name, Base string
	Notes                   []string
}

func (s Service) web() string {
	if s.WebURL == "" {
		return "https://github.com"
	}
	return strings.TrimRight(s.WebURL, "/")
}

// UserError é um problema que o usuário resolve (entrada ou ambiente), com o código HTTP sugerido.
type UserError struct {
	Status int
	Code   string
	Msg    string
	Action string
}

func (e *UserError) Error() string { return e.Msg }

func bad(msg, action string) error { return &UserError{400, "invalid_request", msg, action} }
func conflict(code, msg, action string) error {
	return &UserError{409, code, msg, action}
}

// Prepare executa o modo pedido e devolve o repositório pronto.
func (s Service) Prepare(ctx context.Context, req Request) (Result, error) {
	if err := s.Git.Check(); err != nil {
		return Result{}, conflict("git_missing", err.Error(), "Instale o git no servidor e tente de novo.")
	}
	switch req.Mode {
	case ModeGitHub:
		return s.clone(ctx, req)
	case ModeLocal:
		return s.local(ctx, req)
	case ModeNew:
		return s.create(ctx, req)
	}
	return Result{}, bad("Modo do repositório inválido.", "Escolha clonar, usar pasta existente ou criar novo.")
}

func (s Service) dest(req Request) (rel, abs string, err error) {
	rel = strings.Trim(strings.TrimSpace(req.Path), "/")
	if rel == "" {
		rel = req.Slug
	}
	abs, err = s.Root.Resolve(rel)
	switch {
	case errors.Is(err, ErrDisabled):
		return "", "", conflict("workspace_disabled", err.Error(), "Defina WORKSPACES_DIR (e monte a pasta no container) para usar pastas locais.")
	case err != nil:
		return "", "", bad("Pasta inválida: "+err.Error()+".", "Escolha uma pasta dentro da área de workspaces.")
	}
	return rel, abs, nil
}

func (s Service) clone(ctx context.Context, req Request) (Result, error) {
	if req.Owner == "" || req.Name == "" || req.Base == "" {
		return Result{}, bad("Informe dono, nome e branch base do repositório.", "Escolha um repositório da lista ou preencha os campos.")
	}
	res := Result{Owner: req.Owner, Name: req.Name, Base: req.Base}
	if !s.Root.Enabled() {
		// Sem área de workspaces: o worker clona sozinho (comportamento anterior).
		res.Notes = []string{"Repositório salvo. A fábrica clona quando for executar (WORKSPACES_DIR não configurada)."}
		return res, nil
	}
	rel, abs, err := s.dest(req)
	if err != nil {
		return Result{}, err
	}
	res.Path = rel
	if IsRepo(abs) {
		info, ierr := s.Git.Inspect(ctx, abs)
		if ierr == nil && strings.EqualFold(info.Owner, req.Owner) && strings.EqualFold(info.Name, req.Name) {
			res.Notes = []string{"A pasta " + rel + " já tinha este repositório clonado; foi reaproveitada."}
			return res, nil
		}
		return Result{}, conflict("workspace_not_empty", "A pasta "+rel+" já contém outro repositório.", "Escolha outra pasta ou apague a existente.")
	}
	if !IsEmptyOrMissing(abs) {
		return Result{}, conflict("workspace_not_empty", "A pasta "+rel+" já existe e não está vazia.", "Escolha outra pasta ou esvazie a existente.")
	}
	url := s.web() + "/" + req.Owner + "/" + req.Name + ".git"
	if err := s.Git.Clone(ctx, url, abs, req.Base, req.Token); err != nil {
		return Result{}, &UserError{502, "clone_failed", "O clone falhou: " + err.Error(),
			"Confira o nome do repositório, a branch base e se a conexão GitHub tem acesso a ele."}
	}
	res.Notes = []string{"Repositório clonado em " + rel + "."}
	return res, nil
}

func (s Service) local(ctx context.Context, req Request) (Result, error) {
	if strings.TrimSpace(req.Path) == "" {
		return Result{}, bad("Escolha a pasta do repositório.", "Selecione uma pasta que já seja um repositório git.")
	}
	rel, abs, err := s.dest(req)
	if err != nil {
		return Result{}, err
	}
	if _, err := os.Stat(abs); err != nil {
		return Result{}, bad("A pasta "+rel+" não existe.", "Escolha uma pasta existente dentro da área de workspaces.")
	}
	info, err := s.Git.Inspect(ctx, abs)
	if err != nil {
		return Result{}, bad("A pasta "+rel+" não é um repositório git.", "Escolha uma pasta com um repositório (que tenha .git) ou use “Criar novo”.")
	}
	res := Result{Path: rel, Owner: req.Owner, Name: req.Name, Base: req.Base}
	if res.Owner == "" || res.Name == "" {
		res.Owner, res.Name = info.Owner, info.Name
	}
	if res.Base == "" {
		res.Base = info.Branch
	}
	switch {
	case !info.HasCommits:
		res.Notes = append(res.Notes, "O repositório ainda não tem commits; faça o primeiro commit antes de ativar o projeto.")
	case res.Owner == "":
		res.Notes = append(res.Notes, "A pasta não tem remoto origin no GitHub. Informe dono e nome do repositório (ou crie-o no GitHub) para a fábrica poder abrir PRs.")
	}
	return res, nil
}

func (s Service) create(ctx context.Context, req Request) (Result, error) {
	rel, abs, err := s.dest(req)
	if err != nil {
		return Result{}, err
	}
	if !IsEmptyOrMissing(abs) {
		return Result{}, conflict("workspace_not_empty", "A pasta "+rel+" já existe e não está vazia.", "Escolha outra pasta, ou use “Pasta existente” se ela já for um repositório.")
	}
	base := req.Base
	if base == "" {
		base = "main"
	}
	name := req.Name
	if name == "" {
		name = req.Slug
	}
	if req.CreateRemote && req.Token.IsZero() {
		return Result{}, bad("Para criar o repositório no GitHub, escolha uma conexão GitHub.", "Conecte uma conta em Conexões e selecione-a.")
	}
	existed := false
	if _, err := os.Stat(abs); err == nil {
		existed = true
	}
	id := Identity{}
	if req.Login != "" {
		id = Identity{Name: req.Login, Email: req.Login + "@users.noreply.github.com"}
	}
	if err := s.Git.InitRepo(ctx, abs, base, req.Title, id); err != nil {
		_ = s.undo(abs, existed)
		return Result{}, &UserError{500, "init_failed", "Não consegui criar o repositório: " + err.Error(), "Verifique as permissões da pasta de workspaces."}
	}
	res := Result{Path: rel, Base: base, Owner: req.Owner, Name: name, Notes: []string{"Repositório criado em " + rel + " com o commit inicial."}}
	if !req.CreateRemote {
		res.Notes = append(res.Notes, "Sem repositório no GitHub por enquanto: informe dono e nome depois, ou crie-o e conecte o remoto.")
		return res, nil
	}
	owner := req.Owner
	if owner == "" {
		owner = req.Login
	}
	repo, err := s.GH.CreateRepo(ctx, req.Token, owner, req.Login, name, req.Description, req.Private)
	if err != nil {
		_ = s.undo(abs, existed) // nada foi publicado: volta ao estado anterior
		return Result{}, &UserError{502, "github_create_failed", "Não consegui criar o repositório no GitHub: " + err.Error(),
			"Escolha outro nome ou confira as permissões da conta conectada (escopo repo)."}
	}
	res.Owner, res.Name = repo.Owner, repo.Name
	if err := s.Git.SetOriginAndPush(ctx, abs, s.web()+"/"+repo.Owner+"/"+repo.Name+".git", base, req.Token); err != nil {
		return Result{}, &UserError{502, "push_failed", fmt.Sprintf("O repositório %s foi criado no GitHub, mas o envio inicial falhou: %s", repo.FullName, err.Error()),
			"A pasta " + rel + " foi mantida. Use “Pasta existente” para continuar de onde parou."}
	}
	vis := "público"
	if repo.Private {
		vis = "privado"
	}
	res.Notes = append(res.Notes, "Repositório "+repo.FullName+" criado no GitHub ("+vis+") e commit inicial enviado.")
	return res, nil
}

// undo remove a pasta que esta chamada criou (nunca uma que já existia com o usuário dentro).
func (s Service) undo(abs string, existed bool) error {
	if !existed {
		return os.RemoveAll(abs)
	}
	items, err := os.ReadDir(abs)
	for _, it := range items {
		_ = os.RemoveAll(abs + "/" + it.Name())
	}
	return err
}
