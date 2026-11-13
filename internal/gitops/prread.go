package gitops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ErrRateLimited indica limite de requisições do GitHub; RetryAfter orienta o backoff.
type ErrRateLimited struct{ RetryAfter time.Duration }

func (e *ErrRateLimited) Error() string {
	return fmt.Sprintf("limite de requisições do GitHub atingido (tente de novo em %s)", e.RetryAfter.Round(time.Second))
}

// PRState é o estado de uma PR lido do GitHub (AJ09-01).
type PRState struct {
	Number   int
	State    string // open | closed
	Draft    bool
	Merged   bool
	MergeSHA string
	HeadSHA  string
	HeadRef  string
	Base     string
	MergedAt time.Time
	ClosedAt time.Time
	URL      string
}

// Review é uma revisão formal.
type Review struct {
	ID       int64
	Author   string
	State    string // APPROVED | CHANGES_REQUESTED | COMMENTED | DISMISSED | PENDING
	Body     string
	URL      string
	CommitID string
	At       time.Time
}

// PRComment é um comentário da conversa (ReviewID 0) ou de linha de uma revisão.
type PRComment struct {
	ID       int64
	ReviewID int64
	Author   string
	Body     string
	URL      string
	At       time.Time
	Line     bool
}

type ghActor struct {
	Login string `json:"login"`
}

// getJSON lê com tratamento de limite de requisições.
func (a *GitHubAPI) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.Base+path, nil)
	if err != nil {
		return ErrInvalidRepo
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+a.Token.Reveal())
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%w: API do GitHub indisponível", ErrGit)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusTooManyRequests || (resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0"):
		wait := time.Minute
		if s := resp.Header.Get("Retry-After"); s != "" {
			if n, e := strconv.Atoi(s); e == nil && n > 0 {
				wait = time.Duration(n) * time.Second
			}
		} else if s := resp.Header.Get("X-RateLimit-Reset"); s != "" {
			if n, e := strconv.ParseInt(s, 10, 64); e == nil {
				if d := time.Until(time.Unix(n, 0)); d > 0 {
					wait = d
				}
			}
		}
		return &ErrRateLimited{RetryAfter: wait}
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return ErrAuth
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("%w: API do GitHub respondeu %d", ErrGit, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil || json.Unmarshal(data, out) != nil {
		return fmt.Errorf("%w: resposta da API fora do formato", ErrGit)
	}
	return nil
}

func (a *GitHubAPI) okRepo(owner, name string, number int) error {
	if !ownerPattern.MatchString(owner) || !namePattern.MatchString(name) || number < 1 {
		return ErrInvalidRepo
	}
	return nil
}

// ReadPR lê o estado da PR, com cabeça e SHA de merge.
func (a *GitHubAPI) ReadPR(ctx context.Context, owner, name string, number int) (PRState, error) {
	if err := a.okRepo(owner, name, number); err != nil {
		return PRState{}, err
	}
	var p struct {
		ghPR
		MergeCommitSHA string     `json:"merge_commit_sha"`
		MergedAt       *time.Time `json:"merged_at"`
		ClosedAt       *time.Time `json:"closed_at"`
		Head           struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
	}
	if err := a.getJSON(ctx, fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, name, number), &p); err != nil {
		return PRState{}, err
	}
	s := PRState{Number: p.Number, State: p.State, Draft: p.Draft, Merged: p.Merged || p.MergedAt != nil, MergeSHA: p.MergeCommitSHA,
		HeadSHA: p.Head.SHA, HeadRef: p.Head.Ref, Base: p.ghPR.Base.Ref, URL: p.HTMLURL}
	if p.MergedAt != nil {
		s.MergedAt = *p.MergedAt
	}
	if p.ClosedAt != nil {
		s.ClosedAt = *p.ClosedAt
	}
	return s, nil
}

const maxPages = 5

// page percorre até maxPages páginas de 100; more indica que ainda há itens (próxima leitura continua).
func page[T any](ctx context.Context, a *GitHubAPI, path string, keep func(T) bool) (out []T, more bool, err error) {
	for pg := 1; pg <= maxPages; pg++ {
		var list []T
		sep := "?"
		if len(path) > 0 && containsQuery(path) {
			sep = "&"
		}
		if err = a.getJSON(ctx, path+sep+"per_page=100&page="+strconv.Itoa(pg), &list); err != nil {
			return out, false, err
		}
		for _, it := range list {
			if keep(it) {
				out = append(out, it)
			}
		}
		if len(list) < 100 {
			return out, false, nil
		}
	}
	return out, true, nil
}

func containsQuery(p string) bool {
	for _, c := range p {
		if c == '?' {
			return true
		}
	}
	return false
}

// ListReviews devolve as revisões formais com ID maior que `after`, em ordem crescente.
func (a *GitHubAPI) ListReviews(ctx context.Context, owner, name string, number int, after int64) ([]Review, bool, error) {
	if err := a.okRepo(owner, name, number); err != nil {
		return nil, false, err
	}
	type gh struct {
		ID       int64     `json:"id"`
		User     ghActor   `json:"user"`
		State    string    `json:"state"`
		Body     string    `json:"body"`
		HTMLURL  string    `json:"html_url"`
		CommitID string    `json:"commit_id"`
		At       time.Time `json:"submitted_at"`
	}
	list, more, err := page(ctx, a, fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, name, number), func(r gh) bool { return r.ID > after })
	out := make([]Review, 0, len(list))
	for _, r := range list {
		out = append(out, Review{ID: r.ID, Author: r.User.Login, State: r.State, Body: r.Body, URL: r.HTMLURL, CommitID: r.CommitID, At: r.At})
	}
	return out, more, err
}

// ListConversation devolve comentários da conversa com ID maior que `after`.
func (a *GitHubAPI) ListConversation(ctx context.Context, owner, name string, number int, after int64) ([]PRComment, bool, error) {
	return a.comments(ctx, owner, name, number, after, false)
}

// ListLineComments devolve comentários de linha (vinculados a uma revisão) com ID maior que `after`.
func (a *GitHubAPI) ListLineComments(ctx context.Context, owner, name string, number int, after int64) ([]PRComment, bool, error) {
	return a.comments(ctx, owner, name, number, after, true)
}

func (a *GitHubAPI) comments(ctx context.Context, owner, name string, number int, after int64, line bool) ([]PRComment, bool, error) {
	if err := a.okRepo(owner, name, number); err != nil {
		return nil, false, err
	}
	type gh struct {
		ID       int64     `json:"id"`
		ReviewID int64     `json:"pull_request_review_id"`
		User     ghActor   `json:"user"`
		Body     string    `json:"body"`
		HTMLURL  string    `json:"html_url"`
		At       time.Time `json:"created_at"`
	}
	kind := "issues"
	if line {
		kind = "pulls"
	}
	list, more, err := page(ctx, a, fmt.Sprintf("/repos/%s/%s/%s/%d/comments", owner, name, kind, number), func(c gh) bool { return c.ID > after })
	out := make([]PRComment, 0, len(list))
	for _, c := range list {
		out = append(out, PRComment{ID: c.ID, ReviewID: c.ReviewID, Author: c.User.Login, Body: c.Body, URL: c.HTMLURL, At: c.At, Line: line})
	}
	return out, more, err
}

// Permission devolve a permissão do usuário no repositório (admin, maintain, write, triage, read, none).
func (a *GitHubAPI) Permission(ctx context.Context, owner, name, login string) (string, error) {
	if !ownerPattern.MatchString(owner) || !namePattern.MatchString(name) || !ownerPattern.MatchString(login) {
		return "none", ErrInvalidRepo
	}
	var out struct {
		Permission string `json:"permission"`
	}
	err := a.getJSON(ctx, fmt.Sprintf("/repos/%s/%s/collaborators/%s/permission", owner, name, url.PathEscape(login)), &out)
	if errors.Is(err, ErrNotFound) {
		return "none", nil
	}
	return out.Permission, err
}

// Contains informa se a cabeça `head` contém o commit `ancestor` (comparação do GitHub).
func (a *GitHubAPI) Contains(ctx context.Context, owner, name, ancestor, head string) (bool, error) {
	if !ownerPattern.MatchString(owner) || !namePattern.MatchString(name) || !isSHA(ancestor) || !isSHA(head) {
		return false, ErrInvalidRepo
	}
	var out struct {
		Status string `json:"status"`
	}
	if err := a.getJSON(ctx, fmt.Sprintf("/repos/%s/%s/compare/%s...%s", owner, name, ancestor, head), &out); err != nil {
		return false, err
	}
	return out.Status == "identical" || out.Status == "ahead", nil
}

func isSHA(s string) bool {
	if len(s) < 7 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
