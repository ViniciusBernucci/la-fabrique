package retention

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Padrões.
const (
	DefaultLogDays   = 30
	DefaultGraceDays = 14
)

// FieldError é um erro de entrada.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// ErrIntegrity: depois da limpeza, um checkpoint que deveria continuar não passou na conferência.
var ErrIntegrity = errors.New("a retenção quebrou a integridade de um checkpoint mantido")

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Policy define os prazos.
type Policy struct {
	LogDays   int
	GraceDays int
}

func (p Policy) withDefaults() (Policy, error) {
	if p.LogDays == 0 {
		p.LogDays = DefaultLogDays
	}
	if p.GraceDays == 0 {
		p.GraceDays = DefaultGraceDays
	}
	if p.LogDays < 1 || p.LogDays > 3650 || p.GraceDays < 1 || p.GraceDays > 3650 {
		return p, &FieldError{"policy", "os prazos vão de 1 a 3650 dias"}
	}
	return p, nil
}

// Retention executa a limpeza.
type Retention struct {
	DB          DB
	Checkpoints *checkpoints.Store
	Now         func() time.Time
}

func (r *Retention) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}

// Report é o resultado de uma execução da retenção.
type Report struct {
	RunID                string     `json:"run_id"`
	DryRun               bool       `json:"dry_run"`
	Now                  time.Time  `json:"now"`
	LogsDeleted          int        `json:"logs_deleted"`
	ExecutionsSummarized int        `json:"executions_summarized"`
	CheckpointsDeleted   int        `json:"checkpoints_deleted"`
	CheckpointsKept      int        `json:"checkpoints_kept"`
	ObjectsDeleted       int        `json:"objects_deleted"`
	Decisions            []Decision `json:"decisions"`
}

// Decision é a decisão sobre um item.
type Decision struct {
	Kind   string `json:"kind"`
	Ref    string `json:"ref"`
	Action string `json:"action"` // deleted | would_delete | kept
	Reason string `json:"reason"`
}

// incidentReasons devolve, por execução, por que ela está ligada a um incidente (vazio = livre).
func (r *Retention) incidentReasons(ctx context.Context) (map[string]string, error) {
	out := map[string]string{}
	add := func(id, why string) {
		if id == "" {
			return
		}
		if prev, ok := out[id]; ok {
			out[id] = prev + "; " + why
		} else {
			out[id] = why
		}
	}
	queries := []struct{ sql, why string }{
		{`SELECT DISTINCT execution_id::text FROM decision_requests WHERE state = 'pending'`, "decisão humana pendente"},
		{`SELECT id::text FROM executions WHERE state IN ('waiting', 'uncertain', 'queued', 'running')`, "execução ainda não concluída"},
		{`SELECT DISTINCT substring(business_key from 6 for 36) FROM outbox WHERE state IN ('intent', 'sending', 'uncertain') AND business_key LIKE 'exec:%'`, "efeito externo sem confirmação"},
		{`SELECT DISTINCT execution_id::text FROM queue_items WHERE execution_id IS NOT NULL AND state IN ('running', 'waiting')`, "tarefa da fila em andamento, em espera ou suspensa"},
	}
	for _, q := range queries {
		rows, err := r.DB.Query(ctx, q.sql)
		if err != nil {
			return nil, persistence.MapError(err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, persistence.MapError(err)
			}
			add(id, q.why)
		}
		rows.Close()
	}
	return out, nil
}

// Run aplica a política. Em dry-run nada é removido (as decisões saem como "would_delete").
func (r *Retention) Run(ctx context.Context, pol Policy, dryRun bool) (Report, error) {
	pol, err := pol.withDefaults()
	if err != nil {
		return Report{}, err
	}
	now := r.now()
	rep := Report{DryRun: dryRun, Now: now, Decisions: []Decision{}}
	logCut := now.AddDate(0, 0, -pol.LogDays)
	graceCut := now.AddDate(0, 0, -pol.GraceDays)
	incidents, err := r.incidentReasons(ctx)
	if err != nil {
		return rep, err
	}
	deleteWord := "deleted"
	if dryRun {
		deleteWord = "would_delete"
	}
	note := func(kind, ref, action, reason string) {
		rep.Decisions = append(rep.Decisions, Decision{kind, ref, action, reason})
	}

	// 1. Logs detalhados com mais de LogDays: resumo permanente primeiro, depois a remoção.
	rows, err := r.DB.Query(ctx, `SELECT execution_id::text, project_id::text, count(*), min(seq), max(seq), min(created_at), max(created_at)
		FROM execution_events WHERE created_at < $1 GROUP BY execution_id, project_id ORDER BY execution_id`, logCut)
	if err != nil {
		return rep, persistence.MapError(err)
	}
	type group struct {
		exec, project  string
		n              int
		minSeq, maxSeq int64
		first, last    time.Time
	}
	var groups []group
	for rows.Next() {
		var g group
		if err := rows.Scan(&g.exec, &g.project, &g.n, &g.minSeq, &g.maxSeq, &g.first, &g.last); err != nil {
			rows.Close()
			return rep, persistence.MapError(err)
		}
		groups = append(groups, g)
	}
	rows.Close()
	var cpDelete = map[string][]string{} // projeto → ids
	var cpDecisions []Decision
	// 2. Checkpoints de execuções concluídas, depois da carência.
	crows, err := r.DB.Query(ctx, `SELECT c.id::text, c.project_id::text, c.execution_id::text, c.seq, e.state, e.updated_at,
		EXISTS (SELECT 1 FROM checkpoint_protections p WHERE p.checkpoint_id = c.id) AS blocked,
		EXISTS (SELECT 1 FROM agent_sessions s WHERE s.checkpoint_id = c.id) AS in_session
		FROM checkpoints c JOIN executions e ON e.id = c.execution_id AND e.project_id = c.project_id
		ORDER BY c.execution_id, c.seq`)
	if err != nil {
		return rep, persistence.MapError(err)
	}
	type cp struct {
		id, project, exec, state string
		seq                      int
		updated                  time.Time
		blocked, inSession       bool
	}
	var cps []cp
	for crows.Next() {
		var c cp
		if err := crows.Scan(&c.id, &c.project, &c.exec, &c.seq, &c.state, &c.updated, &c.blocked, &c.inSession); err != nil {
			crows.Close()
			return rep, persistence.MapError(err)
		}
		cps = append(cps, c)
	}
	crows.Close()
	keptByExec := map[string]string{}
	for _, c := range cps {
		switch {
		case c.blocked:
			cpDecisions = append(cpDecisions, Decision{"checkpoint", c.id, "kept", "protegido: bloqueado explicitamente"})
			keptByExec[c.exec] = c.project
		case c.inSession:
			cpDecisions = append(cpDecisions, Decision{"checkpoint", c.id, "kept", "usado por uma sessão do agente"})
			keptByExec[c.exec] = c.project
		case incidents[c.exec] != "":
			cpDecisions = append(cpDecisions, Decision{"checkpoint", c.id, "kept", "ligado a incidente: " + incidents[c.exec]})
			keptByExec[c.exec] = c.project
		case c.state != "succeeded" && c.state != "cancelled":
			cpDecisions = append(cpDecisions, Decision{"checkpoint", c.id, "kept", "a execução não está concluída (" + c.state + ")"})
			keptByExec[c.exec] = c.project
		case c.updated.After(graceCut):
			cpDecisions = append(cpDecisions, Decision{"checkpoint", c.id, "kept", fmt.Sprintf("dentro da carência de %d dias", pol.GraceDays)})
			keptByExec[c.exec] = c.project
		default:
			cpDecisions = append(cpDecisions, Decision{"checkpoint", c.id, deleteWord, "execução concluída há mais de " + fmt.Sprint(pol.GraceDays) + " dias, sem proteção nem incidente"})
			cpDelete[c.project] = append(cpDelete[c.project], c.id)
		}
	}

	// Execução (auditoria + mudanças), numa transação por grupo de logs e uma para os checkpoints.
	for _, g := range groups {
		if why := incidents[g.exec]; why != "" {
			note("logs", g.exec, "kept", "ligado a incidente: "+why)
			continue
		}
		if dryRun {
			note("logs", g.exec, deleteWord, fmt.Sprintf("%d evento(s) com mais de %d dias; o resumo permanente será gravado antes", g.n, pol.LogDays))
			rep.LogsDeleted += g.n
			rep.ExecutionsSummarized++
			continue
		}
		deleted, err := r.compact(ctx, g.exec, g.project, logCut)
		if err != nil {
			return rep, err
		}
		if deleted > 0 {
			note("logs", g.exec, deleteWord, fmt.Sprintf("%d evento(s) com mais de %d dias removidos; resumo permanente gravado", deleted, pol.LogDays))
			rep.LogsDeleted += deleted
			rep.ExecutionsSummarized++
		}
	}
	for _, d := range cpDecisions {
		note(d.Kind, d.Ref, d.Action, d.Reason)
		switch d.Action {
		case "kept":
			rep.CheckpointsKept++
		default:
			rep.CheckpointsDeleted++
		}
	}
	if !dryRun {
		for project, ids := range cpDelete {
			n, objs, err := r.Checkpoints.Delete(ctx, project, ids)
			if err != nil {
				return rep, err
			}
			if n != len(ids) {
				return rep, fmt.Errorf("%w: esperava remover %d checkpoints e removeu %d", ErrIntegrity, len(ids), n)
			}
			rep.ObjectsDeleted += objs
		}
		// Conferência: o que ficou continua íntegro e restaurável.
		for exec, project := range keptByExec {
			if err := r.verify(ctx, project, exec); err != nil {
				return rep, err
			}
		}
	}
	// Auditoria.
	var audit string
	err = r.DB.QueryRow(ctx, `INSERT INTO retention_runs (dry_run, now_ref, log_days, grace_days, logs_deleted, executions_summarized, checkpoints_deleted, checkpoints_kept, objects_deleted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`, dryRun, now, pol.LogDays, pol.GraceDays, rep.LogsDeleted, rep.ExecutionsSummarized,
		rep.CheckpointsDeleted, rep.CheckpointsKept, rep.ObjectsDeleted).Scan(&audit)
	if err != nil {
		return rep, persistence.MapError(err)
	}
	rep.RunID = audit
	for _, d := range rep.Decisions {
		if _, err := r.DB.Exec(ctx, `INSERT INTO retention_decisions (run_id, kind, ref, action, reason) VALUES ($1, $2, $3, $4, $5)`, audit, d.Kind, d.Ref, d.Action, d.Reason); err != nil {
			return rep, persistence.MapError(err)
		}
	}
	return rep, nil
}

// compact grava o resumo permanente da execução e remove os eventos antigos, na mesma transação
// (que também trava a execução, para não competir com AppendEvent).
func (r *Retention) compact(ctx context.Context, execID, projectID string, cut time.Time) (int, error) {
	deleted := 0
	err := persistence.WithTx(ctx, r.DB, func(tx pgx.Tx) error {
		var locked string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM executions WHERE id = $1 AND project_id = $2 FOR UPDATE`, execID, projectID).Scan(&locked); err != nil {
			return persistence.MapError(err)
		}
		var n int
		var first, last *time.Time
		if err := tx.QueryRow(ctx, `SELECT count(*), min(created_at), max(created_at) FROM execution_events WHERE execution_id = $1 AND created_at < $2`, execID, cut).Scan(&n, &first, &last); err != nil {
			return persistence.MapError(err)
		}
		if n == 0 {
			return nil
		}
		var maxSeq, maxDeleted int64
		if err := tx.QueryRow(ctx, `SELECT COALESCE(max(seq), 0) FROM execution_events WHERE execution_id = $1`, execID).Scan(&maxSeq); err != nil {
			return persistence.MapError(err)
		}
		if err := tx.QueryRow(ctx, `SELECT COALESCE(max(seq), 0) FROM execution_events WHERE execution_id = $1 AND created_at < $2`, execID, cut).Scan(&maxDeleted); err != nil {
			return persistence.MapError(err)
		}
		byType := map[string]int{}
		rows, err := tx.Query(ctx, `SELECT type, count(*) FROM execution_events WHERE execution_id = $1 AND created_at < $2 GROUP BY type`, execID, cut)
		if err != nil {
			return persistence.MapError(err)
		}
		for rows.Next() {
			var t string
			var c int
			if err := rows.Scan(&t, &c); err != nil {
				rows.Close()
				return persistence.MapError(err)
			}
			byType[t] = c
		}
		rows.Close()
		var prevTotal int
		var prevSeq int64
		var prevBy []byte
		var prevFirst *time.Time
		err = tx.QueryRow(ctx, `SELECT compacted_total, last_seq, compacted_by_type, first_at FROM execution_summaries WHERE execution_id = $1`, execID).Scan(&prevTotal, &prevSeq, &prevBy, &prevFirst)
		if err != nil && !errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
			return persistence.MapError(err)
		}
		if len(prevBy) > 0 {
			old := map[string]int{}
			_ = json.Unmarshal(prevBy, &old)
			for t, c := range old {
				byType[t] += c
			}
		}
		if prevFirst != nil && (first == nil || prevFirst.Before(*first)) {
			first = prevFirst
		}
		if prevSeq > maxSeq {
			maxSeq = prevSeq
		}
		bj, _ := json.Marshal(byType)
		if _, err := tx.Exec(ctx, `INSERT INTO execution_summaries (execution_id, project_id, last_seq, compacted_through, compacted_total, compacted_by_type, first_at, last_at, compacted_at)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9)
			ON CONFLICT (execution_id) DO UPDATE SET last_seq = $3, compacted_through = GREATEST(execution_summaries.compacted_through, $4), compacted_total = $5,
				compacted_by_type = $6::jsonb, first_at = $7, last_at = GREATEST(execution_summaries.last_at, $8), compacted_at = $9`,
			execID, projectID, maxSeq, maxDeleted, prevTotal+n, string(bj), first, last, r.now()); err != nil {
			return persistence.MapError(err)
		}
		tag, err := tx.Exec(ctx, `DELETE FROM execution_events WHERE execution_id = $1 AND created_at < $2`, execID, cut)
		if err != nil {
			return persistence.MapError(err)
		}
		deleted = int(tag.RowsAffected())
		return nil
	})
	return deleted, err
}

func (r *Retention) verify(ctx context.Context, projectID, execID string) error {
	rows, err := r.DB.Query(ctx, `SELECT id::text FROM checkpoints WHERE project_id = $1 AND execution_id = $2`, projectID, execID)
	if err != nil {
		return persistence.MapError(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return persistence.MapError(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		cp, err := r.Checkpoints.Get(ctx, projectID, id)
		if err != nil {
			return fmt.Errorf("%w: %s: %v", ErrIntegrity, id, err)
		}
		for _, f := range cp.Manifest.Files {
			if f.Kind != checkpoints.KindFile {
				continue
			}
			if _, err := r.Checkpoints.Object(ctx, projectID, f.SHA256); err != nil {
				return fmt.Errorf("%w: %s: %v", ErrIntegrity, id, err)
			}
		}
	}
	return nil
}

// ---- proteção explícita ------------------------------------------------------------

// Protect bloqueia um checkpoint contra a retenção (ex.: ligado a um incidente).
func (r *Retention) Protect(ctx context.Context, projectID, checkpointID, reason, incidentRef, adminID string) error {
	if strings.TrimSpace(reason) == "" || len(reason) > 500 || len(incidentRef) > 200 {
		return &FieldError{"reason", "informe o motivo da proteção (até 500 caracteres)"}
	}
	var one int
	if err := r.DB.QueryRow(ctx, `SELECT 1 FROM checkpoints WHERE id::text = $1 AND project_id = $2`, checkpointID, projectID).Scan(&one); err != nil {
		return persistence.MapError(err)
	}
	var admin any
	if adminID != "" {
		admin = adminID
	}
	_, err := r.DB.Exec(ctx, `INSERT INTO checkpoint_protections (checkpoint_id, reason, incident_ref, created_by) VALUES ($1, $2, $3, $4)
		ON CONFLICT (checkpoint_id) DO UPDATE SET reason = $2, incident_ref = $3, created_by = $4`, checkpointID, strings.TrimSpace(reason), incidentRef, admin)
	return persistence.MapError(err)
}

// Unprotect libera o checkpoint para a retenção.
func (r *Retention) Unprotect(ctx context.Context, projectID, checkpointID string) error {
	_, err := r.DB.Exec(ctx, `DELETE FROM checkpoint_protections p USING checkpoints c WHERE p.checkpoint_id = c.id AND c.id::text = $1 AND c.project_id = $2`, checkpointID, projectID)
	return persistence.MapError(err)
}

// Summary é o resumo permanente de uma execução.
type Summary struct {
	ExecutionID  string         `json:"execution_id"`
	LastSeq      int64          `json:"last_seq"`
	Compacted    int            `json:"compacted_total"`
	ByType       map[string]int `json:"compacted_by_type"`
	CompactedTil int64          `json:"compacted_through"`
	FirstAt      *time.Time     `json:"first_at"`
	LastAt       *time.Time     `json:"last_at"`
}

// SummaryOf devolve o resumo permanente (ErrNotFound se a execução nunca teve logs removidos).
func (r *Retention) SummaryOf(ctx context.Context, projectID, execID string) (Summary, error) {
	var s Summary
	var by []byte
	err := r.DB.QueryRow(ctx, `SELECT execution_id::text, last_seq, compacted_total, compacted_by_type, compacted_through, first_at, last_at FROM execution_summaries
		WHERE project_id = $1 AND execution_id = $2`, projectID, execID).Scan(&s.ExecutionID, &s.LastSeq, &s.Compacted, &by, &s.CompactedTil, &s.FirstAt, &s.LastAt)
	if err != nil {
		return s, persistence.MapError(err)
	}
	_ = json.Unmarshal(by, &s.ByType)
	return s, nil
}
