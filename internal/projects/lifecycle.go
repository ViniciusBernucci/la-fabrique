package projects

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// DB é o que o ciclo de vida precisa do banco (consultas e transações).
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Tipos de alteração registrados na trilha do projeto.
const (
	ChangeCreated       = "created"
	ChangeUpdated       = "updated"
	ChangeSourceChanged = "source_changed"
	ChangeActivated     = "activated"
	ChangeSuspended     = "suspended"
	ChangeResumed       = "resumed"
)

// Change é um registro da trilha. O detalhe guarda só nomes de campos e
// referências de conexão, nunca valores de segredo.
type Change struct {
	ID        int64          `json:"id"`
	Action    string         `json:"action"`
	Detail    map[string]any `json:"detail"`
	CreatedAt time.Time      `json:"created_at"`
}

func record(ctx context.Context, tx persistence.DBTX, projectID, adminID, action string, detail map[string]any) error {
	raw, _ := json.Marshal(detail)
	var admin any
	if adminID != "" {
		admin = adminID
	}
	_, err := tx.Exec(ctx, `INSERT INTO project_changes (project_id, admin_id, action, detail) VALUES ($1, $2, $3, $4::jsonb)`,
		projectID, admin, action, string(raw))
	return persistence.MapError(err)
}

// CreateDraft cria o projeto em rascunho, vazio de vínculos, e registra a criação.
func CreateDraft(ctx context.Context, db DB, adminID string, in NewProject) (Project, error) {
	var out Project
	err := persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		p, err := Create(ctx, tx, in)
		if err != nil {
			return err
		}
		out = p
		return record(ctx, tx, p.ID, adminID, ChangeCreated, map[string]any{"slug": p.Slug})
	})
	return out, err
}

// Configure aplica uma etapa do assistente (ou ajuste de dados) e registra a
// alteração. Mudança de origem sobre uma origem já definida vira `source_changed`.
func Configure(ctx context.Context, db DB, adminID, id string, expectedVersion int, patch Patch) (Project, error) {
	var out Project
	err := persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		before, err := getForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		updated, err := Update(ctx, tx, id, expectedVersion, patch)
		if err != nil {
			return err
		}
		out = updated
		var fields []string
		add := func(name string, changed bool) {
			if changed {
				fields = append(fields, name)
			}
		}
		add("name", updated.Name != before.Name)
		add("description", updated.Description != before.Description)
		add("priority", updated.Priority != before.Priority)
		add("repository", !sameRepo(before.Repository, updated.Repository))
		sourceChanged := !sameSource(before.Source, updated.Source)
		add("source", sourceChanged)
		if len(fields) > 0 {
			if err := record(ctx, tx, id, adminID, ChangeUpdated, map[string]any{"fields": fields}); err != nil {
				return err
			}
		}
		if sourceChanged && !before.Source.empty() {
			return record(ctx, tx, id, adminID, ChangeSourceChanged, map[string]any{
				"from": sourceDetail(before.Source), "to": sourceDetail(updated.Source),
			})
		}
		return nil
	})
	return out, err
}

func sameStr(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func sameRepo(a, b Repository) bool {
	return a.Owner == b.Owner && a.Name == b.Name && a.BaseBranch == b.BaseBranch && sameStr(a.ConnectionID, b.ConnectionID)
}

func sameSource(a, b Source) bool {
	return a.Kind == b.Kind && a.Container == b.Container && sameStr(a.ConnectionID, b.ConnectionID)
}

func sourceDetail(s Source) map[string]any {
	d := map[string]any{"kind": s.Kind, "container": s.Container}
	if s.ConnectionID != nil {
		d["connection_id"] = *s.ConnectionID
	}
	return d
}

// transition valida a versão e o estado de origem, revalida quando o destino é
// "active" e registra a mudança, tudo sob bloqueio da linha do projeto.
func transition(ctx context.Context, db DB, adminID, id string, expectedVersion int, from []string, to, action string) (Project, error) {
	var out Project
	err := persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		cur, err := getForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		if cur.Version != expectedVersion {
			return persistence.ErrVersionConflict
		}
		allowed := false
		for _, f := range from {
			allowed = allowed || cur.State == f
		}
		if !allowed {
			return fmt.Errorf("%w: estado atual %q", ErrInvalidState, cur.State)
		}
		if to == StateActive {
			report, err := Validate(ctx, tx, cur)
			if err != nil {
				return err
			}
			if !report.OK {
				return &ValidationError{Report: report}
			}
		}
		out, err = Update(ctx, tx, id, expectedVersion, Patch{State: &to})
		if err != nil {
			return err
		}
		return record(ctx, tx, id, adminID, action, map[string]any{"from": cur.State, "to": to})
	})
	return out, err
}

// Activate ativa um rascunho se o preenchimento obrigatório estiver válido.
func Activate(ctx context.Context, db DB, adminID, id string, expectedVersion int) (Project, error) {
	return transition(ctx, db, adminID, id, expectedVersion, []string{StateDraft}, StateActive, ChangeActivated)
}

// Suspend suspende um projeto ativo.
func Suspend(ctx context.Context, db DB, adminID, id string, expectedVersion int) (Project, error) {
	return transition(ctx, db, adminID, id, expectedVersion, []string{StateActive}, StatePaused, ChangeSuspended)
}

// Resume reativa um projeto suspenso, revalidando a configuração.
func Resume(ctx context.Context, db DB, adminID, id string, expectedVersion int) (Project, error) {
	return transition(ctx, db, adminID, id, expectedVersion, []string{StatePaused}, StateActive, ChangeResumed)
}

// Changes devolve a trilha do projeto, da mais recente para a mais antiga.
func Changes(ctx context.Context, db persistence.DBTX, projectID string, limit int) ([]Change, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := db.Query(ctx, `SELECT id, action, detail, created_at FROM project_changes
		WHERE project_id = $1 ORDER BY id DESC LIMIT $2`, projectID, limit)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	items := []Change{}
	for rows.Next() {
		var c Change
		var raw []byte
		if err := rows.Scan(&c.ID, &c.Action, &raw, &c.CreatedAt); err != nil {
			return nil, persistence.MapError(err)
		}
		_ = json.Unmarshal(raw, &c.Detail)
		items = append(items, c)
	}
	return items, persistence.MapError(rows.Err())
}
