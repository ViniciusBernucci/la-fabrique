-- +goose Up
-- T04-03: cofre de conexões. O valor do segredo fica cifrado (AES-256-GCM) com
-- chave externa ao banco (VAULT_KEY); o banco nunca vê texto claro. O id da
-- conexão é estável: a rotação troca o segredo e preserva os vínculos futuros.

CREATE TABLE connections (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind              text NOT NULL CHECK (kind IN ('github', 'jira', 'clickup', 'cli', 'api', 'mcp')),
    name              text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 80),
    -- Destino não secreto (ex.: URL do site Jira, organização GitHub).
    target            text NOT NULL DEFAULT '' CHECK (length(target) <= 200),
    -- Escopos mínimos declarados para o destino (documentação; não é segredo).
    scopes            text NOT NULL DEFAULT '' CHECK (length(scopes) <= 200),
    state             text NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'revoked')),
    secret_ciphertext bytea,
    key_id            text,
    secret_version    integer NOT NULL DEFAULT 1 CHECK (secret_version >= 1),
    rotated_at        timestamptz,
    revoked_at        timestamptz,
    last_test_at      timestamptz,
    last_test_status  text NOT NULL DEFAULT 'not_run' CHECK (last_test_status IN ('ok', 'failed', 'not_run')),
    last_test_detail  text NOT NULL DEFAULT '' CHECK (length(last_test_detail) <= 300),
    version           integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT connections_name_key UNIQUE (name),
    -- Revogada = segredo apagado; ativa = segredo presente.
    CONSTRAINT connections_secret_state CHECK (
        (state = 'active' AND secret_ciphertext IS NOT NULL AND key_id IS NOT NULL)
        OR (state = 'revoked' AND secret_ciphertext IS NULL))
);

-- +goose Down
DROP TABLE connections;
