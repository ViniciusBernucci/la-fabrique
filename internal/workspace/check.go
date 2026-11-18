package workspace

import (
	"context"
	"errors"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Estados dos passos da verificação.
const (
	StepOK      = "ok"
	StepFailed  = "failed"
	StepSkipped = "skipped"
	StepWarn    = "warn"
)

// Step é um passo da verificação do repositório.
type Step struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// CheckRequest descreve o repositório configurado do projeto.
type CheckRequest struct {
	Path, Owner, Name, Base string
	Token                   security.Secret
}

// Check prova o repositório de ponta a ponta sem escrever em lugar nenhum: pasta, git, origin,
// branch base, árvore limpa, conta, permissão de push e branch base no GitHub.
// Falhas viram passos "failed" com motivo e ação; só "failed" reprova.
func (s Service) Check(ctx context.Context, req CheckRequest) (bool, []Step) {
	var steps []Step
	add := func(id, label, status, detail string) { steps = append(steps, Step{id, label, status, detail}) }

	dir, local := "", false
	switch {
	case req.Path == "" || !s.Root.Enabled():
		add("folder", "Pasta do repositório", StepSkipped, "Sem pasta local: a fábrica clona sozinha quando for executar.")
	default:
		abs, err := s.Root.Resolve(req.Path)
		if err != nil {
			add("folder", "Pasta do repositório", StepFailed, "O caminho é inválido ou sai da área de workspaces. Escolha a pasta de novo.")
		} else if IsEmptyOrMissing(abs) {
			add("folder", "Pasta do repositório", StepFailed, "A pasta "+req.Path+" não existe ou está vazia. Clone ou escolha o repositório de novo.")
		} else {
			add("folder", "Pasta do repositório", StepOK, "/"+req.Path)
			dir, local = abs, true
		}
	}

	if local {
		if err := s.Git.Check(); err != nil {
			add("git", "Repositório git", StepFailed, err.Error()+".")
			local = false
		}
	}
	if local {
		info, err := s.Git.Inspect(ctx, dir)
		switch {
		case errors.Is(err, ErrNotRepo):
			add("git", "Repositório git", StepFailed, "A pasta não é um repositório git.")
			local = false
		case err != nil:
			add("git", "Repositório git", StepFailed, "Não consegui ler o repositório: "+err.Error())
			local = false
		default:
			add("git", "Repositório git", StepOK, "Branch atual: "+orDash(info.Branch)+".")
			switch {
			case info.Owner == "":
				add("origin", "Remoto origin", StepFailed, "A pasta não tem origin no GitHub. Adicione o remoto ou crie o repositório pela fábrica.")
			case !strings.EqualFold(info.Owner, req.Owner) || !strings.EqualFold(info.Name, req.Name):
				add("origin", "Remoto origin", StepFailed, "O origin aponta para "+info.Owner+"/"+info.Name+", mas o projeto usa "+req.Owner+"/"+req.Name+".")
			default:
				add("origin", "Remoto origin", StepOK, info.Owner+"/"+info.Name)
			}
		}
	}
	if local {
		if s.Git.HasBranch(ctx, dir, req.Base) {
			add("base_local", "Branch base na pasta", StepOK, req.Base)
		} else {
			add("base_local", "Branch base na pasta", StepFailed, "A branch "+req.Base+" não existe na pasta. Faça fetch/checkout ou corrija a branch base.")
		}
		if dirty, err := s.Git.Dirty(ctx, dir); err != nil {
			add("clean", "Árvore limpa", StepWarn, "Não consegui ler o estado da pasta.")
		} else if dirty {
			add("clean", "Árvore limpa", StepWarn, "Há alterações não commitadas na pasta. A fábrica não as altera, mas elas não entram nas tentativas.")
		} else {
			add("clean", "Árvore limpa", StepOK, "Sem alterações pendentes.")
		}
	}

	// Conta e permissões no GitHub.
	if req.Token.IsZero() {
		add("account", "Conta GitHub", StepFailed, "Escolha uma conexão GitHub ativa na etapa Repositório.")
		return finish(steps)
	}
	info, err := s.GH.Repo(ctx, req.Token, req.Owner, req.Name)
	switch {
	case errors.Is(err, ErrGitHubAuth):
		add("account", "Conta GitHub", StepFailed, "O GitHub recusou a conexão. Reconecte a conta em Conexões.")
		return finish(steps)
	case errors.Is(err, ErrGitHubForbidden):
		add("account", "Conta GitHub", StepFailed, "A conta não enxerga "+req.Owner+"/"+req.Name+" (não existe ou sem acesso).")
		return finish(steps)
	case err != nil:
		add("account", "Conta GitHub", StepFailed, "Não consegui consultar o GitHub: "+err.Error())
		return finish(steps)
	}
	add("account", "Conta GitHub", StepOK, "Acessa "+info.FullName+".")
	switch {
	case info.Archived:
		add("push", "Permissão de push", StepFailed, "O repositório está arquivado no GitHub.")
	case !info.CanPush:
		add("push", "Permissão de push", StepFailed, "A conta tem acesso somente leitura. A fábrica precisa de permissão de escrita para abrir PRs.")
	default:
		add("push", "Permissão de push", StepOK, "A conta pode enviar branches.")
	}
	if ok, err := s.GH.BranchExists(ctx, req.Token, req.Owner, req.Name, req.Base); err != nil {
		add("base_remote", "Branch base no GitHub", StepFailed, "Não consegui consultar a branch: "+err.Error())
	} else if !ok {
		add("base_remote", "Branch base no GitHub", StepFailed, "A branch "+req.Base+" não existe no GitHub. Corrija a branch base ou envie-a.")
	} else {
		add("base_remote", "Branch base no GitHub", StepOK, req.Base)
	}
	return finish(steps)
}

func finish(steps []Step) (bool, []Step) {
	ok := true
	for _, st := range steps {
		if st.Status == StepFailed {
			ok = false
		}
	}
	return ok, steps
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
