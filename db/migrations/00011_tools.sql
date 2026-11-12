-- +goose Up
-- T08-03: ferramentas personalizadas (script e integração). As nativas ficam no
-- código. Esquemas de entrada/saída, permissões e limites são dados validados
-- no cadastro; o comando de uma ferramenta de script passa pela política antes
-- de ser aceito e de novo a cada chamada.

CREATE TABLE tools (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL CHECK (name ~ '^[a-z][a-z0-9_]{1,47}$'),
    kind        text NOT NULL CHECK (kind IN ('script', 'integration')),
    description text NOT NULL CHECK (length(btrim(description)) BETWEEN 1 AND 500),
    input       jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(input) = 'array'),
    output      jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(output) = 'array'),
    permissions jsonb NOT NULL CHECK (jsonb_typeof(permissions) = 'array'),
    limits      jsonb NOT NULL CHECK (jsonb_typeof(limits) = 'object'),
    command     text NOT NULL DEFAULT '' CHECK (length(command) <= 2000),
    operation   text NOT NULL DEFAULT '' CHECK (length(operation) <= 32),
    version     integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tools_name_key UNIQUE (name)
);

-- +goose Down
DROP TABLE tools;
