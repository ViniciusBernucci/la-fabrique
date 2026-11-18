-- +goose Up
-- ADR-0065: login oficial das contas de assinatura de IA (Claude e Codex) pelo painel. A API grava a
-- intenção; o worker (com acesso ao Docker) roda o login oficial do cliente num volume por conta e
-- publica aqui só o necessário para a pessoa autorizar (link e código). Nenhum token passa por aqui:
-- a credencial vive só no volume lf-acct-<connection_id>.
CREATE TABLE cli_logins (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider       text NOT NULL CHECK (provider IN ('claude', 'codex')),
    name           text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 80),
    -- Id da futura conexão (tipo cli); também nomeia o volume da conta.
    connection_id  uuid NOT NULL DEFAULT gen_random_uuid(),
    state          text NOT NULL DEFAULT 'pending' CHECK (state IN
        ('pending', 'preparing', 'awaiting_user', 'verifying', 'done', 'failed', 'cancelled')),
    -- Desafio de curta duração: apagados ao terminar (qualquer desfecho).
    challenge_url  text CHECK (length(challenge_url) <= 2048),
    challenge_code text CHECK (length(challenge_code) <= 64),
    needs_code     boolean NOT NULL DEFAULT false,
    submitted_code text CHECK (length(submitted_code) <= 512),
    error          text NOT NULL DEFAULT '' CHECK (length(error) <= 400),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    expires_at     timestamptz NOT NULL
);
CREATE INDEX cli_logins_open_idx ON cli_logins (state) WHERE state IN ('pending', 'preparing', 'awaiting_user', 'verifying');

-- +goose Down
DROP TABLE cli_logins;
