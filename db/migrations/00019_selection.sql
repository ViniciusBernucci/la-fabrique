-- +goose Up
-- T07-04: seleção determinística de fluxo e subfluxo por regra ordenada. Cada
-- projeto tem regras em ordem (a primeira que casa vence), um padrão explícito e
-- um rastro persistido de cada seleção com o motivo.

CREATE TABLE selection_rules (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    position    integer NOT NULL CHECK (position >= 1),
    name        text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 80),
    conditions  jsonb NOT NULL CHECK (jsonb_typeof(conditions) = 'array' AND jsonb_array_length(conditions) BETWEEN 1 AND 10),
    workflow_id uuid NOT NULL REFERENCES workflows (id) ON DELETE RESTRICT,
    subflow_id  uuid REFERENCES workflows (id) ON DELETE RESTRICT,
    enabled     boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT selection_rules_position_key UNIQUE (project_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE selection_defaults (
    project_id  uuid PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    workflow_id uuid NOT NULL REFERENCES workflows (id) ON DELETE RESTRICT,
    subflow_id  uuid REFERENCES workflows (id) ON DELETE RESTRICT
);

CREATE TABLE selection_traces (
    id           bigserial PRIMARY KEY,
    project_id   uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    feature_ref  text NOT NULL DEFAULT '' CHECK (length(feature_ref) <= 200),
    input        jsonb NOT NULL,
    rule_id      uuid,
    rule_position integer,
    used_default boolean NOT NULL,
    workflow_id  uuid,
    subflow_id   uuid,
    matched      jsonb NOT NULL DEFAULT '[]'::jsonb,
    reason       text NOT NULL CHECK (length(reason) BETWEEN 1 AND 1000),
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX selection_traces_project_idx ON selection_traces (project_id, id DESC);

-- +goose Down
DROP TABLE selection_traces;
DROP TABLE selection_defaults;
DROP TABLE selection_rules;
