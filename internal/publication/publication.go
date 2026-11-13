// Package publication liga consolidação, gates e GitHub (AJ06-04, AJ09-03): só publica a cabeça
// candidata depois de os testes integrados e os gates por trilha referenciarem exatamente essa cabeça.
// A feature abre uma única Draft PR; uma correção atualiza a MESMA PR por fast-forward, nunca com força,
// e conserva o vínculo entre rodada, Issue, contratos, commits e cabeça publicada. Se a PR foi integrada
// ou fechada antes do envio, o avanço é cancelado e o trabalho fica preservado para tratamento humano.
package publication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/trackgates"
)

// Erros.
var (
	ErrBlocked   = errors.New("publicação bloqueada pelos gates")
	ErrNoPR      = errors.New("a feature ainda não tem Draft PR publicada")
	ErrPRClosed  = errors.New("a PR foi integrada ou fechada: o avanço da correção foi cancelado e o trabalho preservado")
	ErrNotAhead  = errors.New("a cabeça candidata não descende da cabeça publicada: nada é enviado sem força")
	ErrStaleHead = errors.New("a cabeça candidata mudou depois dos gates: revalide")
)

// BlockedError carrega os motivos do bloqueio.
type BlockedError struct{ Problems []string }

func (e *BlockedError) Error() string {
	return ErrBlocked.Error() + ": " + strings.Join(e.Problems, "; ")
}
func (e *BlockedError) Unwrap() error { return ErrBlocked }

// Candidates devolve a cabeça candidata e as tentativas cuja evidência perdeu validade (consolidate).
type Candidates interface {
	Candidate(ctx context.Context, featureID string) (head string, revalidate []string, err error)
	Descends(ctx context.Context, ancestor, head string) (bool, error)
	Range(ctx context.Context, from, to string) ([]string, error)
}

// Pusher publica a branch da feature (gitops.Publisher).
type Pusher interface {
	Push(ctx context.Context, req gitops.PushRequest) (effects.Result, error)
}

// PRs abre a Draft PR (gitops.PRPublisher).
type PRs interface {
	OpenDraft(ctx context.Context, req gitops.PRRequest) (effects.Result, gitops.PullRequest, error)
}

// PRState consulta o estado atual da PR para revalidar antes de publicar.
type PRState interface {
	ReadPR(ctx context.Context, number int) (gitops.PRState, error)
}

// DB é o que o serviço precisa do PostgreSQL.
type DB interface{ persistence.DBTX }

// Service publica.
type Service struct {
	DB    DB
	Gates *trackgates.Store
	Cands Candidates
	Push  Pusher
	PRs   PRs
	State PRState
}

// Request é o pedido da Draft PR.
type Request struct {
	ProjectID, FeatureID string
	Attempt              int
	Base                 string
	Title, Body          string
	Required             []flow.Track
	AttemptTrack         map[string]flow.Track // tentativa → trilha (para as que a consolidação alterou)
}

// Result é a publicação.
type Result struct {
	ID       string `json:"id"`
	Head     string `json:"head"`
	Branch   string `json:"branch"`
	PRNumber int    `json:"pr_number"`
	PRURL    string `json:"pr_url"`
	Existing bool   `json:"existing"` // a Draft PR já existia (repetição)
}

func (s *Service) gate(ctx context.Context, featureID, head string, req []flow.Track, attemptTrack map[string]flow.Track, reval []string) (trackgates.Result, error) {
	changed := map[flow.Track]bool{}
	for _, a := range reval {
		if tr, ok := attemptTrack[a]; ok {
			changed[tr] = true
		}
	}
	return s.Gates.Gate(ctx, trackgates.Input{FeatureID: featureID, Head: head, Required: req, Changed: changed})
}

// PublishDraft envia a cabeça candidata e abre (ou encontra) a única Draft PR da feature.
func (s *Service) PublishDraft(ctx context.Context, r Request) (Result, error) {
	var res Result
	head, reval, err := s.Cands.Candidate(ctx, r.FeatureID)
	if err != nil {
		return res, err
	}
	g, err := s.gate(ctx, r.FeatureID, head, r.Required, r.AttemptTrack, reval)
	if err != nil {
		return res, err
	}
	if !g.Allowed {
		return res, &BlockedError{Problems: g.Problems}
	}
	branch, err := gitops.FeatureBranch(r.FeatureID, r.Attempt)
	if err != nil {
		return res, err
	}
	// Revalida: a cabeça candidata não pode ter mudado entre os gates e o envio.
	if cur, _, err := s.Cands.Candidate(ctx, r.FeatureID); err != nil || cur != head {
		return res, ErrStaleHead
	}
	if _, err := s.Push.Push(ctx, gitops.PushRequest{ProjectID: r.ProjectID, FeatureID: r.FeatureID, Attempt: r.Attempt, Branch: branch, SHA: head}); err != nil {
		return res, err
	}
	_, pr, err := s.PRs.OpenDraft(ctx, gitops.PRRequest{ProjectID: r.ProjectID, FeatureID: r.FeatureID, Attempt: r.Attempt, Branch: branch, Base: r.Base, Title: r.Title, Body: r.Body})
	if err != nil {
		return res, err
	}
	gj, _ := json.Marshal(g)
	var id string
	var existing bool
	err = s.DB.QueryRow(ctx, `INSERT INTO publications (project_id, feature_id, kind, branch, head_sha, pr_number, pr_url, gate) VALUES ($1,$2,'draft',$3,$4,$5,$6,$7)
		ON CONFLICT (feature_id) WHERE kind = 'draft' AND state = 'published' DO UPDATE SET pr_number = publications.pr_number RETURNING id::text, (xmax <> 0)`,
		r.ProjectID, r.FeatureID, branch, head, pr.Number, pr.URL, string(gj)).Scan(&id, &existing)
	if err != nil {
		return res, persistence.MapError(err)
	}
	return Result{ID: id, Head: head, Branch: branch, PRNumber: pr.Number, PRURL: pr.URL, Existing: existing}, nil
}

// Correction é o pedido de publicar os commits de uma correção na PR original.
type Correction struct {
	ProjectID, FeatureID string
	Attempt              int
	RoundKey, IssueKey   string
	Contracts            []string // hashes dos contratos usados pela correção
	Required             []flow.Track
	AttemptTrack         map[string]flow.Track
}

// PublishCorrection atualiza a PR original. Antes de enviar revalida o estado da PR; se foi integrada ou
// fechada, cancela o avanço (o trabalho segue preservado na branch/cabeça candidata).
func (s *Service) PublishCorrection(ctx context.Context, c Correction) (Result, error) {
	var res Result
	var prev struct {
		head, branch, url string
		pr                int
	}
	err := s.DB.QueryRow(ctx, `SELECT head_sha, branch, pr_url, COALESCE(pr_number,0) FROM publications
		WHERE feature_id::text=$1 AND state='published' ORDER BY created_at DESC LIMIT 1`, c.FeatureID).Scan(&prev.head, &prev.branch, &prev.url, &prev.pr)
	if err != nil {
		return res, ErrNoPR
	}
	head, reval, err := s.Cands.Candidate(ctx, c.FeatureID)
	if err != nil {
		return res, err
	}
	if head == prev.head {
		return res, fmt.Errorf("%w: a correção não produziu commits novos", persistence.ErrInvalidInput)
	}
	g, err := s.gate(ctx, c.FeatureID, head, c.Required, c.AttemptTrack, reval)
	if err != nil {
		return res, err
	}
	if !g.Allowed {
		return res, &BlockedError{Problems: g.Problems}
	}
	// Revalida o estado da PR imediatamente antes de publicar.
	st, err := s.State.ReadPR(ctx, prev.pr)
	if err != nil {
		return res, fmt.Errorf("não foi possível revalidar a PR antes de publicar: %w", err)
	}
	if st.Merged || st.State == "closed" {
		gj, _ := json.Marshal(g)
		_, _ = s.DB.Exec(ctx, `INSERT INTO publications (project_id, feature_id, kind, state, round_key, issue_key, branch, head_sha, previous_head, pr_number, pr_url, gate, detail)
			VALUES ($1,$2,'correction','cancelled',$3,$4,$5,$6,$7,$8,$9,$10,$11)`, c.ProjectID, c.FeatureID, c.RoundKey, c.IssueKey, prev.branch, head, prev.head, prev.pr, prev.url, string(gj),
			"PR integrada ou fechada antes da publicação: trabalho preservado para tratamento humano")
		return res, ErrPRClosed
	}
	if ok, err := s.Cands.Descends(ctx, prev.head, head); err != nil || !ok {
		if err != nil {
			return res, err
		}
		return res, ErrNotAhead
	}
	commits, err := s.Cands.Range(ctx, prev.head, head)
	if err != nil {
		return res, err
	}
	if _, err := s.Push.Push(ctx, gitops.PushRequest{ProjectID: c.ProjectID, FeatureID: c.FeatureID, Attempt: c.Attempt, Branch: prev.branch, SHA: head}); err != nil {
		return res, err
	}
	cj, _ := json.Marshal(commits)
	kj, _ := json.Marshal(nonNil(c.Contracts))
	gj, _ := json.Marshal(g)
	var id string
	if err := s.DB.QueryRow(ctx, `INSERT INTO publications (project_id, feature_id, kind, round_key, issue_key, branch, head_sha, previous_head, pr_number, pr_url, commits, contracts, gate)
		VALUES ($1,$2,'correction',$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (feature_id, kind, head_sha) WHERE state = 'published' DO UPDATE SET detail = publications.detail RETURNING id::text`,
		c.ProjectID, c.FeatureID, c.RoundKey, c.IssueKey, prev.branch, head, prev.head, prev.pr, prev.url, string(cj), string(kj), string(gj)).Scan(&id); err != nil {
		return res, persistence.MapError(err)
	}
	// O vínculo com a rodada: a cabeça publicada é o que o merge deverá conter para a correção contar como incorporada.
	if _, err := s.DB.Exec(ctx, `UPDATE pr_rounds SET published_head = $2 WHERE round_key = $1 AND watch_id IN (SELECT id FROM pr_watch WHERE feature_id::text = $3)`, c.RoundKey, head, c.FeatureID); err != nil {
		return res, persistence.MapError(err)
	}
	return Result{ID: id, Head: head, Branch: prev.branch, PRNumber: prev.pr, PRURL: prev.url}, nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
