package delivery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checks"
	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/review"
)

// Gate é uma condição de entrega (checks, revisão, E2E…).
type Gate interface {
	Name() string
	Check(ctx context.Context) (reasons []string, err error)
}

// GateFunc adapta uma função a Gate.
type GateFunc struct {
	Label string
	Fn    func(ctx context.Context) ([]string, error)
}

// Name implementa Gate.
func (g GateFunc) Name() string { return g.Label }

// Check implementa Gate.
func (g GateFunc) Check(ctx context.Context) ([]string, error) { return g.Fn(ctx) }

// ChecksGate exige os checks da revisão aprovados (T14-01).
func ChecksGate(s *checks.Store, featureID, revision string) Gate {
	return GateFunc{"checks", func(ctx context.Context) ([]string, error) {
		g, err := s.Gate(ctx, featureID, revision)
		if err != nil || !g.Blocked {
			return nil, err
		}
		return g.Reasons, nil
	}}
}

// ReviewGate exige a revisão independente aprovada para cada tarefa informada (T14-03).
func ReviewGate(s *review.Store, projectID, featureID, revision string, tasks []string, required bool) Gate {
	return GateFunc{"revisão", func(ctx context.Context) ([]string, error) {
		var out []string
		for _, t := range tasks {
			g, err := s.Gate(ctx, projectID, featureID, t, revision, required)
			if err != nil {
				return nil, err
			}
			for _, r := range g.Reasons {
				out = append(out, t+": "+r)
			}
		}
		return out, nil
	}}
}

// Origin publica "Aguardando merge" na plataforma de origem (tasksync.Publisher implementa).
type Origin interface {
	PublishDraftPR(ctx context.Context, project, keyBase, itemID, prURL string) error
}

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
}

// Deliverer entrega a feature.
type Deliverer struct {
	DB  DB
	PRs *gitops.PRPublisher
	Now func() time.Time
}

func (d *Deliverer) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

var shaRe = regexp.MustCompile(`^[0-9a-f]{40}$`)

// RecordCommit registra o commit de uma tarefa (idempotente por feature e SHA).
func (d *Deliverer) RecordCommit(ctx context.Context, projectID, featureID, taskKey, sha, branch, subject string) error {
	if !shaRe.MatchString(sha) || strings.TrimSpace(taskKey) == "" || strings.TrimSpace(branch) == "" {
		return &FieldError{"commit", "tarefa, branch e SHA (40 hexadecimais) são obrigatórios"}
	}
	if len(subject) > 300 {
		subject = subject[:300]
	}
	_, err := d.DB.Exec(ctx, `INSERT INTO feature_commits (project_id, feature_id, task_key, sha, branch, subject) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (feature_id, sha) DO NOTHING`,
		projectID, featureID, taskKey, sha, branch, subject)
	return persistence.MapError(err)
}

// FieldError é um erro de entrada.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// Request pede a entrega da tentativa da feature.
type Request struct {
	ProjectID, FeatureID string
	Attempt              int
	Branch, Base         string
	Revision             string // SHA da ponta da branch (a revisão entregue)
	Title                string
	ChangedFiles         []string // diff da feature, incluindo a documentação
	DocContent           string   // conteúdo do histórico commitado (se informado, precisa ser o gerado)
	Rules                []DocRule
	Gates                []Gate
	Origin               Origin
	OriginKeyBase        string
	OriginItemID         string
}

// Outcome é o resultado da entrega.
type Outcome struct {
	DeliveryID string              `json:"delivery_id"`
	Blocked    bool                `json:"blocked"`
	Reasons    []string            `json:"reasons"`
	DocGaps    []Gap               `json:"doc_gaps"`
	DocPath    string              `json:"doc_path"`
	Document   string              `json:"document"` // o histórico gerado, para ser commitado
	PR         *gitops.PullRequest `json:"pr,omitempty"`
	Reused     bool                `json:"reused"` // a PR já existia (repetição ou resposta perdida)
}

// Document gera o histórico vivo da feature (para o commit) e o caminho dele.
func (d *Deliverer) Document(ctx context.Context, projectID, featureID, revision, branch string) (path, content string, err error) {
	rep, err := BuildReport(ctx, d.DB, projectID, featureID, revision, branch)
	if err != nil {
		return "", "", err
	}
	return FeatureDocPath(rep.ExternalKey), rep.Render(), nil
}

// Deliver confere os gates e abre a Draft PR (idempotente). Sem documentação (do módulo e o
// histórico da feature) ou com algum gate pendente, a entrega fica bloqueada e nenhuma PR é aberta.
func (d *Deliverer) Deliver(ctx context.Context, req Request) (Outcome, error) {
	out := Outcome{Reasons: []string{}, DocGaps: []Gap{}}
	if strings.TrimSpace(req.Revision) == "" || strings.TrimSpace(req.Title) == "" {
		return out, &FieldError{"request", "revisão e título são obrigatórios"}
	}
	path, doc, err := d.Document(ctx, req.ProjectID, req.FeatureID, req.Revision, req.Branch)
	if err != nil {
		return out, err
	}
	out.DocPath, out.Document = path, doc
	rules := req.Rules
	if rules == nil {
		rules = DefaultDocRules()
	}
	// 1. Documentação por módulo e histórico da feature.
	out.DocGaps = DocsGaps(req.ChangedFiles, rules)
	for _, g := range out.DocGaps {
		out.Reasons = append(out.Reasons, fmt.Sprintf("o módulo %s mudou sem documentação (inclua um destes: %s)", g.Module, strings.Join(g.Expected, ", ")))
	}
	hasDoc := false
	for _, f := range req.ChangedFiles {
		hasDoc = hasDoc || f == path
	}
	switch {
	case !hasDoc:
		out.Reasons = append(out.Reasons, fmt.Sprintf("o histórico da feature (%s) não está no diff: gere-o e inclua-o no commit", path))
	case req.DocContent != "" && req.DocContent != doc:
		out.Reasons = append(out.Reasons, fmt.Sprintf("o histórico da feature (%s) está desatualizado em relação ao que foi registrado: gere-o de novo", path))
	}
	// 2. Gates de checks, revisão, E2E…
	for _, g := range req.Gates {
		rs, err := g.Check(ctx)
		if err != nil {
			return out, err
		}
		for _, r := range rs {
			out.Reasons = append(out.Reasons, g.Name()+": "+r)
		}
	}
	reasonsJSON := func() string { b, _ := json.Marshal(out.Reasons); return string(b) }
	if len(out.Reasons) > 0 {
		out.Blocked = true
		id, err := d.record(ctx, req, path, doc, "blocked", reasonsJSON(), nil, "")
		out.DeliveryID = id
		return out, err
	}
	// 3. Draft PR pelo ledger.
	body := d.prBody(ctx, req, path)
	res, pr, err := d.PRs.OpenDraft(ctx, gitops.PRRequest{ProjectID: req.ProjectID, FeatureID: req.FeatureID, Attempt: req.Attempt, Branch: req.Branch, Base: req.Base, Title: req.Title, Body: body})
	if err != nil {
		if errors.Is(err, effects.ErrBlocked) {
			out.Blocked = true
			out.Reasons = append(out.Reasons, "o estado da PR é inconclusivo: a repetição automática está bloqueada até uma decisão")
		}
		return out, err
	}
	out.PR = &pr
	out.Reused = res.Outcome == effects.OutcomeAlreadyConfirmed
	out.Reasons = []string{}
	n := pr.Number
	id, err := d.record(ctx, req, path, doc, "draft_open", "[]", &n, pr.URL)
	out.DeliveryID = id
	if err != nil {
		return out, err
	}
	if req.Origin != nil && req.OriginItemID != "" && pr.URL != "" {
		if err := req.Origin.PublishDraftPR(ctx, req.ProjectID, req.OriginKeyBase, req.OriginItemID, pr.URL); err != nil {
			return out, fmt.Errorf("a Draft PR existe, mas a origem não foi atualizada (repita: nada é duplicado): %w", err)
		}
	}
	return out, nil
}

func (d *Deliverer) record(ctx context.Context, req Request, docPath, doc, state, reasons string, pr *int, url string) (string, error) {
	var id string
	sum := sha256Hex(doc)
	err := d.DB.QueryRow(ctx, `INSERT INTO deliveries (project_id, feature_id, attempt, branch, revision, state, reasons, doc_path, doc_hash, pr_number, pr_url, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $12)
		ON CONFLICT (feature_id, attempt) DO UPDATE SET branch = $4, revision = $5, state = $6, reasons = $7::jsonb, doc_path = $8, doc_hash = $9, pr_number = $10, pr_url = $11, updated_at = $12
		RETURNING id`, req.ProjectID, req.FeatureID, req.Attempt, req.Branch, req.Revision, state, reasons, docPath, sum, pr, url, d.now()).Scan(&id)
	return id, persistence.MapError(err)
}

func (d *Deliverer) prBody(ctx context.Context, req Request, docPath string) string {
	rep, err := BuildReport(ctx, d.DB, req.ProjectID, req.FeatureID, req.Revision, req.Branch)
	var b strings.Builder
	b.WriteString("Rascunho aberto pela La fabrique. O merge é humano.\n\n")
	if err == nil {
		fmt.Fprintf(&b, "Feature: %s — %s\n\n", rep.ExternalKey, rep.Title)
		b.WriteString("Commits por tarefa:\n")
		for _, t := range rep.Tasks {
			for _, c := range t.Commits {
				sha := c.SHA
				if len(sha) > 12 {
					sha = sha[:12]
				}
				fmt.Fprintf(&b, "- %s: `%s` %s\n", t.Key, sha, c.Subject)
			}
		}
		b.WriteString("\nChecks da revisão:\n")
		for _, c := range rep.Checks {
			fmt.Fprintf(&b, "- %s: %s\n", c.Name, c.Status)
		}
	}
	fmt.Fprintf(&b, "\nHistórico da feature: `%s`\n", docPath)
	return b.String()
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
