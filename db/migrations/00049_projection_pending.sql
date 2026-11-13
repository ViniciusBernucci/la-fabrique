-- +goose Up
-- LF-AJ-01 (AJ01-04): pendências da projeção de status na origem (mapeamento, conflito, escrita),
-- expostas na página Issues até a confirmação. Uma linha por intenção (chave de negócio da outbox).

CREATE TABLE projection_pending (
    project_id   uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    business_key text NOT NULL,
    issue_key    text NOT NULL,
    version      integer NOT NULL,
    target_code  text NOT NULL,
    reason       text NOT NULL CHECK (reason IN ('mapping', 'conflict', 'write', 'unregistered', 'uncertain')),
    detail       text NOT NULL DEFAULT '',
    attempts     integer NOT NULL DEFAULT 0,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, business_key)
);
CREATE INDEX projection_pending_issue ON projection_pending (project_id, issue_key);

-- +goose Down
DROP TABLE projection_pending;
