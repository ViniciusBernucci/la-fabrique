-- +goose Up
-- T15-04: revisão externa e PR fechada. Cada comentário autorizado da PR é registrado uma vez
-- (por comentário e conteúdo) e vira um único item de correção; a PR fechada sem merge passa a
-- oferecer três ações (cancelar, reabrir, nova tentativa), todas com histórico.

ALTER TABLE deliveries DROP CONSTRAINT deliveries_state_check;
ALTER TABLE deliveries ADD CONSTRAINT deliveries_state_check CHECK (state IN ('blocked', 'draft_open', 'merged', 'closed', 'cancelled'));

CREATE TABLE pr_comments (
    id            bigserial PRIMARY KEY,
    delivery_id   uuid NOT NULL REFERENCES deliveries (id) ON DELETE CASCADE,
    comment_id    bigint NOT NULL CHECK (comment_id > 0),
    author        text NOT NULL CHECK (author <> '' AND length(author) <= 100),
    body          text NOT NULL CHECK (length(body) <= 4000),
    body_hash     text NOT NULL,
    state         text NOT NULL CHECK (state IN ('queued', 'held', 'fixed', 'dismissed')),
    queue_item_id uuid REFERENCES queue_items (id) ON DELETE SET NULL,
    discovered_at timestamptz NOT NULL DEFAULT now(),
    handled_at    timestamptz,
    CONSTRAINT pr_comments_key UNIQUE (delivery_id, comment_id, body_hash)
);
CREATE INDEX pr_comments_state_idx ON pr_comments (delivery_id, state);

CREATE TABLE delivery_actions (
    id          bigserial PRIMARY KEY,
    delivery_id uuid NOT NULL REFERENCES deliveries (id) ON DELETE CASCADE,
    action      text NOT NULL CHECK (action IN ('offered', 'cancel', 'reopen', 'retry')),
    actor       text NOT NULL DEFAULT '' CHECK (length(actor) <= 100),
    note        text NOT NULL DEFAULT '' CHECK (length(note) <= 500),
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX delivery_actions_idx ON delivery_actions (delivery_id, id);

-- +goose Down
DROP TABLE delivery_actions;
DROP TABLE pr_comments;
UPDATE deliveries SET state = 'closed' WHERE state = 'cancelled';
ALTER TABLE deliveries DROP CONSTRAINT deliveries_state_check;
ALTER TABLE deliveries ADD CONSTRAINT deliveries_state_check CHECK (state IN ('blocked', 'draft_open', 'merged', 'closed'));
