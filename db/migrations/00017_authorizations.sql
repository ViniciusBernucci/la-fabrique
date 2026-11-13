-- +goose Up
-- T13-03: autorizações com escopo. Uma autorização nasce de uma aprovação humana e
-- vale para UMA ação canônica (ferramenta, operação, recurso, destino e hash dos
-- parâmetros), por tempo e número de usos limitados. O consumo é um UPDATE
-- condicional (atômico): na corrida pelo último uso, um só vence. Revogar impede
-- novos usos e, havendo uso em andamento, dispara a parada segura.

CREATE TABLE authorizations (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    uuid NOT NULL,
    execution_id  uuid NOT NULL,
    node_id       text NOT NULL DEFAULT '' CHECK (length(node_id) <= 64),
    decision_id   uuid REFERENCES decision_requests (id) ON DELETE RESTRICT,
    tool          text NOT NULL CHECK (length(tool) BETWEEN 1 AND 64),
    operation     text NOT NULL CHECK (length(operation) BETWEEN 1 AND 64),
    resource      text NOT NULL DEFAULT '' CHECK (length(resource) <= 1024),
    target        text NOT NULL DEFAULT '' CHECK (length(target) <= 255),
    params_hash   text NOT NULL CHECK (params_hash ~ '^[0-9a-f]{64}$'),
    max_uses      integer NOT NULL CHECK (max_uses BETWEEN 1 AND 100),
    uses          integer NOT NULL DEFAULT 0,
    expires_at    timestamptz NOT NULL,
    state         text NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'revoked')),
    revoked_at    timestamptz,
    revoked_by    uuid REFERENCES admins (id) ON DELETE SET NULL,
    revoke_reason text NOT NULL DEFAULT '' CHECK (length(revoke_reason) <= 500),
    granted_by    uuid REFERENCES admins (id) ON DELETE SET NULL,
    version       integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT authorizations_uses_ck CHECK (uses >= 0 AND uses <= max_uses),
    CONSTRAINT authorizations_revoked_ck CHECK ((state = 'revoked') = (revoked_at IS NOT NULL)),
    CONSTRAINT authorizations_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT authorizations_project_id_key UNIQUE (project_id, id)
);
CREATE INDEX authorizations_execution_idx ON authorizations (execution_id, state);

CREATE TABLE authorization_uses (
    id               bigserial PRIMARY KEY,
    authorization_id uuid NOT NULL REFERENCES authorizations (id) ON DELETE CASCADE,
    state            text NOT NULL DEFAULT 'started' CHECK (state IN ('started', 'done', 'failed')),
    started_at       timestamptz NOT NULL,
    finished_at      timestamptz
);
CREATE INDEX authorization_uses_open_idx ON authorization_uses (authorization_id) WHERE state = 'started';

CREATE TABLE authorization_events (
    id               bigserial PRIMARY KEY,
    authorization_id uuid NOT NULL REFERENCES authorizations (id) ON DELETE CASCADE,
    type             text NOT NULL CHECK (type IN ('created', 'consumed', 'refused', 'revoked', 'halted', 'halt_failed')),
    detail           jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at       timestamptz NOT NULL
);
CREATE INDEX authorization_events_idx ON authorization_events (authorization_id, id);

-- +goose Down
DROP TABLE authorization_events;
DROP TABLE authorization_uses;
DROP TABLE authorizations;
