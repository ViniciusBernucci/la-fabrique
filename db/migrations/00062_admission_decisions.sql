-- +goose Up
-- LF-AJ-03 (AJ03-03): auditoria das decisões de admissão (ordem da fila, admitido ou espera e o motivo).
CREATE TABLE admission_decisions (
    id         bigserial PRIMARY KEY,
    project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    candidate  text NOT NULL,
    stage      text NOT NULL,
    position   integer NOT NULL,
    admitted   boolean NOT NULL,
    wait       text NOT NULL DEFAULT '',
    dry_run    boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX admission_decisions_project_idx ON admission_decisions (project_id, id DESC);

-- +goose Down
DROP TABLE admission_decisions;
