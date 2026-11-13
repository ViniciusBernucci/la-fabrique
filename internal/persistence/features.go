package persistence

import (
	"context"
	"fmt"
)

// Feature é uma feature importada da origem externa do projeto.
type Feature struct {
	ID               string
	ProjectID        string
	ExternalID       string
	ExternalRevision string
	Title            string
	State            string
	Version          int
}

// CreateFeature cria a feature; (projeto, external_id) é único.
func CreateFeature(ctx context.Context, db DBTX, projectID, externalID, title string) (Feature, error) {
	var f Feature
	var rev *string
	err := db.QueryRow(ctx, `
		INSERT INTO features (project_id, external_id, title) VALUES ($1, $2, $3)
		RETURNING id, project_id, external_id, external_revision, title, state, version`,
		projectID, externalID, title).
		Scan(&f.ID, &f.ProjectID, &f.ExternalID, &rev, &f.Title, &f.State, &f.Version)
	if rev != nil {
		f.ExternalRevision = *rev
	}
	return f, MapError(err)
}

// Execution é uma tentativa de executar uma feature.
type Execution struct {
	ID         string
	ProjectID  string
	FeatureID  string
	State      string
	Attempt    int
	PreviousID string
	Version    int
}

// CreateExecution cria a próxima tentativa da feature, ligada à anterior. A FK
// composta recusa feature de outro projeto (ErrInvalidScope).
func CreateExecution(ctx context.Context, db DBTX, projectID, featureID string) (Execution, error) {
	var e Execution
	err := db.QueryRow(ctx, `
		WITH prev AS (
			SELECT id, attempt FROM executions WHERE feature_id = $2 ORDER BY attempt DESC LIMIT 1
		)
		INSERT INTO executions (project_id, feature_id, attempt, previous_execution_id)
		VALUES ($1, $2, COALESCE((SELECT attempt FROM prev), 0) + 1, (SELECT id FROM prev))
		RETURNING id, project_id, feature_id, state, attempt, COALESCE(previous_execution_id::text, ''), version`,
		projectID, featureID).
		Scan(&e.ID, &e.ProjectID, &e.FeatureID, &e.State, &e.Attempt, &e.PreviousID, &e.Version)
	return e, MapError(err)
}

// UpdateExecutionState altera o estado com controle de versão.
func UpdateExecutionState(ctx context.Context, db DBTX, projectID, id string, expectedVersion int, state string) (Execution, error) {
	var e Execution
	err := db.QueryRow(ctx, `
		UPDATE executions SET state = $4, version = version + 1, updated_at = now()
		WHERE project_id = $1 AND id = $2 AND version = $3
		RETURNING id, project_id, feature_id, state, attempt, COALESCE(previous_execution_id::text, ''), version`,
		projectID, id, expectedVersion, state).
		Scan(&e.ID, &e.ProjectID, &e.FeatureID, &e.State, &e.Attempt, &e.PreviousID, &e.Version)
	if mapped := MapError(err); mapped == ErrNotFound {
		var exists bool
		_ = db.QueryRow(ctx, `SELECT true FROM executions WHERE project_id = $1 AND id = $2`, projectID, id).Scan(&exists)
		if exists {
			return Execution{}, ErrVersionConflict
		}
		return Execution{}, fmt.Errorf("%w: execução", ErrNotFound)
	} else if mapped != nil {
		return Execution{}, mapped
	}
	return e, nil
}
