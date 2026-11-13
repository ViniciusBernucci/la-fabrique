-- +goose Up
-- T08-01: catálogo global de agentes. Um agente reúne instruções, provedor,
-- conta (conexão do cofre, só a referência), modelo, skills e ferramentas por
-- nome, cadeia de fallback e limites. Sobrescritas por nó NÃO ficam aqui: ficam
-- no nó do workflow e são resolvidas em tempo de execução, preservando a base.

CREATE TABLE agents (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name            text NOT NULL CHECK (name ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    description     text NOT NULL DEFAULT '' CHECK (length(description) <= 500),
    instructions    text NOT NULL CHECK (length(btrim(instructions)) BETWEEN 1 AND 20000),
    provider        text NOT NULL CHECK (provider IN ('claude', 'codex', 'api')),
    connection_id   uuid REFERENCES connections (id) ON DELETE RESTRICT,
    model           text NOT NULL DEFAULT '' CHECK (length(model) <= 80),
    skills          jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(skills) = 'array'),
    tools           jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(tools) = 'array'),
    fallback        jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(fallback) = 'array'),
    timeout_minutes integer NOT NULL DEFAULT 30 CHECK (timeout_minutes BETWEEN 1 AND 720),
    version         integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT agents_name_key UNIQUE (name)
);

-- +goose Down
DROP TABLE agents;
