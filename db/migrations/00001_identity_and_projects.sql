-- +goose Up
-- Identidade do administrador, sessões revogáveis e projetos.
-- IDs opacos (uuid v4). Datas em UTC (timestamptz). Mutações usam `version`.

CREATE TABLE admins (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username      text NOT NULL CHECK (username ~ '^[a-z0-9][a-z0-9._-]{2,63}$'),
    password_hash text NOT NULL,
    totp_secret   bytea,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT admins_username_key UNIQUE (username)
);
-- Invariante do MVP: existe no máximo um administrador.
CREATE UNIQUE INDEX admins_single_row ON admins ((true));

CREATE TABLE admin_sessions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id    uuid NOT NULL REFERENCES admins (id) ON DELETE CASCADE,
    token_hash  bytea NOT NULL CHECK (octet_length(token_hash) = 32),
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz,
    reauth_at   timestamptz,
    CONSTRAINT admin_sessions_token_key UNIQUE (token_hash),
    CONSTRAINT admin_sessions_expiry_check CHECK (expires_at > created_at)
);
CREATE INDEX admin_sessions_admin_idx ON admin_sessions (admin_id);

CREATE TABLE projects (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        text NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 63),
    name        text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    state       text NOT NULL DEFAULT 'draft' CHECK (state IN ('draft', 'active', 'paused', 'archived')),
    -- Faixa finita 0..4 da fila global (ARQUITETURA, "Estados, fila e exclusão mútua").
    priority    smallint NOT NULL DEFAULT 2 CHECK (priority BETWEEN 0 AND 4),
    version     integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT projects_slug_key UNIQUE (slug)
);
CREATE INDEX projects_created_idx ON projects (created_at DESC, id DESC);

-- +goose Down
DROP TABLE projects;
DROP TABLE admin_sessions;
DROP TABLE admins;
