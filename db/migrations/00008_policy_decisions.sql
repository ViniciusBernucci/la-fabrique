-- +goose Up
-- T13-01: trilha de decisões da política de risco. Guarda a ação (ferramenta,
-- operação, recurso, destino), a decisão e a regra; os parâmetros entram só
-- redigidos (chaves sensíveis e valores com cara de segredo viram [REDACTED]) e
-- como hash dos parâmetros canônicos. Nunca valores de segredo.

CREATE TABLE policy_decisions (
    id              bigserial PRIMARY KEY,
    project_id      uuid REFERENCES projects (id) ON DELETE SET NULL,
    tool            text NOT NULL CHECK (length(tool) BETWEEN 1 AND 40),
    operation       text NOT NULL CHECK (length(operation) <= 200),
    resource        text NOT NULL DEFAULT '' CHECK (length(resource) <= 300),
    target          text NOT NULL DEFAULT '' CHECK (length(target) <= 200),
    effect          text NOT NULL CHECK (effect IN ('allow', 'approve', 'deny')),
    risk            text NOT NULL CHECK (risk IN ('low', 'sensitive', 'forbidden')),
    rule            text NOT NULL DEFAULT '',
    reason          text NOT NULL DEFAULT '' CHECK (length(reason) <= 500),
    params_hash     text NOT NULL CHECK (length(params_hash) = 64),
    params_redacted jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX policy_decisions_project_idx ON policy_decisions (project_id, id DESC);

-- +goose Down
DROP TABLE policy_decisions;
