-- +goose Up
-- LF-AJ-08 (AJ08-04): configuração versionada do ambiente de teste/visual por projeto.
CREATE TABLE env_configs (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    version         integer NOT NULL CHECK (version >= 1),
    state           text NOT NULL DEFAULT 'proposal' CHECK (state IN ('proposal', 'validating', 'validated', 'invalid', 'needs_revalidation')),
    content         jsonb NOT NULL,
    content_hash    text NOT NULL,
    material_hash   text NOT NULL,
    base_sha        text NOT NULL DEFAULT '',
    evidence        jsonb,
    diagnostic      text NOT NULL DEFAULT '',
    approved_by     text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    validated_at    timestamptz,
    UNIQUE (project_id, version)
);

-- A versão aprovada que as próximas execuções fixam.
CREATE TABLE env_active (
    project_id  uuid PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    config_id   uuid NOT NULL REFERENCES env_configs (id) ON DELETE RESTRICT,
    approved_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE env_active;
DROP TABLE env_configs;
