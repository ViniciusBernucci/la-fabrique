package gitops

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
)

// PullRequest é o que a fábrica precisa saber de uma PR.
type PullRequest struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
	Draft  bool   `json:"draft"`
	State  string `json:"state"` // open | closed
	Merged bool   `json:"merged"`
	Head   string `json:"head"`
	Base   string `json:"base"`
	Title  string `json:"title"`
}

// PRInput descreve a PR a abrir. A fábrica só abre Draft: o merge é humano.
type PRInput struct {
	Head  string
	Base  string
	Title string
	Body  string
}

// PRAPI é o que o publicador de PR precisa do GitHub (GitHubAPI implementa).
type PRAPI interface {
	FindPR(ctx context.Context, owner, name, head, base string) (*PullRequest, error)
	CreateDraftPR(ctx context.Context, owner, name string, in PRInput) (PullRequest, error)
}

// ErrPRExists: o GitHub recusou a criação porque já existe uma PR aberta para a branch.
var ErrPRExists = errors.New("já existe uma PR aberta para esta branch")

type ghPR struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	Draft   bool   `json:"draft"`
	State   string `json:"state"`
	Merged  bool   `json:"merged"`
	Title   string `json:"title"`
	Head    struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

func (p ghPR) toPR() PullRequest {
	return PullRequest{Number: p.Number, URL: p.HTMLURL, Draft: p.Draft, State: p.State, Merged: p.Merged, Head: p.Head.Ref, Base: p.Base.Ref, Title: p.Title}
}

// FindPR procura uma PR ABERTA da branch `head` para `base` (nil se não houver).
func (a *GitHubAPI) FindPR(ctx context.Context, owner, name, head, base string) (*PullRequest, error) {
	if !ownerPattern.MatchString(owner) || !namePattern.MatchString(name) || validRefName(head) != nil || validRefName(base) != nil {
		return nil, ErrInvalidRepo
	}
	q := url.Values{"head": {owner + ":" + head}, "base": {base}, "state": {"open"}}
	var list []ghPR
	if err := a.get(ctx, "/repos/"+owner+"/"+name+"/pulls?"+q.Encode(), &list); err != nil {
		return nil, err
	}
	for _, p := range list {
		if p.Head.Ref == head && p.Base.Ref == base && p.State == "open" {
			pr := p.toPR()
			return &pr, nil
		}
	}
	return nil, nil
}

// CreateDraftPR abre a PR como rascunho.
func (a *GitHubAPI) CreateDraftPR(ctx context.Context, owner, name string, in PRInput) (PullRequest, error) {
	if !ownerPattern.MatchString(owner) || !namePattern.MatchString(name) || validRefName(in.Head) != nil || validRefName(in.Base) != nil || in.Head == in.Base {
		return PullRequest{}, ErrInvalidRepo
	}
	body, _ := json.Marshal(map[string]any{"title": in.Title, "head": in.Head, "base": in.Base, "body": in.Body, "draft": true})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.Base+"/repos/"+owner+"/"+name+"/pulls", bytes.NewReader(body))
	if err != nil {
		return PullRequest{}, ErrInvalidRepo
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+a.Token.Reveal())
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return PullRequest{}, fmt.Errorf("%w: API do GitHub indisponível", ErrGit)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return PullRequest{}, ErrAuth
	case resp.StatusCode == http.StatusNotFound:
		return PullRequest{}, ErrNotFound
	case resp.StatusCode == http.StatusUnprocessableEntity:
		if strings.Contains(string(data), "pull request already exists") {
			return PullRequest{}, ErrPRExists
		}
		return PullRequest{}, fmt.Errorf("%w: a API recusou a PR (422)", ErrRejected)
	case resp.StatusCode != http.StatusCreated:
		return PullRequest{}, fmt.Errorf("%w: API do GitHub respondeu %d", ErrGit, resp.StatusCode)
	}
	var pr ghPR
	if json.Unmarshal(data, &pr) != nil || pr.Number == 0 {
		return PullRequest{}, fmt.Errorf("%w: resposta da API fora do formato", ErrGit)
	}
	return pr.toPR(), nil
}

// PRRequest pede a abertura (idempotente) da Draft PR da feature.
type PRRequest struct {
	ProjectID string
	FeatureID string
	Attempt   int
	Branch    string
	Base      string
	Title     string
	Body      string
}

// PRPublisher abre a Draft PR pelo ledger de efeitos: a chave é o efeito lógico (a branch da
// tentativa), então repetir encontra a mesma PR e a prova por busca decide quando a resposta se perde.
type PRPublisher struct {
	API    PRAPI
	Owner  string
	Name   string
	Ledger *effects.Ledger
}

// BusinessKey é a chave estável da PR da tentativa da feature.
func (r PRRequest) BusinessKey() string {
	return fmt.Sprintf("%s:%s:%d:pr:%s", r.ProjectID, r.FeatureID, r.Attempt, r.Branch)
}

func (r PRRequest) guard() error {
	want, err := FeatureBranch(r.FeatureID, r.Attempt)
	if err != nil {
		return err
	}
	if r.Branch != want || r.Branch == r.Base || ValidFeatureBranch(r.Branch) != nil {
		return fmt.Errorf("%w: %q (esperada %q)", ErrBranchForbidden, r.Branch, want)
	}
	if r.ProjectID == "" || strings.TrimSpace(r.Title) == "" || len(r.Title) > 250 || len(r.Body) > 60000 {
		return fmt.Errorf("%w: título ou corpo da PR", ErrBadCommit)
	}
	return nil
}

// OpenDraft abre (ou encontra) a Draft PR. O resultado traz a PR e como foi obtida.
func (p *PRPublisher) OpenDraft(ctx context.Context, req PRRequest) (effects.Result, PullRequest, error) {
	if err := req.guard(); err != nil {
		return effects.Result{}, PullRequest{}, err
	}
	body := map[string]any{"repo": p.Owner + "/" + p.Name, "branch": req.Branch, "base": req.Base, "feature": req.FeatureID, "attempt": req.Attempt}
	res, err := p.Ledger.Run(ctx, effects.Spec{ProjectID: req.ProjectID, BusinessKey: req.BusinessKey(), Operation: "create_pr", Request: body}, &prAdapter{p: p, req: req})
	if err != nil {
		return res, PullRequest{}, err
	}
	var pr PullRequest
	var proof map[string]any
	if res.Entry.Proof != nil {
		_ = json.Unmarshal(res.Entry.Proof, &proof)
	}
	if n, ok := proof["number"].(float64); ok {
		pr.Number = int(n)
	}
	pr.URL, _ = proof["url"].(string)
	pr.Draft, _ = proof["draft"].(bool)
	pr.Head, pr.Base = req.Branch, req.Base
	return res, pr, nil
}

type prAdapter struct {
	p   *PRPublisher
	req PRRequest
}

func (a *prAdapter) Apply(ctx context.Context, op effects.Op) (any, error) {
	if err := a.req.guard(); err != nil {
		return nil, fmt.Errorf("%w: %v", effects.ErrNotApplied, err)
	}
	pr, err := a.p.API.CreateDraftPR(ctx, a.p.Owner, a.p.Name, PRInput{Head: a.req.Branch, Base: a.req.Base, Title: a.req.Title, Body: a.req.Body})
	switch {
	case err == nil:
		return map[string]any{"number": pr.Number, "url": pr.URL, "draft": pr.Draft}, nil
	case errors.Is(err, ErrPRExists):
		// A PR já existe (uma tentativa anterior chegou ao destino): a prova por busca confirma.
		return nil, err
	case errors.Is(err, ErrAuth), errors.Is(err, ErrNotFound), errors.Is(err, ErrRejected), errors.Is(err, ErrInvalidRepo):
		return nil, fmt.Errorf("%w: %v", effects.ErrNotApplied, err)
	}
	return nil, err // timeout ou queda: resultado desconhecido, o ledger verifica
}

func (a *prAdapter) Verify(ctx context.Context, op effects.Op) (effects.Verdict, error) {
	pr, err := a.p.API.FindPR(ctx, a.p.Owner, a.p.Name, a.req.Branch, a.req.Base)
	if err != nil {
		return effects.Verdict{Status: effects.VerdictInconclusive, Evidence: "não foi possível consultar as PRs da branch: " + err.Error()}, nil
	}
	if pr == nil {
		return effects.Verdict{Status: effects.VerdictAbsent}, nil
	}
	return effects.Verdict{Status: effects.VerdictConfirmed, Proof: map[string]any{"number": pr.Number, "url": pr.URL, "draft": pr.Draft}}, nil
}
