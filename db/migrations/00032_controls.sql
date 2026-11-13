-- +goose Up
-- T10-04: controles operacionais, globais e por projeto, persistentes (sobrevivem ao
-- reboot) e auditados. `draining` = pausar após a feature atual (nenhuma feature nova
-- inicia; o que já começou termina); `suspended` = suspender agora (nada novo inicia e
-- o que roda é parado com checkpoint seguro).

CREATE TABLE controls (
    scope      text NOT NULL CHECK (scope IN ('global', 'project')),
    project_id uuid REFERENCES projects (id) ON DELETE CASCADE,
    mode       text NOT NULL CHECK (mode IN ('active', 'draining', 'suspended')),
    reason     text NOT NULL DEFAULT '' CHECK (length(reason) <= 500),
    changed_by uuid,
    version    integer NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT controls_scope_ck CHECK ((scope = 'global') = (project_id IS NULL))
);
CREATE UNIQUE INDEX controls_global_key ON controls ((1)) WHERE scope = 'global';
CREATE UNIQUE INDEX controls_project_key ON controls (project_id) WHERE scope = 'project';

CREATE TABLE control_events (
    id         bigserial PRIMARY KEY,
    scope      text NOT NULL,
    project_id uuid,
    from_mode  text NOT NULL,
    to_mode    text NOT NULL,
    admin_id   uuid,
    detail     text NOT NULL DEFAULT '' CHECK (length(detail) <= 500),
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE queue_items DROP CONSTRAINT queue_items_wait_reason_check;
ALTER TABLE queue_items ADD CONSTRAINT queue_items_wait_reason_check CHECK (wait_reason IN ('merge', 'provider', 'decision', 'resources', 'suspended'));

-- +goose Down
UPDATE queue_items SET state = 'queued', wait_reason = NULL, slot_held = false WHERE wait_reason = 'suspended';
ALTER TABLE queue_items DROP CONSTRAINT queue_items_wait_reason_check;
ALTER TABLE queue_items ADD CONSTRAINT queue_items_wait_reason_check CHECK (wait_reason IN ('merge', 'provider', 'decision', 'resources'));
DROP TABLE control_events;
DROP TABLE controls;
