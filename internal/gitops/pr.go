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

// MergeInfo é o que se sabe do merge de uma PR.
type MergeInfo struct {
	PullRequest
	MergeSHA string
}

// GetPR lê uma PR pelo número, com o SHA do commit de integração quando houver merge.
func (a *GitHubAPI) GetPR(ctx context.Context, owner, name string, number int) (MergeInfo, error) {
	if !ownerPattern.MatchString(owner) || !namePattern.MatchString(name) || number < 1 {
		return MergeInfo{}, ErrInvalidRepo
	}
	var p struct {
		ghPR
		MergeCommitSHA string `json:"merge_commit_sha"`
	}
	if err := a.get(ctx, fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, name, number), &p); err != nil {
		return MergeInfo{}, err
	}
	return MergeInfo{PullRequest: p.toPR(), MergeSHA: p.MergeCommitSHA}, nil
}

// PRNumberFromURL extrai o número de uma URL de PR (.../pull/<n>).
func PRNumberFromURL(u string) (int, bool) {
	i := strings.LastIndex(u, "/pull/")
	if i < 0 {
		return 0, false
	}
	n := 0
	rest := u[i+len("/pull/"):]
	if rest == "" {
		return 0, false
	}
	for _, c := range rest {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
		if n > 1<<30 {
			return 0, false
		}
	}
	return n, n > 0
}

// Comment é um comentário de revisão ou de conversa da PR.
type Comment struct {
	ID     int64
	Author string
	Body   string
	Review bool // comentário de linha de revisão (e não da conversa)
}

type ghComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

// ListComments lê os comentários da conversa e da revisão da PR (até 100 de cada, os mais
// recentes primeiro não são garantidos: o chamador deduplica por identificador).
func (a *GitHubAPI) ListComments(ctx context.Context, owner, name string, number int) ([]Comment, error) {
	if !ownerPattern.MatchString(owner) || !namePattern.MatchString(name) || number < 1 {
		return nil, ErrInvalidRepo
	}
	var out []Comment
	for _, src := range []struct {
		path   string
		review bool
	}{{"issues", false}, {"pulls", true}} {
		var list []ghComment
		if err := a.get(ctx, fmt.Sprintf("/repos/%s/%s/%s/%d/comments?per_page=100", owner, name, src.path, number), &list); err != nil {
			return nil, err
		}
		for _, c := range list {
			out = append(out, Comment{ID: c.ID, Author: c.User.Login, Body: c.Body, Review: src.review})
		}
	}
	return out, nil
}

// ReopenPR reabre uma PR fechada sem merge. É idempotente (reabrir uma aberta não muda nada).
func (a *GitHubAPI) ReopenPR(ctx context.Context, owner, name string, number int) error {
	if !ownerPattern.MatchString(owner) || !namePattern.MatchString(name) || number < 1 {
		return ErrInvalidRepo
	}
	body, _ := json.Marshal(map[string]any{"state": "open"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, fmt.Sprintf("%s/repos/%s/%s/pulls/%d", a.Base, owner, name, number), bytes.NewReader(body))
	if err != nil {
		return ErrInvalidRepo
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+a.Token.Reveal())
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%w: API do GitHub indisponível", ErrGit)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return ErrAuth
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode == http.StatusUnprocessableEntity:
		return fmt.Errorf("%w: o GitHub recusou reabrir a PR (a branch pode ter sido apagada)", ErrRejected)
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("%w: API do GitHub respondeu %d", ErrGit, resp.StatusCode)
	}
	return nil
}
