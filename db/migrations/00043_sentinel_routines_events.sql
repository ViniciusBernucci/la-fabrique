-- +goose Up
-- LF-AJ-03 (AJ03-01): eventos internos duráveis e estado de rotinas agendadas, com relógio e
-- cursor independentes por rotina e alvo (escopo). Aditiva.

CREATE TABLE sentinel_events (
    id          bigserial PRIMARY KEY,
    project_id  uuid REFERENCES projects (id) ON DELETE CASCADE,
    kind        text NOT NULL,
    dedupe_key  text NOT NULL,
    payload     jsonb NOT NULL DEFAULT '{}'::jsonb,
    attempts    integer NOT NULL DEFAULT 0,
    last_error  text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    handled_at  timestamptz
);
CREATE UNIQUE INDEX sentinel_events_dedupe ON sentinel_events (coalesce(project_id::text, ''), kind, dedupe_key);
CREATE INDEX sentinel_events_pending ON sentinel_events (id) WHERE handled_at IS NULL;

CREATE TABLE sentinel_routine_state (
    routine         text NOT NULL,
    scope           text NOT NULL DEFAULT '',
    cursor          text NOT NULL DEFAULT '',
    last_run_at     timestamptz,
    last_success_at timestamptz,
    last_error      text NOT NULL DEFAULT '',
    fail_count      integer NOT NULL DEFAULT 0,
    next_run_at     timestamptz,
    running_until   timestamptz,
    runs            integer NOT NULL DEFAULT 0,
    PRIMARY KEY (routine, scope)
);

-- +goose Down
DROP TABLE sentinel_routine_state;
DROP TABLE sentinel_events;
