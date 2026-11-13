-- +goose Up
-- LF-AJ-06 (AJ06-03): consolidação serializada por feature, equivalência de commits, conflitos preservados
-- e cabeça candidata de entrega.
CREATE TABLE consolidation_runs (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    feature_id      uuid NOT NULL,
    delivery_branch text NOT NULL,
    state           text NOT NULL DEFAULT 'running' CHECK (state IN ('running', 'done', 'conflict', 'stale', 'failed')),
    expected_head   text NOT NULL DEFAULT '',
    new_head        text NOT NULL DEFAULT '',
    detail          text NOT NULL DEFAULT '',
    lease_until     timestamptz NOT NULL,
    started_at      timestamptz NOT NULL DEFAULT now(),
    finished_at     timestamptz
);
CREATE UNIQUE INDEX consolidation_one_running ON consolidation_runs (feature_id) WHERE state = 'running';

CREATE TABLE consolidated_commits (
    feature_id     uuid NOT NULL,
    original_sha   text NOT NULL,
    integrated_sha text NOT NULL,
    changed        boolean NOT NULL,
    attempt_ref    text NOT NULL DEFAULT '',
    run_id         uuid NOT NULL REFERENCES consolidation_runs (id) ON DELETE CASCADE,
    created_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (feature_id, original_sha)
);

CREATE TABLE delivery_candidates (
    feature_id   uuid NOT NULL,
    head_sha     text NOT NULL,
    base_head    text NOT NULL,
    run_id       uuid NOT NULL REFERENCES consolidation_runs (id) ON DELETE CASCADE,
    revalidate   jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (feature_id, head_sha)
);

CREATE TABLE consolidation_conflicts (
    id           bigserial PRIMARY KEY,
    run_id       uuid NOT NULL REFERENCES consolidation_runs (id) ON DELETE CASCADE,
    feature_id   uuid NOT NULL,
    attempt_ref  text NOT NULL,
    commit_sha   text NOT NULL,
    files        jsonb NOT NULL DEFAULT '[]'::jsonb,
    resolved_at  timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE consolidation_conflicts;
DROP TABLE delivery_candidates;
DROP TABLE consolidated_commits;
DROP TABLE consolidation_runs;
