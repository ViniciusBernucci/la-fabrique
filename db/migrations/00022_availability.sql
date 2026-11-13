-- +goose Up
-- T09-04: disponibilidade de contas e fallback. A saúde de cada conta combina o
-- sinal oficial (cota observada, erro de autenticação) com o histórico de
-- falhas; o fallback segue a cadeia explícita do agente e fica auditado; a
-- tarefa sem alternativa espera (liberando a vaga só com checkpoint seguro).

CREATE TABLE provider_health (
    connection_id        uuid PRIMARY KEY REFERENCES connections (id) ON DELETE CASCADE,
    state                text NOT NULL CHECK (state IN ('available','auth','quota','unavailable','billing')),
    source               text NOT NULL DEFAULT 'history' CHECK (source IN ('official','history')),
    reason               text NOT NULL DEFAULT '' CHECK (length(reason) <= 500),
    until                timestamptz,
    consecutive_failures integer NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
    last_failure_at      timestamptz,
    last_success_at      timestamptz,
    updated_at           timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE provider_events (
    id            bigserial PRIMARY KEY,
    connection_id uuid NOT NULL REFERENCES connections (id) ON DELETE CASCADE,
    kind          text NOT NULL CHECK (kind IN ('failure','success','cleared')),
    class         text NOT NULL DEFAULT '' CHECK (class IN ('','auth','quota','unavailable','billing')),
    source        text NOT NULL DEFAULT 'history',
    detail        text NOT NULL DEFAULT '' CHECK (length(detail) <= 500),
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX provider_events_conn_idx ON provider_events (connection_id, id DESC);

CREATE TABLE fallback_events (
    id            bigserial PRIMARY KEY,
    project_id    uuid NOT NULL,
    execution_id  uuid NOT NULL,
    node_key      text NOT NULL DEFAULT '',
    action        text NOT NULL CHECK (action IN ('switch','wait','hold')),
    from_conn     uuid,
    from_model    text NOT NULL DEFAULT '',
    to_conn       uuid,
    to_model      text NOT NULL DEFAULT '',
    reason        text NOT NULL DEFAULT '' CHECK (length(reason) <= 1000),
    checkpoint_id uuid REFERENCES checkpoints (id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fallback_events_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE RESTRICT
);
CREATE INDEX fallback_events_exec_idx ON fallback_events (execution_id, id);

CREATE TABLE waiting_tasks (
    execution_id  uuid PRIMARY KEY,
    project_id    uuid NOT NULL,
    node_key      text NOT NULL DEFAULT '',
    reason        text NOT NULL DEFAULT '' CHECK (length(reason) <= 1000),
    retry_at      timestamptz,
    attempts      integer NOT NULL DEFAULT 1 CHECK (attempts >= 1),
    slot_released boolean NOT NULL DEFAULT false,
    checkpoint_id uuid REFERENCES checkpoints (id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT waiting_tasks_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE CASCADE
);
CREATE INDEX waiting_tasks_due_idx ON waiting_tasks (retry_at);

-- +goose Down
DROP TABLE waiting_tasks;
DROP TABLE fallback_events;
DROP TABLE provider_events;
DROP TABLE provider_health;
