-- +goose Up
-- T10-01: plano de decomposição de uma feature, versionado. Cada versão guarda o
-- plano validado (tarefas, critérios verificáveis, risco, DAG) com o hash do
-- conteúdo. Plano inválido ou conflitante é REGISTRADO como rejeitado e a versão
-- vigente continua intacta. A identidade da tarefa é (feature, chave) e atravessa
-- as versões: o estado e o item remoto não se perdem ao replanejar.

CREATE TABLE plans (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL,
    feature_id   uuid NOT NULL,
    version      integer NOT NULL CHECK (version >= 1),
    status       text NOT NULL CHECK (status IN ('current', 'superseded', 'rejected')),
    content      jsonb NOT NULL,
    content_hash text NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    problems     jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(problems) = 'array'),
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT plans_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT plans_version_key UNIQUE (feature_id, version)
);
CREATE UNIQUE INDEX plans_current_idx ON plans (feature_id) WHERE status = 'current';

CREATE TABLE plan_tasks (
    project_id   uuid NOT NULL,
    feature_id   uuid NOT NULL,
    task_key     text NOT NULL CHECK (task_key ~ '^[a-z0-9][a-z0-9_-]{0,39}$'),
    title        text NOT NULL,
    content_hash text NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    state        text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'running', 'done', 'blocked')),
    removed      boolean NOT NULL DEFAULT false,
    remote_id    text NOT NULL DEFAULT '',
    remote_key   text NOT NULL DEFAULT '',
    plan_version integer NOT NULL CHECK (plan_version >= 1),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (feature_id, task_key),
    CONSTRAINT plan_tasks_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT
);

-- +goose Down
DROP TABLE plan_tasks;
DROP TABLE plans;
