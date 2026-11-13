package reqchange

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
	"github.com/ViniciusBernucci/le-fabrique/internal/resume"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Impactos.
const (
	ImpactInitial  = "initial"
	ImpactMetadata = "metadata"
	ImpactContent  = "content"
)

// Ações e estados.
const (
	ActionRecord   = "record"
	ActionUpdate   = "update"
	ActionDefer    = "defer"
	ActionReplan   = "replan"
	ActionDecision = "decision"

	StateApplied     = "applied"
	StateDeferred    = "deferred"
	StateNeedsReplan = "needs_replan"
	StateConflict    = "conflict"
	StateRejected    = "rejected"
	StateSuperseded  = "superseded"
)

// Erros.
var (
	ErrNoRevision  = errors.New("a revisão não está em um estado que permita esta operação")
	ErrNotResolved = errors.New("a decisão ainda não foi resolvida")
	ErrNoExecution = errors.New("a feature não tem execução para receber a decisão")
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

// Handler trata as revisões.
type Handler struct {
	DB       DB
	Redactor *security.Redactor
	Now      func() time.Time
}

func (h *Handler) now() time.Time {
	if h.Now != nil {
		return h.Now().UTC()
	}
	return time.Now().UTC()
}

// Revision é uma revisão gravada.
type Revision struct {
	ID         string             `json:"id"`
	FeatureID  string             `json:"feature_id"`
	Seq        int                `json:"seq"`
	Content    resume.Requirement `json:"content"`
	Hash       string             `json:"content_hash"`
	Impact     string             `json:"impact"`
	Action     string             `json:"action"`
	State      string             `json:"state"`
	Diff       []resume.Diff      `json:"diff"`
	DecisionID string             `json:"decision_id,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	ResolvedAt *time.Time         `json:"resolved_at,omitempty"`
}

const revCols = `id, feature_id, seq, content, content_hash, impact, action, state, diff, COALESCE(decision_id::text, ''), created_at, resolved_at`

func scanRev(row interface{ Scan(...any) error }) (Revision, error) {
	var r Revision
	var content, diff []byte
	if err := row.Scan(&r.ID, &r.FeatureID, &r.Seq, &content, &r.Hash, &r.Impact, &r.Action, &r.State, &diff, &r.DecisionID, &r.CreatedAt, &r.ResolvedAt); err != nil {
		return r, persistence.MapError(err)
	}
	_ = json.Unmarshal(content, &r.Content)
	r.Diff = []resume.Diff{}
	_ = json.Unmarshal(diff, &r.Diff)
	return r, nil
}

func normalize(r resume.Requirement) resume.Requirement {
	r.Title, r.Description = strings.TrimSpace(r.Title), strings.TrimSpace(r.Description)
	r.Labels = append([]string{}, r.Labels...)
	r.Dependencies = append([]string{}, r.Dependencies...)
	for i := range r.Labels {
		r.Labels[i] = strings.TrimSpace(r.Labels[i])
	}
	for i := range r.Dependencies {
		r.Dependencies[i] = strings.TrimSpace(r.Dependencies[i])
	}
	sort.Strings(r.Labels)
	sort.Strings(r.Dependencies)
	return r
}

// HashOf é o hash do requisito normalizado.
func HashOf(r resume.Requirement) string {
	b, _ := json.Marshal(normalize(r))
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// Classify compara duas revisões do requisito: só metadado (prioridade, rótulos,
// estado) é compatível; título, descrição ou dependências são conteúdo.
func Classify(prev, cur resume.Requirement) (string, []resume.Diff) {
	ds := resume.Reconcile(resume.Inputs{Requirement: normalize(prev)}, resume.Inputs{Requirement: normalize(cur)}, resume.Options{})
	if len(ds) == 0 {
		return "", ds
	}
	if resume.HasConflict(ds) {
		return ImpactContent, ds
	}
	return ImpactMetadata, ds
}

// Outcome é o resultado de tratar uma revisão.
type Outcome struct {
	Revision  Revision           `json:"revision"`
	Unchanged bool               `json:"unchanged"`
	Decision  *decisions.Request `json:"decision,omitempty"`
}

// Current devolve a revisão aplicada (vigente) da feature.
func (h *Handler) Current(ctx context.Context, projectID, featureID string) (Revision, error) {
	return scanRev(h.DB.QueryRow(ctx, `SELECT `+revCols+` FROM requirement_revisions WHERE project_id = $1 AND feature_id = $2 AND state = 'applied'`, projectID, featureID))
}

// History lista todas as revisões, da mais nova para a mais antiga.
func (h *Handler) History(ctx context.Context, projectID, featureID string) ([]Revision, error) {
	rows, err := h.DB.Query(ctx, `SELECT `+revCols+` FROM requirement_revisions WHERE project_id = $1 AND feature_id = $2 ORDER BY seq DESC`, projectID, featureID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Revision{}
	for rows.Next() {
		r, err := scanRev(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, persistence.MapError(rows.Err())
}

// inFlight informa se há trabalho iniciado (tarefa fora de "pendente") e se alguma está em execução.
func (h *Handler) inFlight(ctx context.Context, q persistence.DBTX, projectID, featureID string) (started, running bool, err error) {
	ts, err := planning.TasksOf(ctx, q, projectID, featureID)
	if err != nil {
		return false, false, err
	}
	for _, t := range ts {
		if t.Removed {
			continue
		}
		if t.State != planning.StatePending {
			started = true
		}
		if t.State == planning.StateRunning {
			running = true
		}
	}
	return started, running, nil
}

// Handle trata uma nova leitura do requisito da feature. O conteúdo igual ao
// vigente não gera revisão. A primeira leitura vira a revisão inicial.
func (h *Handler) Handle(ctx context.Context, projectID, featureID string, cur resume.Requirement) (Outcome, error) {
	cur = normalize(cur)
	hash := HashOf(cur)
	var out Outcome
	err := persistence.WithTx(ctx, h.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 11))`, featureID); err != nil {
			return persistence.MapError(err)
		}
		var one int
		if err := tx.QueryRow(ctx, `SELECT 1 FROM features WHERE project_id = $1 AND id = $2`, projectID, featureID).Scan(&one); err != nil {
			return persistence.MapError(err)
		}
		// Última revisão conhecida (qualquer estado) para não repetir o mesmo conteúdo.
		last, err := scanRev(tx.QueryRow(ctx, `SELECT `+revCols+` FROM requirement_revisions WHERE feature_id = $1 ORDER BY seq DESC LIMIT 1`, featureID))
		hasLast := err == nil
		if err != nil && !errors.Is(err, persistence.ErrNotFound) {
			return err
		}
		if hasLast && last.Hash == hash {
			out = Outcome{Revision: last, Unchanged: true}
			return nil
		}
		applied, aerr := scanRev(tx.QueryRow(ctx, `SELECT `+revCols+` FROM requirement_revisions WHERE feature_id = $1 AND state = 'applied'`, featureID))
		hasApplied := aerr == nil
		if aerr != nil && !errors.Is(aerr, persistence.ErrNotFound) {
			return aerr
		}
		var seq int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(max(seq), 0) + 1 FROM requirement_revisions WHERE feature_id = $1`, featureID).Scan(&seq); err != nil {
			return persistence.MapError(err)
		}
		content, _ := json.Marshal(cur)
		insert := func(impact, action, state string, diff []resume.Diff, decisionID any) (Revision, error) {
			dj, _ := json.Marshal(diff)
			return scanRev(tx.QueryRow(ctx, `INSERT INTO requirement_revisions (project_id, feature_id, seq, content, content_hash, impact, action, state, diff, decision_id, created_at)
				VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8, $9::jsonb, $10, $11) RETURNING `+revCols,
				projectID, featureID, seq, string(content), hash, impact, action, state, string(dj), decisionID, h.now()))
		}
		supersede := func() error {
			if hasApplied {
				_, err := tx.Exec(ctx, `UPDATE requirement_revisions SET state = 'superseded', resolved_at = $2 WHERE id = $1`, applied.ID, h.now())
				return persistence.MapError(err)
			}
			return nil
		}
		setRevision := func(r Revision) error {
			_, err := tx.Exec(ctx, `UPDATE features SET external_revision = $3, title = $4, updated_at = $5 WHERE project_id = $1 AND id = $2`,
				projectID, featureID, r.Hash, titleOrKeep(cur.Title), h.now())
			return persistence.MapError(err)
		}
		if !hasApplied {
			// Sem revisão vigente (primeira leitura, ou a anterior aguarda replanejamento): vira a base.
			if hasLast && last.State == StateNeedsReplan {
				// Novo conteúdo antes de o replanejamento terminar: substitui o pendente.
				if _, err := tx.Exec(ctx, `UPDATE requirement_revisions SET state = 'superseded', resolved_at = $2 WHERE id = $1`, last.ID, h.now()); err != nil {
					return persistence.MapError(err)
				}
				r, err := insert(ImpactContent, ActionReplan, StateNeedsReplan, []resume.Diff{}, nil)
				out = Outcome{Revision: r}
				return err
			}
			r, err := insert(ImpactInitial, ActionRecord, StateApplied, []resume.Diff{}, nil)
			if err != nil {
				return err
			}
			out = Outcome{Revision: r}
			return setRevision(r)
		}
		impact, diff := Classify(applied.Content, cur)
		if impact == "" { // só ordem/espaço: o mesmo requisito
			out = Outcome{Revision: applied, Unchanged: true}
			return nil
		}
		started, running, err := h.inFlight(ctx, tx, projectID, featureID)
		if err != nil {
			return err
		}
		switch {
		case impact == ImpactMetadata && running:
			// Marco seguro: só quando nenhuma tarefa está em execução.
			r, err := insert(impact, ActionDefer, StateDeferred, diff, nil)
			out = Outcome{Revision: r}
			return err
		case impact == ImpactMetadata:
			if err := supersede(); err != nil {
				return err
			}
			r, err := insert(impact, ActionUpdate, StateApplied, diff, nil)
			if err != nil {
				return err
			}
			out = Outcome{Revision: r}
			return setRevision(r)
		case !started:
			// Conteúdo mudou e nada começou: o plano precisa ser refeito.
			r, err := insert(impact, ActionReplan, StateNeedsReplan, diff, nil)
			out = Outcome{Revision: r}
			return err
		}
		// Conteúdo mudou com trabalho iniciado: conflito. Nada é alterado; abre decisão.
		exID, exVersion, exState, err := latestExecution(ctx, tx, projectID, featureID)
		if err != nil {
			return err
		}
		if exState != "waiting" {
			if exState == "cancelled" {
				return fmt.Errorf("%w: a execução foi cancelada", persistence.ErrInvalidInput)
			}
			if _, err := persistence.UpdateExecutionState(ctx, tx, projectID, exID, exVersion, "waiting"); err != nil {
				return err
			}
		}
		ev := make([]decisions.Evidence, 0, len(diff))
		for _, d := range diff {
			ev = append(ev, decisions.Evidence{Label: d.Kind + "." + d.Field, Text: fmt.Sprintf("antes: %s\nagora: %s\n%s", d.Before, d.After, d.Reason)})
		}
		dec, _, err := decisions.Create(ctx, tx, h.Redactor, decisions.NewRequest{
			ProjectID: projectID, ExecutionID: exID, NodeID: "", Kind: decisions.KindConflict,
			Title:    "O requisito mudou com trabalho em andamento",
			Problem:  "O requisito da feature mudou depois que parte do plano já foi iniciada ou concluída.",
			Impact:   "Nada foi alterado: o plano, o requisito vigente e o trabalho feito foram preservados até a decisão.",
			Evidence: ev,
			Options: []decisions.Option{
				{ID: "keep_work", Label: "Manter o plano atual", Description: "Ignorar a mudança e seguir com o plano em andamento."},
				{ID: "replan", Label: "Replanejar", Description: "Adotar o novo requisito e refazer o plano."},
			},
		})
		if err != nil {
			return err
		}
		r, err := insert(impact, ActionDecision, StateConflict, diff, dec.ID)
		out = Outcome{Revision: r, Decision: &dec}
		return err
	})
	return out, err
}

func titleOrKeep(t string) string {
	if strings.TrimSpace(t) == "" {
		return "(sem título)"
	}
	return t
}

func latestExecution(ctx context.Context, q persistence.DBTX, projectID, featureID string) (id string, version int, state string, err error) {
	err = q.QueryRow(ctx, `SELECT id, version, state FROM executions WHERE project_id = $1 AND feature_id = $2 ORDER BY attempt DESC LIMIT 1`, projectID, featureID).Scan(&id, &version, &state)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return "", 0, "", ErrNoExecution
	}
	return id, version, state, persistence.MapError(err)
}

// AtMilestone aplica as revisões de metadado adiadas, num marco seguro (nenhuma
// tarefa em execução). Devolve a revisão aplicada, se houve.
func (h *Handler) AtMilestone(ctx context.Context, projectID, featureID string) (*Revision, error) {
	var applied *Revision
	err := persistence.WithTx(ctx, h.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 11))`, featureID); err != nil {
			return persistence.MapError(err)
		}
		_, running, err := h.inFlight(ctx, tx, projectID, featureID)
		if err != nil || running {
			return err
		}
		// A mais recente adiada vence; as anteriores ficam substituídas.
		def, err := scanRev(tx.QueryRow(ctx, `SELECT `+revCols+` FROM requirement_revisions WHERE feature_id = $1 AND state = 'deferred' ORDER BY seq DESC LIMIT 1`, featureID))
		if errors.Is(err, persistence.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE requirement_revisions SET state = 'superseded', resolved_at = $2 WHERE feature_id = $1 AND state IN ('applied', 'deferred') AND id <> $3`, featureID, h.now(), def.ID); err != nil {
			return persistence.MapError(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE requirement_revisions SET state = 'applied', action = 'update', resolved_at = $2 WHERE id = $1`, def.ID, h.now()); err != nil {
			return persistence.MapError(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE features SET external_revision = $3, updated_at = $4 WHERE project_id = $1 AND id = $2`, projectID, featureID, def.Hash, h.now()); err != nil {
			return persistence.MapError(err)
		}
		def.State = StateApplied
		applied = &def
		return nil
	})
	return applied, err
}

// Settle conclui o conflito conforme a resolução da decisão: "keep_work" mantém o
// requisito vigente (a revisão da mudança é rejeitada, mas permanece no histórico);
// "replan" adota a mudança e marca a revisão como precisando de replanejamento.
func (h *Handler) Settle(ctx context.Context, projectID, revisionID string) (Revision, error) {
	var out Revision
	err := persistence.WithTx(ctx, h.DB, func(tx pgx.Tx) error {
		r, err := scanRev(tx.QueryRow(ctx, `SELECT `+revCols+` FROM requirement_revisions WHERE project_id = $1 AND id = $2 FOR UPDATE`, projectID, revisionID))
		if err != nil {
			return err
		}
		if r.State != StateConflict || r.DecisionID == "" {
			return ErrNoRevision
		}
		dec, err := decisions.Get(ctx, tx, r.DecisionID)
		if err != nil {
			return err
		}
		if dec.State != decisions.StateResolved {
			return ErrNotResolved
		}
		switch dec.ResolutionOption {
		case "keep_work":
			_, err = tx.Exec(ctx, `UPDATE requirement_revisions SET state = 'rejected', resolved_at = $2 WHERE id = $1`, r.ID, h.now())
			r.State = StateRejected
		case "replan":
			if _, err = tx.Exec(ctx, `UPDATE requirement_revisions SET state = 'superseded', resolved_at = $2 WHERE feature_id = $1 AND state = 'applied'`, r.FeatureID, h.now()); err != nil {
				return persistence.MapError(err)
			}
			_, err = tx.Exec(ctx, `UPDATE requirement_revisions SET state = 'needs_replan', action = 'replan', resolved_at = NULL WHERE id = $1`, r.ID)
			r.State = StateNeedsReplan
		default:
			return fmt.Errorf("%w: opção %q", ErrNoRevision, dec.ResolutionOption)
		}
		out = r
		return persistence.MapError(err)
	})
	return out, err
}

// CompleteReplan aplica a revisão que aguardava replanejamento, depois que um
// novo plano foi aceito (versão informada). A revisão passa a ser a vigente.
func (h *Handler) CompleteReplan(ctx context.Context, projectID, featureID string) (Revision, error) {
	var out Revision
	err := persistence.WithTx(ctx, h.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 11))`, featureID); err != nil {
			return persistence.MapError(err)
		}
		r, err := scanRev(tx.QueryRow(ctx, `SELECT `+revCols+` FROM requirement_revisions WHERE project_id = $1 AND feature_id = $2 AND state = 'needs_replan' ORDER BY seq DESC LIMIT 1`, projectID, featureID))
		if errors.Is(err, persistence.ErrNotFound) {
			return ErrNoRevision
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE requirement_revisions SET state = 'superseded', resolved_at = $2 WHERE feature_id = $1 AND state IN ('applied', 'needs_replan', 'deferred') AND id <> $3`, featureID, h.now(), r.ID); err != nil {
			return persistence.MapError(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE requirement_revisions SET state = 'applied', resolved_at = $2 WHERE id = $1`, r.ID, h.now()); err != nil {
			return persistence.MapError(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE features SET external_revision = $3, updated_at = $4 WHERE project_id = $1 AND id = $2`, projectID, featureID, r.Hash, h.now()); err != nil {
			return persistence.MapError(err)
		}
		r.State = StateApplied
		out = r
		return nil
	})
	return out, err
}
