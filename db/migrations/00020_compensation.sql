-- +goose Up
-- T13-05: compensação de efeitos parciais por risco. Cada tentativa de compensar
-- um efeito confirmado fica registrada (aplicada, recusada ou falha); falha
-- conserva o trabalho e abre um bloqueio que impede novas operações agravantes
-- da execução até ser resolvido.

CREATE TABLE compensations (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       uuid NOT NULL,
    execution_id     uuid NOT NULL,
    outbox_id        uuid NOT NULL REFERENCES outbox (id) ON DELETE RESTRICT,
    operation        text NOT NULL,
    inverse          text NOT NULL DEFAULT '',
    risk             text NOT NULL CHECK (risk IN ('low', 'sensitive', 'irreversible')),
    state            text NOT NULL CHECK (state IN ('applied', 'refused', 'failed')),
    reason           text NOT NULL DEFAULT '' CHECK (length(reason) <= 1000),
    authorization_id uuid REFERENCES authorizations (id) ON DELETE RESTRICT,
    attempts         integer NOT NULL DEFAULT 1 CHECK (attempts >= 1),
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT compensations_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT compensations_outbox_key UNIQUE (outbox_id)
);

CREATE TABLE compensation_events (
    id              bigserial PRIMARY KEY,
    compensation_id uuid NOT NULL REFERENCES compensations (id) ON DELETE CASCADE,
    type            text NOT NULL CHECK (type IN ('requested', 'refused', 'applied', 'failed', 'blocked', 'block_resolved')),
    detail          jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE compensation_blocks (
    id              bigserial PRIMARY KEY,
    project_id      uuid NOT NULL,
    execution_id    uuid NOT NULL,
    compensation_id uuid NOT NULL REFERENCES compensations (id) ON DELETE CASCADE,
    created_at      timestamptz NOT NULL DEFAULT now(),
    resolved_at     timestamptz,
    resolved_by     uuid REFERENCES admins (id) ON DELETE SET NULL,
    note            text NOT NULL DEFAULT '' CHECK (length(note) <= 500),
    CONSTRAINT compensation_blocks_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE RESTRICT
);
CREATE UNIQUE INDEX compensation_blocks_open_key ON compensation_blocks (compensation_id) WHERE resolved_at IS NULL;
CREATE INDEX compensation_blocks_execution_idx ON compensation_blocks (execution_id) WHERE resolved_at IS NULL;

-- +goose Down
DROP TABLE compensation_blocks;
DROP TABLE compensation_events;
DROP TABLE compensations;
