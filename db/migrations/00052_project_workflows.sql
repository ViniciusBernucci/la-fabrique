-- +goose Up
-- LF-AJ-05 (AJ05-01): cada projeto tem o seu workflow estrutural (rascunho padrão criado no cadastro).
-- O vínculo marca o workflow como gerido: a publicação exige as guardas obrigatórias da Sentinela.

CREATE TABLE project_workflows (
    project_id  uuid PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    workflow_id uuid NOT NULL UNIQUE REFERENCES workflows (id) ON DELETE RESTRICT,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE project_workflows;
