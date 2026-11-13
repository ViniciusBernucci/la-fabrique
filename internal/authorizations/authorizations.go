package authorizations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
)

// Erros que impedem a operação.
var (
	ErrExpired   = errors.New("autorização expirada")
	ErrRevoked   = errors.New("autorização revogada")
	ErrExhausted = errors.New("autorização sem usos restantes")
	ErrDecision  = errors.New("a decisão não está resolvida ou não pertence a esta execução")
)

// MismatchError diz QUAL parte da ação diverge do que foi autorizado.
type MismatchError struct{ Field string } // tool | operation | resource | target | params

func (e *MismatchError) Error() string {
	return "a ação diverge do que foi autorizado (" + e.Field + ")"
}

// FieldError é um campo inválido com mensagem segura.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// DB é o que o Store precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Halter faz a parada segura de uma execução: cancela o que está rodando,
// confirma a quiescência e grava o checkpoint (T11-03/T12-01).
type Halter interface {
	Halt(ctx context.Context, projectID, executionID string) error
}

// Store guarda as autorizações.
type Store struct {
	DB     DB
	Halter Halter // opcional; sem ele, a revogação com uso em andamento falha explicitamente
	Now    func() time.Time
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// Authorization é uma autorização vigente ou não.
type Authorization struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"project_id"`
	ExecutionID  string     `json:"execution_id"`
	NodeID       string     `json:"node_id"`
	DecisionID   *string    `json:"decision_id"`
	Tool         string     `json:"tool"`
	Operation    string     `json:"operation"`
	Resource     string     `json:"resource"`
	Target       string     `json:"target"`
	ParamsHash   string     `json:"params_hash"`
	MaxUses      int        `json:"max_uses"`
	Uses         int        `json:"uses"`
	ExpiresAt    time.Time  `json:"expires_at"`
	State        string     `json:"state"`
	RevokedAt    *time.Time `json:"revoked_at"`
	RevokeReason string     `json:"revoke_reason"`
	Version      int        `json:"version"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Remaining são os usos que ainda restam (0 se revogada ou vencida).
func (a Authorization) Remaining(now time.Time) int {
	if a.State != "active" || !a.ExpiresAt.After(now) {
		return 0
	}
	return a.MaxUses - a.Uses
}

const cols = `id, project_id, execution_id, node_id, decision_id, tool, operation, resource, target, params_hash, max_uses, uses, expires_at, state, revoked_at, revoke_reason, version, created_at`

func scan(row interface{ Scan(...any) error }) (Authorization, error) {
	var a Authorization
	err := row.Scan(&a.ID, &a.ProjectID, &a.ExecutionID, &a.NodeID, &a.DecisionID, &a.Tool, &a.Operation, &a.Resource, &a.Target, &a.ParamsHash,
		&a.MaxUses, &a.Uses, &a.ExpiresAt, &a.State, &a.RevokedAt, &a.RevokeReason, &a.Version, &a.CreatedAt)
	return a, persistence.MapError(err)
}

// NewAuthorization reúne os dados de criação. A ação é a MESMA estrutura que a
// política avalia: o que foi aprovado é exatamente o que será exigido na execução.
type NewAuthorization struct {
	ProjectID   string
	ExecutionID string
	NodeID      string
	DecisionID  string // opcional: a decisão humana que originou a autorização
	Action      policy.Action
	MaxUses     int
	TTL         time.Duration
	GrantedBy   string // administrador (UUID); opcional em testes
}

// MaxTTL é o prazo máximo de uma autorização.
const MaxTTL = 24 * time.Hour

// bound reduz a ação à identidade que a autorização cobre.
type bound struct{ tool, operation, resource, target, hash string }

func boundOf(a policy.Action) bound {
	_, h := policy.Canonical(a.Params)
	return bound{tool: strings.TrimSpace(a.Tool), operation: strings.TrimSpace(a.Operation),
		resource: strings.TrimSpace(a.Resource), target: strings.ToLower(strings.TrimSpace(a.Target)), hash: h}
}

func (s *Store) event(ctx context.Context, q persistence.DBTX, id, typ string, detail any) {
	b, _ := json.Marshal(detail)
	_, _ = q.Exec(ctx, `INSERT INTO authorization_events (authorization_id, type, detail, created_at) VALUES ($1, $2, $3::jsonb, $4)`, id, typ, string(b), s.now())
}

// Create cria a autorização. Com DecisionID, a decisão precisa estar resolvida e
// ser da mesma execução.
func (s *Store) Create(ctx context.Context, in NewAuthorization) (Authorization, error) {
	b := boundOf(in.Action)
	switch {
	case b.tool == "" || b.operation == "":
		return Authorization{}, &FieldError{"action", "Ferramenta e operação são obrigatórias."}
	case in.MaxUses < 1 || in.MaxUses > 100:
		return Authorization{}, &FieldError{"max_uses", "O número de usos deve estar entre 1 e 100."}
	case in.TTL <= 0 || in.TTL > MaxTTL:
		return Authorization{}, &FieldError{"ttl", "O prazo deve ser positivo e de até 24 horas."}
	case in.ProjectID == "" || in.ExecutionID == "":
		return Authorization{}, &FieldError{"execution", "Projeto e execução são obrigatórios."}
	}
	var decision, granted any
	if in.DecisionID != "" {
		decision = in.DecisionID
	}
	if in.GrantedBy != "" {
		granted = in.GrantedBy
	}
	var out Authorization
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if in.DecisionID != "" {
			var state, proj, exec string
			err := tx.QueryRow(ctx, `SELECT state, project_id::text, execution_id::text FROM decision_requests WHERE id = $1`, in.DecisionID).Scan(&state, &proj, &exec)
			if errors.Is(persistence.MapError(err), persistence.ErrNotFound) || (err == nil && (state != "resolved" || proj != in.ProjectID || exec != in.ExecutionID)) {
				return ErrDecision
			}
			if err != nil {
				return persistence.MapError(err)
			}
		}
		var err error
		out, err = scan(tx.QueryRow(ctx, `INSERT INTO authorizations (project_id, execution_id, node_id, decision_id, tool, operation, resource, target, params_hash, max_uses, expires_at, granted_by)
			VALUES ($1, $2, $3, $4::uuid, $5, $6, $7, $8, $9, $10, $11, $12::uuid) RETURNING `+cols,
			in.ProjectID, in.ExecutionID, in.NodeID, decision, b.tool, b.operation, b.resource, b.target, b.hash, in.MaxUses, s.now().Add(in.TTL), granted))
		if err != nil {
			return err
		}
		s.event(ctx, tx, out.ID, "created", map[string]any{"tool": b.tool, "operation": b.operation, "max_uses": in.MaxUses})
		return nil
	})
	return out, err
}

// Get lê uma autorização do projeto.
func (s *Store) Get(ctx context.Context, projectID, id string) (Authorization, error) {
	return scan(s.DB.QueryRow(ctx, `SELECT `+cols+` FROM authorizations WHERE id = $1 AND project_id = $2`, id, projectID))
}

// Use é um consumo registrado.
type Use struct {
	ID            int64 `json:"id"`
	UsesRemaining int   `json:"uses_remaining"`
}

// Consume gasta UM uso, de forma atômica, só se a ação for exatamente a autorizada,
// a autorização estiver ativa, dentro do prazo e com usos restantes. Qualquer
// divergência recusa SEM gastar uso e registra a recusa.
func (s *Store) Consume(ctx context.Context, projectID, id string, action policy.Action) (Use, error) {
	b := boundOf(action)
	now := s.now()
	var u Use
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		var uses, maxUses int
		err := tx.QueryRow(ctx, `UPDATE authorizations SET uses = uses + 1, version = version + 1
			WHERE id = $1 AND project_id = $2 AND state = 'active' AND uses < max_uses AND expires_at > $3
			  AND tool = $4 AND operation = $5 AND resource = $6 AND target = $7 AND params_hash = $8
			RETURNING uses, max_uses`, id, projectID, now, b.tool, b.operation, b.resource, b.target, b.hash).Scan(&uses, &maxUses)
		if err == nil {
			if err := tx.QueryRow(ctx, `INSERT INTO authorization_uses (authorization_id, started_at) VALUES ($1, $2) RETURNING id`, id, now).Scan(&u.ID); err != nil {
				return persistence.MapError(err)
			}
			u.UsesRemaining = maxUses - uses
			s.event(ctx, tx, id, "consumed", map[string]any{"use": u.ID, "remaining": u.UsesRemaining})
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return persistence.MapError(err)
		}
		// Nenhuma linha: descobre o motivo (sem gastar uso) e registra a recusa.
		cur, gerr := scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM authorizations WHERE id = $1 AND project_id = $2`, id, projectID))
		if gerr != nil {
			return gerr
		}
		u = Use{}
		return diagnose(cur, b, now)
	})
	if err != nil && (errors.Is(err, ErrExpired) || errors.Is(err, ErrRevoked) || errors.Is(err, ErrExhausted) || isMismatch(err)) {
		// A transação foi desfeita junto com o erro; grava a recusa à parte para a trilha.
		s.event(context.WithoutCancel(ctx), s.DB, id, "refused", map[string]any{"reason": reasonOf(err)})
	}
	return u, err
}

func isMismatch(err error) bool { var m *MismatchError; return errors.As(err, &m) }

func reasonOf(err error) string {
	var m *MismatchError
	switch {
	case errors.As(err, &m):
		return "mismatch:" + m.Field
	case errors.Is(err, ErrRevoked):
		return "revoked"
	case errors.Is(err, ErrExpired):
		return "expired"
	case errors.Is(err, ErrExhausted):
		return "exhausted"
	}
	return "unknown"
}

func diagnose(a Authorization, b bound, now time.Time) error {
	switch {
	case a.State == "revoked":
		return ErrRevoked
	case !a.ExpiresAt.After(now):
		return ErrExpired
	case a.Tool != b.tool:
		return &MismatchError{"tool"}
	case a.Operation != b.operation:
		return &MismatchError{"operation"}
	case a.Resource != b.resource:
		return &MismatchError{"resource"}
	case a.Target != b.target:
		return &MismatchError{"target"}
	case a.ParamsHash != b.hash:
		return &MismatchError{"params"}
	case a.Uses >= a.MaxUses:
		return ErrExhausted
	}
	return errors.New("autorização indisponível")
}

// Finish marca o uso como concluído (ok) ou falho. Uso não concluído fica "em
// andamento": é o que a revogação procura para decidir pela parada segura.
func (s *Store) Finish(ctx context.Context, useID int64, ok bool) error {
	state := "done"
	if !ok {
		state = "failed"
	}
	_, err := s.DB.Exec(ctx, `UPDATE authorization_uses SET state = $2, finished_at = $3 WHERE id = $1 AND state = 'started'`, useID, state, s.now())
	return persistence.MapError(err)
}

// Do consome um uso e roda a operação; o uso fica "em andamento" até `run` voltar.
// A operação só roda se o consumo for autorizado.
func (s *Store) Do(ctx context.Context, projectID, id string, action policy.Action, run func(ctx context.Context) error) error {
	u, err := s.Consume(ctx, projectID, id, action)
	if err != nil {
		return err
	}
	runErr := run(ctx)
	if ferr := s.Finish(context.WithoutCancel(ctx), u.ID, runErr == nil); ferr != nil && runErr == nil {
		return ferr
	}
	return runErr
}

// InFlight conta os usos consumidos e ainda não concluídos.
func (s *Store) InFlight(ctx context.Context, id string) (int, error) {
	var n int
	err := s.DB.QueryRow(ctx, `SELECT count(*) FROM authorization_uses WHERE authorization_id = $1 AND state = 'started'`, id).Scan(&n)
	return n, persistence.MapError(err)
}

// Revocation é o resultado da revogação.
type Revocation struct {
	Authorization  Authorization `json:"authorization"`
	InFlight       int           `json:"in_flight"`
	Halted         bool          `json:"halted"`
	AlreadyRevoked bool          `json:"already_revoked"`
}

// ErrHaltFailed: a revogação foi gravada (nenhum uso novo passa), mas a parada
// segura da execução com uso em andamento falhou e precisa de atenção.
var ErrHaltFailed = errors.New("revogada, mas a parada segura falhou")

// Revoke revoga a autorização. É imediato: novos consumos falham a partir do
// commit. Havendo uso em andamento, dispara a parada segura (Halter); se ela
// falhar, a revogação continua valendo e o erro ErrHaltFailed sinaliza o pendente.
func (s *Store) Revoke(ctx context.Context, projectID, id, adminID, reason string) (Revocation, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) > 500 {
		return Revocation{}, &FieldError{"reason", "Motivo longo demais (até 500)."}
	}
	var admin any
	if adminID != "" {
		admin = adminID
	}
	var rev Revocation
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		cur, err := scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM authorizations WHERE id = $1 AND project_id = $2 FOR UPDATE`, id, projectID))
		if err != nil {
			return err
		}
		if cur.State == "revoked" {
			rev = Revocation{Authorization: cur, AlreadyRevoked: true}
			return nil
		}
		rev.Authorization, err = scan(tx.QueryRow(ctx, `UPDATE authorizations SET state = 'revoked', revoked_at = $3, revoked_by = $4::uuid, revoke_reason = $5, version = version + 1
			WHERE id = $1 AND project_id = $2 RETURNING `+cols, id, projectID, s.now(), admin, reason))
		if err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM authorization_uses WHERE authorization_id = $1 AND state = 'started'`, id).Scan(&rev.InFlight); err != nil {
			return persistence.MapError(err)
		}
		s.event(ctx, tx, id, "revoked", map[string]any{"in_flight": rev.InFlight, "reason": reason})
		return nil
	})
	if err != nil || rev.AlreadyRevoked || rev.InFlight == 0 {
		return rev, err
	}
	// Uso em andamento: parada segura (fora da transação da revogação, que já valeu).
	bg := context.WithoutCancel(ctx)
	if s.Halter == nil {
		s.event(bg, s.DB, id, "halt_failed", map[string]any{"error": "sem executor de parada segura"})
		return rev, fmt.Errorf("%w: sem executor de parada segura configurado", ErrHaltFailed)
	}
	if herr := s.Halter.Halt(bg, projectID, rev.Authorization.ExecutionID); herr != nil {
		s.event(bg, s.DB, id, "halt_failed", map[string]any{"error": herr.Error()})
		return rev, fmt.Errorf("%w: %v", ErrHaltFailed, herr)
	}
	rev.Halted = true
	s.event(bg, s.DB, id, "halted", map[string]any{"execution": rev.Authorization.ExecutionID})
	return rev, nil
}

// Event é uma linha da trilha.
type Event struct {
	Type      string          `json:"type"`
	Detail    json.RawMessage `json:"detail"`
	CreatedAt time.Time       `json:"created_at"`
}

// Events devolve a trilha da autorização em ordem.
func (s *Store) Events(ctx context.Context, id string) ([]Event, error) {
	rows, err := s.DB.Query(ctx, `SELECT type, detail, created_at FROM authorization_events WHERE authorization_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		var d []byte
		if err := rows.Scan(&e.Type, &d, &e.CreatedAt); err != nil {
			return nil, persistence.MapError(err)
		}
		e.Detail = d
		out = append(out, e)
	}
	return out, persistence.MapError(rows.Err())
}
