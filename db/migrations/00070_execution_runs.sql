-- +goose Up
-- Execução de um workflow publicado: estado serializado do motor, a tarefa pedida, a pasta de trabalho
-- e a posse temporária (lease) do worker que a conduz. A execução (executions) segue sendo o registro
-- público; esta tabela guarda o que o motor precisa para avançar e retomar.
CREATE TABLE execution_runs (
    execution_id        uuid PRIMARY KEY REFERENCES executions (id) ON DELETE RESTRICT,
    project_id          uuid NOT NULL REFERENCES projects (id) ON DELETE RESTRICT,
    workflow_version_id uuid NOT NULL,
    title               text NOT NULL CHECK (length(btrim(title)) > 0),
    task                text NOT NULL DEFAULT '',
    workspace           text NOT NULL DEFAULT '',
    status              text NOT NULL DEFAULT 'queued' CHECK (status IN (
                            'queued', 'running', 'waiting', 'awaiting_decision', 'completed', 'failed', 'cancelled')),
    outcome             text NOT NULL DEFAULT '',
    failure             text NOT NULL DEFAULT '',
    state               jsonb NOT NULL,
    wake_at             timestamptz,
    owner               text,
    lease_until         timestamptz,
    created_by          text NOT NULL DEFAULT '',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX execution_runs_active_idx ON execution_runs (status, wake_at) WHERE status IN ('queued', 'running', 'waiting');
CREATE INDEX execution_runs_project_idx ON execution_runs (project_id, created_at DESC);

-- +goose Down
DROP TABLE execution_runs;
