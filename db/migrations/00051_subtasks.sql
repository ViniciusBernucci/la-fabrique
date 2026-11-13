-- +goose Up
-- LF-AJ-05 (AJ05-02): subtarefas técnicas materializadas SOMENTE na fábrica, uma linha por tarefa
-- e versão do plano, com trilha, risco, dependências e recursos compartilhados.

CREATE TABLE subtasks (
    project_id     uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    feature_id     uuid NOT NULL REFERENCES features (id) ON DELETE CASCADE,
    plan_version   integer NOT NULL CHECK (plan_version >= 1),
    task_key       text NOT NULL,
    track          text NOT NULL CHECK (track IN ('FRONTEND', 'BACKEND')),
    risk           text NOT NULL CHECK (risk IN ('low', 'medium', 'high')),
    depends_on     jsonb NOT NULL DEFAULT '[]'::jsonb,
    resources      jsonb NOT NULL DEFAULT '[]'::jsonb,
    exclusive_both boolean NOT NULL DEFAULT false,
    content_hash   text NOT NULL,
    state          text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'running', 'done', 'blocked')),
    created_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (feature_id, plan_version, task_key)
);

-- +goose Down
DROP TABLE subtasks;
