-- +goose Up
-- LF-AJ-07 (AJ07-03): orientações do administrador ao agente, com o estado comprovado da entrega.
CREATE TABLE agent_guidance (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL,
    execution_id uuid NOT NULL,
    seq          integer NOT NULL,
    author       text NOT NULL,
    body         text NOT NULL,
    mode         text NOT NULL CHECK (mode IN ('immediate', 'next_safe_point', 'none')),
    state        text NOT NULL DEFAULT 'queued' CHECK (state IN ('queued', 'sent', 'acknowledged', 'uncertain', 'undeliverable')),
    detail       text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    sent_at      timestamptz,
    acked_at     timestamptz,
    UNIQUE (execution_id, seq),
    FOREIGN KEY (project_id, execution_id) REFERENCES executions (project_id, id) ON DELETE RESTRICT
);

-- +goose Down
DROP TABLE agent_guidance;
