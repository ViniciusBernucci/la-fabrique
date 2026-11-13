// Package prcycle é o ciclo GitHub da Sentinela (LF-AJ-09): consulta as PRs gerenciadas, fecha
// rodadas de revisão, cria a Issue REPROVADO_PR por rodada na origem e conclui após o merge.
//
// Comentários, comandos e textos das revisões são insumos: autoria e permissão são verificadas no
// GitHub e nada aqui executa correção — liberar a correção é decisão humana na origem (AJ01-03).
package prcycle

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// Reader é o que o ciclo precisa do GitHub (GitHubReader implementa).
type Reader interface {
	Repo() string
	ReadPR(ctx context.Context, number int) (gitops.PRState, error)
	ListReviews(ctx context.Context, number int, after int64) ([]gitops.Review, bool, error)
	ListConversation(ctx context.Context, number int, after int64) ([]gitops.PRComment, bool, error)
	ListLineComments(ctx context.Context, number int, after int64) ([]gitops.PRComment, bool, error)
	Permission(ctx context.Context, login string) (string, error)
	Contains(ctx context.Context, ancestor, head string) (bool, error)
}

// Origin é o que o ciclo precisa da origem (Jira/ClickUp): criar a Issue e provar que existe.
type Origin interface {
	integrations.ItemCreator
	ListItems(ctx context.Context, container string, opts integrations.ListOptions) (integrations.ItemPage, error)
}

// Target reúne o que é específico do projeto.
type Target struct {
	Reader    Reader
	Origin    Origin
	Container string          // chave do projeto Jira ou lista ClickUp
	Map       flow.ProjectMap // para o nome bruto do estado REPROVADO_PR
	// Register rastreia a Issue criada (issuesync.RegisterCorrection) vinculada à Issue original.
	Register func(ctx context.Context, it integrations.Item, parentKey string) error
}

// GitHubReader adapta gitops.GitHubAPI ao Reader de um repositório.
type GitHubReader struct {
	API         *gitops.GitHubAPI
	Owner, Name string
}

func (g GitHubReader) Repo() string { return g.Owner + "/" + g.Name }
func (g GitHubReader) ReadPR(ctx context.Context, n int) (gitops.PRState, error) {
	return g.API.ReadPR(ctx, g.Owner, g.Name, n)
}
func (g GitHubReader) ListReviews(ctx context.Context, n int, after int64) ([]gitops.Review, bool, error) {
	return g.API.ListReviews(ctx, g.Owner, g.Name, n, after)
}
func (g GitHubReader) ListConversation(ctx context.Context, n int, after int64) ([]gitops.PRComment, bool, error) {
	return g.API.ListConversation(ctx, g.Owner, g.Name, n, after)
}
func (g GitHubReader) ListLineComments(ctx context.Context, n int, after int64) ([]gitops.PRComment, bool, error) {
	return g.API.ListLineComments(ctx, g.Owner, g.Name, n, after)
}
func (g GitHubReader) Permission(ctx context.Context, login string) (string, error) {
	return g.API.Permission(ctx, g.Owner, g.Name, login)
}
func (g GitHubReader) Contains(ctx context.Context, a, h string) (bool, error) {
	return g.API.Contains(ctx, g.Owner, g.Name, a, h)
}

// DB é o que o serviço precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Service é o ciclo de PRs.
type Service struct {
	DB       DB
	Ledger   *effects.Ledger
	Sentinel *sentinel.Coordinator
	Now      func() time.Time
	Batch    int // PRs por execução da rotina (limite de leitura; padrão 20)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) batch() int {
	if s.Batch > 0 {
		return s.Batch
	}
	return 20
}

// commandPattern reconhece o comando explícito de encerramento de rodada.
var commandPattern = regexp.MustCompile(`(?mi)^\s*/fabrique\s+(fix|corrigir)\b`)

// IsCommand informa se o texto contém o comando de encerramento de rodada.
func IsCommand(body string) bool { return commandPattern.MatchString(body) }

func canWrite(perm string) bool {
	switch perm {
	case "admin", "maintain", "write":
		return true
	}
	return false
}

// Watch é uma PR gerenciada.
type Watch struct {
	ID, ProjectID, IssueKey, FeatureID, Repo, Base, Branch, State, HeadSHA string
	Number                                                                 int
	Draft                                                                  bool
	MergeSHA, MergedHead                                                   string
	ReviewCursor, CommentCursor, LineCursor                                int64
	ConcludedAt                                                            *time.Time
}

// Link registra (idempotente) uma PR como gerenciada.
func (s *Service) Link(ctx context.Context, projectID, featureID string, number int, issueKey, branch string) error {
	if number < 1 || strings.TrimSpace(issueKey) == "" {
		return fmt.Errorf("%w: PR e Issue original são obrigatórias", persistence.ErrInvalidInput)
	}
	var fid any
	if featureID != "" {
		fid = featureID
	}
	_, err := s.DB.Exec(ctx, `INSERT INTO pr_watch (project_id, pr_number, feature_id, issue_key, branch) VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (project_id, pr_number) DO NOTHING`, projectID, number, fid, issueKey, branch)
	return persistence.MapError(err)
}

// SyncDeliveries traz para a lista as PRs das entregas (Draft aberta) cuja feature tem Issue rastreada.
func (s *Service) SyncDeliveries(ctx context.Context, projectID string) (int, error) {
	tag, err := s.DB.Exec(ctx, `INSERT INTO pr_watch (project_id, pr_number, feature_id, issue_key, branch)
		SELECT d.project_id, d.pr_number, d.feature_id, fi.external_key, d.branch
		FROM deliveries d JOIN flow_issues fi ON fi.project_id = d.project_id AND fi.feature_id = d.feature_id AND fi.kind = 'feature'
		WHERE d.project_id = $1 AND d.pr_number IS NOT NULL
		ON CONFLICT (project_id, pr_number) DO NOTHING`, projectID)
	return int(tag.RowsAffected()), persistence.MapError(err)
}

const watchCols = `id::text, project_id::text, issue_key, COALESCE(feature_id::text,''), repo, base, branch, state, head_sha, pr_number, draft, merge_sha, merged_head_sha, review_cursor, comment_cursor, line_cursor, concluded_at`

type scanner interface{ Scan(dest ...any) error }

func scanWatch(r scanner) (Watch, error) {
	var w Watch
	err := r.Scan(&w.ID, &w.ProjectID, &w.IssueKey, &w.FeatureID, &w.Repo, &w.Base, &w.Branch, &w.State, &w.HeadSHA, &w.Number, &w.Draft, &w.MergeSHA, &w.MergedHead, &w.ReviewCursor, &w.CommentCursor, &w.LineCursor, &w.ConcludedAt)
	return w, err
}

// Report resume uma passagem de consulta.
type Report struct {
	Polled  int      `json:"polled"`
	Rounds  int      `json:"rounds"`
	Issues  int      `json:"issues"`
	Merged  int      `json:"merged"`
	Pending []string `json:"pending,omitempty"`
}

// Poll consulta até Batch PRs a partir do cursor (número da última PR lida; "" recomeça).
// Devolve o novo cursor ("" quando a passagem terminou). Limite de requisições devolve o cursor
// atual com o erro, para o backoff da rotina; falha de uma PR não impede as demais.
func (s *Service) Poll(ctx context.Context, projectID string, t Target, cursor string) (string, Report, error) {
	var rep Report
	after := 0
	if cursor != "" {
		if _, err := fmt.Sscanf(cursor, "%d", &after); err != nil {
			after = 0
		}
	}
	if _, err := s.SyncDeliveries(ctx, projectID); err != nil {
		return cursor, rep, err
	}
	rows, err := s.DB.Query(ctx, `SELECT `+watchCols+` FROM pr_watch
		WHERE project_id = $1 AND pr_number > $2 AND (state IN ('open','closed') OR (state = 'merged' AND concluded_at IS NULL))
		ORDER BY pr_number LIMIT $3`, projectID, after, s.batch())
	if err != nil {
		return cursor, rep, persistence.MapError(err)
	}
	var ws []Watch
	for rows.Next() {
		w, e := scanWatch(rows)
		if e != nil {
			rows.Close()
			return cursor, rep, e
		}
		ws = append(ws, w)
	}
	rows.Close()
	last := cursor
	for _, w := range ws {
		r, err := s.pollOne(ctx, t, w)
		rep.Polled++
		rep.Rounds += r.Rounds
		rep.Issues += r.Issues
		rep.Merged += r.Merged
		rep.Pending = append(rep.Pending, r.Pending...)
		var rl *gitops.ErrRateLimited
		if errors.As(err, &rl) {
			return last, rep, err
		}
		if err != nil {
			s.noteError(ctx, w.ID, err)
			continue
		}
		s.noteError(ctx, w.ID, nil)
		last = fmt.Sprintf("%d", w.Number)
	}
	if len(ws) < s.batch() {
		return "", rep, nil
	}
	return last, rep, nil
}

// PollAll faz uma passagem completa (ação manual do painel).
func (s *Service) PollAll(ctx context.Context, projectID string, t Target) (Report, error) {
	var total Report
	cursor := ""
	for i := 0; i < 50; i++ {
		next, r, err := s.Poll(ctx, projectID, t, cursor)
		total.Polled += r.Polled
		total.Rounds += r.Rounds
		total.Issues += r.Issues
		total.Merged += r.Merged
		total.Pending = append(total.Pending, r.Pending...)
		if err != nil {
			return total, err
		}
		if next == "" {
			return total, nil
		}
		cursor = next
	}
	return total, nil
}

func (s *Service) noteError(ctx context.Context, id string, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
		if len(msg) > 300 {
			msg = msg[:300]
		}
	}
	_, _ = s.DB.Exec(ctx, `UPDATE pr_watch SET last_error = $2, last_polled_at = $3 WHERE id = $1`, id, msg, s.now())
}

// Routine é a rotina da Sentinela com agenda própria (independente de Jira/ClickUp).
func (s *Service) Routine(resolve func(ctx context.Context, projectID string) (Target, error), every time.Duration) sentinel.Routine {
	return sentinel.Routine{Name: sentinel.RoutinePRs, Every: every, Lease: 5 * time.Minute, Backoff: time.Minute,
		Run: func(ctx context.Context, scope, cursor string) (string, error) {
			t, err := resolve(ctx, scope)
			if errors.Is(err, ErrNoTarget) {
				return "", nil
			}
			if err != nil {
				return cursor, err
			}
			next, _, err := s.Poll(ctx, scope, t, cursor)
			return next, err
		}}
}
