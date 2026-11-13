-- +goose Up
-- LF-AJ-03 (AJ03-03): reservas atômicas de vaga por projeto (FRONTEND, BACKEND, PLANNING:n) com
-- token de geração. A geração impede dupla execução e escrita tardia de quem perdeu a reserva.

CREATE SEQUENCE reservation_generation_seq;

CREATE TABLE track_reservations (
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    slot        text NOT NULL,
    holder      text NOT NULL,
    generation  bigint NOT NULL,
    acquired_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, slot)
);

-- +goose Down
DROP TABLE track_reservations;
DROP SEQUENCE reservation_generation_seq;
