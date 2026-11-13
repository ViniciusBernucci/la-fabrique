package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/tasksync"
)

// Estados de uma tarefa.
const (
	StatePending = "pending"
	StateRunning = "running"
	StateDone    = "done"
	StateBlocked = "blocked"
)

// Erros.
var (
	ErrRejected = errors.New("plano rejeitado: a versão anterior foi preservada")
	ErrNoPlan   = errors.New("a feature ainda não tem plano")
)

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Store guarda os planos.
type Store struct {
	DB  DB
	Now func() time.Time
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// Stored é uma versão gravada.
type Stored struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	Status    string    `json:"status"`
	Hash      string    `json:"content_hash"`
	Plan      Plan      `json:"plan"`
	Problems  []Problem `json:"problems"`
	CreatedAt time.Time `json:"created_at"`
}

// Outcome é o resultado de propor um plano.
type Outcome struct {
	Accepted  bool      `json:"accepted"`
	Unchanged bool      `json:"unchanged"` // o conteúdo é igual ao vigente: nenhuma versão nova
	Stored    Stored    `json:"stored"`
	Problems  []Problem `json:"problems"`
}

// RejectedError é devolvido junto do Outcome quando o plano não é aceito.
type RejectedError struct{ Problems []Problem }

func (e *RejectedError) Error() string {
	parts := make([]string, 0, len(e.Problems))
	for _, p := range e.Problems {
		parts = append(parts, p.String())
	}
	return ErrRejected.Error() + ": " + strings.Join(parts, "; ")
}
func (e *RejectedError) Unwrap() error { return ErrRejected }

// Task é a tarefa gravada (identidade estável).
type StoredTask struct {
	Key       string `json:"key"`
	Title     string `json:"title"`
	State     string `json:"state"`
	Removed   bool   `json:"removed"`
	RemoteID  string `json:"remote_id"`
	RemoteKey string `json:"remote_key"`
	Hash      string `json:"content_hash"`
	FromPlanV int    `json:"plan_version"`
}

const planCols = `id, version, status, content, content_hash, problems, created_at`

func scanPlan(row interface{ Scan(...any) error }) (Stored, error) {
	var s Stored
	var content, problems []byte
	if err := row.Scan(&s.ID, &s.Version, &s.Status, &content, &s.Hash, &problems, &s.CreatedAt); err != nil {
		return s, persistence.MapError(err)
	}
	if err := json.Unmarshal(content, &s.Plan); err != nil {
		return s, fmt.Errorf("plano ilegível: %w", err)
	}
	s.Problems = []Problem{}
	_ = json.Unmarshal(problems, &s.Problems)
	return s, nil
}

// Current devolve a versão vigente.
func (s *Store) Current(ctx context.Context, projectID, featureID string) (Stored, error) {
	st, err := scanPlan(s.DB.QueryRow(ctx, `SELECT `+planCols+` FROM plans WHERE project_id = $1 AND feature_id = $2 AND status = 'current'`, projectID, featureID))
	if errors.Is(err, persistence.ErrNotFound) {
		return Stored{}, ErrNoPlan
	}
	return st, err
}

// Versions lista todas as versões (inclusive rejeitadas), da mais nova para a mais antiga.
func (s *Store) Versions(ctx context.Context, projectID, featureID string) ([]Stored, error) {
	rows, err := s.DB.Query(ctx, `SELECT `+planCols+` FROM plans WHERE project_id = $1 AND feature_id = $2 ORDER BY version DESC`, projectID, featureID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Stored{}
	for rows.Next() {
		st, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, persistence.MapError(rows.Err())
}

// Tasks lista as tarefas (identidade estável) da feature.
func (s *Store) Tasks(ctx context.Context, projectID, featureID string) ([]StoredTask, error) {
	return s.tasksQ(ctx, s.DB, projectID, featureID)
}

// conflicts compara o plano novo com as tarefas já gravadas: uma tarefa que saiu
// de "pendente" não pode ser alterada nem removida.
func conflicts(existing []StoredTask, p Plan) []Problem {
	in := map[string]Task{}
	for _, t := range p.Tasks {
		in[t.Key] = t
	}
	var out []Problem
	for _, e := range existing {
		if e.State == StatePending || e.Removed {
			continue
		}
		t, ok := in[e.Key]
		switch {
		case !ok:
			out = append(out, Problem{e.Key, "key", fmt.Sprintf("a tarefa já está %s e não pode ser removida", stateLabel(e.State))})
		case TaskHash(t) != e.Hash:
			out = append(out, Problem{e.Key, "content", fmt.Sprintf("a tarefa já está %s e não pode ser alterada", stateLabel(e.State))})
		}
	}
	return out
}

func stateLabel(s string) string {
	switch s {
	case StateRunning:
		return "em execução"
	case StateDone:
		return "concluída"
	case StateBlocked:
		return "bloqueada"
	}
	return s
}

// Propose valida e grava um plano. Inválido ou conflitante: registrado como
// rejeitado, a versão vigente fica intacta e o erro é *RejectedError.
func (s *Store) Propose(ctx context.Context, projectID, featureID string, plan Plan) (Outcome, error) {
	problems := Validate(plan)
	norm := Normalize(plan)
	hash := Hash(plan)
	var out Outcome
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 10))`, featureID); err != nil {
			return persistence.MapError(err)
		}
		var cur *Stored
		if c, err := scanPlan(tx.QueryRow(ctx, `SELECT `+planCols+` FROM plans WHERE feature_id = $1 AND status = 'current'`, featureID)); err == nil {
			cur = &c
		} else if !errors.Is(err, persistence.ErrNotFound) {
			return err
		}
		existing, err := s.tasksQ(ctx, tx, projectID, featureID)
		if err != nil {
			return err
		}
		if len(problems) == 0 {
			problems = conflicts(existing, norm)
		}
		if len(problems) == 0 && cur != nil && cur.Hash == hash {
			out = Outcome{Accepted: true, Unchanged: true, Stored: *cur, Problems: []Problem{}}
			return nil
		}
		var version int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(max(version), 0) + 1 FROM plans WHERE feature_id = $1`, featureID).Scan(&version); err != nil {
			return persistence.MapError(err)
		}
		content, _ := json.Marshal(norm)
		pj, _ := json.Marshal(nonNil(problems))
		status := "current"
		if len(problems) > 0 {
			status = "rejected"
		} else if cur != nil {
			if _, err := tx.Exec(ctx, `UPDATE plans SET status = 'superseded' WHERE id = $1`, cur.ID); err != nil {
				return persistence.MapError(err)
			}
		}
		var st Stored
		st, err = scanPlan(tx.QueryRow(ctx, `INSERT INTO plans (project_id, feature_id, version, status, content, content_hash, problems, created_at)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7::jsonb, $8) RETURNING `+planCols,
			projectID, featureID, version, status, string(content), hash, string(pj), s.now()))
		if err != nil {
			return err
		}
		out = Outcome{Accepted: status == "current", Stored: st, Problems: nonNil(problems)}
		if status != "current" {
			return nil
		}
		inPlan := map[string]bool{}
		for _, t := range norm.Tasks {
			inPlan[t.Key] = true
			if _, err := tx.Exec(ctx, `
				INSERT INTO plan_tasks (project_id, feature_id, task_key, title, content_hash, plan_version, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (feature_id, task_key) DO UPDATE SET title = $4, content_hash = $5, removed = false, plan_version = $6, updated_at = $7`,
				projectID, featureID, t.Key, t.Title, TaskHash(t), version, s.now()); err != nil {
				return persistence.MapError(err)
			}
		}
		for _, e := range existing {
			if !inPlan[e.Key] && !e.Removed {
				if _, err := tx.Exec(ctx, `UPDATE plan_tasks SET removed = true, updated_at = $3 WHERE feature_id = $1 AND task_key = $2`, featureID, e.Key, s.now()); err != nil {
					return persistence.MapError(err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return Outcome{}, err
	}
	if !out.Accepted {
		return out, &RejectedError{Problems: out.Problems}
	}
	return out, nil
}

func nonNil(p []Problem) []Problem {
	if p == nil {
		return []Problem{}
	}
	return p
}

// TasksOf lê as tarefas com qualquer consulta (inclusive dentro de uma transação).
func TasksOf(ctx context.Context, q persistence.DBTX, projectID, featureID string) ([]StoredTask, error) {
	return (&Store{}).tasksQ(ctx, q, projectID, featureID)
}

func (s *Store) tasksQ(ctx context.Context, q persistence.DBTX, projectID, featureID string) ([]StoredTask, error) {
	rows, err := q.Query(ctx, `SELECT task_key, title, state, removed, remote_id, remote_key, content_hash, plan_version FROM plan_tasks
		WHERE project_id = $1 AND feature_id = $2 ORDER BY task_key`, projectID, featureID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []StoredTask{}
	for rows.Next() {
		var t StoredTask
		if err := rows.Scan(&t.Key, &t.Title, &t.State, &t.Removed, &t.RemoteID, &t.RemoteKey, &t.Hash, &t.FromPlanV); err != nil {
			return nil, persistence.MapError(err)
		}
		out = append(out, t)
	}
	return out, persistence.MapError(rows.Err())
}

var stateFlow = map[string][]string{StatePending: {StateRunning, StateBlocked}, StateRunning: {StateDone, StateBlocked, StatePending}, StateBlocked: {StatePending, StateRunning}}

// SetTaskState avança o estado de uma tarefa; transição inválida é recusada.
func (s *Store) SetTaskState(ctx context.Context, projectID, featureID, key, to string) error {
	return persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 10))`, featureID); err != nil {
			return persistence.MapError(err)
		}
		var cur string
		if err := tx.QueryRow(ctx, `SELECT state FROM plan_tasks WHERE project_id = $1 AND feature_id = $2 AND task_key = $3 AND NOT removed`, projectID, featureID, key).Scan(&cur); err != nil {
			return persistence.MapError(err)
		}
		ok := false
		for _, n := range stateFlow[cur] {
			ok = ok || n == to
		}
		if !ok {
			return fmt.Errorf("%w: transição %s → %s não permitida", persistence.ErrInvalidInput, cur, to)
		}
		_, err := tx.Exec(ctx, `UPDATE plan_tasks SET state = $4, updated_at = $5 WHERE project_id = $1 AND feature_id = $2 AND task_key = $3`, projectID, featureID, key, to, s.now())
		return persistence.MapError(err)
	})
}

// --- publicação na origem -------------------------------------------------------

// PublishReport descreve o que a publicação fez.
type PublishReport struct {
	Subtasks     map[string]string `json:"subtasks"` // chave → ID remoto
	Created      int               `json:"created"`
	AlreadyThere int               `json:"already_there"`
	Dependencies int               `json:"dependencies"`
	Failed       []string          `json:"failed"`
}

// BusinessKey é a chave estável da subtarefa de uma tarefa: a mesma em toda versão
// do plano e específica da identidade remota da feature, então depois de uma troca de
// origem (nova identidade) a publicação cria os itens na nova origem em vez de dar o
// efeito da antiga como feito.
func BusinessKey(featureID, remoteID, taskKey string) string {
	return "plan:" + featureID + "@" + remoteID + ":" + taskKey
}

// Publish cria, pelo ledger, UMA subtarefa por tarefa e UMA relação por
// dependência na origem. Repetir (inclusive depois de timeout) não duplica.
// Tarefa removida do plano não é apagada na origem.
func (s *Store) Publish(ctx context.Context, projectID, featureID, parentRemoteID string, pub *tasksync.Publisher) (PublishReport, error) {
	cur, err := s.Current(ctx, projectID, featureID)
	if err != nil {
		return PublishReport{}, err
	}
	var remoteID string
	if err := s.DB.QueryRow(ctx, `SELECT external_id FROM features WHERE project_id = $1 AND id = $2`, projectID, featureID).Scan(&remoteID); err != nil {
		return PublishReport{}, persistence.MapError(err)
	}
	rep := PublishReport{Subtasks: map[string]string{}, Failed: []string{}}
	by := map[string]Task{}
	for _, t := range cur.Plan.Tasks {
		by[t.Key] = t
	}
	for _, key := range Order(cur.Plan) {
		t := by[key]
		bk := BusinessKey(featureID, remoteID, key)
		res, err := pub.CreateSubtask(ctx, projectID, bk, parentRemoteID, t.Title, describe(t))
		if err != nil || (res.Outcome != effects.OutcomeApplied && res.Outcome != effects.OutcomeAlreadyConfirmed && res.Outcome != effects.OutcomeReconciled) {
			rep.Failed = append(rep.Failed, key)
			continue
		}
		id, err := pub.RemoteIDFor(ctx, projectID, bk+":subtask")
		if err != nil || id == "" {
			rep.Failed = append(rep.Failed, key)
			continue
		}
		rep.Subtasks[key] = id
		if res.Outcome == effects.OutcomeAlreadyConfirmed {
			rep.AlreadyThere++
		} else {
			rep.Created++
		}
		if _, err := s.DB.Exec(ctx, `UPDATE plan_tasks SET remote_id = $4, updated_at = $5 WHERE project_id = $1 AND feature_id = $2 AND task_key = $3`,
			projectID, featureID, key, id, s.now()); err != nil {
			return rep, persistence.MapError(err)
		}
	}
	for _, key := range Order(cur.Plan) {
		t := by[key]
		for _, d := range sortedCopy(t.DependsOn) {
			item, dep := rep.Subtasks[key], rep.Subtasks[d]
			if item == "" || dep == "" {
				continue // a subtarefa ainda não existe: a próxima publicação completa
			}
			res, err := pub.AddDependency(ctx, projectID, BusinessKey(featureID, remoteID, key), item, dep)
			if err != nil || (res.Outcome != effects.OutcomeApplied && res.Outcome != effects.OutcomeAlreadyConfirmed && res.Outcome != effects.OutcomeReconciled) {
				rep.Failed = append(rep.Failed, key+">"+d)
				continue
			}
			rep.Dependencies++
		}
	}
	return rep, nil
}

func sortedCopy(in []string) []string {
	out := append([]string{}, in...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func describe(t Task) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(t.Description))
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	fmt.Fprintf(&b, "Risco inicial: %s.\nCritérios de aceite:\n", t.Risk)
	for _, c := range t.Criteria {
		fmt.Fprintf(&b, "- %s\n", strings.TrimSpace(c.Text))
	}
	return strings.TrimSpace(b.String())
}
