-- +goose Up
-- T09-03: orçamento e uso. Gasto em API (custo) e cota de assinatura (unidades
-- observadas) são registrados SEPARADAMENTE e nunca se somam. A reserva de custo
-- é atômica: quem disputa o mesmo orçamento é serializado pelo bloqueio da linha
-- do orçamento, então a soma de reservas e gastos nunca passa do teto.

-- Preço por milhão de tokens (micro-USD), com a fonte. Sem preço, o modelo de API
-- não pode ser orçado e a reserva é recusada (falha fechada).
ALTER TABLE models
    ADD COLUMN input_micros_per_mtok  bigint CHECK (input_micros_per_mtok BETWEEN 0 AND 1000000000),
    ADD COLUMN output_micros_per_mtok bigint CHECK (output_micros_per_mtok BETWEEN 0 AND 1000000000),
    ADD COLUMN price_source           text NOT NULL DEFAULT '' CHECK (length(price_source) <= 200),
    ADD CONSTRAINT models_price_pair_ck CHECK ((input_micros_per_mtok IS NULL) = (output_micros_per_mtok IS NULL));

CREATE TABLE budgets (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    scope        text NOT NULL CHECK (scope IN ('global', 'agent')),
    agent_id     uuid REFERENCES agents (id) ON DELETE CASCADE,
    period       text NOT NULL CHECK (period IN ('day', 'month')),
    limit_micros bigint NOT NULL CHECK (limit_micros >= 0),
    state        text NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'disabled')),
    version      integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT budgets_scope_ck CHECK ((scope = 'global') = (agent_id IS NULL))
);
CREATE UNIQUE INDEX budgets_global_key ON budgets (period) WHERE scope = 'global';
CREATE UNIQUE INDEX budgets_agent_key ON budgets (agent_id, period) WHERE scope = 'agent';

CREATE TABLE usage_records (
    id            bigserial PRIMARY KEY,
    kind          text NOT NULL CHECK (kind IN ('api_cost', 'subscription')),
    agent_id      uuid REFERENCES agents (id) ON DELETE SET NULL,
    connection_id uuid REFERENCES connections (id) ON DELETE SET NULL,
    model         text NOT NULL DEFAULT '' CHECK (length(model) <= 80),
    input_tokens  bigint NOT NULL DEFAULT 0 CHECK (input_tokens >= 0),
    output_tokens bigint NOT NULL DEFAULT 0 CHECK (output_tokens >= 0),
    cost_micros   bigint NOT NULL DEFAULT 0 CHECK (cost_micros >= 0),
    quota_units   bigint NOT NULL DEFAULT 0 CHECK (quota_units >= 0),
    quota_unit    text NOT NULL DEFAULT '' CHECK (length(quota_unit) <= 32),
    source        text NOT NULL CHECK (source IN ('provider_reported', 'estimated', 'reconciliation')),
    reservation_id uuid,
    overrun_micros bigint NOT NULL DEFAULT 0 CHECK (overrun_micros >= 0),
    observed_at   timestamptz NOT NULL DEFAULT now(),
    created_at    timestamptz NOT NULL DEFAULT now(),
    -- Assinatura nunca carrega custo; custo de API nunca carrega cota.
    CONSTRAINT usage_kind_ck CHECK (
        (kind = 'api_cost' AND quota_units = 0) OR (kind = 'subscription' AND cost_micros = 0))
);
CREATE INDEX usage_records_window_idx ON usage_records (kind, observed_at);
CREATE INDEX usage_records_agent_idx ON usage_records (agent_id, kind, observed_at);

CREATE TABLE usage_reservations (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id      uuid REFERENCES agents (id) ON DELETE SET NULL,
    connection_id uuid REFERENCES connections (id) ON DELETE SET NULL,
    amount_micros bigint NOT NULL CHECK (amount_micros >= 0),
    state         text NOT NULL DEFAULT 'reserved' CHECK (state IN ('reserved', 'settled', 'released')),
    expires_at    timestamptz NOT NULL,
    usage_id      bigint REFERENCES usage_records (id),
    created_at    timestamptz NOT NULL DEFAULT now(),
    settled_at    timestamptz,
    CONSTRAINT usage_reservations_settled_ck CHECK ((state = 'settled') = (usage_id IS NOT NULL))
);
CREATE INDEX usage_reservations_active_idx ON usage_reservations (agent_id, state, expires_at);

-- Cotas de assinatura OBSERVADAS (com fonte e horário); nunca estimadas.
CREATE TABLE quota_observations (
    id            bigserial PRIMARY KEY,
    connection_id uuid NOT NULL REFERENCES connections (id) ON DELETE CASCADE,
    used          bigint NOT NULL CHECK (used >= 0),
    quota_limit   bigint NOT NULL CHECK (quota_limit >= 0),
    unit          text NOT NULL CHECK (length(unit) BETWEEN 1 AND 32),
    resets_at     timestamptz,
    source        text NOT NULL CHECK (length(source) BETWEEN 1 AND 100),
    observed_at   timestamptz NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX quota_observations_conn_idx ON quota_observations (connection_id, observed_at DESC);

-- Reconciliação com o total informado pelo provedor num período.
CREATE TABLE usage_reconciliations (
    id              bigserial PRIMARY KEY,
    connection_id   uuid NOT NULL REFERENCES connections (id) ON DELETE CASCADE,
    period_start    timestamptz NOT NULL,
    period_end      timestamptz NOT NULL CHECK (period_end > period_start),
    recorded_micros bigint NOT NULL CHECK (recorded_micros >= 0),
    reported_micros bigint NOT NULL CHECK (reported_micros >= 0),
    drift_micros    bigint NOT NULL,
    adjustment_id   bigint REFERENCES usage_records (id),
    source          text NOT NULL CHECK (length(source) BETWEEN 1 AND 100),
    created_at      timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE usage_reconciliations;
DROP TABLE quota_observations;
DROP TABLE usage_reservations;
DROP TABLE usage_records;
DROP TABLE budgets;
ALTER TABLE models DROP CONSTRAINT models_price_pair_ck,
    DROP COLUMN input_micros_per_mtok, DROP COLUMN output_micros_per_mtok, DROP COLUMN price_source;
