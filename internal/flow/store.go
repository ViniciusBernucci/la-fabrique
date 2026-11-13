package flow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// ErrInvalidMap indica mapa que não pode ser salvo; os motivos vêm em MapError.
var ErrInvalidMap = errors.New("mapa de status inválido")

// ErrStaleMap indica gravação sobre versão antiga do mapa.
var ErrStaleMap = errors.New("o mapa mudou desde a leitura")

// MapError carrega os campos que impedem a projeção.
type MapError struct{ Problems []MapProblem }

func (e *MapError) Error() string {
	ps := make([]string, 0, len(e.Problems))
	for _, p := range e.Problems {
		ps = append(ps, p.String())
	}
	return ErrInvalidMap.Error() + ": " + strings.Join(ps, "; ")
}
func (e *MapError) Unwrap() error { return ErrInvalidMap }

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Store persiste o catálogo e os mapas.
type Store struct{ DB DB }

// EnsureCatalog grava o catálogo, os aliases e as transições. Idempotente: reaplicar mantém IDs,
// vínculos e histórico (nada é apagado nem renumerado).
func (s *Store) EnsureCatalog(ctx context.Context) error {
	return persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		for _, st := range Catalog {
			if _, err := tx.Exec(ctx, `INSERT INTO status_catalog (id, code, label, ordinal) VALUES ($1,$2,$3,$4) ON CONFLICT (id) DO NOTHING`,
				st.ID, st.Code, st.Label, st.Ordinal); err != nil {
				return err
			}
		}
		for alias, code := range Aliases {
			if _, err := tx.Exec(ctx, `INSERT INTO status_aliases (alias, status_id) SELECT $1, id FROM status_catalog WHERE code = $2 ON CONFLICT (alias) DO NOTHING`, alias, code); err != nil {
				return err
			}
		}
		for _, tr := range Transitions {
			if _, err := tx.Exec(ctx, `INSERT INTO status_transitions (from_id, to_id)
				SELECT f.id, t.id FROM status_catalog f, status_catalog t WHERE f.code=$1 AND t.code=$2 ON CONFLICT DO NOTHING`, tr.From, tr.To); err != nil {
				return err
			}
		}
		return nil
	})
}

// SavedMap é o mapa gravado de um projeto.
type SavedMap struct {
	ProjectMap
	Version int `json:"version"`
}

// ProjectedCodes são os status que a fábrica projeta na origem.
func ProjectedCodes() []string {
	return []string{PRDDoc, PRDDocConcluido, Planejamento, Implementacao, Review, DraftPR, Concluido, ReprovadoPR, CorrecaoPR}
}

// SaveMap valida e grava o mapa. expectedVersion 0 cria; senão exige a versão atual.
func (s *Store) SaveMap(ctx context.Context, projectID string, expectedVersion int, m ProjectMap) (SavedMap, error) {
	if ps := m.Validate(ProjectedCodes()); len(ps) > 0 {
		return SavedMap{}, &MapError{ps}
	}
	remote := map[string]string{}
	for k, v := range m.Remote {
		n, _ := Normalize(v)
		remote[k] = n.Code // sempre o código canônico; o bruto é a chave
	}
	moves := map[string]string{}
	for k, v := range m.Moves {
		if n, ok := Normalize(k); ok {
			moves[n.Code] = v
		}
	}
	rj, _ := json.Marshal(remote)
	mj, _ := json.Marshal(moves)
	var out SavedMap
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		var cur int
		err := tx.QueryRow(ctx, `SELECT version FROM project_status_maps WHERE project_id=$1 FOR UPDATE`, projectID).Scan(&cur)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			if expectedVersion != 0 {
				return ErrStaleMap
			}
			_, err = tx.Exec(ctx, `INSERT INTO project_status_maps (project_id, source, remote, moves) VALUES ($1,$2,$3,$4)`, projectID, m.Source, rj, mj)
			out = SavedMap{ProjectMap{m.Source, remote, moves}, 1}
			return err
		case err != nil:
			return err
		}
		if expectedVersion != cur {
			return ErrStaleMap
		}
		_, err = tx.Exec(ctx, `UPDATE project_status_maps SET source=$2, remote=$3, moves=$4, version=version+1, updated_at=now() WHERE project_id=$1`, projectID, m.Source, rj, mj)
		out = SavedMap{ProjectMap{m.Source, remote, moves}, cur + 1}
		return err
	})
	if err != nil {
		return SavedMap{}, persistence.MapError(err)
	}
	return out, nil
}

// GetMap lê o mapa do projeto. ok=false se não houver.
func (s *Store) GetMap(ctx context.Context, projectID string) (SavedMap, bool, error) {
	var m SavedMap
	var rj, mj []byte
	err := s.DB.QueryRow(ctx, `SELECT source, remote, moves, version FROM project_status_maps WHERE project_id=$1`, projectID).Scan(&m.Source, &rj, &mj, &m.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return SavedMap{}, false, nil
	}
	if err != nil {
		return SavedMap{}, false, fmt.Errorf("ler mapa de status: %w", err)
	}
	_ = json.Unmarshal(rj, &m.Remote)
	_ = json.Unmarshal(mj, &m.Moves)
	return m, true, nil
}
