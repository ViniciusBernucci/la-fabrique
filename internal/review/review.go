package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Severidades.
const (
	Critical = "critical"
	Relevant = "relevant"
	Low      = "low"
)

// Estados da rodada.
const (
	StatusReviewing        = "reviewing"
	StatusChangesRequested = "changes_requested"
	StatusFixing           = "fixing"
	StatusApproved         = "approved"
	StatusExhausted        = "exhausted"
)

// DefaultMaxRounds é o limite padrão de rodadas.
const DefaultMaxRounds = 2

// Erros.
var (
	ErrNotIndependent = errors.New("a primeira revisão precisa ser de um revisor diferente de quem implementou")
	ErrNoReviewer     = errors.New("nenhum revisor independente disponível na cadeia configurada")
	ErrBadState       = errors.New("a revisão não está num estado que permita esta operação")
	ErrChecksNotGreen = errors.New("a correção só segue com os checks da nova revisão aprovados")
	ErrActive         = errors.New("já há uma revisão em andamento para esta tarefa")
)

// FieldError é um erro de entrada.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// ChecksGate diz se os checks da revisão estão aprovados (T14-01: checks.Store.Gate).
type ChecksGate func(ctx context.Context, featureID, revision string) (blocked bool, reasons []string, err error)

// Checkpointer produz o checkpoint seguro da tarefa para suspendê-la (para o processo, espera a
// quietude e captura). Devolve o id do checkpoint e se o sandbox está quieto.
type Checkpointer func(ctx context.Context, rv Review) (checkpointID string, quiescent bool, err error)

// Store conduz as revisões.
type Store struct {
	DB           DB
	Redactor     *security.Redactor
	Checks       ChecksGate
	Queue        *queue.Queue // opcional: suspende o item ao esgotar
	Checkpointer Checkpointer // exigido para suspender
	Now          func() time.Time
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// Config é a configuração do revisor de um projeto.
type Config struct {
	Reviewer  string   `json:"reviewer"`
	Fallback  []string `json:"fallback"`
	MaxRounds int      `json:"max_rounds"`
}

// SetConfig grava o revisor, o fallback e o limite de rodadas (padrão 2).
func (s *Store) SetConfig(ctx context.Context, projectID string, c Config) (Config, error) {
	c.Reviewer = strings.TrimSpace(c.Reviewer)
	if c.Reviewer == "" || len(c.Reviewer) > 100 {
		return Config{}, &FieldError{"reviewer", "informe o revisor"}
	}
	if c.MaxRounds == 0 {
		c.MaxRounds = DefaultMaxRounds
	}
	if c.MaxRounds < 1 || c.MaxRounds > 10 {
		return Config{}, &FieldError{"max_rounds", "o limite de rodadas vai de 1 a 10"}
	}
	seen := map[string]bool{c.Reviewer: true}
	fb := []string{}
	for _, a := range c.Fallback {
		a = strings.TrimSpace(a)
		if a == "" || len(a) > 100 || seen[a] {
			return Config{}, &FieldError{"fallback", fmt.Sprintf("alternativa inválida ou repetida: %q", a)}
		}
		seen[a] = true
		fb = append(fb, a)
	}
	c.Fallback = fb
	fj, _ := json.Marshal(fb)
	_, err := s.DB.Exec(ctx, `INSERT INTO review_configs (project_id, reviewer, fallback, max_rounds) VALUES ($1, $2, $3::jsonb, $4)
		ON CONFLICT (project_id) DO UPDATE SET reviewer = $2, fallback = $3::jsonb, max_rounds = $4, updated_at = now()`, projectID, c.Reviewer, string(fj), c.MaxRounds)
	return c, persistence.MapError(err)
}

// GetConfig devolve a configuração do projeto (ErrNotFound se não houver).
func (s *Store) GetConfig(ctx context.Context, projectID string) (Config, error) {
	var c Config
	var fb []byte
	if err := s.DB.QueryRow(ctx, `SELECT reviewer, fallback, max_rounds FROM review_configs WHERE project_id = $1`, projectID).Scan(&c.Reviewer, &fb, &c.MaxRounds); err != nil {
		return Config{}, persistence.MapError(err)
	}
	_ = json.Unmarshal(fb, &c.Fallback)
	if c.Fallback == nil {
		c.Fallback = []string{}
	}
	return c, nil
}

// Pick escolhe o primeiro revisor da cadeia que seja DIFERENTE do implementador e esteja
// disponível (`available` nil = todos).
func (s *Store) Pick(ctx context.Context, projectID, implementer string, available func(agent string) bool) (string, error) {
	c, err := s.GetConfig(ctx, projectID)
	if err != nil {
		return "", err
	}
	for _, a := range append([]string{c.Reviewer}, c.Fallback...) {
		if a == implementer {
			continue
		}
		if available == nil || available(a) {
			return a, nil
		}
	}
	return "", ErrNoReviewer
}

// Review é uma rodada.
type Review struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"project_id"`
	FeatureID    string     `json:"feature_id"`
	TaskKey      string     `json:"task_key"`
	Round        int        `json:"round"`
	Revision     string     `json:"revision"`
	Implementer  string     `json:"implementer"`
	Reviewer     string     `json:"reviewer"`
	Fixer        string     `json:"fixer,omitempty"`
	Status       string     `json:"status"`
	FixRevision  string     `json:"fix_revision,omitempty"`
	CheckpointID string     `json:"checkpoint_id,omitempty"`
	Findings     []Finding  `json:"findings"`
	CreatedAt    time.Time  `json:"created_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

// Finding é um achado.
type Finding struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	File        string `json:"file,omitempty"`
	Line        *int   `json:"line,omitempty"`
	State       string `json:"state"`
	FixRevision string `json:"fix_revision,omitempty"`
}

// Blocking: crítico e relevante bloqueiam.
func (f Finding) Blocking() bool { return f.Severity == Critical || f.Severity == Relevant }

// FindingInput é o achado informado pelo revisor.
type FindingInput struct {
	Severity string
	Title    string
	Body     string
	File     string
	Line     *int
}

const revCols = `id, project_id, feature_id, task_key, round, revision, implementer, reviewer, fixer, status, fix_revision, COALESCE(checkpoint_id::text, ''), created_at, finished_at`

func scanReview(row interface{ Scan(...any) error }) (Review, error) {
	var r Review
	err := row.Scan(&r.ID, &r.ProjectID, &r.FeatureID, &r.TaskKey, &r.Round, &r.Revision, &r.Implementer, &r.Reviewer, &r.Fixer, &r.Status, &r.FixRevision, &r.CheckpointID, &r.CreatedAt, &r.FinishedAt)
	r.Findings = []Finding{}
	return r, persistence.MapError(err)
}

func (s *Store) loadFindings(ctx context.Context, q persistence.DBTX, r *Review) error {
	rows, err := q.Query(ctx, `SELECT id, severity, title, body, file, line, state, fix_revision FROM review_findings WHERE review_id = $1 ORDER BY created_at, id`, r.ID)
	if err != nil {
		return persistence.MapError(err)
	}
	defer rows.Close()
	r.Findings = []Finding{}
	for rows.Next() {
		var f Finding
		if err := rows.Scan(&f.ID, &f.Severity, &f.Title, &f.Body, &f.File, &f.Line, &f.State, &f.FixRevision); err != nil {
			return persistence.MapError(err)
		}
		r.Findings = append(r.Findings, f)
	}
	return rows.Err()
}

// Get devolve a rodada com os achados.
func (s *Store) Get(ctx context.Context, projectID, id string) (Review, error) {
	r, err := scanReview(s.DB.QueryRow(ctx, `SELECT `+revCols+` FROM reviews WHERE project_id = $1 AND id = $2`, projectID, id))
	if err != nil {
		return r, err
	}
	return r, s.loadFindings(ctx, s.DB, &r)
}

// Start abre a primeira rodada sobre a revisão do código. O revisor TEM que ser diferente de
// quem implementou (e, se informadas, de sessão diferente).
func (s *Store) Start(ctx context.Context, projectID, featureID, taskKey, revision, implementer, reviewer string) (Review, error) {
	if strings.TrimSpace(revision) == "" || len(revision) > 100 || strings.TrimSpace(taskKey) == "" {
		return Review{}, &FieldError{"revision", "informe a tarefa e a revisão do código a revisar"}
	}
	implementer, reviewer = strings.TrimSpace(implementer), strings.TrimSpace(reviewer)
	if implementer == "" || reviewer == "" {
		return Review{}, &FieldError{"reviewer", "informe quem implementou e quem revisa"}
	}
	if implementer == reviewer {
		return Review{}, fmt.Errorf("%w: %q implementou e revisaria", ErrNotIndependent, implementer)
	}
	r, err := scanReview(s.DB.QueryRow(ctx, `INSERT INTO reviews (project_id, feature_id, task_key, round, revision, implementer, reviewer, status, created_at)
		VALUES ($1, $2, $3, 1, $4, $5, $6, 'reviewing', $7) RETURNING `+revCols, projectID, featureID, taskKey, revision, implementer, reviewer, s.now()))
	if errors.Is(err, persistence.ErrDuplicate) {
		return Review{}, ErrActive
	}
	return r, err
}

func (s *Store) maxRounds(ctx context.Context, q persistence.DBTX, projectID string) int {
	var n int
	if err := q.QueryRow(ctx, `SELECT max_rounds FROM review_configs WHERE project_id = $1`, projectID).Scan(&n); err != nil {
		return DefaultMaxRounds
	}
	return n
}

// Submit registra os achados da rodada. Com crítico ou relevante a rodada pede correção (ou, no
// limite de rodadas, esgota e suspende); só com baixos ou sem achados, aprova — os baixos ficam
// registrados.
func (s *Store) Submit(ctx context.Context, projectID, reviewID string, in []FindingInput) (Review, error) {
	for i, f := range in {
		if f.Severity != Critical && f.Severity != Relevant && f.Severity != Low {
			return Review{}, &FieldError{"severity", fmt.Sprintf("achado %d: use critical, relevant ou low", i+1)}
		}
		if t := strings.TrimSpace(f.Title); t == "" || len(t) > 200 {
			return Review{}, &FieldError{"title", fmt.Sprintf("achado %d: título obrigatório (até 200 caracteres)", i+1)}
		}
		if len(f.Body) > 4000 || len(f.File) > 300 || (f.Line != nil && *f.Line < 1) {
			return Review{}, &FieldError{"finding", fmt.Sprintf("achado %d: corpo, arquivo ou linha inválidos", i+1)}
		}
	}
	var out Review
	exhausted := false
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		r, err := scanReview(tx.QueryRow(ctx, `SELECT `+revCols+` FROM reviews WHERE project_id = $1 AND id = $2 FOR UPDATE`, projectID, reviewID))
		if err != nil {
			return err
		}
		if r.Status != StatusReviewing {
			return fmt.Errorf("%w: estado %s", ErrBadState, r.Status)
		}
		blocking := false
		for _, f := range in {
			state := "open"
			if f.Severity == Low {
				state = "noted"
			} else {
				blocking = true
			}
			if _, err := tx.Exec(ctx, `INSERT INTO review_findings (review_id, severity, title, body, file, line, state) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				reviewID, f.Severity, strings.TrimSpace(f.Title), s.Redactor.Redact(f.Body), f.File, f.Line, state); err != nil {
				return persistence.MapError(err)
			}
		}
		status := StatusApproved
		if blocking {
			status = StatusChangesRequested
			if r.Round >= s.maxRounds(ctx, tx, r.ProjectID) {
				status, exhausted = StatusExhausted, true
			}
		}
		fin := any(nil)
		if status != StatusChangesRequested {
			fin = s.now()
		}
		r, err = scanReview(tx.QueryRow(ctx, `UPDATE reviews SET status = $2, finished_at = $3 WHERE id = $1 RETURNING `+revCols, reviewID, status, fin))
		if err != nil {
			return err
		}
		if err := s.loadFindings(ctx, tx, &r); err != nil {
			return err
		}
		out = r
		return nil
	})
	if err != nil {
		return Review{}, err
	}
	if exhausted {
		return s.onExhausted(ctx, out)
	}
	return out, nil
}

// onExhausted suspende o item da fila com checkpoint seguro e abre a decisão humana.
func (s *Store) onExhausted(ctx context.Context, rv Review) (Review, error) {
	if s.Queue == nil || s.Checkpointer == nil {
		return rv, nil
	}
	var itemID, execID string
	err := s.DB.QueryRow(ctx, `SELECT id::text, COALESCE(execution_id::text, '') FROM queue_items WHERE project_id = $1 AND feature_id = $2 AND task_key = $3 AND state = 'running' ORDER BY seq DESC LIMIT 1`,
		rv.ProjectID, rv.FeatureID, rv.TaskKey).Scan(&itemID, &execID)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return rv, nil // a tarefa não está em execução na fila: nada a suspender
	}
	if err != nil {
		return rv, persistence.MapError(err)
	}
	cp, quiet, err := s.Checkpointer(ctx, rv)
	if err != nil {
		return rv, fmt.Errorf("revisão esgotada, mas o checkpoint falhou (o item segue em execução): %w", err)
	}
	if _, err := s.Queue.SuspendOne(ctx, itemID, cp, quiet); err != nil {
		return rv, fmt.Errorf("revisão esgotada, mas a suspensão não foi segura: %w", err)
	}
	if _, err := s.DB.Exec(ctx, `UPDATE reviews SET checkpoint_id = $2 WHERE id = $1`, rv.ID, cp); err != nil {
		return rv, persistence.MapError(err)
	}
	rv.CheckpointID = cp
	if execID != "" {
		var state string
		var version int
		if e := s.DB.QueryRow(ctx, `SELECT state, version FROM executions WHERE project_id = $1 AND id = $2`, rv.ProjectID, execID).Scan(&state, &version); e == nil && state != "cancelled" {
			if state != "waiting" {
				_, _ = persistence.UpdateExecutionState(ctx, s.DB, rv.ProjectID, execID, version, "waiting")
			}
			ev := []decisions.Evidence{}
			for _, f := range rv.Findings {
				if f.Blocking() {
					ev = append(ev, decisions.Evidence{Label: f.Severity + ": " + f.Title, Text: f.Body})
				}
			}
			_, _, _ = decisions.Create(ctx, s.DB, s.Redactor, decisions.NewRequest{
				ProjectID: rv.ProjectID, ExecutionID: execID, NodeID: rv.TaskKey, Kind: decisions.KindOther,
				Title:    "Revisão esgotou as rodadas com achados bloqueantes",
				Problem:  fmt.Sprintf("A tarefa %s continua com achados críticos ou relevantes depois de %d rodada(s). A tarefa foi suspensa com checkpoint.", rv.TaskKey, rv.Round),
				Impact:   "A tarefa fica parada até a decisão; nada segue sozinho.",
				Evidence: ev,
				Options: []decisions.Option{
					{ID: "more_rounds", Label: "Mais uma rodada", Description: "Retomar a tarefa e permitir nova correção e revisão."},
					{ID: "stop", Label: "Manter suspensa", Description: "Não retomar sem investigação."},
				},
			})
		}
	}
	return rv, nil
}

// BeginFix registra quem vai corrigir (pode ser o próprio revisor, em etapa posterior à revisão inicial).
func (s *Store) BeginFix(ctx context.Context, projectID, reviewID, fixer string) (Review, error) {
	fixer = strings.TrimSpace(fixer)
	if fixer == "" || len(fixer) > 100 {
		return Review{}, &FieldError{"fixer", "informe quem corrige"}
	}
	r, err := scanReview(s.DB.QueryRow(ctx, `UPDATE reviews SET status = 'fixing', fixer = $3 WHERE project_id = $1 AND id = $2 AND status = 'changes_requested' RETURNING `+revCols, projectID, reviewID, fixer))
	if errors.Is(err, persistence.ErrNotFound) {
		return Review{}, fmt.Errorf("%w: só uma rodada com correção pedida pode ser corrigida", ErrBadState)
	}
	if err != nil {
		return Review{}, err
	}
	return r, s.loadFindings(ctx, s.DB, &r)
}

// CompleteFix aceita a correção SÓ se os checks da nova revisão estiverem aprovados, marca os
// achados bloqueantes como corrigidos e abre a próxima rodada (revalidação) sobre a nova revisão,
// com o mesmo revisor.
func (s *Store) CompleteFix(ctx context.Context, projectID, reviewID, newRevision string) (Review, error) {
	if strings.TrimSpace(newRevision) == "" || len(newRevision) > 100 {
		return Review{}, &FieldError{"revision", "informe a revisão com a correção"}
	}
	cur, err := s.Get(ctx, projectID, reviewID)
	if err != nil {
		return Review{}, err
	}
	if cur.Status != StatusFixing {
		return Review{}, fmt.Errorf("%w: estado %s", ErrBadState, cur.Status)
	}
	if newRevision == cur.Revision {
		return Review{}, &FieldError{"revision", "a correção precisa gerar uma revisão nova"}
	}
	if s.Checks == nil {
		return Review{}, fmt.Errorf("%w: sem como conferir os checks", ErrChecksNotGreen)
	}
	blocked, reasons, err := s.Checks(ctx, cur.FeatureID, newRevision)
	if err != nil {
		return Review{}, err
	}
	if blocked {
		return Review{}, fmt.Errorf("%w: %s", ErrChecksNotGreen, strings.Join(reasons, "; "))
	}
	var next Review
	err = persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		r, err := scanReview(tx.QueryRow(ctx, `UPDATE reviews SET fix_revision = $3, finished_at = $4, status = 'changes_requested' WHERE project_id = $1 AND id = $2 AND status = 'fixing' RETURNING `+revCols,
			projectID, reviewID, newRevision, s.now()))
		if errors.Is(err, persistence.ErrNotFound) {
			return ErrBadState
		}
		if err != nil {
			return err
		}
		// A rodada anterior fica fechada (corrigida) para liberar a vaga de "revisão ativa".
		if _, err := tx.Exec(ctx, `UPDATE review_findings SET state = 'fixed', fix_revision = $2 WHERE review_id = $1 AND state = 'open'`, reviewID, newRevision); err != nil {
			return persistence.MapError(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE reviews SET status = 'approved' WHERE id = $1`, reviewID); err != nil {
			return persistence.MapError(err)
		}
		_ = r
		next, err = scanReview(tx.QueryRow(ctx, `INSERT INTO reviews (project_id, feature_id, task_key, round, revision, implementer, reviewer, status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'reviewing', $8) RETURNING `+revCols, cur.ProjectID, cur.FeatureID, cur.TaskKey, cur.Round+1, newRevision, cur.Implementer, cur.Reviewer, s.now()))
		return err
	})
	return next, err
}

// Result é o veredito da revisão para uma revisão do código.
type Result struct {
	Blocked bool      `json:"blocked"`
	Reasons []string  `json:"reasons"`
	Status  string    `json:"status"`
	Round   int       `json:"round"`
	Noted   []Finding `json:"noted"` // baixa severidade: registrada, não bloqueia
}

// Gate confere a revisão da REVISÃO do código. `required` vem do nível de risco (médio e alto
// exigem). Aprovação de outra revisão não vale: uma alteração depois exige nova revisão.
func (s *Store) Gate(ctx context.Context, projectID, featureID, taskKey, revision string, required bool) (Result, error) {
	res := Result{Reasons: []string{}, Noted: []Finding{}}
	r, err := scanReview(s.DB.QueryRow(ctx, `SELECT `+revCols+` FROM reviews WHERE project_id = $1 AND feature_id = $2 AND task_key = $3 AND revision = $4 ORDER BY seq DESC LIMIT 1`,
		projectID, featureID, taskKey, revision))
	if errors.Is(err, persistence.ErrNotFound) {
		if required {
			res.Blocked = true
			res.Status = "missing"
			res.Reasons = append(res.Reasons, "revisão independente obrigatória ainda não feita para esta revisão do código")
		}
		return res, nil
	}
	if err != nil {
		return res, err
	}
	if err := s.loadFindings(ctx, s.DB, &r); err != nil {
		return res, err
	}
	res.Status, res.Round = r.Status, r.Round
	for _, f := range r.Findings {
		if f.State == "noted" {
			res.Noted = append(res.Noted, f)
		}
	}
	switch r.Status {
	case StatusApproved:
		// aprovada só vale se não restou bloqueante em aberto
		for _, f := range r.Findings {
			if f.Blocking() && f.State == "open" {
				res.Blocked = true
				res.Reasons = append(res.Reasons, fmt.Sprintf("achado %s em aberto: %s", f.Severity, f.Title))
			}
		}
	case StatusExhausted:
		res.Blocked = true
		res.Reasons = append(res.Reasons, fmt.Sprintf("a revisão esgotou as rodadas com achados bloqueantes (%d)", r.Round))
	default:
		res.Blocked = true
		res.Reasons = append(res.Reasons, fmt.Sprintf("a revisão está em andamento (%s, rodada %d)", r.Status, r.Round))
	}
	return res, nil
}

// History lista as rodadas da tarefa, da mais nova para a mais antiga.
func (s *Store) History(ctx context.Context, projectID, featureID, taskKey string) ([]Review, error) {
	rows, err := s.DB.Query(ctx, `SELECT `+revCols+` FROM reviews WHERE project_id = $1 AND feature_id = $2 AND task_key = $3 ORDER BY seq DESC`, projectID, featureID, taskKey)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	var out []Review
	for rows.Next() {
		r, err := scanReview(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, r)
	}
	rows.Close()
	for i := range out {
		if err := s.loadFindings(ctx, s.DB, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}
