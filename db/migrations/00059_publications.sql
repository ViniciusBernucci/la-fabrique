-- +goose Up
-- LF-AJ-06 (AJ06-04) e LF-AJ-09 (AJ09-03): publicações na PR da feature (Draft inicial e commits de correção)
-- com a cabeça publicada, os gates conferidos e os vínculos com rodada, Issue, contratos e commits.
CREATE TABLE publications (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    feature_id    uuid NOT NULL,
    kind          text NOT NULL CHECK (kind IN ('draft', 'correction')),
    state         text NOT NULL DEFAULT 'published' CHECK (state IN ('published', 'cancelled')),
    round_key     text NOT NULL DEFAULT '',
    issue_key     text NOT NULL DEFAULT '',
    branch        text NOT NULL,
    head_sha      text NOT NULL,
    previous_head text NOT NULL DEFAULT '',
    pr_number     integer,
    pr_url        text NOT NULL DEFAULT '',
    commits       jsonb NOT NULL DEFAULT '[]'::jsonb,
    contracts     jsonb NOT NULL DEFAULT '[]'::jsonb,
    gate          jsonb NOT NULL DEFAULT '{}'::jsonb,
    detail        text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now()
);
-- Uma Draft PR por feature; uma publicação por (feature, kind, cabeça).
CREATE UNIQUE INDEX publications_one_draft ON publications (feature_id) WHERE kind = 'draft' AND state = 'published';
CREATE UNIQUE INDEX publications_head_key ON publications (feature_id, kind, head_sha) WHERE state = 'published';

-- +goose Down
DROP TABLE publications;
