-- +goose Up
-- T04-01/T04-02: segundo fator, códigos de recuperação, desafios de login,
-- limitação de tentativas, trilha de autenticação e metadados de sessão.

-- O acesso de dev provisório (ADR-0004) é removido: apaga o admin sem senha real.
DELETE FROM admins WHERE password_hash = 'dev-auth-no-password';

ALTER TABLE admins
    -- totp_secret (já existente) passa a guardar o segredo CIFRADO (AES-256-GCM, chave fora do banco).
    ADD COLUMN totp_pending_secret bytea,
    ADD COLUMN totp_confirmed_at   timestamptz,
    -- Último passo de 30 s aceito: impede reutilizar o mesmo código (replay).
    ADD COLUMN totp_last_step      bigint NOT NULL DEFAULT 0,
    -- Verdadeiro após entrar com código de recuperação: o 2FA deve ser substituído.
    ADD COLUMN totp_reset_required boolean NOT NULL DEFAULT false,
    ADD CONSTRAINT admins_totp_consistent CHECK (totp_confirmed_at IS NULL OR totp_secret IS NOT NULL);

ALTER TABLE admin_sessions
    ADD COLUMN csrf_token   text NOT NULL DEFAULT '',
    ADD COLUMN ip           text NOT NULL DEFAULT '',
    ADD COLUMN user_agent   text NOT NULL DEFAULT '',
    ADD COLUMN auth_method  text NOT NULL DEFAULT 'totp' CHECK (auth_method IN ('totp', 'recovery'));

-- Códigos de uso único: só o SHA-256 é guardado (códigos têm 80 bits de entropia).
CREATE TABLE recovery_codes (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id  uuid NOT NULL REFERENCES admins (id) ON DELETE CASCADE,
    code_hash bytea NOT NULL CHECK (octet_length(code_hash) = 32),
    used_at   timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT recovery_codes_hash_key UNIQUE (admin_id, code_hash)
);

-- Senha validada, segundo fator pendente. Só o hash do token é guardado.
CREATE TABLE auth_challenges (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id    uuid NOT NULL REFERENCES admins (id) ON DELETE CASCADE,
    token_hash  bytea NOT NULL CHECK (octet_length(token_hash) = 32),
    attempts    integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    CONSTRAINT auth_challenges_token_key UNIQUE (token_hash)
);

-- Limitação de tentativas por chave (conta e IP).
CREATE TABLE auth_throttle (
    key               text PRIMARY KEY CHECK (length(key) <= 128),
    failures          integer NOT NULL DEFAULT 0 CHECK (failures >= 0),
    window_started_at timestamptz NOT NULL DEFAULT now(),
    locked_until      timestamptz
);

-- Trilha de autenticação (nunca registra senha, código, token nem nome tentado).
CREATE TABLE auth_events (
    id             bigserial PRIMARY KEY,
    admin_id       uuid REFERENCES admins (id) ON DELETE SET NULL,
    type           text NOT NULL CHECK (length(btrim(type)) > 0),
    ip             text NOT NULL DEFAULT '',
    user_agent     text NOT NULL DEFAULT '',
    correlation_id text NOT NULL DEFAULT '',
    detail         jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX auth_events_created_idx ON auth_events (created_at DESC, id DESC);

-- +goose Down
DROP TABLE auth_events;
DROP TABLE auth_throttle;
DROP TABLE auth_challenges;
DROP TABLE recovery_codes;
ALTER TABLE admin_sessions
    DROP COLUMN auth_method, DROP COLUMN user_agent, DROP COLUMN ip, DROP COLUMN csrf_token;
ALTER TABLE admins
    DROP CONSTRAINT admins_totp_consistent,
    DROP COLUMN totp_reset_required, DROP COLUMN totp_last_step,
    DROP COLUMN totp_confirmed_at, DROP COLUMN totp_pending_secret;
