package decisions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Tipos de pedido.
const (
	KindConflict           = "conflict"
	KindMissingRequirement = "missing_requirement"
	KindSensitiveOperation = "sensitive_operation"
	KindUncertainResult    = "uncertain_result"
	KindPRClosed           = "pr_closed"
	KindOther              = "other"
)

var validKinds = map[string]bool{KindConflict: true, KindMissingRequirement: true, KindSensitiveOperation: true, KindUncertainResult: true, KindPRClosed: true, KindOther: true}

// Estados.
const (
	StatePending  = "pending"
	StateResolved = "resolved"
	StateStale    = "stale" // a execução mudou depois do pedido
)

// Erros com resultado definido.
var (
	ErrAlreadyResolved = errors.New("o pedido já foi resolvido com outra opção")
	ErrStale           = errors.New("a execução mudou desde o pedido: ele ficou obsoleto")
	ErrInvalidOption   = errors.New("opção inexistente neste pedido")
)

// FieldError é um campo inválido.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// Option é uma alternativa de resolução.
type Option struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Evidence é um item de evidência (texto já redigido).
type Evidence struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

// Request é um pedido de decisão.
type Request struct {
	ID               string     `json:"id"`
	ProjectID        string     `json:"project_id"`
	ExecutionID      string     `json:"execution_id"`
	NodeID           string     `json:"node_id"`
	Kind             string     `json:"kind"`
	Title            string     `json:"title"`
	Problem          string     `json:"problem"`
	Impact           string     `json:"impact"`
	Evidence         []Evidence `json:"evidence"`
	Options          []Option   `json:"options"`
	State            string     `json:"state"`
	ExecutionVersion int        `json:"execution_version"`
	ResolutionOption string     `json:"resolution_option"`
	ResolutionNote   string     `json:"resolution_note"`
	ResolvedAt       *time.Time `json:"resolved_at"`
	Version          int        `json:"version"`
	CreatedAt        time.Time  `json:"created_at"`
}

// DB é o que o pacote precisa do banco.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// NewRequest reúne os dados de criação.
type NewRequest struct {
	ProjectID, ExecutionID, NodeID, Kind string
	Title, Problem, Impact               string
	Evidence                             []Evidence
	Options                              []Option
}

var optionID = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

const columns = `id, project_id, execution_id, node_id, kind, title, problem, impact, evidence, options, state, execution_version,
	resolution_option, resolution_note, resolved_at, version, created_at`

func scan(row interface{ Scan(...any) error }) (Request, error) {
	var r Request
	var ev, op []byte
	if err := row.Scan(&r.ID, &r.ProjectID, &r.ExecutionID, &r.NodeID, &r.Kind, &r.Title, &r.Problem, &r.Impact, &ev, &op, &r.State,
		&r.ExecutionVersion, &r.ResolutionOption, &r.ResolutionNote, &r.ResolvedAt, &r.Version, &r.CreatedAt); err != nil {
		return r, persistence.MapError(err)
	}
	r.Evidence, r.Options = []Evidence{}, []Option{}
	_ = json.Unmarshal(ev, &r.Evidence)
	_ = json.Unmarshal(op, &r.Options)
	return r, nil
}

// Create abre um pedido para a execução (que precisa estar `waiting`). O mesmo
// problema (execução, nó e tipo) enquanto pendente não gera um segundo pedido:
// devolve o existente com created=false. A evidência é redigida com o redator do cofre.
func Create(ctx context.Context, db DB, rd *security.Redactor, in NewRequest) (req Request, created bool, err error) {
	if !validKinds[in.Kind] {
		return Request{}, false, &FieldError{"kind", "Tipo de pedido desconhecido."}
	}
	in.Title, in.Problem, in.Impact = strings.TrimSpace(in.Title), strings.TrimSpace(in.Problem), strings.TrimSpace(in.Impact)
	if in.Title == "" || len(in.Title) > 160 {
		return Request{}, false, &FieldError{"title", "O título deve ter entre 1 e 160 caracteres."}
	}
	if in.Problem == "" || len(in.Problem) > 2000 {
		return Request{}, false, &FieldError{"problem", "O problema deve ter entre 1 e 2000 caracteres."}
	}
	if len(in.Impact) > 1000 || len(in.NodeID) > 64 {
		return Request{}, false, &FieldError{"impact", "Texto longo demais."}
	}
	if len(in.Options) < 2 || len(in.Options) > 6 {
		return Request{}, false, &FieldError{"options", "Informe de 2 a 6 alternativas."}
	}
	seen := map[string]bool{}
	for _, o := range in.Options {
		if !optionID.MatchString(o.ID) || seen[o.ID] || strings.TrimSpace(o.Label) == "" || len(o.Label) > 80 || len(o.Description) > 300 {
			return Request{}, false, &FieldError{"options", "Alternativa inválida (id, rótulo até 80 e descrição até 300)."}
		}
		seen[o.ID] = true
	}
	if len(in.Evidence) > 20 {
		return Request{}, false, &FieldError{"evidence", "Evidência demais."}
	}
	if rd == nil {
		rd = security.NewRedactor()
	}
	ev := make([]Evidence, 0, len(in.Evidence))
	for _, e := range in.Evidence {
		t := rd.Redact(e.Text)
		if len(t) > 2000 {
			t = t[:2000]
		}
		ev = append(ev, Evidence{Label: clip(e.Label, 80), Text: t})
	}
	evJSON, _ := json.Marshal(ev)
	opJSON, _ := json.Marshal(in.Options)
	err = persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		var state string
		var version int
		if err := tx.QueryRow(ctx, `SELECT state, version FROM executions WHERE id = $1 AND project_id = $2 FOR UPDATE`, in.ExecutionID, in.ProjectID).Scan(&state, &version); err != nil {
			return persistence.MapError(err)
		}
		if state != "waiting" {
			return &FieldError{"execution_id", "A execução precisa estar aguardando (waiting) para receber um pedido."}
		}
		existing, gerr := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM decision_requests WHERE execution_id = $1 AND node_id = $2 AND kind = $3 AND state = 'pending'`,
			in.ExecutionID, in.NodeID, in.Kind))
		if gerr == nil {
			req, created = existing, false
			return nil
		}
		if !errors.Is(gerr, persistence.ErrNotFound) {
			return gerr
		}
		r, err := scan(tx.QueryRow(ctx, `
			INSERT INTO decision_requests (project_id, execution_id, node_id, kind, title, problem, impact, evidence, options, execution_version)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10) RETURNING `+columns,
			in.ProjectID, in.ExecutionID, in.NodeID, in.Kind, in.Title, in.Problem, in.Impact, string(evJSON), string(opJSON), version))
		if err != nil {
			return err
		}
		if _, err := persistence.AppendEvent(ctx, tx, in.ProjectID, in.ExecutionID, "decision_requested", map[string]any{"request_id": r.ID, "kind": in.Kind, "node_id": in.NodeID}); err != nil {
			return err
		}
		req, created = r, true
		return nil
	})
	return req, created, err
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// Get lê um pedido.
func Get(ctx context.Context, db persistence.DBTX, id string) (Request, error) {
	return scan(db.QueryRow(ctx, `SELECT `+columns+` FROM decision_requests WHERE id = $1`, id))
}

// List devolve pedidos por estado ("" = todos), mais recentes primeiro.
func List(ctx context.Context, db persistence.DBTX, state string) ([]Request, error) {
	if state != "" && state != StatePending && state != StateResolved && state != StateStale {
		return nil, &FieldError{"state", "Estado desconhecido."}
	}
	q := `SELECT ` + columns + ` FROM decision_requests`
	args := []any{}
	if state != "" {
		q += ` WHERE state = $1`
		args = append(args, state)
	}
	rows, err := db.Query(ctx, q+` ORDER BY created_at DESC, id`, args...)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Request{}
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, persistence.MapError(rows.Err())
}

// Outcome descreve o que a resolução fez.
type Outcome struct {
	Request  Request `json:"request"`
	Replayed bool    `json:"replayed"` // a mesma decisão já tinha sido registrada: nada mudou
}

// Resolve registra a decisão. Tudo numa transação, restrito à execução do pedido:
//   - repetir a MESMA opção é idempotente (nada muda, nenhum segundo sinal);
//   - outra opção depois de resolvido → ErrAlreadyResolved;
//   - execução que mudou desde o pedido (versão ou estado) → o pedido vira `stale`
//     e a resolução é recusada com ErrStale;
//   - caso contrário grava a resolução, os eventos `decision_resolved` e
//     `resume_requested` e move SÓ essa execução de waiting para queued.
func Resolve(ctx context.Context, db DB, adminID, id, option, note string) (Outcome, error) {
	note = strings.TrimSpace(note)
	if len(note) > 500 {
		return Outcome{}, &FieldError{"note", "A observação deve ter até 500 caracteres."}
	}
	var out Outcome
	var stale bool
	err := persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		req, err := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM decision_requests WHERE id = $1 FOR UPDATE`, id))
		if err != nil {
			return err
		}
		switch req.State {
		case StateResolved:
			if req.ResolutionOption == option {
				out = Outcome{Request: req, Replayed: true}
				return nil
			}
			return ErrAlreadyResolved
		case StateStale:
			return ErrStale
		}
		valid := false
		for _, o := range req.Options {
			valid = valid || o.ID == option
		}
		if !valid {
			return ErrInvalidOption
		}
		var execState string
		var execVersion int
		if err := tx.QueryRow(ctx, `SELECT state, version FROM executions WHERE id = $1 AND project_id = $2 FOR UPDATE`, req.ExecutionID, req.ProjectID).Scan(&execState, &execVersion); err != nil {
			return persistence.MapError(err)
		}
		if execState != "waiting" || execVersion != req.ExecutionVersion {
			if _, err := tx.Exec(ctx, `UPDATE decision_requests SET state = 'stale', version = version + 1 WHERE id = $1`, id); err != nil {
				return persistence.MapError(err)
			}
			stale = true
			return nil
		}
		var admin any
		if adminID != "" {
			admin = adminID
		}
		upd, err := scan(tx.QueryRow(ctx, `
			UPDATE decision_requests SET state = 'resolved', resolution_option = $2, resolution_note = $3, resolved_by = $4::uuid,
				resolved_at = now(), version = version + 1 WHERE id = $1 RETURNING `+columns, id, option, note, admin))
		if err != nil {
			return err
		}
		payload := map[string]any{"request_id": id, "node_id": req.NodeID, "kind": req.Kind, "option": option}
		if _, err := persistence.AppendEvent(ctx, tx, req.ProjectID, req.ExecutionID, "decision_resolved", payload); err != nil {
			return err
		}
		if _, err := persistence.AppendEvent(ctx, tx, req.ProjectID, req.ExecutionID, "resume_requested", payload); err != nil {
			return err
		}
		if _, err := persistence.UpdateExecutionState(ctx, tx, req.ProjectID, req.ExecutionID, execVersion, "queued"); err != nil {
			return fmt.Errorf("sinal de retomada: %w", err)
		}
		out = Outcome{Request: upd}
		return nil
	})
	if err != nil {
		return Outcome{}, err
	}
	if stale {
		return Outcome{}, ErrStale
	}
	return out, nil
}
