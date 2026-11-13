-- +goose Up
-- LF-AJ-04 (AJ04-02): resultado do gate do PRD por revisão da Issue, com as pendências expostas.

CREATE TABLE prd_gate_results (
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    issue_key   text NOT NULL,
    revision    text NOT NULL,
    complete    boolean NOT NULL,
    problems    jsonb NOT NULL DEFAULT '[]'::jsonb,
    contract_id uuid REFERENCES execution_contracts (id),
    evaluated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, issue_key, revision)
);

-- +goose Down
DROP TABLE prd_gate_results;
