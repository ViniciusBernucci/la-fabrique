package queue

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Modos de controle.
const (
	ModeActive    = "active"
	ModeDraining  = "draining"  // pausar após a feature atual
	ModeSuspended = "suspended" // suspender agora, com checkpoint

	WaitSuspended = "suspended"
)

// Scope é o alcance de um controle: global ou um projeto.
type Scope struct {
	Global    bool
	ProjectID string
}

func (s Scope) valid() error {
	if s.Global == (s.ProjectID != "") {
		return &FieldError{"scope", "informe global OU um projeto"}
	}
	return nil
}

func (s Scope) name() string {
	if s.Global {
		return "global"
	}
	return "project"
}

// Control é o estado de um controle.
type Control struct {
	Scope     string `json:"scope"`
	ProjectID string `json:"project_id,omitempty"`
	Mode      string `json:"mode"`
	Reason    string `json:"reason"`
	Version   int    `json:"version"`
}

// Mode devolve o modo do escopo (active se nunca foi alterado).
func (q *Queue) Mode(ctx context.Context, s Scope) (string, error) {
	return q.modeOf(ctx, q.DB, s)
}

func (q *Queue) modeOf(ctx context.Context, db persistence.DBTX, s Scope) (string, error) {
	if err := s.valid(); err != nil {
		return "", err
	}
	var mode string
	var err error
	if s.Global {
		err = db.QueryRow(ctx, `SELECT mode FROM controls WHERE scope = 'global'`).Scan(&mode)
	} else {
		err = db.QueryRow(ctx, `SELECT mode FROM controls WHERE scope = 'project' AND project_id = $1`, s.ProjectID).Scan(&mode)
	}
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return ModeActive, nil
	}
	return mode, persistence.MapError(err)
}

// Controls lista os controles que não estão ativos (os que alteram o comportamento).
func (q *Queue) Controls(ctx context.Context) ([]Control, error) {
	rows, err := q.DB.Query(ctx, `SELECT scope, COALESCE(project_id::text, ''), mode, reason, version FROM controls WHERE mode <> 'active' ORDER BY scope DESC, project_id`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Control{}
	for rows.Next() {
		var c Control
		if err := rows.Scan(&c.Scope, &c.ProjectID, &c.Mode, &c.Reason, &c.Version); err != nil {
			return nil, persistence.MapError(err)
		}
		out = append(out, c)
	}
	return out, persistence.MapError(rows.Err())
}

// SetMode persiste o controle e audita a mudança. Não altera o que já está em
// execução: parar o que roda é SuspendNow; "draining" só impede começar feature nova.
func (q *Queue) SetMode(ctx context.Context, s Scope, mode, adminID, why string) (Control, error) {
	if err := s.valid(); err != nil {
		return Control{}, err
	}
	if mode != ModeActive && mode != ModeDraining && mode != ModeSuspended {
		return Control{}, &FieldError{"mode", "use active, draining ou suspended"}
	}
	if len(why) > 500 {
		return Control{}, &FieldError{"reason", "justificativa longa demais"}
	}
	var out Control
	err := persistence.WithTx(ctx, q.DB, func(tx pgx.Tx) error {
		return q.setModeTx(ctx, tx, s, mode, adminID, why, &out)
	})
	return out, err
}

func (q *Queue) setModeTx(ctx context.Context, tx pgx.Tx, s Scope, mode, adminID, why string, out *Control) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('controls', 16))`); err != nil {
		return persistence.MapError(err)
	}
	from, err := q.modeOf(ctx, tx, s)
	if err != nil {
		return err
	}
	var admin, proj any
	if adminID != "" {
		admin = adminID
	}
	if !s.Global {
		proj = s.ProjectID
	}
	now := q.now()
	var c Control
	if s.Global {
		err = tx.QueryRow(ctx, `INSERT INTO controls (scope, project_id, mode, reason, changed_by, updated_at) VALUES ('global', NULL, $1, $2, $3, $4)
			ON CONFLICT ((1)) WHERE scope = 'global' DO UPDATE SET mode = $1, reason = $2, changed_by = $3, updated_at = $4, version = controls.version + 1
			RETURNING scope, COALESCE(project_id::text, ''), mode, reason, version`, mode, why, admin, now).Scan(&c.Scope, &c.ProjectID, &c.Mode, &c.Reason, &c.Version)
	} else {
		err = tx.QueryRow(ctx, `INSERT INTO controls (scope, project_id, mode, reason, changed_by, updated_at) VALUES ('project', $5, $1, $2, $3, $4)
			ON CONFLICT (project_id) WHERE scope = 'project' DO UPDATE SET mode = $1, reason = $2, changed_by = $3, updated_at = $4, version = controls.version + 1
			RETURNING scope, COALESCE(project_id::text, ''), mode, reason, version`, mode, why, admin, now, proj).Scan(&c.Scope, &c.ProjectID, &c.Mode, &c.Reason, &c.Version)
	}
	if err != nil {
		return persistence.MapError(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO control_events (scope, project_id, from_mode, to_mode, admin_id, detail, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.name(), proj, from, mode, admin, why, now); err != nil {
		return persistence.MapError(err)
	}
	*out = c
	return nil
}

// controlReasons devolve os motivos de controle que impedem o item de iniciar: suspensão
// (global ou do projeto) e, em "pausa após a feature atual", começar uma feature nova.
func (q *Queue) controlReasons(ctx context.Context, db persistence.DBTX, it Item) ([]string, error) {
	g, err := q.modeOf(ctx, db, Scope{Global: true})
	if err != nil {
		return nil, err
	}
	p, err := q.modeOf(ctx, db, Scope{ProjectID: it.ProjectID})
	if err != nil {
		return nil, err
	}
	var out []string
	if g == ModeSuspended {
		out = append(out, "suspensão global ativa")
	}
	if p == ModeSuspended {
		out = append(out, "o projeto está suspenso")
	}
	if g == ModeDraining || p == ModeDraining {
		var started int
		if err := db.QueryRow(ctx, `SELECT count(*) FROM plan_tasks WHERE project_id = $1 AND feature_id = $2 AND NOT removed AND state <> 'pending'`, it.ProjectID, it.FeatureID).Scan(&started); err != nil {
			return nil, persistence.MapError(err)
		}
		if started == 0 && it.Kind == KindTask {
			where := "do projeto"
			if g == ModeDraining {
				where = "global"
			}
			out = append(out, fmt.Sprintf("pausa após a feature atual (%s): nenhuma feature nova inicia", where))
		}
	}
	return out, nil
}

// Suspender para o que está rodando e devolve o checkpoint seguro. Em produção: cancela o
// processo, espera o sandbox ficar quieto e captura (checkpoints.Store.CaptureAndRelease).
type Suspender interface {
	Suspend(ctx context.Context, it Item) (checkpointID string, quiescent bool, err error)
}

// SuspendFailure é um item que não pôde ser suspendido com segurança (segue em execução).
type SuspendFailure struct {
	Item   Item   `json:"item"`
	Reason string `json:"reason"`
}

// SuspendResult separa o que foi suspendido do que não pôde ser.
type SuspendResult struct {
	Control   Control          `json:"control"`
	Suspended []Item           `json:"suspended"`
	Failed    []SuspendFailure `json:"failed"`
}

// SuspendNow persiste a suspensão (nada novo inicia) e para o que roda no escopo: cada item
// é suspendido SÓ com checkpoint seguro (íntegro, da execução, sandbox quieto); a vaga é
// devolvida e o dono antigo perde o direito de gravar (a geração sobe e o item sai de
// "running"). Sem checkpoint seguro o item segue em execução e é reportado.
func (q *Queue) SuspendNow(ctx context.Context, s Scope, adminID, why string, sp Suspender) (SuspendResult, error) {
	res := SuspendResult{Suspended: []Item{}, Failed: []SuspendFailure{}}
	ctl, err := q.SetMode(ctx, s, ModeSuspended, adminID, why)
	if err != nil {
		return res, err
	}
	res.Control = ctl
	query := `SELECT ` + cols + ` FROM queue_items WHERE state = 'running'`
	args := []any{}
	if !s.Global {
		query += ` AND project_id = $1`
		args = append(args, s.ProjectID)
	}
	rows, err := q.DB.Query(ctx, query+` ORDER BY seq`, args...)
	if err != nil {
		return res, persistence.MapError(err)
	}
	var running []Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			rows.Close()
			return res, err
		}
		running = append(running, it)
	}
	rows.Close()
	for _, it := range running {
		cp, quiet, err := sp.Suspend(ctx, it)
		if err != nil {
			res.Failed = append(res.Failed, SuspendFailure{it, "falha ao parar: " + err.Error()})
			continue
		}
		safe, err := q.safeCheckpoint(ctx, it, cp, quiet)
		if err != nil || !safe {
			why := "sem checkpoint seguro"
			if err != nil {
				why = err.Error()
			}
			res.Failed = append(res.Failed, SuspendFailure{it, why})
			continue
		}
		done, err := q.suspendItem(ctx, it.ID, cp)
		if err != nil {
			res.Failed = append(res.Failed, SuspendFailure{it, err.Error()})
			continue
		}
		res.Suspended = append(res.Suspended, done)
	}
	return res, nil
}

func (q *Queue) suspendItem(ctx context.Context, itemID, checkpointID string) (Item, error) {
	var out Item
	err := persistence.WithTx(ctx, q.DB, func(tx pgx.Tx) error {
		first, err := scanItem(tx.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1`, itemID))
		if err != nil {
			return err
		}
		if err := lockProject(ctx, tx, first.ProjectID); err != nil {
			return err
		}
		out, err = scanItem(tx.QueryRow(ctx, `UPDATE queue_items SET state = 'waiting', wait_reason = 'suspended', wait_ref = '', checkpoint_id = $2, slot_held = false,
			lease_owner = NULL, lease_expires_at = NULL, generation = generation + 1, version = version + 1
			WHERE id = $1 AND state = 'running' RETURNING `+cols, itemID, checkpointID))
		if errors.Is(err, persistence.ErrNotFound) {
			return fmt.Errorf("%w: o item deixou de estar em execução", ErrBadState)
		}
		if err != nil {
			return err
		}
		if out.Kind == KindTask {
			_, err = tx.Exec(ctx, `UPDATE plan_tasks SET state = 'blocked', updated_at = $4 WHERE project_id = $1 AND feature_id = $2 AND task_key = $3 AND state = 'running'`,
				out.ProjectID, out.FeatureID, out.TaskKey, q.now())
		}
		return persistence.MapError(err)
	})
	return out, err
}

// ResumeScope reativa o escopo e devolve à fila os itens suspensos que não continuam sob
// outra suspensão (a global não reativa o que o projeto ainda mantém suspenso, e vice-versa).
func (q *Queue) ResumeScope(ctx context.Context, s Scope, adminID, why string) (Control, []Item, error) {
	if err := s.valid(); err != nil {
		return Control{}, nil, err
	}
	var ctl Control
	if err := persistence.WithTx(ctx, q.DB, func(tx pgx.Tx) error {
		return q.setModeTx(ctx, tx, s, ModeActive, adminID, why, &ctl)
	}); err != nil {
		return Control{}, nil, err
	}
	rows, err := q.DB.Query(ctx, `SELECT `+cols+` FROM queue_items WHERE state = 'waiting' AND wait_reason = 'suspended' ORDER BY seq`)
	if err != nil {
		return ctl, nil, persistence.MapError(err)
	}
	var cand []Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			rows.Close()
			return ctl, nil, err
		}
		if (s.Global || it.ProjectID == s.ProjectID) && it.State == StateWaiting {
			cand = append(cand, it)
		}
	}
	rows.Close()
	out := []Item{}
	for _, it := range cand {
		g, err := q.modeOf(ctx, q.DB, Scope{Global: true})
		if err != nil {
			return ctl, out, err
		}
		p, err := q.modeOf(ctx, q.DB, Scope{ProjectID: it.ProjectID})
		if err != nil {
			return ctl, out, err
		}
		if g == ModeSuspended || p == ModeSuspended {
			continue
		}
		back, err := q.requeue(ctx, it.ID)
		if err != nil {
			return ctl, out, err
		}
		out = append(out, back)
	}
	return ctl, out, nil
}

func (q *Queue) requeue(ctx context.Context, itemID string) (Item, error) {
	var out Item
	err := persistence.WithTx(ctx, q.DB, func(tx pgx.Tx) error {
		first, err := scanItem(tx.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1`, itemID))
		if err != nil {
			return err
		}
		if err := lockProject(ctx, tx, first.ProjectID); err != nil {
			return err
		}
		out, err = scanItem(tx.QueryRow(ctx, `UPDATE queue_items SET state = 'queued', wait_reason = NULL, wait_ref = '', slot_held = false, version = version + 1
			WHERE id = $1 AND state = 'waiting' AND wait_reason = 'suspended' RETURNING `+cols, itemID))
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

// SuspendOne suspende UM item em execução com um checkpoint seguro já feito (íntegro, da execução do
// item, sandbox quieto): devolve a vaga, perde a lease e sobe a geração. É o que o esgotamento
// da revisão usa; ResumeScope (ou o escopo do projeto) o devolve à fila depois da decisão.
func (q *Queue) SuspendOne(ctx context.Context, itemID, checkpointID string, quiescent bool) (Item, error) {
	it, err := scanItem(q.DB.QueryRow(ctx, `SELECT `+cols+` FROM queue_items WHERE id = $1`, itemID))
	if err != nil {
		return Item{}, err
	}
	if it.State != StateRunning {
		return Item{}, fmt.Errorf("%w: só um item em execução é suspenso (estado %s)", ErrBadState, it.State)
	}
	safe, err := q.safeCheckpoint(ctx, it, checkpointID, quiescent)
	if err != nil {
		return Item{}, err
	}
	if !safe {
		return Item{}, fmt.Errorf("%w: sem checkpoint seguro", ErrNotSafe)
	}
	return q.suspendItem(ctx, itemID, checkpointID)
}
