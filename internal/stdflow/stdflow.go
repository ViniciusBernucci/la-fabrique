// Package stdflow provisiona o workflow estrutural de cada projeto (AJ05-01). O cadastro cria um
// rascunho com gate do PRD, planejamento, implementação e review por trilha, consolidação, checks,
// Draft PR, espera pelo merge e o caminho de correção. Repetir o provisionamento nunca cria cópia
// nem sobrescreve o que o administrador editou.
package stdflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Provision garante o workflow do projeto. created=false quando já existia (intacto).
func Provision(ctx context.Context, db DB, projectID, projectName string) (wf workflows.Workflow, created bool, err error) {
	err = persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "project-workflow|"+projectID); err != nil {
			return err
		}
		var id string
		err := tx.QueryRow(ctx, `SELECT workflow_id::text FROM project_workflows WHERE project_id=$1`, projectID).Scan(&id)
		switch {
		case err == nil:
			wf, err = workflows.Get(ctx, tx, id)
			return err
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		w, err := workflows.Create(ctx, tx, "Fluxo padrão — "+projectName, workflows.KindFeature, "Fluxo estrutural do projeto: PRD, planejamento, trilhas, review, Draft PR, merge e correções.")
		if err != nil {
			return err
		}
		w, err = workflows.Save(ctx, tx, w.ID, w.Version, w.Name, w.Description, workflows.StandardGraph())
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO project_workflows (project_id, workflow_id) VALUES ($1,$2)`, projectID, w.ID); err != nil {
			return err
		}
		wf, created = w, true
		return nil
	})
	if err != nil {
		return workflows.Workflow{}, false, fmt.Errorf("provisionar o workflow do projeto: %w", persistence.MapError(err))
	}
	return wf, created, nil
}

// Of devolve o ID do workflow do projeto ("" se ainda não provisionado).
func Of(ctx context.Context, db persistence.DBTX, projectID string) (string, error) {
	var id string
	err := db.QueryRow(ctx, `SELECT workflow_id::text FROM project_workflows WHERE project_id=$1`, projectID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// Managed informa se o workflow pertence a um projeto (exige as guardas na publicação).
func Managed(ctx context.Context, db persistence.DBTX, workflowID string) (bool, error) {
	var n int
	err := db.QueryRow(ctx, `SELECT count(*) FROM project_workflows WHERE workflow_id=$1`, workflowID).Scan(&n)
	return n > 0, err
}
