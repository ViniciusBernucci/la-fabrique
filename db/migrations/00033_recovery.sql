-- +goose Up
-- T12-03: recuperação de tentativas interrompidas (crash do worker, reboot da VPS).
-- Cada classe de falha tem o seu contador de tentativas (infraestrutura, IA, revisão),
-- com limites separados, e toda decisão automática ou escalada fica na trilha.

CREATE TABLE recovery_attempts (
    item_id    uuid NOT NULL REFERENCES queue_items (id) ON DELETE CASCADE,
    class      text NOT NULL CHECK (class IN ('infra', 'ai', 'review')),
    attempts   integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (item_id, class)
);

CREATE TABLE recovery_events (
    id            bigserial PRIMARY KEY,
    item_id       uuid NOT NULL REFERENCES queue_items (id) ON DELETE CASCADE,
    project_id    uuid NOT NULL,
    action        text NOT NULL CHECK (action IN ('recovered', 'decision', 'limit', 'deferred')),
    class         text NOT NULL CHECK (class IN ('infra', 'ai', 'review')),
    checkpoint_id uuid,
    slot_released boolean NOT NULL DEFAULT false,
    detail        jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX recovery_events_item_idx ON recovery_events (item_id, id);

-- +goose Down
DROP TABLE recovery_events;
DROP TABLE recovery_attempts;
