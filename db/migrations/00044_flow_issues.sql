-- +goose Up
-- LF-AJ-03 (AJ03-02): Issues com status canônico e auditoria de toda decisão da Sentinela.
-- A transição, a evidência e a intenção de projeção são gravadas na mesma transação.

CREATE TABLE flow_issues (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    external_key     text NOT NULL,
    kind             text NOT NULL CHECK (kind IN ('feature', 'correction')),
    parent_issue_id  uuid REFERENCES flow_issues (id),
    feature_id       uuid REFERENCES features (id),
    status_id        text NOT NULL REFERENCES status_catalog (id),
    raw_status       text NOT NULL DEFAULT '',
    priority         integer,
    version          integer NOT NULL DEFAULT 1,
    last_observed_at timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, external_key)
);

CREATE TABLE flow_transitions (
    id              bigserial PRIMARY KEY,
    issue_id        uuid NOT NULL REFERENCES flow_issues (id) ON DELETE CASCADE,
    from_code       text NOT NULL,
    to_code         text NOT NULL,
    decision        text NOT NULL CHECK (decision IN ('applied', 'rejected', 'refused', 'stale')),
    reasons         jsonb NOT NULL DEFAULT '[]'::jsonb,
    actor           text NOT NULL,
    observer        text NOT NULL DEFAULT '',
    observation_key text NOT NULL,
    raw_status      text NOT NULL DEFAULT '',
    evidence        jsonb NOT NULL DEFAULT '{}'::jsonb,
    version_after   integer NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (issue_id, observation_key, decision)
);

-- +goose Down
DROP TABLE flow_transitions;
DROP TABLE flow_issues;
