package prcycle

import (
	"context"
	"fmt"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/issuesync"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Resolver monta o Target de um projeto: GitHub e origem com os segredos do cofre (o segredo nunca
// é logado nem devolvido). Projeto sem repositório, origem ou mapa devolve ErrNoTarget.
type Resolver struct {
	DB           DB
	Vault        *security.Vault
	Syncer       *issuesync.Syncer
	GitHubBase   string // vazio = api.github.com
	AllowPrivate bool   // somente testes
}

// Target resolve o alvo do projeto.
func (r *Resolver) Target(ctx context.Context, projectID string) (Target, error) {
	p, err := projects.Get(ctx, r.DB, projectID)
	if err != nil {
		return Target{}, err
	}
	rp, src := p.Repository, p.Source
	if p.State != "active" || rp.ConnectionID == nil || rp.Owner == "" || rp.Name == "" || src.Kind == "" || src.ConnectionID == nil || src.Container == "" {
		return Target{}, ErrNoTarget
	}
	m, ok, err := (&flow.Store{DB: r.DB}).GetMap(ctx, projectID)
	if err != nil {
		return Target{}, err
	}
	if !ok {
		return Target{}, ErrNoTarget
	}
	gs, gc, err := r.Vault.Resolve(ctx, *rp.ConnectionID)
	if err != nil {
		return Target{}, err
	}
	if gc.Kind != "github" {
		return Target{}, ErrNoTarget
	}
	api, err := gitops.NewGitHubAPI(gs, r.GitHubBase, r.AllowPrivate)
	if err != nil {
		return Target{}, err
	}
	os, oc, err := r.Vault.Resolve(ctx, *src.ConnectionID)
	if err != nil {
		return Target{}, err
	}
	tr, err := integrations.NewTracker(integrations.Config{Kind: oc.Kind, Target: oc.Target, Secret: os})
	if err != nil {
		return Target{}, err
	}
	creator, ok := tr.(Origin)
	if !ok {
		return Target{}, fmt.Errorf("a origem %s não permite criar Issues", oc.Kind)
	}
	source := issuesync.Source{Kind: src.Kind, Container: src.Container, Map: m.ProjectMap}
	return Target{
		Reader: GitHubReader{API: api, Owner: rp.Owner, Name: rp.Name}, Origin: creator, Container: src.Container, Map: m.ProjectMap,
		Register: func(ctx context.Context, it integrations.Item, parent string) error {
			return r.Syncer.RegisterCorrection(ctx, projectID, source, it, parent)
		},
	}, nil
}
