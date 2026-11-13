-- +goose Up
-- Núcleo de execução: features, execuções, eventos monotônicos, outbox e
-- metadados de artefatos. FKs compostas (project_id, ...) impedem vínculo
-- entre projetos diferentes.

CREATE TABLE features (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id        uuid NOT NULL REFERENCES projects (id) ON DELETE RESTRICT,
    external_id       text NOT NULL CHECK (length(btrim(external_id)) > 0),
    external_revision text,
    title             text NOT NULL CHECK (length(btrim(title)) > 0),
    state             text NOT NULL DEFAULT 'imported' CHECK (state IN (
                          'imported', 'planning', 'developing', 'final_validation',
                          'awaiting_merge', 'completed', 'needs_decision')),
    version           integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    -- Unicidade por projeto e identidade externa.
    CONSTRAINT features_project_external_key UNIQUE (project_id, external_id),
    -- Alvo das FKs compostas.
    CONSTRAINT features_project_id_key UNIQUE (project_id, id)
);

CREATE TABLE executions (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id            uuid NOT NULL,
    feature_id            uuid NOT NULL,
    state                 text NOT NULL DEFAULT 'queued' CHECK (state IN (
                              'queued', 'running', 'waiting', 'succeeded', 'failed', 'cancelled', 'uncertain')),
    attempt               integer NOT NULL DEFAULT 1 CHECK (attempt >= 1),
    previous_execution_id uuid REFERENCES executions (id),
    version               integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT executions_feature_fk FOREIGN KEY (project_id, feature_id)
        REFERENCES features (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT executions_attempt_key UNIQUE (feature_id, attempt),
    CONSTRAINT executions_project_id_key UNIQUE (project_id, id)
);

CREATE TABLE execution_events (
    execution_id uuid NOT NULL,
    seq          bigint NOT NULL CHECK (seq >= 1),
    project_id   uuid NOT NULL,
    type         text NOT NULL CHECK (length(btrim(type)) > 0),
    payload      jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (execution_id, seq),
    CONSTRAINT execution_events_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE RESTRICT
);

CREATE TABLE outbox (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL REFERENCES projects (id) ON DELETE RESTRICT,
    -- Chave de negócio única por efeito lógico, independente da tentativa.
    business_key text NOT NULL CHECK (length(btrim(business_key)) > 0),
    destination  text NOT NULL CHECK (length(btrim(destination)) > 0),
    request      jsonb NOT NULL DEFAULT '{}'::jsonb,
    state        text NOT NULL DEFAULT 'intent' CHECK (state IN (
                     'intent', 'sending', 'confirmed', 'failed', 'uncertain')),
    proof        jsonb,
    attempts     integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    version      integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT outbox_business_key_key UNIQUE (project_id, business_key)
);
CREATE INDEX outbox_pending_idx ON outbox (created_at) WHERE state IN ('intent', 'sending', 'uncertain');

CREATE TABLE artifacts (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     uuid NOT NULL REFERENCES projects (id) ON DELETE RESTRICT,
    execution_id   uuid,
    kind           text NOT NULL CHECK (length(btrim(kind)) > 0),
    sha256         text NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    size_bytes     bigint NOT NULL CHECK (size_bytes >= 0),
    -- Referência relativa ao volume de artefatos; nunca caminho absoluto nem '..'.
    storage_ref    text NOT NULL CHECK (storage_ref !~ '^/' AND storage_ref !~ '(^|/)\.\.(/|$)' AND length(storage_ref) > 0),
    schema_version integer NOT NULL DEFAULT 1 CHECK (schema_version >= 1),
    retention_until timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT artifacts_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT artifacts_content_key UNIQUE (project_id, kind, sha256)
);

-- +goose Down
DROP TABLE artifacts;
DROP TABLE outbox;
DROP TABLE execution_events;
DROP TABLE executions;
DROP TABLE features;
