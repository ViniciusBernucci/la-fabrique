package contract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// ErrTampered indica snapshot cujo conteúdo não confere com o hash.
var ErrTampered = errors.New("o contrato não confere com o hash registrado")

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Store publica e lê snapshots.
type Store struct{ DB DB }

// Snapshot é um contrato publicado.
type Snapshot struct {
	ID       string   `json:"id"`
	Scope    string   `json:"scope_key"`
	Version  int      `json:"version"`
	Hash     string   `json:"hash"`
	Contract Contract `json:"contract"`
}

// ScopeKey identifica a sequência de versões: a subtarefa, ou a Issue (contrato inicial).
func ScopeKey(d Deterministic) string {
	if d.SubtaskID != "" {
		return "subtask:" + d.SubtaskID
	}
	return "issue:" + d.IssueID + ":initial"
}

// Publish valida e publica. Conteúdo igual ao vigente não cria versão nova (unchanged=true);
// conteúdo diferente cria a próxima versão encadeada pelo hash da anterior; as antigas permanecem.
func (s *Store) Publish(ctx context.Context, d Deterministic, p Proposal, riskFloor string, refs References, g Graph) (snap Snapshot, unchanged bool, err error) {
	if ps := Validate(d, p, g); len(ps) > 0 {
		return Snapshot{}, false, &RejectedError{ps}
	}
	c := Build(d, p, riskFloor, refs)
	scope := ScopeKey(d)
	err = persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, d.ProjectID+"|"+scope); err != nil {
			return err
		}
		var prevVer int
		var prevHash string
		var prevBody []byte
		var prevID string
		qerr := tx.QueryRow(ctx, `SELECT id::text, version, hash, body FROM execution_contracts WHERE project_id=$1 AND scope_key=$2 ORDER BY version DESC LIMIT 1`,
			d.ProjectID, scope).Scan(&prevID, &prevVer, &prevHash, &prevBody)
		if qerr != nil && !errors.Is(qerr, pgx.ErrNoRows) {
			return qerr
		}
		if qerr == nil {
			var prev Contract
			if err := json.Unmarshal(prevBody, &prev); err != nil {
				return err
			}
			// Compara o conteúdo ignorando versão e encadeamento.
			cmp := c
			cmp.Version, cmp.ParentHash = prev.Version, prev.ParentHash
			if cmp.Hash() == prev.Hash() {
				snap, unchanged = Snapshot{ID: prevID, Scope: scope, Version: prevVer, Hash: prevHash, Contract: prev}, true
				return nil
			}
		}
		c.Version, c.ParentHash = prevVer+1, prevHash
		h := c.Hash()
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO execution_contracts (project_id, scope_key, version, hash, parent_hash, body)
			VALUES ($1,$2,$3,$4,$5,$6) RETURNING id::text`, d.ProjectID, scope, c.Version, h, c.ParentHash, c.Canonical()).Scan(&id); err != nil {
			return err
		}
		snap = Snapshot{ID: id, Scope: scope, Version: c.Version, Hash: h, Contract: c}
		return nil
	})
	if err != nil {
		return Snapshot{}, false, persistence.MapError(err)
	}
	return snap, unchanged, nil
}

// Get lê uma versão específica (0 = a mais recente) e confere a integridade.
func (s *Store) Get(ctx context.Context, projectID, scope string, version int) (Snapshot, error) {
	q := `SELECT id::text, version, hash, body FROM execution_contracts WHERE project_id=$1 AND scope_key=$2`
	args := []any{projectID, scope}
	if version > 0 {
		q += ` AND version=$3`
		args = append(args, version)
	} else {
		q += ` ORDER BY version DESC LIMIT 1`
	}
	var sn Snapshot
	var body []byte
	if err := s.DB.QueryRow(ctx, q, args...).Scan(&sn.ID, &sn.Version, &sn.Hash, &body); err != nil {
		return Snapshot{}, persistence.MapError(err)
	}
	sn.Scope = scope
	if err := json.Unmarshal(body, &sn.Contract); err != nil {
		return Snapshot{}, fmt.Errorf("contrato v%s ilegível: %w", strconv.Itoa(sn.Version), err)
	}
	if sn.Contract.Hash() != sn.Hash {
		return Snapshot{}, ErrTampered
	}
	return sn, nil
}

// Chain devolve todas as versões, da primeira à última, conferindo o encadeamento de hashes.
func (s *Store) Chain(ctx context.Context, projectID, scope string) ([]Snapshot, error) {
	rows, err := s.DB.Query(ctx, `SELECT version FROM execution_contracts WHERE project_id=$1 AND scope_key=$2 ORDER BY version`, projectID, scope)
	if err != nil {
		return nil, err
	}
	var vs []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return nil, err
		}
		vs = append(vs, v)
	}
	rows.Close()
	var out []Snapshot
	prev := ""
	for _, v := range vs {
		sn, err := s.Get(ctx, projectID, scope, v)
		if err != nil {
			return nil, err
		}
		if sn.Contract.ParentHash != prev {
			return nil, ErrTampered
		}
		prev = sn.Hash
		out = append(out, sn)
	}
	return out, nil
}
