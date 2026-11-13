package checkpoints

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// DB é o que o Store precisa do PostgreSQL (*pgxpool.Pool implementa).
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Store grava e lê checkpoints.
type Store struct {
	DB DB
	// Hook é uma emenda de teste para injetar falha no meio da gravação
	// (estágios "object" e "checkpoint"). Em produção fica nil.
	Hook func(stage string, i int) error
	// Guard (opcional) roda DENTRO da transação de Save, antes de gravar. Serve para
	// recusar um escritor obsoleto (lease vencida, vaga dada a outro): a checagem e a
	// gravação são atômicas.
	Guard func(ctx context.Context, tx pgx.Tx, in Input) error
}

// Input é o que o motor entrega junto ao conteúdo do workspace.
type Input struct {
	ProjectID   string
	ExecutionID string
	Label       string
	State       json.RawMessage   // estado serializado dos nós (RunState)
	Refs        map[string]string // referências externas: ramo, PR, IDs remotos…
}

// Checkpoint é um checkpoint gravado.
type Checkpoint struct {
	ID           string            `json:"id"`
	ProjectID    string            `json:"project_id"`
	ExecutionID  string            `json:"execution_id"`
	Seq          int               `json:"seq"`
	Label        string            `json:"label"`
	Manifest     Manifest          `json:"manifest"`
	ManifestHash string            `json:"manifest_hash"`
	TreeHash     string            `json:"tree_hash"`
	State        json.RawMessage   `json:"state"`
	StateHash    string            `json:"state_hash"`
	Refs         map[string]string `json:"refs"`
	TotalBytes   int64             `json:"total_bytes"`
	CreatedAt    time.Time         `json:"created_at"`
}

// Save grava o checkpoint e todos os seus objetos numa ÚNICA transação: ou o novo
// existe por inteiro, ou nada muda e o anterior segue válido.
func (s *Store) Save(ctx context.Context, snap Snapshot, in Input) (Checkpoint, error) {
	if err := snap.Manifest.Validate(); err != nil {
		return Checkpoint{}, err
	}
	for _, e := range snap.Manifest.Files {
		if e.Kind == KindFile {
			if d, ok := snap.Objects[e.SHA256]; !ok || sum(d) != e.SHA256 {
				return Checkpoint{}, fmt.Errorf("%w: objeto de %q", ErrCorrupt, e.Path)
			}
		}
	}
	state := in.State
	if len(state) == 0 {
		state = json.RawMessage(`{}`)
	}
	if !json.Valid(state) {
		return Checkpoint{}, errors.New("estado dos nós inválido (JSON)")
	}
	refs := in.Refs
	if refs == nil {
		refs = map[string]string{}
	}
	manifest, _ := json.Marshal(snap.Manifest)
	refsJSON, _ := json.Marshal(refs)
	var cp Checkpoint
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 12))`, in.ExecutionID); err != nil {
			return persistence.MapError(err)
		}
		if s.Guard != nil {
			if err := s.Guard(ctx, tx, in); err != nil {
				return err
			}
		}
		i := 0
		for hash, data := range snap.Objects {
			if s.Hook != nil {
				if err := s.Hook("object", i); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(ctx, `INSERT INTO checkpoint_objects (project_id, hash, size, data) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
				in.ProjectID, hash, len(data), data); err != nil {
				return persistence.MapError(err)
			}
			i++
		}
		if s.Hook != nil {
			if err := s.Hook("checkpoint", 0); err != nil {
				return err
			}
		}
		var seq int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(max(seq), 0) + 1 FROM checkpoints WHERE execution_id = $1`, in.ExecutionID).Scan(&seq); err != nil {
			return persistence.MapError(err)
		}
		err := tx.QueryRow(ctx, `
			INSERT INTO checkpoints (project_id, execution_id, seq, label, manifest, manifest_hash, tree_hash, state, state_hash, refs, total_bytes)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8::jsonb, $9, $10::jsonb, $11)
			RETURNING id, created_at`,
			in.ProjectID, in.ExecutionID, seq, in.Label, string(manifest), snap.Manifest.Hash(), snap.Manifest.TreeHash,
			string(state), stateHash(state), string(refsJSON), snap.Manifest.TotalBytes).Scan(&cp.ID, &cp.CreatedAt)
		if err != nil {
			return persistence.MapError(err)
		}
		cp.Seq = seq
		return nil
	})
	if err != nil {
		return Checkpoint{}, err
	}
	return s.Get(ctx, in.ProjectID, cp.ID) // devolve o que foi gravado, já reconferido
}

const cpColumns = `id, project_id, execution_id, seq, label, manifest, manifest_hash, tree_hash, state, state_hash, refs, total_bytes, created_at`

func scanCP(row interface{ Scan(...any) error }) (Checkpoint, error) {
	var c Checkpoint
	var manifest, state, refs []byte
	if err := row.Scan(&c.ID, &c.ProjectID, &c.ExecutionID, &c.Seq, &c.Label, &manifest, &c.ManifestHash, &c.TreeHash, &state, &c.StateHash, &refs, &c.TotalBytes, &c.CreatedAt); err != nil {
		return c, persistence.MapError(err)
	}
	if err := json.Unmarshal(manifest, &c.Manifest); err != nil {
		return c, fmt.Errorf("%w: manifesto ilegível", ErrCorrupt)
	}
	c.State = json.RawMessage(state)
	_ = json.Unmarshal(refs, &c.Refs)
	// Toda leitura reconfere o que foi gravado.
	if c.Manifest.Hash() != c.ManifestHash || c.Manifest.TreeHash != c.TreeHash || stateHash(c.State) != c.StateHash {
		return c, fmt.Errorf("%w: manifesto ou estado", ErrCorrupt)
	}
	if err := c.Manifest.Validate(); err != nil {
		return c, err
	}
	return c, nil
}

// Get lê um checkpoint do projeto, reconferindo os hashes.
func (s *Store) Get(ctx context.Context, projectID, id string) (Checkpoint, error) {
	return scanCP(s.DB.QueryRow(ctx, `SELECT `+cpColumns+` FROM checkpoints WHERE id = $1 AND project_id = $2`, id, projectID))
}

// Latest devolve o checkpoint mais recente da execução.
func (s *Store) Latest(ctx context.Context, projectID, executionID string) (Checkpoint, error) {
	return scanCP(s.DB.QueryRow(ctx, `SELECT `+cpColumns+` FROM checkpoints WHERE execution_id = $1 AND project_id = $2 ORDER BY seq DESC LIMIT 1`, executionID, projectID))
}

// Object lê um objeto e reconfere o hash.
func (s *Store) Object(ctx context.Context, projectID, hash string) ([]byte, error) {
	var data []byte
	err := s.DB.QueryRow(ctx, `SELECT data FROM checkpoint_objects WHERE project_id = $1 AND hash = $2`, projectID, hash).Scan(&data)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return nil, fmt.Errorf("%w: objeto %s ausente", ErrCorrupt, hash[:12])
	}
	if err != nil {
		return nil, persistence.MapError(err)
	}
	if sum(data) != hash {
		return nil, fmt.Errorf("%w: objeto %s", ErrCorrupt, hash[:12])
	}
	return data, nil
}

// Prune mantém só os `keep` checkpoints mais recentes da execução e remove os
// objetos que nenhum checkpoint do projeto referencia mais.
func (s *Store) Prune(ctx context.Context, projectID, executionID string, keep int) (removed int, err error) {
	if keep < 1 {
		return 0, errors.New("keep deve ser pelo menos 1")
	}
	err = persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 12))`, executionID); err != nil {
			return persistence.MapError(err)
		}
		tag, err := tx.Exec(ctx, `DELETE FROM checkpoints WHERE project_id = $1 AND execution_id = $2 AND seq <= (
			SELECT COALESCE(max(seq), 0) - $3 FROM checkpoints WHERE project_id = $1 AND execution_id = $2)`, projectID, executionID, keep)
		if err != nil {
			return persistence.MapError(err)
		}
		removed = int(tag.RowsAffected())
		rows, err := tx.Query(ctx, `SELECT manifest FROM checkpoints WHERE project_id = $1`, projectID)
		if err != nil {
			return persistence.MapError(err)
		}
		live := map[string]bool{}
		for rows.Next() {
			var raw []byte
			if err := rows.Scan(&raw); err != nil {
				rows.Close()
				return persistence.MapError(err)
			}
			var m Manifest
			if json.Unmarshal(raw, &m) != nil {
				rows.Close()
				return fmt.Errorf("%w: manifesto ilegível", ErrCorrupt)
			}
			for _, f := range m.Files {
				live[f.SHA256] = true
			}
			if m.Git.PatchHash != "" {
				live[m.Git.PatchHash] = true
			}
		}
		rows.Close()
		hashes := make([]string, 0, len(live))
		for h := range live {
			hashes = append(hashes, h)
		}
		_, err = tx.Exec(ctx, `DELETE FROM checkpoint_objects WHERE project_id = $1 AND NOT (hash = ANY($2))`, projectID, hashes)
		return persistence.MapError(err)
	})
	return removed, err
}
