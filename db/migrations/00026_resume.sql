-- +goose Up
-- T12-04: retomada seletiva. O baseline guarda os insumos (refs, requisito,
-- configuração) como estavam no último ponto reconciliado da execução; a
-- retomada compara com o estado atual, incorpora o que é compatível e abre uma
-- decisão no conflito. Toda retomada, bloqueio e incorporação fica na trilha.

CREATE TABLE resume_baselines (
    execution_id  uuid PRIMARY KEY,
    project_id    uuid NOT NULL,
    checkpoint_id uuid REFERENCES checkpoints (id) ON DELETE SET NULL,
    inputs        jsonb NOT NULL,
    inputs_hash   text NOT NULL CHECK (inputs_hash ~ '^[0-9a-f]{64}$'),
    recorded_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT resume_baselines_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE CASCADE
);

CREATE TABLE resume_events (
    id           bigserial PRIMARY KEY,
    project_id   uuid NOT NULL,
    execution_id uuid NOT NULL,
    node_key     text NOT NULL DEFAULT '',
    action       text NOT NULL CHECK (action IN ('resumed', 'blocked', 'incorporated')),
    detail       jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT resume_events_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE CASCADE
);
CREATE INDEX resume_events_exec_idx ON resume_events (execution_id, id);

-- +goose Down
DROP TABLE resume_events;
DROP TABLE resume_baselines;
