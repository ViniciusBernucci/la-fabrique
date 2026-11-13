-- +goose Up
-- T10-03: limite global persistente, leases com heartbeat e token de geração, e
-- envelhecimento determinístico. A geração sobe a cada vaga reclamada: um escritor
-- que perdeu a vaga (lease vencida e vaga dada a outro) não consegue mais gravar.

CREATE TABLE scheduler_settings (
    id            integer PRIMARY KEY CHECK (id = 1),
    global_limit  integer NOT NULL DEFAULT 2 CHECK (global_limit BETWEEN 1 AND 64),
    aging_minutes integer NOT NULL DEFAULT 30 CHECK (aging_minutes BETWEEN 1 AND 10080),
    version       integer NOT NULL DEFAULT 1,
    updated_at    timestamptz NOT NULL DEFAULT now()
);
INSERT INTO scheduler_settings (id) VALUES (1);

ALTER TABLE queue_items
    ADD COLUMN lease_owner      text CHECK (lease_owner IS NULL OR length(lease_owner) BETWEEN 1 AND 100),
    ADD COLUMN lease_expires_at timestamptz,
    ADD COLUMN heartbeat_at     timestamptz,
    ADD COLUMN generation       bigint NOT NULL DEFAULT 0 CHECK (generation >= 0),
    ADD CONSTRAINT queue_items_lease_ck CHECK ((lease_owner IS NULL) = (lease_expires_at IS NULL));

CREATE TABLE queue_events (
    id         bigserial PRIMARY KEY,
    item_id    uuid NOT NULL REFERENCES queue_items (id) ON DELETE CASCADE,
    project_id uuid NOT NULL,
    action     text NOT NULL CHECK (action IN ('claimed', 'heartbeat_refused', 'lease_expired', 'write_refused')),
    owner      text NOT NULL DEFAULT '',
    generation bigint NOT NULL DEFAULT 0,
    detail     text NOT NULL DEFAULT '' CHECK (length(detail) <= 500),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX queue_events_item_idx ON queue_events (item_id, id);

-- +goose Down
DROP TABLE queue_events;
ALTER TABLE queue_items DROP CONSTRAINT queue_items_lease_ck, DROP COLUMN generation, DROP COLUMN heartbeat_at, DROP COLUMN lease_expires_at, DROP COLUMN lease_owner;
DROP TABLE scheduler_settings;
