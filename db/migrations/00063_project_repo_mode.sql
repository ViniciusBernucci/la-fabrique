-- +goose Up
-- ADR-0061: o repositório do projeto pode vir do GitHub (clonar), de uma pasta existente
-- ou ser criado do zero. repo_path é relativo à raiz de workspaces (WORKSPACES_DIR).
ALTER TABLE projects
    ADD COLUMN repo_mode text NOT NULL DEFAULT 'github' CHECK (repo_mode IN ('github', 'local', 'new')),
    ADD COLUMN repo_path text NOT NULL DEFAULT '' CHECK (length(repo_path) <= 300);

-- +goose Down
ALTER TABLE projects
    DROP COLUMN repo_path,
    DROP COLUMN repo_mode;
