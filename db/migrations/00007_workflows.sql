-- +goose Up
-- T07-01: rascunhos de workflow. O grafo (nós, portas e conexões) é um documento
-- JSON versionado por `version` (controle otimista). Publicação imutável e
-- congelamento por execução chegam em T07-03 (tabelas próprias).

CREATE TABLE workflows (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    kind        text NOT NULL CHECK (kind IN ('feature', 'subflow')),
    description text NOT NULL DEFAULT '' CHECK (length(description) <= 500),
    graph       jsonb NOT NULL DEFAULT '{"nodes":[],"edges":[]}'::jsonb
                CHECK (jsonb_typeof(graph) = 'object' AND pg_column_size(graph) <= 1048576),
    version     integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT workflows_name_key UNIQUE (name)
);

-- +goose Down
DROP TABLE workflows;
