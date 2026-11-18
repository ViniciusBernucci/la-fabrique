-- +goose Up
-- ADR-0062 (fase 2): registro de operações com operationId. Repetir uma provisão com o mesmo ID
-- (retry após queda) devolve a mesma tentativa e nunca cria outra.
CREATE TABLE worktree_operations (
    operation_id text NOT NULL CHECK (length(operation_id) BETWEEN 1 AND 100),
    project_id   uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    op           text NOT NULL CHECK (op IN ('provision')),
    attempt_id   uuid NOT NULL REFERENCES worktree_attempts (id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, operation_id, op)
);

-- +goose Down
DROP TABLE worktree_operations;
