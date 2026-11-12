package persistence

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"
)

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Artifact são os metadados de um artefato durável. O conteúdo vive no volume
// de artefatos, referenciado por StorageRef (caminho relativo).
type Artifact struct {
	ID            string
	ProjectID     string
	ExecutionID   string // vazio quando não pertence a uma execução
	Kind          string
	SHA256        string
	SizeBytes     int64
	StorageRef    string
	SchemaVersion int
}

// ValidateStorageRef recusa caminhos absolutos, traversal e referências vazias.
func ValidateStorageRef(ref string) error {
	if ref == "" || strings.HasPrefix(ref, "/") || strings.Contains(ref, "\x00") || strings.Contains(ref, `\`) {
		return fmt.Errorf("%w: storage_ref inválida", ErrInvalidInput)
	}
	if clean := path.Clean(ref); clean != ref || clean == "." || strings.HasPrefix(clean, "..") {
		return fmt.Errorf("%w: storage_ref deve ser relativa e normalizada", ErrInvalidInput)
	}
	return nil
}

// RegisterArtifact grava os metadados após o arquivo ter sido gravado de forma
// íntegra. É idempotente por (projeto, tipo, sha256): repetir devolve o existente.
func RegisterArtifact(ctx context.Context, db DBTX, a Artifact) (Artifact, bool, error) {
	if !sha256Pattern.MatchString(a.SHA256) {
		return Artifact{}, false, fmt.Errorf("%w: sha256 deve ter 64 hexadecimais minúsculos", ErrInvalidInput)
	}
	if err := ValidateStorageRef(a.StorageRef); err != nil {
		return Artifact{}, false, err
	}
	if a.SchemaVersion == 0 {
		a.SchemaVersion = 1
	}
	var execID any
	if a.ExecutionID != "" {
		execID = a.ExecutionID
	}
	const returning = ` RETURNING id, project_id, COALESCE(execution_id::text, ''), kind, sha256, size_bytes, storage_ref, schema_version`
	row := db.QueryRow(ctx, `
		INSERT INTO artifacts (project_id, execution_id, kind, sha256, size_bytes, storage_ref, schema_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (project_id, kind, sha256) DO NOTHING`+returning,
		a.ProjectID, execID, a.Kind, a.SHA256, a.SizeBytes, a.StorageRef, a.SchemaVersion)
	var out Artifact
	err := row.Scan(&out.ID, &out.ProjectID, &out.ExecutionID, &out.Kind, &out.SHA256, &out.SizeBytes, &out.StorageRef, &out.SchemaVersion)
	if err == nil {
		return out, true, nil
	}
	if mapped := MapError(err); mapped != ErrNotFound {
		return Artifact{}, false, mapped
	}
	err = db.QueryRow(ctx, `
		SELECT id, project_id, COALESCE(execution_id::text, ''), kind, sha256, size_bytes, storage_ref, schema_version
		FROM artifacts WHERE project_id = $1 AND kind = $2 AND sha256 = $3`,
		a.ProjectID, a.Kind, a.SHA256).
		Scan(&out.ID, &out.ProjectID, &out.ExecutionID, &out.Kind, &out.SHA256, &out.SizeBytes, &out.StorageRef, &out.SchemaVersion)
	return out, false, MapError(err)
}
