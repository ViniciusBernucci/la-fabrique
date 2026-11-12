-- +goose Up
-- T05-03: perfil técnico e recursos do projeto. O perfil guarda comandos (com a
-- origem de cada valor) e referências de segredo (nunca o valor). As associações
-- escolhem, de forma explícita, recursos GLOBAIS (workflow, agente, skill,
-- ferramenta, modelo); a sobrescrita local vive na associação e nunca altera o
-- recurso global.

CREATE TABLE project_profiles (
    project_id    uuid PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    commands      jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(commands) = 'object'),
    services      jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(services) = 'array'),
    secret_refs   jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(secret_refs) = 'object'),
    discovery     jsonb,
    discovered_at timestamptz,
    version       integer NOT NULL DEFAULT 0 CHECK (version >= 0), -- 0 = só descoberta; 1+ = perfil escolhido
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE project_resources (
    project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('workflow', 'agent', 'skill', 'tool', 'model')),
    ref        text NOT NULL CHECK (length(ref) BETWEEN 1 AND 200),
    override   jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(override) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, kind, ref)
);

-- +goose Down
DROP TABLE project_resources;
DROP TABLE project_profiles;
