-- +goose Up
-- T06-04: sincronização manual e preflight. O último estado conhecido de cada
-- item da origem fica gravado (identidade remota, versão, hash do conteúdo
-- normalizado) para que uma mudança remota seja detectada UMA vez, com o que
-- mudou, quando e de onde. Edição humana local (prioridade) nunca é sobrescrita
-- em silêncio: vira conflito aberto.

CREATE TABLE source_items (
    project_id        uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    remote_id         text NOT NULL CHECK (length(remote_id) BETWEEN 1 AND 200),
    item_key          text NOT NULL DEFAULT '',
    title             text NOT NULL DEFAULT '',
    description       text NOT NULL DEFAULT '',
    status            text NOT NULL DEFAULT '',
    category          text NOT NULL DEFAULT '',
    priority          text NOT NULL DEFAULT '',
    mapped_priority   integer CHECK (mapped_priority BETWEEN 0 AND 4),
    labels            jsonb NOT NULL DEFAULT '[]'::jsonb,
    dependencies      jsonb NOT NULL DEFAULT '[]'::jsonb,
    is_subtask        boolean NOT NULL DEFAULT false,
    eligible          boolean NOT NULL DEFAULT false,
    version           text NOT NULL DEFAULT '',
    content_hash      text NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    url               text NOT NULL DEFAULT '',
    state             text NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'gone')),
    -- Edição humana local: {"mapped_priority": 0..4}. Prevalece sobre a origem.
    local_overrides   jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(local_overrides) = 'object'),
    conflict          jsonb,
    first_seen_at     timestamptz NOT NULL,
    last_seen_at      timestamptz NOT NULL,
    last_changed_at   timestamptz NOT NULL,
    PRIMARY KEY (project_id, remote_id)
);

CREATE TABLE source_syncs (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    seq            bigserial NOT NULL,
    project_id     uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    trigger        text NOT NULL CHECK (trigger IN ('manual', 'preflight')),
    source_kind    text NOT NULL DEFAULT '',
    container      text NOT NULL DEFAULT '',
    outcome        text NOT NULL CHECK (outcome IN ('ok', 'failed')),
    error          text NOT NULL DEFAULT '' CHECK (length(error) <= 500),
    read_count     integer NOT NULL DEFAULT 0,
    new_count      integer NOT NULL DEFAULT 0,
    changed_count  integer NOT NULL DEFAULT 0,
    gone_count     integer NOT NULL DEFAULT 0,
    conflict_count integer NOT NULL DEFAULT 0,
    snapshot_hash  text NOT NULL DEFAULT '',
    started_at     timestamptz NOT NULL,
    finished_at    timestamptz NOT NULL
);
CREATE INDEX source_syncs_project_idx ON source_syncs (project_id, seq DESC);

CREATE TABLE source_changes (
    id             bigserial PRIMARY KEY,
    sync_id        uuid NOT NULL REFERENCES source_syncs (id) ON DELETE CASCADE,
    project_id     uuid NOT NULL,
    remote_id      text NOT NULL,
    item_key       text NOT NULL DEFAULT '',
    kind           text NOT NULL CHECK (kind IN ('new', 'changed', 'gone', 'reappeared', 'conflict')),
    fields         jsonb NOT NULL DEFAULT '{}'::jsonb,
    before_version text NOT NULL DEFAULT '',
    after_version  text NOT NULL DEFAULT '',
    detail         text NOT NULL DEFAULT '' CHECK (length(detail) <= 500),
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX source_changes_sync_idx ON source_changes (sync_id, id);

-- +goose Down
DROP TABLE source_changes;
DROP TABLE source_syncs;
DROP TABLE source_items;
