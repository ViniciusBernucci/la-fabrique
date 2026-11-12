-- +goose Up
-- T05-01: cadastro de projeto em rascunho. Os campos do assistente ficam no
-- próprio projeto (vazio = ainda não preenchido) e nenhum agente, workflow ou
-- recurso global é associado automaticamente. Credenciais só por referência
-- (connections.id); nunca o valor.

ALTER TABLE projects
    ADD COLUMN description text NOT NULL DEFAULT '' CHECK (length(description) <= 500),
    ADD COLUMN wizard_step text NOT NULL DEFAULT 'basics'
        CHECK (wizard_step IN ('basics', 'repository', 'source', 'review')),
    ADD COLUMN repo_owner text NOT NULL DEFAULT '' CHECK (length(repo_owner) <= 39),
    ADD COLUMN repo_name text NOT NULL DEFAULT '' CHECK (length(repo_name) <= 100),
    ADD COLUMN base_branch text NOT NULL DEFAULT '' CHECK (length(base_branch) <= 200),
    ADD COLUMN repo_connection_id uuid REFERENCES connections (id) ON DELETE RESTRICT,
    ADD COLUMN source_kind text CHECK (source_kind IN ('jira', 'clickup')),
    ADD COLUMN source_connection_id uuid REFERENCES connections (id) ON DELETE RESTRICT,
    ADD COLUMN source_container text NOT NULL DEFAULT '' CHECK (length(source_container) <= 64),
    ADD COLUMN activated_at timestamptz;

-- Trilha de ativação, suspensão e alterações de configuração/origem. O detalhe
-- guarda só nomes de campos e referências, nunca valores de segredo.
CREATE TABLE project_changes (
    id          bigserial PRIMARY KEY,
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    admin_id    uuid REFERENCES admins (id) ON DELETE SET NULL,
    action      text NOT NULL CHECK (action IN ('created', 'updated', 'source_changed', 'activated', 'suspended', 'resumed')),
    detail      jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX project_changes_project_idx ON project_changes (project_id, id DESC);

-- +goose Down
DROP TABLE project_changes;
ALTER TABLE projects
    DROP COLUMN activated_at,
    DROP COLUMN source_container,
    DROP COLUMN source_connection_id,
    DROP COLUMN source_kind,
    DROP COLUMN repo_connection_id,
    DROP COLUMN base_branch,
    DROP COLUMN repo_name,
    DROP COLUMN repo_owner,
    DROP COLUMN wizard_step,
    DROP COLUMN description;
