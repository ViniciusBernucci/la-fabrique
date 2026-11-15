-- +goose Up
-- ADR-0062: verificação do repositório (pasta, origin, branch base, conta e permissão de push) vinculada à
-- configuração verificada, e unicidade da pasta do repositório entre projetos.
ALTER TABLE projects
    ADD COLUMN repo_check_ok boolean,
    ADD COLUMN repo_check_fingerprint text NOT NULL DEFAULT '',
    ADD COLUMN repo_checked_at timestamptz;

CREATE UNIQUE INDEX projects_repo_path_unique ON projects (lower(repo_path)) WHERE repo_path <> '';

ALTER TABLE project_changes DROP CONSTRAINT project_changes_action_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_action_check
    CHECK (action IN ('created', 'updated', 'source_changed', 'source_checked', 'repository_checked', 'activated', 'suspended', 'resumed'));

-- +goose Down
DELETE FROM project_changes WHERE action = 'repository_checked';
ALTER TABLE project_changes DROP CONSTRAINT project_changes_action_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_action_check
    CHECK (action IN ('created', 'updated', 'source_changed', 'source_checked', 'activated', 'suspended', 'resumed'));
DROP INDEX projects_repo_path_unique;
ALTER TABLE projects
    DROP COLUMN repo_checked_at,
    DROP COLUMN repo_check_fingerprint,
    DROP COLUMN repo_check_ok;
