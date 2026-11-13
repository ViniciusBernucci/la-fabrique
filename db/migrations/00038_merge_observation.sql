-- +goose Up
-- T15-03: observação do merge. A entrega passa a registrar o que o GitHub informou
-- (aberta, fechada sem merge, integrada com SHA), a projeção do status na origem (com a
-- falha visível e reconciliável) e cada observação fica auditada.

ALTER TABLE deliveries DROP CONSTRAINT deliveries_state_check;
ALTER TABLE deliveries ADD CONSTRAINT deliveries_state_check CHECK (state IN ('blocked', 'draft_open', 'merged', 'closed'));
ALTER TABLE deliveries
    ADD COLUMN merge_sha        text NOT NULL DEFAULT '' CHECK (merge_sha = '' OR merge_sha ~ '^[0-9a-f]{7,64}$'),
    ADD COLUMN merged_at        timestamptz,
    ADD COLUMN closed_at        timestamptz,
    ADD COLUMN observed_at      timestamptz,
    ADD COLUMN projection       text NOT NULL DEFAULT 'none' CHECK (projection IN ('none', 'pending', 'done', 'failed')),
    ADD COLUMN projection_error text NOT NULL DEFAULT '' CHECK (length(projection_error) <= 500);

CREATE TABLE delivery_observations (
    id          bigserial PRIMARY KEY,
    delivery_id uuid NOT NULL REFERENCES deliveries (id) ON DELETE CASCADE,
    outcome     text NOT NULL CHECK (outcome IN ('open', 'closed', 'merged', 'unknown')),
    detail      text NOT NULL DEFAULT '' CHECK (length(detail) <= 500),
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX delivery_observations_idx ON delivery_observations (delivery_id, id);

-- +goose Down
DROP TABLE delivery_observations;
ALTER TABLE deliveries DROP COLUMN projection_error, DROP COLUMN projection, DROP COLUMN observed_at, DROP COLUMN closed_at, DROP COLUMN merged_at, DROP COLUMN merge_sha;
UPDATE deliveries SET state = 'draft_open' WHERE state IN ('merged', 'closed');
ALTER TABLE deliveries DROP CONSTRAINT deliveries_state_check;
ALTER TABLE deliveries ADD CONSTRAINT deliveries_state_check CHECK (state IN ('blocked', 'draft_open'));
