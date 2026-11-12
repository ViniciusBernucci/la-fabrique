-- +goose Up
-- T05-02 (parte da origem): critérios de elegibilidade, mapas de status e
-- prioridade, escopo de leitura/escrita e o resultado da última verificação
-- remota da origem. A verificação vale só para a configuração que a produziu
-- (impressão digital); mudar origem, conexão, segredo ou mapas a invalida.

ALTER TABLE projects
    ADD COLUMN eligibility_marker text NOT NULL DEFAULT '' CHECK (length(eligibility_marker) <= 50),
    ADD COLUMN eligible_statuses jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(eligible_statuses) = 'array'),
    ADD COLUMN priority_map jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(priority_map) = 'object'),
    ADD COLUMN source_access text NOT NULL DEFAULT 'read' CHECK (source_access IN ('read', 'read_write')),
    ADD COLUMN source_check_ok boolean,
    ADD COLUMN source_check_fingerprint text NOT NULL DEFAULT '',
    ADD COLUMN source_checked_at timestamptz;

ALTER TABLE project_changes DROP CONSTRAINT project_changes_action_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_action_check
    CHECK (action IN ('created', 'updated', 'source_changed', 'source_checked', 'activated', 'suspended', 'resumed'));

-- +goose Down
DELETE FROM project_changes WHERE action = 'source_checked';
ALTER TABLE project_changes DROP CONSTRAINT project_changes_action_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_action_check
    CHECK (action IN ('created', 'updated', 'source_changed', 'activated', 'suspended', 'resumed'));
ALTER TABLE projects
    DROP COLUMN source_checked_at,
    DROP COLUMN source_check_fingerprint,
    DROP COLUMN source_check_ok,
    DROP COLUMN source_access,
    DROP COLUMN priority_map,
    DROP COLUMN eligible_statuses,
    DROP COLUMN eligibility_marker;
