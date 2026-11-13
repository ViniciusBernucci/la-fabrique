-- +goose Up
-- LF-AJ-04 (AJ04-03): cada despacho fixa a versão do contrato que recebeu. Evidências ligam
-- versão do contrato, execução e resultado. Linha só avança de running para um estado final.

CREATE TABLE dispatches (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    execution_id     text NOT NULL,
    scope_key        text NOT NULL,
    contract_id      uuid NOT NULL REFERENCES execution_contracts (id),
    contract_version integer NOT NULL,
    contract_hash    text NOT NULL,
    executor_kind    text NOT NULL CHECK (executor_kind IN ('cli', 'api')),
    state            text NOT NULL DEFAULT 'running' CHECK (state IN ('running', 'done', 'failed', 'cancelled')),
    result           jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at       timestamptz NOT NULL DEFAULT now(),
    finished_at      timestamptz,
    UNIQUE (project_id, execution_id)
);
CREATE INDEX dispatches_scope ON dispatches (project_id, scope_key, created_at);

-- +goose Down
DROP TABLE dispatches;
