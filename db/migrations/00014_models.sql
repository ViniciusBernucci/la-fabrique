-- +goose Up
-- T09-02: catálogo de modelos cadastrados manualmente. Um modelo só é usado
-- quando compatível com o provedor, a conta e as capacidades exigidas.

CREATE TABLE models (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider          text NOT NULL CHECK (provider IN ('claude', 'codex', 'api')),
    gateway           text NOT NULL DEFAULT '' CHECK (gateway IN ('', 'openrouter', 'vercel')),
    name              text NOT NULL CHECK (name ~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{0,79}$'),
    label             text NOT NULL DEFAULT '' CHECK (length(label) <= 120),
    supports_tools    boolean NOT NULL DEFAULT false,
    supports_streaming boolean NOT NULL DEFAULT false,
    context_window    integer NOT NULL DEFAULT 0 CHECK (context_window BETWEEN 0 AND 10000000),
    max_output_tokens integer NOT NULL DEFAULT 0 CHECK (max_output_tokens BETWEEN 0 AND 1000000),
    params            jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(params) = 'object'),
    state             text NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'disabled')),
    version           integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    -- gateway só existe no provedor api; api exige gateway.
    CONSTRAINT models_gateway_ck CHECK ((provider = 'api') = (gateway <> '')),
    CONSTRAINT models_identity_key UNIQUE (provider, gateway, name)
);

-- +goose Down
DROP TABLE models;
