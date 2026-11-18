package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/stdflow"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// ErrCancelled interrompe o avanço de uma execução cancelada.
var ErrCancelled = errors.New("execução cancelada")

// Tipos de execução.
const (
	KindManual = "manual"
	KindTest   = "test"
)

// Status de uma execução do fluxo (espelha o do motor, mais "queued" e "cancelled").
const (
	StatusQueued    = "queued"
	StatusCancelled = "cancelled"
)

// Tipos de evento gravados em execution_events.
const (
	EventStarted  = "run_started"
	EventStep     = "run_step"
	EventDecision = "run_decision"
	EventFinished = "run_finished"
	EventCancel   = "run_cancelled"
	EventNode     = "node_event" // eventos livres de um nó (ex.: saída do agente)
)

// StartInput descreve a execução pedida.
type StartInput struct {
	ProjectID  string
	WorkflowID string // vazio = o fluxo do projeto
	Title      string
	Task       string // o que a fábrica deve fazer, em texto
	Workspace  string // pasta do repositório, relativa à raiz de workspaces
	CreatedBy  string
	Kind       string // "manual" (padrão) ou "test"
	// Variáveis de partida do fluxo padrão.
	Correction bool // fabrica.correcao
	PRDDone    bool // prd.concluido
}

// Run é uma execução do fluxo.
type Run struct {
	ExecutionID string                `json:"execution_id"`
	ProjectID   string                `json:"project_id"`
	VersionID   string                `json:"workflow_version_id"`
	Title       string                `json:"title"`
	Task        string                `json:"task"`
	Workspace   string                `json:"workspace"`
	Kind        string                `json:"kind"`
	Status      string                `json:"status"`
	Outcome     string                `json:"outcome"`
	Failure     string                `json:"failure"`
	Pending     []string              `json:"pending"`
	Trace       []workflows.TraceStep `json:"trace"`
	WakeAt      *time.Time            `json:"wake_at"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`

	State workflows.RunState `json:"-"`
}

// FieldError é um campo inválido com mensagem segura.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

const runColumns = `r.execution_id, r.project_id, r.workflow_version_id, r.title, r.task, r.workspace, r.kind, r.status, r.outcome, r.failure, r.state, r.wake_at, r.created_at, r.updated_at`

func scanRun(row interface{ Scan(...any) error }) (Run, error) {
	var r Run
	var raw []byte
	if err := row.Scan(&r.ExecutionID, &r.ProjectID, &r.VersionID, &r.Title, &r.Task, &r.Workspace, &r.Kind, &r.Status, &r.Outcome, &r.Failure,
		&raw, &r.WakeAt, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return r, persistence.MapError(err)
	}
	if err := json.Unmarshal(raw, &r.State); err != nil {
		return r, fmt.Errorf("estado da execução ilegível: %w", err)
	}
	r.Pending = append([]string{}, r.State.Pending...)
	r.Trace = append([]workflows.TraceStep{}, r.State.Trace...)
	return r, nil
}

// InitialVars monta as variáveis de partida (o que as condições do fluxo padrão leem).
func InitialVars(in StartInput) map[string]any {
	return map[string]any{
		"fabrica": map[string]any{"correcao": in.Correction, "titulo": in.Title, "tarefa": in.Task},
		"prd":     map[string]any{"concluido": in.PRDDone},
	}
}

func randomSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Start cria a execução a partir da versão PUBLICADA do workflow (rascunho não executa).
func Start(ctx context.Context, db DB, in StartInput) (Run, error) {
	in.Title, in.Task = strings.TrimSpace(in.Title), strings.TrimSpace(in.Task)
	if in.Title == "" || len(in.Title) > 200 {
		return Run{}, &FieldError{"title", "Informe um título de até 200 caracteres."}
	}
	if len(in.Task) > 20000 {
		return Run{}, &FieldError{"task", "A tarefa deve ter até 20000 caracteres."}
	}
	if in.WorkflowID == "" {
		id, err := stdflow.Of(ctx, db, in.ProjectID)
		if err != nil {
			return Run{}, persistence.MapError(err)
		}
		if id == "" {
			return Run{}, fmt.Errorf("%w: o projeto ainda não tem workflow", persistence.ErrInvalidInput)
		}
		in.WorkflowID = id
	}
	if in.Kind == "" {
		in.Kind = KindManual
	}
	if in.Kind != KindManual && in.Kind != KindTest {
		return Run{}, &FieldError{"kind", "Tipo de execução inválido."}
	}
	feat, err := persistence.CreateFeature(ctx, db, in.ProjectID, "manual-"+randomSuffix(), in.Title)
	if err != nil {
		return Run{}, err
	}
	ex, ver, err := releases.StartExecution(ctx, db, in.ProjectID, feat.ID, in.WorkflowID)
	if err != nil {
		_, _ = db.Exec(ctx, `DELETE FROM features WHERE id = $1`, feat.ID)
		return Run{}, err
	}
	st, err := workflows.NewRun(ver.FlatGraph, InitialVars(in))
	if err != nil {
		return Run{}, err
	}
	raw, _ := json.Marshal(st)
	var run Run
	err = persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO execution_runs (execution_id, project_id, workflow_version_id, title, task, workspace, state, created_by, kind)
			VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9)
			RETURNING `+strings.ReplaceAll(runColumns, "r.", ""),
			ex.ID, in.ProjectID, ver.ID, in.Title, in.Task, in.Workspace, string(raw), in.CreatedBy, in.Kind)
		var err error
		if run, err = scanRun(row); err != nil {
			return err
		}
		_, err = persistence.AppendEvent(ctx, tx, in.ProjectID, ex.ID, EventStarted, map[string]any{
			"title": in.Title, "workflow_version": ver.Number, "workflow": ver.Name,
		})
		return err
	})
	return run, err
}

// Get lê uma execução do projeto.
func Get(ctx context.Context, db persistence.DBTX, projectID, executionID string) (Run, error) {
	return scanRun(db.QueryRow(ctx, `SELECT `+runColumns+` FROM execution_runs r WHERE r.execution_id = $1 AND r.project_id = $2`, executionID, projectID))
}

// List devolve as execuções do projeto, as mais novas primeiro.
func List(ctx context.Context, db persistence.DBTX, projectID string, limit int) ([]Run, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := db.Query(ctx, `SELECT `+runColumns+` FROM execution_runs r WHERE r.project_id = $1 ORDER BY r.created_at DESC LIMIT $2`, projectID, limit)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Claim reclama a próxima execução que precisa de um worker: nova, em andamento com a posse vencida
// (o worker anterior caiu) ou dormindo com o prazo da espera vencido. Devolve nil se não há nenhuma.
func Claim(ctx context.Context, db DB, owner string, lease time.Duration) (*Run, error) {
	var run Run
	var found bool
	err := persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		var id string
		err := tx.QueryRow(ctx, `
			SELECT execution_id FROM execution_runs
			WHERE (lease_until IS NULL OR lease_until < now())
			  AND (status IN ('queued', 'running') OR (status = 'waiting' AND wake_at <= now()))
			ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		row := tx.QueryRow(ctx, `
			UPDATE execution_runs r SET status = 'running', owner = $2, lease_until = now() + make_interval(secs => $3), updated_at = now()
			WHERE r.execution_id = $1 RETURNING `+runColumns, id, owner, lease.Seconds())
		if run, err = scanRun(row); err != nil {
			return err
		}
		found = true
		_, err = tx.Exec(ctx, `UPDATE executions SET state = 'running', updated_at = now(), version = version + 1 WHERE id = $1`, id)
		return err
	})
	if err != nil || !found {
		return nil, persistence.MapError(err)
	}
	return &run, nil
}

// executionState traduz o estado do motor no da execução.
func executionState(st workflows.RunState, status string) string {
	switch {
	case status == StatusCancelled:
		return "cancelled"
	case st.Status == workflows.RunCompleted && st.Outcome == "success":
		return "succeeded"
	case st.Status == workflows.RunCompleted, st.Status == workflows.RunFailed:
		return "failed"
	case st.Status == workflows.RunWaiting, st.Status == workflows.RunAwaitingDecision:
		return "waiting"
	}
	return "running"
}

func runStatus(st workflows.RunState) string {
	switch st.Status {
	case workflows.RunCompleted, workflows.RunFailed, workflows.RunWaiting, workflows.RunAwaitingDecision:
		return st.Status
	}
	return "running"
}

func wakeAt(st workflows.RunState) *time.Time {
	var min *time.Time
	for _, t := range st.WaitUntil {
		t := t
		if min == nil || t.Before(*min) {
			min = &t
		}
	}
	return min
}

// save grava o estado, o resumo e os passos novos como eventos. Recusa gravar sobre uma execução cancelada.
func save(ctx context.Context, db DB, run *Run, st workflows.RunState, seen int, lease time.Duration) error {
	raw, _ := json.Marshal(st)
	status := runStatus(st)
	return persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		var leaseSecs any
		if status == "running" {
			leaseSecs = lease.Seconds()
		}
		tag, err := tx.Exec(ctx, `
			UPDATE execution_runs SET state = $2::jsonb, status = $3, outcome = $4, failure = $5, wake_at = $6,
			       lease_until = CASE WHEN $7::double precision IS NULL THEN NULL ELSE now() + make_interval(secs => $7) END,
			       owner = CASE WHEN $7::double precision IS NULL THEN NULL ELSE owner END, updated_at = now()
			WHERE execution_id = $1 AND status <> 'cancelled'`,
			run.ExecutionID, string(raw), status, st.Outcome, st.Failure, wakeAt(st), leaseSecs)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrCancelled
		}
		if _, err := tx.Exec(ctx, `UPDATE executions SET state = $2, updated_at = now(), version = version + 1 WHERE id = $1`,
			run.ExecutionID, executionState(st, status)); err != nil {
			return err
		}
		for _, step := range st.Trace[seen:] {
			if _, err := persistence.AppendEvent(ctx, tx, run.ProjectID, run.ExecutionID, EventStep, step); err != nil {
				return err
			}
		}
		if status == "completed" || status == "failed" {
			if _, err := persistence.AppendEvent(ctx, tx, run.ProjectID, run.ExecutionID, EventFinished,
				map[string]any{"status": status, "outcome": st.Outcome, "failure": st.Failure}); err != nil {
				return err
			}
		}
		return nil
	})
}

// Emit grava um evento livre da execução (usado pelos executores, ex.: a saída do terminal do agente).
func Emit(ctx context.Context, db DB, projectID, executionID, eventType string, payload any) error {
	return persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		_, err := persistence.AppendEvent(ctx, tx, projectID, executionID, eventType, payload)
		return err
	})
}

// Options configura o avanço.
type Options struct {
	Exec  workflows.Executor
	Lease time.Duration
	Now   func() time.Time
	// WatchEvery é de quanto em quanto tempo se confere se a execução foi cancelada enquanto um nó roda.
	WatchEvery time.Duration
}

type ctxKey struct{}

// Info é o que um executor precisa saber da execução em curso.
type Info struct {
	Run  Run
	Emit func(eventType string, payload any)
}

// InfoFrom devolve a execução em curso (ok=false fora de uma execução).
func InfoFrom(ctx context.Context) (Info, bool) { i, ok := ctx.Value(ctxKey{}).(Info); return i, ok }

// Advance conduz a execução reclamada até ela concluir, falhar, dormir ou esperar uma decisão.
func Advance(ctx context.Context, db DB, run Run, opt Options) (Run, error) {
	if opt.Lease <= 0 {
		opt.Lease = 2 * time.Minute
	}
	if opt.WatchEvery <= 0 {
		opt.WatchEvery = 2 * time.Second
	}
	ver, err := releases.Get(ctx, db, run.VersionID)
	if err != nil {
		return run, err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go watchRun(ctx, db, run.ExecutionID, opt.WatchEvery, opt.Lease, cancel)

	st := run.State
	seen := len(st.Trace)
	info := Info{Run: run, Emit: func(t string, p any) {
		_ = Emit(context.WithoutCancel(ctx), db, run.ProjectID, run.ExecutionID, t, p)
	}}
	ctx = context.WithValue(ctx, ctxKey{}, info)
	eng := &workflows.Engine{Exec: opt.Exec, Now: opt.Now}
	if run.Kind == KindTest {
		// o teste do fluxo não dorme: a espera programada (ex.: um dia pelo merge) vira um instante
		eng.WaitFor = func(int) time.Duration { return 0 }
	}
	eng.OnStep = func(ctx context.Context, s *workflows.RunState) error {
		if err := save(ctx, db, &run, *s, seen, opt.Lease); err != nil {
			return err
		}
		seen = len(s.Trace)
		return nil
	}
	err = eng.Advance(ctx, ver.FlatGraph, &st)
	if errors.Is(err, context.Canceled) && cancelled(context.WithoutCancel(ctx), db, run.ExecutionID) {
		err = ErrCancelled
	}
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return run, ErrCancelled
		}
		return run, err
	}
	if err := save(context.WithoutCancel(ctx), db, &run, st, seen, opt.Lease); err != nil {
		return run, err
	}
	return Get(context.WithoutCancel(ctx), db, run.ProjectID, run.ExecutionID)
}

func cancelled(ctx context.Context, db DB, id string) bool {
	var s string
	return db.QueryRow(ctx, `SELECT status FROM execution_runs WHERE execution_id = $1`, id).Scan(&s) == nil && s == StatusCancelled
}

// watchRun confere a cada tick se a execução foi cancelada (interrompendo o nó em andamento) e renova a
// posse (lease): um nó longo, como um agente, não pode deixar outro worker reclamar a execução.
func watchRun(ctx context.Context, db DB, id string, every, lease time.Duration, cancel context.CancelFunc) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if cancelled(ctx, db, id) {
				cancel()
				return
			}
			_, _ = db.Exec(ctx, `UPDATE execution_runs SET lease_until = now() + make_interval(secs => $2) WHERE execution_id = $1 AND status = 'running'`, id, lease.Seconds())
		}
	}
}

// Cancel cancela uma execução ainda aberta. O nó que estiver rodando é interrompido pelo worker.
func Cancel(ctx context.Context, db DB, projectID, executionID, by string) (Run, error) {
	err := persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE execution_runs SET status = 'cancelled', lease_until = NULL, updated_at = now()
			WHERE execution_id = $1 AND project_id = $2 AND status IN ('queued', 'running', 'waiting', 'awaiting_decision')`,
			executionID, projectID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("%w: a execução já terminou ou não existe", persistence.ErrInvalidInput)
		}
		if _, err := tx.Exec(ctx, `UPDATE executions SET state = 'cancelled', updated_at = now(), version = version + 1 WHERE id = $1`, executionID); err != nil {
			return err
		}
		_, err = persistence.AppendEvent(ctx, tx, projectID, executionID, EventCancel, map[string]any{"by": by})
		return err
	})
	if err != nil {
		return Run{}, err
	}
	return Get(ctx, db, projectID, executionID)
}

// Decide registra a resposta de uma pessoa a um nó "Decisão humana" pendente e devolve a execução à fila.
func Decide(ctx context.Context, db DB, projectID, executionID, node, choice, by string) (Run, error) {
	err := persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		var raw []byte
		var status, versionID string
		err := tx.QueryRow(ctx, `SELECT state, status, workflow_version_id::text FROM execution_runs WHERE execution_id = $1 AND project_id = $2 FOR UPDATE`,
			executionID, projectID).Scan(&raw, &status, &versionID)
		if err != nil {
			return persistence.MapError(err)
		}
		if status != workflows.RunAwaitingDecision {
			return fmt.Errorf("%w: a execução não está esperando uma decisão", persistence.ErrInvalidInput)
		}
		var st workflows.RunState
		if err := json.Unmarshal(raw, &st); err != nil {
			return err
		}
		ver, err := releases.Get(ctx, tx, versionID)
		if err != nil {
			return err
		}
		eng := &workflows.Engine{}
		if err := eng.Decide(ver.FlatGraph, &st, node, choice); err != nil {
			return fmt.Errorf("%w: %v", persistence.ErrInvalidInput, err)
		}
		out, _ := json.Marshal(st)
		if _, err := tx.Exec(ctx, `UPDATE execution_runs SET state = $2::jsonb, status = 'queued', lease_until = NULL, updated_at = now() WHERE execution_id = $1`,
			executionID, string(out)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE executions SET state = 'queued', updated_at = now(), version = version + 1 WHERE id = $1`, executionID); err != nil {
			return err
		}
		_, err = persistence.AppendEvent(ctx, tx, projectID, executionID, EventDecision, map[string]any{"node": node, "choice": choice, "by": by})
		return err
	})
	if err != nil {
		return Run{}, err
	}
	return Get(ctx, db, projectID, executionID)
}
