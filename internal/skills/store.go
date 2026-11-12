package skills

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Origens e resultados de importação.
const (
	SourceManual  = "manual"
	SourceFile    = "file"
	SourcePackage = "package"
	SourceGit     = "git"

	OutcomeImported  = "imported"
	OutcomeUnchanged = "unchanged"
	OutcomeReplaced  = "replaced"
	OutcomeRejected  = "rejected"
)

// DB é o que o catálogo precisa do banco.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Skill é uma skill do catálogo.
type Skill struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Executor      string    `json:"executor"`
	SourceKind    string    `json:"source_kind"`
	SourceRef     string    `json:"source_ref"`
	Revision      string    `json:"revision"`
	ContentHash   string    `json:"content_hash"`
	Body          string    `json:"body"`
	Manifest      Manifest  `json:"manifest"`
	NeedsApproval bool      `json:"needs_approval"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// FileInfo descreve um arquivo guardado (sem o conteúdo).
type FileInfo struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int    `json:"size"`
}

const columns = `id, name, description, executor, source_kind, source_ref, revision, content_hash, body, manifest, needs_approval, version, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (Skill, error) {
	var s Skill
	var raw []byte
	if err := row.Scan(&s.ID, &s.Name, &s.Description, &s.Executor, &s.SourceKind, &s.SourceRef, &s.Revision, &s.ContentHash,
		&s.Body, &raw, &s.NeedsApproval, &s.Version, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return s, persistence.MapError(err)
	}
	_ = json.Unmarshal(raw, &s.Manifest)
	return s, nil
}

func record(ctx context.Context, db persistence.DBTX, skillID *string, name, kind, ref, outcome, reason, hash string) error {
	_, err := db.Exec(ctx, `INSERT INTO skill_imports (skill_id, name, source_kind, source_ref, outcome, reason, content_hash)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)`, skillID, clipTo(name, 100), kind, clipTo(ref, 300), outcome, clipTo(reason, 500), hash)
	return persistence.MapError(err)
}

func clipTo(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// RecordRejected registra uma importação recusada (sem o conteúdo do pacote).
func RecordRejected(ctx context.Context, db persistence.DBTX, name, kind, ref, reason string) error {
	return record(ctx, db, nil, name, kind, ref, OutcomeRejected, reason, "")
}

// Save guarda a skill validada. Mesmo nome com mesmo conteúdo é idempotente
// (`unchanged`); conteúdo diferente exige replace=true e gera nova versão.
func Save(ctx context.Context, db DB, pkg Package, kind, ref, revision string, replace bool) (Skill, string, error) {
	var out Skill
	var outcome string
	manifest, _ := json.Marshal(pkg.Manifest)
	err := persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		cur, err := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM skills WHERE name = $1 FOR UPDATE`, pkg.Manifest.Name))
		switch {
		case err == nil && cur.ContentHash == pkg.Hash:
			out, outcome = cur, OutcomeUnchanged
		case err == nil && !replace:
			return persistence.ErrDuplicate
		case err == nil:
			if _, err := tx.Exec(ctx, `DELETE FROM skill_files WHERE skill_id = $1`, cur.ID); err != nil {
				return persistence.MapError(err)
			}
			out, err = scan(tx.QueryRow(ctx, `
				UPDATE skills SET description = $2, executor = $3, source_kind = $4, source_ref = $5, revision = $6, content_hash = $7,
					body = $8, manifest = $9::jsonb, needs_approval = $10, version = version + 1, updated_at = now()
				WHERE id = $1 RETURNING `+columns, cur.ID, pkg.Manifest.Description, pkg.Manifest.Executor, kind, clipTo(ref, 300), clipTo(revision, 80),
				pkg.Hash, pkg.Body, string(manifest), pkg.NeedsApproval))
			if err != nil {
				return err
			}
			outcome = OutcomeReplaced
		case errors.Is(err, persistence.ErrNotFound):
			out, err = scan(tx.QueryRow(ctx, `
				INSERT INTO skills (name, description, executor, source_kind, source_ref, revision, content_hash, body, manifest, needs_approval)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10) RETURNING `+columns,
				pkg.Manifest.Name, pkg.Manifest.Description, pkg.Manifest.Executor, kind, clipTo(ref, 300), clipTo(revision, 80),
				pkg.Hash, pkg.Body, string(manifest), pkg.NeedsApproval))
			if err != nil {
				return err
			}
			outcome = OutcomeImported
		default:
			return err
		}
		if outcome != OutcomeUnchanged {
			for _, f := range pkg.Files {
				sum := sha256.Sum256(f.Content)
				if _, err := tx.Exec(ctx, `INSERT INTO skill_files (skill_id, path, sha256, size, content) VALUES ($1, $2, $3, $4, $5)`,
					out.ID, f.Path, hex.EncodeToString(sum[:]), len(f.Content), f.Content); err != nil {
					return persistence.MapError(err)
				}
			}
		}
		id := out.ID
		return record(ctx, tx, &id, out.Name, kind, ref, outcome, "", pkg.Hash)
	})
	return out, outcome, err
}

// Get lê uma skill por ID.
func Get(ctx context.Context, db persistence.DBTX, id string) (Skill, error) {
	return scan(db.QueryRow(ctx, `SELECT `+columns+` FROM skills WHERE id = $1`, id))
}

// GetByName lê uma skill por nome.
func GetByName(ctx context.Context, db persistence.DBTX, name string) (Skill, error) {
	return scan(db.QueryRow(ctx, `SELECT `+columns+` FROM skills WHERE name = $1`, name))
}

// List devolve as skills por nome.
func List(ctx context.Context, db persistence.DBTX) ([]Skill, error) {
	rows, err := db.Query(ctx, `SELECT `+columns+` FROM skills ORDER BY name`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Skill{}
	for rows.Next() {
		s, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, persistence.MapError(rows.Err())
}

// Files lista os arquivos guardados da skill.
func Files(ctx context.Context, db persistence.DBTX, id string) ([]FileInfo, error) {
	rows, err := db.Query(ctx, `SELECT path, sha256, size FROM skill_files WHERE skill_id = $1 ORDER BY path`, id)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []FileInfo{}
	for rows.Next() {
		var f FileInfo
		if err := rows.Scan(&f.Path, &f.SHA256, &f.Size); err != nil {
			return nil, persistence.MapError(err)
		}
		out = append(out, f)
	}
	return out, persistence.MapError(rows.Err())
}

// Import é uma linha da trilha de importações.
type Import struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	SourceKind  string    `json:"source_kind"`
	SourceRef   string    `json:"source_ref"`
	Outcome     string    `json:"outcome"`
	Reason      string    `json:"reason"`
	ContentHash string    `json:"content_hash"`
	CreatedAt   time.Time `json:"created_at"`
}

// Imports devolve a trilha (mais recente primeiro).
func Imports(ctx context.Context, db persistence.DBTX, limit int) ([]Import, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := db.Query(ctx, `SELECT id, name, source_kind, source_ref, outcome, reason, content_hash, created_at FROM skill_imports ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Import{}
	for rows.Next() {
		var i Import
		if err := rows.Scan(&i.ID, &i.Name, &i.SourceKind, &i.SourceRef, &i.Outcome, &i.Reason, &i.ContentHash, &i.CreatedAt); err != nil {
			return nil, persistence.MapError(err)
		}
		out = append(out, i)
	}
	return out, persistence.MapError(rows.Err())
}
