package queue

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
)

// Tipos e estados.
const (
	KindTask  = "task"
	KindPRFix = "pr_fix"

	StateQueued    = "queued"
	StateRunning   = "running"
	StateWaiting   = "waiting"
	StateDone      = "done"
	StateCancelled = "cancelled"
)

// Motivos de espera.
const (
	WaitMerge     = "merge"
	WaitProvider  = "provider"
	WaitDecision  = "decision"
	WaitResources = "resources"
)

var waitReasons = map[string]bool{WaitMerge: true, WaitProvider: true, WaitDecision: true, WaitResources: true}

// Erros.
var (
	ErrNoGate   = errors.New("não há porta configurada para este motivo de espera: o retorno é recusado")
	ErrNotReady = errors.New("a condição de espera ainda não foi atendida")
	ErrBadState = errors.New("o item não está num estado que permita esta operação")
	ErrNotSafe  = errors.New("checkpoint inválido ou de outra execução")
)

// NotEligibleError lista os motivos de um item não poder iniciar.
type NotEligibleError struct{ Reasons []string }

func (e *NotEligibleError) Error() string {
	return "o item não está elegível: " + strings.Join(e.Reasons, "; ")
}

// FieldError é um erro de entrada.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Gate diz se a condição de uma espera foi atendida (merge confirmado, provedor
// de volta, decisão resolvida, recursos livres).
type Gate interface {
	Ready(ctx context.Context, it Item) (ok bool, reason string, err error)
}

// Queue é a fila.
type Queue struct {
	DB          DB
	Checkpoints *checkpoints.Store
	Gates       map[string]Gate // por motivo de espera
	Now         func() time.Time
}

func (q *Queue) now() time.Time {
	if q.Now != nil {
		return q.Now().UTC()
	}
	return time.Now().UTC()
}

// Item é um item da fila.
type Item struct {
	ID           string     `json:"id"`
	Seq          int64      `json:"seq"`
	ProjectID    string     `json:"project_id"`
	FeatureID    string     `json:"feature_id"`
	TaskKey      string     `json:"task_key"`
	Kind         string     `json:"kind"`
	Ref          string     `json:"ref"`
	Priority     int        `json:"priority"`
	State        string     `json:"state"`
	WaitReason   string     `json:"wait_reason,omitempty"`
	WaitRef      string     `json:"wait_ref,omitempty"`
	ExecutionID  string     `json:"execution_id,omitempty"`
	CheckpointID string     `json:"checkpoint_id,omitempty"`
	SlotHeld     bool       `json:"slot_held"`
	EnqueuedAt   time.Time  `json:"enqueued_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	LeaseOwner   string     `json:"lease_owner,omitempty"`
	LeaseExpires *time.Time `json:"lease_expires_at,omitempty"`
	Generation   int64      `json:"generation"`
}

const cols = `id, seq, project_id, feature_id, task_key, kind, ref, priority, state, COALESCE(wait_reason, ''), wait_ref,
	COALESCE(execution_id::text, ''), COALESCE(checkpoint_id::text, ''), slot_held, enqueued_at, started_at, finished_at, COALESCE(lease_owner, ''), lease_expires_at, generation`

func scanItem(row interface{ Scan(...any) error }) (Item, error) {
	var it Item
	err := row.Scan(&it.ID, &it.Seq, &it.ProjectID, &it.FeatureID, &it.TaskKey, &it.Kind, &it.Ref, &it.Priority, &it.State, &it.WaitReason, &it.WaitRef,
		&it.ExecutionID, &it.CheckpointID, &it.SlotHeld, &it.EnqueuedAt, &it.StartedAt, &it.FinishedAt, &it.LeaseOwner, &it.LeaseExpires, &it.Generation)
	return it, persistence.MapError(err)
}

func lockProject(ctx context.Context, tx pgx.Tx, projectID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 14))`, projectID)
	return persistence.MapError(err)
}

// Enqueue enfileira uma tarefa do plano. Repetir devolve o item ativo existente.
func (q *Queue) Enqueue(ctx context.Context, projectID, featureID, taskKey string, priority int) (Item, error) {
	return q.enqueue(ctx, projectID, featureID, taskKey, KindTask, "", priority)
}

// EnqueuePRFix enfileira a correção de uma PR (vai à frente na próxima vaga elegível).
func (q *Queue) EnqueuePRFix(ctx context.Context, projectID, featureID, taskKey, prRef string, priority int) (Item, error) {
	if strings.TrimSpace(prRef) == "" || len(prRef) > 300 {
		return Item{}, &FieldError{"ref", "informe a PR a corrigir"}
	}
	return q.enqueue(ctx, projectID, featureID, taskKey, KindPRFix, strings.TrimSpace(prRef), priority)
}

func (q *Queue) enqueue(ctx context.Context, projectID, featureID, taskKey, kind, ref string, priority int) (Item, error) {
	if priority < 0 || priority > 4 {
		return Item{}, &FieldError{"priority", "a prioridade vai de 0 (mais urgente) a 4"}
	}
	var out Item
	err := persistence.WithTx(ctx, q.DB, func(tx pgx.Tx) error {
		var one int
		if err := tx.QueryRow(ctx, `SELECT 1 FROM plan_tasks WHERE project_id = $1 AND feature_id = $2 AND task_key = $3 AND NOT removed`, projectID, featureID, taskKey).Scan(&one); err != nil {
			if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
				return &FieldError{"task_key", fmt.Sprintf("a tarefa %q não existe no plano vigente da feature", taskKey)}
			}
			return persistence.MapError(err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO queue_items (project_id, feature_id, task_key, kind, ref, priority, enqueued_at) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (feature_id, task_key, kind, ref) WHERE state IN ('queued', 'running', 'waiting') DO NOTHING`,
			projectID, featureID, taskKey, kind, ref, priority, q.now()); err != nil {
			return persistence.MapError(err)
		}
		it, err := scanItem(tx.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE feature_id = $1 AND task_key = $2 AND kind = $3 AND ref = $4 AND state IN ('queued', 'running', 'waiting')`,
			featureID, taskKey, kind, ref))
		out = it
		return err
	})
	return out, err
}

// EnqueuePlan enfileira todas as tarefas pendentes do plano vigente (idempotente).
func (q *Queue) EnqueuePlan(ctx context.Context, projectID, featureID string, priority int) ([]Item, error) {
	tasks, err := planning.TasksOf(ctx, q.DB, projectID, featureID)
	if err != nil {
		return nil, err
	}
	out := []Item{}
	for _, t := range tasks {
		if t.Removed || t.State != planning.StatePending {
			continue
		}
		it, err := q.Enqueue(ctx, projectID, featureID, t.Key, priority)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

// orderer define a ordem da fila: correção de PR; prioridade EFETIVA (a do item,
// reduzida em 1 a cada período de espera, mínimo 0: envelhecimento determinístico);
// prioridade do projeto; chegada.
type orderer struct {
	now      time.Time
	aging    time.Duration
	projPrio map[string]int
}

func (o orderer) effective(it Item) int {
	e := it.Priority
	if o.aging > 0 {
		e -= int(o.now.Sub(it.EnqueuedAt) / o.aging)
	}
	if e < 0 {
		e = 0
	}
	return e
}

func (o orderer) less(a, b Item) bool {
	if (a.Kind == KindPRFix) != (b.Kind == KindPRFix) {
		return a.Kind == KindPRFix
	}
	if ea, eb := o.effective(a), o.effective(b); ea != eb {
		return ea < eb
	}
	if pa, pb := o.projPrio[a.ProjectID], o.projPrio[b.ProjectID]; pa != pb {
		return pa < pb
	}
	if !a.EnqueuedAt.Equal(b.EnqueuedAt) {
		return a.EnqueuedAt.Before(b.EnqueuedAt)
	}
	return a.Seq < b.Seq
}

// orderer carrega o envelhecimento (configuração persistente) e as prioridades dos projetos.
func (q *Queue) orderer(ctx context.Context, db persistence.DBTX, items []Item) (orderer, error) {
	st, err := q.settings(ctx, db)
	if err != nil {
		return orderer{}, err
	}
	o := orderer{now: q.now(), aging: time.Duration(st.AgingMinutes) * time.Minute, projPrio: map[string]int{}}
	ids := map[string]bool{}
	for _, it := range items {
		ids[it.ProjectID] = true
	}
	for id := range ids {
		var p int
		if err := db.QueryRow(ctx, `SELECT priority FROM projects WHERE id = $1`, id).Scan(&p); err != nil {
			return orderer{}, persistence.MapError(err)
		}
		o.projPrio[id] = p
	}
	return o, nil
}

// reasons lista por que o item NÃO pode iniciar agora (vazio = elegível), sem considerar a ordem.
func (q *Queue) reasons(ctx context.Context, db persistence.DBTX, it Item) ([]string, error) {
	var out []string
	var state string
	if err := db.QueryRow(ctx, `SELECT state FROM projects WHERE id = $1`, it.ProjectID).Scan(&state); err != nil {
		return nil, persistence.MapError(err)
	}
	if state != projects.StateActive {
		out = append(out, fmt.Sprintf("o projeto não está ativo (%s)", state))
	}
	var holder string
	err := db.QueryRow(ctx, `SELECT task_key FROM queue_items WHERE project_id = $1 AND slot_held AND id <> $2`, it.ProjectID, it.ID).Scan(&holder)
	switch {
	case err == nil:
		out = append(out, fmt.Sprintf("a vaga do projeto está ocupada pela tarefa %q", holder))
	case !errors.Is(persistence.MapError(err), persistence.ErrNotFound):
		return nil, persistence.MapError(err)
	}
	var pending int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM requirement_revisions WHERE feature_id = $1 AND state IN ('needs_replan', 'conflict')`, it.FeatureID).Scan(&pending); err != nil {
		return nil, persistence.MapError(err)
	}
	if pending > 0 {
		out = append(out, "o requisito da feature mudou: replanejamento ou decisão pendente")
	}
	if it.Kind != KindTask {
		return out, nil
	}
	tasks, err := planning.TasksOf(ctx, db, it.ProjectID, it.FeatureID)
	if err != nil {
		return nil, err
	}
	state2 := map[string]string{}
	for _, t := range tasks {
		if !t.Removed {
			state2[t.Key] = t.State
		}
	}
	if s := state2[it.TaskKey]; s != "" && s != planning.StatePending && it.State == StateQueued {
		out = append(out, fmt.Sprintf("a tarefa já está %s", s))
	}
	cur, err := (&planning.Store{DB: nil}).CurrentOf(ctx, db, it.ProjectID, it.FeatureID)
	if err != nil {
		return nil, err
	}
	for _, t := range cur.Plan.Tasks {
		if t.Key != it.TaskKey {
			continue
		}
		deps := append([]string{}, t.DependsOn...)
		sort.Strings(deps)
		for _, d := range deps {
			if state2[d] != planning.StateDone {
				out = append(out, fmt.Sprintf("o pré-requisito %q ainda não foi concluído", d))
			}
		}
	}
	return out, nil
}

// Explain devolve os motivos de um item não iniciar agora (vazio = pode iniciar, respeitada a ordem).
func (q *Queue) Explain(ctx context.Context, itemID string) ([]string, error) {
	it, err := scanItem(q.DB.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1`, itemID))
	if err != nil {
		return nil, err
	}
	if it.State != StateQueued {
		return []string{fmt.Sprintf("o item já está %s", it.State)}, nil
	}
	rs, err := q.reasons(ctx, q.DB, it)
	if err != nil {
		return nil, err
	}
	if len(rs) == 0 {
		if best, err := q.best(ctx, q.DB, it.ProjectID); err == nil && best != nil && best.ID != it.ID {
			rs = append(rs, fmt.Sprintf("a vez é da tarefa %q (%s)", best.TaskKey, orderWhy(*best)))
		}
	}
	return rs, nil
}

func orderWhy(it Item) string {
	if it.Kind == KindPRFix {
		return "correção de PR"
	}
	return fmt.Sprintf("prioridade %d, chegada anterior", it.Priority)
}

// best devolve o primeiro item ELEGÍVEL do projeto na ordem da fila (nil se nenhum).
func (q *Queue) best(ctx context.Context, db persistence.DBTX, projectID string) (*Item, error) {
	rows, err := db.Query(ctx, `SELECT `+cols+` FROM queue_items WHERE project_id = $1 AND state = 'queued'`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	var items []Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, it)
	}
	rows.Close()
	ord, err := q.orderer(ctx, db, items)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return ord.less(items[i], items[j]) })
	for _, it := range items {
		rs, err := q.reasons(ctx, db, it)
		if err != nil {
			return nil, err
		}
		if len(rs) == 0 {
			c := it
			return &c, nil
		}
	}
	return nil, nil
}

// Candidates devolve no máximo UM item por projeto (o primeiro elegível), na ordem
// global: correções de PR, prioridade e chegada. Projetos diferentes podem iniciar em paralelo.
func (q *Queue) Candidates(ctx context.Context) ([]Item, error) { return q.candidatesIn(ctx, q.DB) }

func (q *Queue) candidatesIn(ctx context.Context, db persistence.DBTX) ([]Item, error) {
	rows, err := db.Query(ctx, `SELECT DISTINCT project_id FROM queue_items WHERE state = 'queued'`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, persistence.MapError(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	out := []Item{}
	for _, id := range ids {
		b, err := q.best(ctx, db, id)
		if err != nil {
			return nil, err
		}
		if b != nil {
			out = append(out, *b)
		}
	}
	ord, err := q.orderer(ctx, db, out)
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return ord.less(out[i], out[j]) })
	return out, nil
}

// Start inicia o item SE ele for elegível agora e for o primeiro da ordem do projeto,
// reconferido sob o bloqueio do projeto. Ocupa a vaga e marca a tarefa como em execução.
// Não aplica o limite global nem lease: quem reclama vagas em produção usa Claim.
func (q *Queue) Start(ctx context.Context, itemID, executionID string) (Item, error) {
	var out Item
	err := persistence.WithTx(ctx, q.DB, func(tx pgx.Tx) error {
		var err error
		out, err = q.startTx(ctx, tx, itemID, executionID, "", 0)
		return err
	})
	return out, err
}

// startTx inicia o item dentro da transação do chamador. Com `owner` informado,
// grava a lease e sobe a geração.
func (q *Queue) startTx(ctx context.Context, tx pgx.Tx, itemID, executionID, owner string, lease time.Duration) (Item, error) {
	first, err := scanItem(tx.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1`, itemID))
	if err != nil {
		return Item{}, err
	}
	if err := lockProject(ctx, tx, first.ProjectID); err != nil {
		return Item{}, err
	}
	it, err := scanItem(tx.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1 FOR UPDATE`, itemID))
	if err != nil {
		return Item{}, err
	}
	if it.State != StateQueued {
		return Item{}, fmt.Errorf("%w: estado %s", ErrBadState, it.State)
	}
	rs, err := q.reasons(ctx, tx, it)
	if err != nil {
		return Item{}, err
	}
	if len(rs) == 0 {
		if best, err := q.best(ctx, tx, it.ProjectID); err != nil {
			return Item{}, err
		} else if best != nil && best.ID != it.ID {
			rs = append(rs, fmt.Sprintf("a vez é da tarefa %q (%s)", best.TaskKey, orderWhy(*best)))
		}
	}
	if len(rs) > 0 {
		return Item{}, &NotEligibleError{Reasons: rs}
	}
	var exec any
	if executionID != "" {
		exec = executionID
	}
	now := q.now()
	var ownerArg, expArg any
	gen := it.Generation
	if owner != "" {
		ownerArg, expArg = owner, now.Add(lease)
		gen++
	}
	it, err = scanItem(tx.QueryRow(ctx, `UPDATE queue_items SET state = 'running', slot_held = true, execution_id = $2, started_at = $3, version = version + 1,
		lease_owner = $4, lease_expires_at = $5, heartbeat_at = $3, generation = $6
		WHERE id = $1 RETURNING `+cols, itemID, exec, now, ownerArg, expArg, gen))
	if err != nil {
		return Item{}, err
	}
	if it.Kind == KindTask {
		if _, err := tx.Exec(ctx, `UPDATE plan_tasks SET state = 'running', updated_at = $4 WHERE project_id = $1 AND feature_id = $2 AND task_key = $3 AND state = 'pending'`,
			it.ProjectID, it.FeatureID, it.TaskKey, now); err != nil {
			return Item{}, persistence.MapError(err)
		}
	}
	return it, nil
}

// safeCheckpoint confere que o checkpoint é íntegro e da execução do item.
func (q *Queue) safeCheckpoint(ctx context.Context, it Item, checkpointID string, quiescent bool) (bool, error) {
	if checkpointID == "" || !quiescent {
		return false, nil
	}
	if q.Checkpoints == nil {
		return false, fmt.Errorf("%w: sem armazenamento de checkpoints", ErrNotSafe)
	}
	cp, err := q.Checkpoints.Get(ctx, it.ProjectID, checkpointID)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrNotSafe, err)
	}
	if it.ExecutionID == "" || cp.ExecutionID != it.ExecutionID {
		return false, fmt.Errorf("%w: o checkpoint não é da execução do item", ErrNotSafe)
	}
	return true, nil
}

// Block coloca o item em espera por um motivo (merge, provedor, decisão, recursos). A
// vaga só é devolvida com um checkpoint seguro (íntegro, da execução, sandbox quieto);
// sem ele o item espera segurando a vaga.
func (q *Queue) Block(ctx context.Context, itemID, reason, ref, checkpointID string, quiescent bool) (Item, error) {
	if !waitReasons[reason] {
		return Item{}, &FieldError{"reason", "use merge, provider, decision ou resources"}
	}
	var out Item
	err := persistence.WithTx(ctx, q.DB, func(tx pgx.Tx) error {
		first, err := scanItem(tx.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1`, itemID))
		if err != nil {
			return err
		}
		if err := lockProject(ctx, tx, first.ProjectID); err != nil {
			return err
		}
		it, err := scanItem(tx.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1 FOR UPDATE`, itemID))
		if err != nil {
			return err
		}
		if it.State != StateRunning {
			return fmt.Errorf("%w: só um item em execução pode esperar (estado %s)", ErrBadState, it.State)
		}
		if err := q.checkFence(ctx, it); err != nil {
			return err
		}
		safe, err := q.safeCheckpoint(ctx, it, checkpointID, quiescent)
		if err != nil {
			return err
		}
		var cp any
		if safe {
			cp = checkpointID
		}
		it, err = scanItem(tx.QueryRow(ctx, `UPDATE queue_items SET state = 'waiting', wait_reason = $2, wait_ref = $3, checkpoint_id = $4, slot_held = $5, lease_owner = NULL, lease_expires_at = NULL, version = version + 1
			WHERE id = $1 RETURNING `+cols, itemID, reason, ref, cp, !safe))
		if err != nil {
			return err
		}
		if it.Kind == KindTask {
			_, err = tx.Exec(ctx, `UPDATE plan_tasks SET state = 'blocked', updated_at = $4 WHERE project_id = $1 AND feature_id = $2 AND task_key = $3 AND state = 'running'`,
				it.ProjectID, it.FeatureID, it.TaskKey, q.now())
		}
		out = it
		return persistence.MapError(err)
	})
	return out, err
}

// ReleaseSlot devolve a vaga de um item que esperava segurando-a, depois de um checkpoint seguro.
func (q *Queue) ReleaseSlot(ctx context.Context, itemID, checkpointID string, quiescent bool) (Item, error) {
	var out Item
	err := persistence.WithTx(ctx, q.DB, func(tx pgx.Tx) error {
		first, err := scanItem(tx.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1`, itemID))
		if err != nil {
			return err
		}
		if err := lockProject(ctx, tx, first.ProjectID); err != nil {
			return err
		}
		it, err := scanItem(tx.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1 FOR UPDATE`, itemID))
		if err != nil {
			return err
		}
		if it.State != StateWaiting || !it.SlotHeld {
			return fmt.Errorf("%w: o item não está esperando com a vaga", ErrBadState)
		}
		safe, err := q.safeCheckpoint(ctx, it, checkpointID, quiescent)
		if err != nil {
			return err
		}
		if !safe {
			return fmt.Errorf("%w: informe um checkpoint com o sandbox quieto", ErrNotSafe)
		}
		out, err = scanItem(tx.QueryRow(ctx, `UPDATE queue_items SET slot_held = false, checkpoint_id = $2, version = version + 1 WHERE id = $1 RETURNING `+cols, itemID, checkpointID))
		return err
	})
	return out, err
}

// Resume devolve o item à fila quando a porta do seu motivo de espera está liberada.
// Sem porta configurada para o motivo, recusa (falha fechada). O item volta a precisar
// de elegibilidade para iniciar.
func (q *Queue) Resume(ctx context.Context, itemID string) (Item, error) {
	it, err := scanItem(q.DB.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1`, itemID))
	if err != nil {
		return Item{}, err
	}
	if it.State != StateWaiting {
		return Item{}, fmt.Errorf("%w: o item não está esperando", ErrBadState)
	}
	gate := q.Gates[it.WaitReason]
	if gate == nil {
		return Item{}, fmt.Errorf("%w: %s", ErrNoGate, it.WaitReason)
	}
	ok, why, err := gate.Ready(ctx, it)
	if err != nil {
		return Item{}, err
	}
	if !ok {
		return Item{}, fmt.Errorf("%w: %s", ErrNotReady, why)
	}
	var out Item
	err = persistence.WithTx(ctx, q.DB, func(tx pgx.Tx) error {
		if err := lockProject(ctx, tx, it.ProjectID); err != nil {
			return err
		}
		out, err = scanItem(tx.QueryRow(ctx, `UPDATE queue_items SET state = 'queued', wait_reason = NULL, wait_ref = '', slot_held = false, version = version + 1
			WHERE id = $1 AND state = 'waiting' RETURNING `+cols, itemID))
		if err != nil {
			return err
		}
		if out.Kind == KindTask {
			_, err = tx.Exec(ctx, `UPDATE plan_tasks SET state = 'pending', updated_at = $4 WHERE project_id = $1 AND feature_id = $2 AND task_key = $3 AND state = 'blocked'`,
				out.ProjectID, out.FeatureID, out.TaskKey, q.now())
		}
		return persistence.MapError(err)
	})
	return out, err
}

// Complete conclui o item em execução, devolve a vaga e marca a tarefa como concluída.
func (q *Queue) Complete(ctx context.Context, itemID string) (Item, error) {
	return q.finish(ctx, itemID, StateDone)
}

// Cancel cancela um item que ainda não terminou e devolve a vaga.
func (q *Queue) Cancel(ctx context.Context, itemID string) (Item, error) {
	return q.finish(ctx, itemID, StateCancelled)
}

func (q *Queue) finish(ctx context.Context, itemID, to string) (Item, error) {
	var out Item
	err := persistence.WithTx(ctx, q.DB, func(tx pgx.Tx) error {
		first, err := scanItem(tx.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1`, itemID))
		if err != nil {
			return err
		}
		if err := lockProject(ctx, tx, first.ProjectID); err != nil {
			return err
		}
		it, err := scanItem(tx.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1 FOR UPDATE`, itemID))
		if err != nil {
			return err
		}
		if to == StateDone && it.State != StateRunning {
			return fmt.Errorf("%w: só um item em execução conclui (estado %s)", ErrBadState, it.State)
		}
		if to == StateDone {
			if err := q.checkFence(ctx, it); err != nil {
				return err
			}
		}
		if it.State == StateDone || it.State == StateCancelled {
			return fmt.Errorf("%w: o item já terminou", ErrBadState)
		}
		out, err = scanItem(tx.QueryRow(ctx, `UPDATE queue_items SET state = $2, slot_held = false, wait_reason = NULL, lease_owner = NULL, lease_expires_at = NULL, finished_at = $3, version = version + 1 WHERE id = $1 RETURNING `+cols, itemID, to, q.now()))
		if err != nil {
			return err
		}
		if out.Kind == KindTask && to == StateDone {
			_, err = tx.Exec(ctx, `UPDATE plan_tasks SET state = 'done', updated_at = $4 WHERE project_id = $1 AND feature_id = $2 AND task_key = $3`,
				out.ProjectID, out.FeatureID, out.TaskKey, q.now())
		}
		return persistence.MapError(err)
	})
	return out, err
}

// List devolve os itens do projeto (todos os estados), na ordem de chegada.
func (q *Queue) List(ctx context.Context, projectID string) ([]Item, error) {
	rows, err := q.DB.Query(ctx, `SELECT `+cols+` FROM queue_items WHERE project_id = $1 ORDER BY seq`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, persistence.MapError(rows.Err())
}
