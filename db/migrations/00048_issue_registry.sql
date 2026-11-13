-- +goose Up
-- LF-AJ-01 (AJ01-01/02): registro das Issues da origem (Jira/ClickUp) com identidade externa
-- estável e revisão observada, e o histórico de cada sincronização (geral, manual ou direcionada).

CREATE TABLE issue_registry (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    source         text NOT NULL CHECK (source IN ('jira', 'clickup')),
    remote_id      text NOT NULL,
    issue_key      text NOT NULL,
    title          text NOT NULL DEFAULT '',
    description    text NOT NULL DEFAULT '',
    raw_status     text NOT NULL DEFAULT '',
    status_code    text NOT NULL DEFAULT '',
    revision       text NOT NULL DEFAULT '',
    priority       text NOT NULL DEFAULT '',
    local_priority integer,
    kind           text NOT NULL CHECK (kind IN ('feature', 'correction')),
    url            text NOT NULL DEFAULT '',
    first_seen_at  timestamptz NOT NULL DEFAULT now(),
    last_seen_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, source, remote_id)
);
CREATE INDEX issue_registry_status ON issue_registry (project_id, status_code);

CREATE TABLE issue_sync_runs (
    id          bigserial PRIMARY KEY,
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    trigger     text NOT NULL CHECK (trigger IN ('daily', 'manual', 'targeted')),
    scope       text NOT NULL DEFAULT 'all',
    outcome     text NOT NULL CHECK (outcome IN ('ok', 'partial', 'failed')),
    read_count  integer NOT NULL DEFAULT 0,
    new_count   integer NOT NULL DEFAULT 0,
    updated     integer NOT NULL DEFAULT 0,
    ignored     integer NOT NULL DEFAULT 0,
    error       text NOT NULL DEFAULT '',
    started_at  timestamptz NOT NULL,
    finished_at timestamptz NOT NULL
);
CREATE INDEX issue_sync_runs_project ON issue_sync_runs (project_id, id DESC);

-- +goose Down
DROP TABLE issue_sync_runs;
DROP TABLE issue_registry;
