package gitops

import (
	"context"
	"fmt"

	"github.com/ViniciusBernucci/le-fabrique/internal/tasksync"
)

// PRInfoAPI é o que o verificador precisa do GitHub (GitHubAPI implementa).
type PRInfoAPI interface {
	GetPR(ctx context.Context, owner, name string, number int) (MergeInfo, error)
}

// MergeVerifier consulta a PR no GitHub e implementa tasksync.MergeVerifier. É leitura: só
// roda quando alguém pede (D89), nunca por polling.
type MergeVerifier struct {
	API   PRInfoAPI
	Owner string
	Name  string
}

// Merged implementa tasksync.MergeVerifier.
func (v MergeVerifier) Merged(ctx context.Context, prURL string) (tasksync.MergeState, error) {
	n, ok := PRNumberFromURL(prURL)
	if !ok {
		return tasksync.MergeState{}, fmt.Errorf("%w: URL de PR inválida", ErrInvalidRepo)
	}
	info, err := v.API.GetPR(ctx, v.Owner, v.Name, n)
	if err != nil {
		return tasksync.MergeState{}, err
	}
	return tasksync.MergeState{Merged: info.Merged, Closed: info.State == "closed", SHA: info.MergeSHA}, nil
}

// PRClient lê comentários e reabre PRs de um repositório (GitHubAPI implementa a API).
type PRClient struct {
	API   PRClientAPI
	Owner string
	Name  string
}

// PRClientAPI é o que o cliente precisa do GitHub.
type PRClientAPI interface {
	ListComments(ctx context.Context, owner, name string, number int) ([]Comment, error)
	ReopenPR(ctx context.Context, owner, name string, number int) error
}

// Comments lista os comentários da PR (só leitura, sob demanda).
func (c PRClient) Comments(ctx context.Context, prURL string) ([]Comment, error) {
	n, ok := PRNumberFromURL(prURL)
	if !ok {
		return nil, fmt.Errorf("%w: URL de PR inválida", ErrInvalidRepo)
	}
	return c.API.ListComments(ctx, c.Owner, c.Name, n)
}

// Reopen reabre a PR.
func (c PRClient) Reopen(ctx context.Context, prURL string) error {
	n, ok := PRNumberFromURL(prURL)
	if !ok {
		return fmt.Errorf("%w: URL de PR inválida", ErrInvalidRepo)
	}
	return c.API.ReopenPR(ctx, c.Owner, c.Name, n)
}
