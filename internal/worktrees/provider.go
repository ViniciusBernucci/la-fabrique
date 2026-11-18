package worktrees

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/workspace"
)

// WorktreesDir é a pasta (oculta, fora da navegação) das worktrees dentro da raiz de workspaces.
const WorktreesDir = ".lf-worktrees"

// Provider monta o Manager de cada projeto a partir do checkout principal do projeto (que a fábrica
// nunca altera) e da raiz de workspaces; as worktrees nascem em <raiz>/.lf-worktrees.
type Provider struct {
	DB            DB
	Root          workspace.Root
	MaxPerProject int
	MaxPerHost    int
}

// For devolve o Manager do projeto. Falha se a raiz não está configurada, se o projeto não tem pasta
// ou se a pasta não é um repositório git dentro da raiz.
func (p Provider) For(ctx context.Context, projectID string) (*Manager, error) {
	if !p.Root.Enabled() {
		return nil, fmt.Errorf("%w: a área de workspaces não está configurada", ErrInvalid)
	}
	var rel string
	if err := p.DB.QueryRow(ctx, `SELECT repo_path FROM projects WHERE id::text = $1`, projectID).Scan(&rel); err != nil {
		return nil, persistence.MapError(err)
	}
	if rel == "" {
		return nil, fmt.Errorf("%w: o projeto não tem pasta de repositório", ErrInvalid)
	}
	repo, err := p.Root.Resolve(rel)
	if err != nil {
		return nil, fmt.Errorf("%w: pasta do projeto fora da área permitida", ErrInvalid)
	}
	if !workspace.IsRepo(repo) {
		return nil, fmt.Errorf("%w: a pasta do projeto não é um repositório git", ErrInvalid)
	}
	base, err := p.Root.Resolve("")
	if err != nil {
		return nil, err
	}
	return &Manager{DB: p.DB, Repo: repo, Root: filepath.Join(base, WorktreesDir),
		MaxPerProject: p.MaxPerProject, MaxPerHost: p.MaxPerHost}, nil
}
