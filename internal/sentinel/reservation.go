package sentinel

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// ErrSlotBusy indica vaga já reservada por outro.
var ErrSlotBusy = errors.New("vaga ocupada")

// ErrStaleGeneration indica escrita de quem perdeu a reserva (geração antiga).
var ErrStaleGeneration = errors.New("geração da reserva obsoleta: escrita recusada")

// Reservation é uma vaga reservada com seu token de geração.
type Reservation struct {
	Slot       string `json:"slot"`
	Holder     string `json:"holder"`
	Generation int64  `json:"generation"`
}

// Reserver reserva vagas atomicamente.
type Reserver struct{ DB DB }

// Acquire reserva todas as vagas pedidas ou nenhuma (útil para trabalho que exige as duas trilhas).
func (r *Reserver) Acquire(ctx context.Context, projectID, holder string, slots ...string) ([]Reservation, error) {
	var out []Reservation
	err := persistence.WithTx(ctx, r.DB, func(tx pgx.Tx) error {
		out = nil
		for _, s := range slots {
			var gen int64
			err := tx.QueryRow(ctx, `INSERT INTO track_reservations (project_id, slot, holder, generation)
				VALUES ($1,$2,$3,nextval('reservation_generation_seq')) ON CONFLICT (project_id, slot) DO NOTHING RETURNING generation`,
				projectID, s, holder).Scan(&gen)
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: %s", ErrSlotBusy, s)
			}
			if err != nil {
				return err
			}
			out = append(out, Reservation{s, holder, gen})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Validate confere que a reserva ainda é do titular com aquela geração (antes de qualquer escrita).
func (r *Reserver) Validate(ctx context.Context, projectID string, res Reservation) error {
	var gen int64
	err := r.DB.QueryRow(ctx, `SELECT generation FROM track_reservations WHERE project_id=$1 AND slot=$2 AND holder=$3`, projectID, res.Slot, res.Holder).Scan(&gen)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && gen != res.Generation {
		return ErrStaleGeneration
	}
	return err
}

// Release libera a vaga se a geração conferir. Liberar uma reserva já substituída não afeta a nova.
func (r *Reserver) Release(ctx context.Context, projectID string, res Reservation) (bool, error) {
	tag, err := r.DB.Exec(ctx, `DELETE FROM track_reservations WHERE project_id=$1 AND slot=$2 AND holder=$3 AND generation=$4`, projectID, res.Slot, res.Holder, res.Generation)
	return tag.RowsAffected() == 1, err
}

// Takeover retira a reserva do titular atual (após comprovar que o executor anterior parou) e
// entrega uma nova geração; o antigo passa a falhar em Validate.
func (r *Reserver) Takeover(ctx context.Context, projectID, slot, newHolder string) (Reservation, error) {
	var gen int64
	err := r.DB.QueryRow(ctx, `INSERT INTO track_reservations (project_id, slot, holder, generation)
		VALUES ($1,$2,$3,nextval('reservation_generation_seq'))
		ON CONFLICT (project_id, slot) DO UPDATE SET holder=$3, generation=nextval('reservation_generation_seq'), acquired_at=now()
		RETURNING generation`, projectID, slot, newHolder).Scan(&gen)
	return Reservation{slot, newHolder, gen}, err
}
