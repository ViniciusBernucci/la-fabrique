-- +goose Up
-- ADR-0062 (fase 3): setup do repositório com aprovação vinculada ao hash e política de limpeza.
CREATE TABLE project_setup_approvals (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    ref           text NOT NULL CHECK (length(ref) BETWEEN 1 AND 200),
    script_sha256 text NOT NULL CHECK (script_sha256 ~ '^[0-9a-f]{64}$'),
    approved_by   uuid,
    approved_at   timestamptz NOT NULL DEFAULT now(),
    revoked_at    timestamptz
);
CREATE UNIQUE INDEX project_setup_approvals_active ON project_setup_approvals (project_id, ref, script_sha256) WHERE revoked_at IS NULL;

CREATE TABLE project_setup_runs (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    ref           text NOT NULL,
    script_sha256 text NOT NULL,
    status        text NOT NULL CHECK (status IN ('ok', 'failed', 'timeout')),
    exit_code     integer NOT NULL,
    output_tail   text NOT NULL DEFAULT '',
    started_at    timestamptz NOT NULL DEFAULT now(),
    finished_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX project_setup_runs_project ON project_setup_runs (project_id, started_at DESC);

CREATE TABLE worktree_cleanup_policy (
    project_id           uuid PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    enabled              boolean NOT NULL DEFAULT false,
    archive_idle_days    integer NOT NULL DEFAULT 7 CHECK (archive_idle_days BETWEEN 1 AND 365),
    remove_archived_days integer NOT NULL DEFAULT 30 CHECK (remove_archived_days BETWEEN 1 AND 3650),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE project_changes DROP CONSTRAINT project_changes_action_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_action_check
    CHECK (action IN ('created', 'updated', 'source_changed', 'source_checked', 'repository_checked', 'activated', 'suspended', 'resumed',
                      'setup_approved', 'setup_revoked', 'setup_ran', 'cleanup_policy_changed', 'cleanup_applied'));

-- +goose Down
DELETE FROM project_changes WHERE action IN ('setup_approved', 'setup_revoked', 'setup_ran', 'cleanup_policy_changed', 'cleanup_applied');
ALTER TABLE project_changes DROP CONSTRAINT project_changes_action_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_action_check
    CHECK (action IN ('created', 'updated', 'source_changed', 'source_checked', 'repository_checked', 'activated', 'suspended', 'resumed'));
DROP TABLE worktree_cleanup_policy;
DROP TABLE project_setup_runs;
DROP TABLE project_setup_approvals;
