-- +goose Up
-- LF-AJ-06 (AJ06-01): uma worktree e uma branch temporária por tentativa que escreve código.
CREATE TABLE worktree_attempts (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    feature_id  uuid NOT NULL,
    issue_key   text NOT NULL DEFAULT '',
    track       text NOT NULL CHECK (track IN ('FRONTEND', 'BACKEND')),
    kind        text NOT NULL CHECK (kind IN ('implementation', 'correction')),
    attempt     integer NOT NULL CHECK (attempt >= 1),
    branch      text NOT NULL,
    path        text NOT NULL,
    base_sha    text NOT NULL,
    paths       jsonb NOT NULL DEFAULT '[]'::jsonb,
    resources   jsonb NOT NULL DEFAULT '[]'::jsonb,
    state       text NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'released')),
    writer_active boolean NOT NULL DEFAULT false,
    archive_ref text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    released_at timestamptz,
    UNIQUE (feature_id, track, kind, attempt)
);
CREATE UNIQUE INDEX worktree_attempts_branch_key ON worktree_attempts (project_id, branch);

CREATE TABLE worktree_checkpoints (
    id         bigserial PRIMARY KEY,
    attempt_id uuid NOT NULL REFERENCES worktree_attempts (id) ON DELETE CASCADE,
    seq        integer NOT NULL,
    ref        text NOT NULL,
    sha        text NOT NULL,
    files      integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (attempt_id, seq)
);

-- +goose Down
DROP TABLE worktree_checkpoints;
DROP TABLE worktree_attempts;
