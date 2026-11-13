-- +goose Up
-- T10-02: fila local por projeto. Cada item é uma tarefa técnica do plano (ou uma
-- correção de PR). Só um item por projeto ocupa a vaga (running, ou waiting sem
-- checkpoint seguro); a vaga só é devolvida depois de um checkpoint seguro.

CREATE TABLE queue_items (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    seq           bigserial NOT NULL,
    project_id    uuid NOT NULL,
    feature_id    uuid NOT NULL,
    task_key      text NOT NULL,
    kind          text NOT NULL DEFAULT 'task' CHECK (kind IN ('task', 'pr_fix')),
    ref           text NOT NULL DEFAULT '' CHECK (length(ref) <= 300), -- PR de uma correção
    priority      integer NOT NULL DEFAULT 2 CHECK (priority BETWEEN 0 AND 4),
    state         text NOT NULL DEFAULT 'queued' CHECK (state IN ('queued', 'running', 'waiting', 'done', 'cancelled')),
    wait_reason   text CHECK (wait_reason IN ('merge', 'provider', 'decision', 'resources')),
    wait_ref      text NOT NULL DEFAULT '' CHECK (length(wait_ref) <= 300),
    execution_id  uuid,
    checkpoint_id uuid REFERENCES checkpoints (id) ON DELETE SET NULL,
    slot_held     boolean NOT NULL DEFAULT false,
    version       integer NOT NULL DEFAULT 1,
    enqueued_at   timestamptz NOT NULL DEFAULT now(),
    started_at    timestamptz,
    finished_at   timestamptz,
    CONSTRAINT queue_items_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT queue_items_wait_ck CHECK ((state = 'waiting') = (wait_reason IS NOT NULL)),
    CONSTRAINT queue_items_slot_ck CHECK (NOT slot_held OR state IN ('running', 'waiting'))
);
-- Um item ativo por (feature, tarefa, tipo, referência): enfileirar de novo é idempotente.
CREATE UNIQUE INDEX queue_items_active_key ON queue_items (feature_id, task_key, kind, ref) WHERE state IN ('queued', 'running', 'waiting');
-- Uma vaga por projeto.
CREATE UNIQUE INDEX queue_items_one_slot ON queue_items (project_id) WHERE slot_held;
CREATE INDEX queue_items_project_idx ON queue_items (project_id, state);

-- +goose Down
DROP TABLE queue_items;
