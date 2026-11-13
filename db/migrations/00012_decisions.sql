-- +goose Up
-- T13-02: pedidos de decisão humana. Um pedido pertence a UMA execução (FK
-- composta com o projeto) e registra a versão da execução no momento do pedido:
-- se a execução mudar, o pedido fica obsoleto e a resolução é recusada. A
-- resolução e o sinal de retomada acontecem na mesma transação.

CREATE TABLE decision_requests (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id        uuid NOT NULL,
    execution_id      uuid NOT NULL,
    node_id           text NOT NULL DEFAULT '' CHECK (length(node_id) <= 64),
    kind              text NOT NULL CHECK (kind IN ('conflict', 'missing_requirement', 'sensitive_operation', 'uncertain_result', 'pr_closed', 'other')),
    title             text NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 160),
    problem           text NOT NULL CHECK (length(btrim(problem)) BETWEEN 1 AND 2000),
    impact            text NOT NULL DEFAULT '' CHECK (length(impact) <= 1000),
    evidence          jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(evidence) = 'array'),
    options           jsonb NOT NULL CHECK (jsonb_typeof(options) = 'array' AND jsonb_array_length(options) BETWEEN 2 AND 6),
    state             text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'resolved', 'stale')),
    execution_version integer NOT NULL CHECK (execution_version >= 1),
    resolution_option text NOT NULL DEFAULT '',
    resolution_note   text NOT NULL DEFAULT '' CHECK (length(resolution_note) <= 500),
    resolved_by       uuid REFERENCES admins (id) ON DELETE SET NULL,
    resolved_at       timestamptz,
    version           integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT decision_requests_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT decision_requests_resolved_check CHECK (
        (state = 'resolved' AND resolution_option <> '' AND resolved_at IS NOT NULL) OR (state <> 'resolved'))
);
-- Um pedido pendente por execução, nó e tipo (o mesmo problema não vira dois pedidos).
CREATE UNIQUE INDEX decision_requests_pending_key ON decision_requests (execution_id, node_id, kind) WHERE state = 'pending';
CREATE INDEX decision_requests_state_idx ON decision_requests (state, created_at DESC);

-- +goose Down
DROP TABLE decision_requests;
