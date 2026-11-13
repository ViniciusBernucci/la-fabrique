-- +goose Up
-- LF-AJ-09 (AJ09-01/02/04): PRs gerenciadas, eventos observados, rodadas de revisão e prova de incorporação.

-- Uma PR gerenciada por (projeto, número). Os cursores de leitura são por PR (paginação e retomada).
CREATE TABLE pr_watch (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    pr_number       integer NOT NULL CHECK (pr_number > 0),
    feature_id      uuid REFERENCES features (id),
    issue_key       text NOT NULL,
    repo            text NOT NULL DEFAULT '',
    base            text NOT NULL DEFAULT '',
    branch          text NOT NULL DEFAULT '',
    state           text NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'merged', 'closed')),
    draft           boolean NOT NULL DEFAULT true,
    head_sha        text NOT NULL DEFAULT '',
    merge_sha       text NOT NULL DEFAULT '',
    merged_head_sha text NOT NULL DEFAULT '',
    merged_at       timestamptz,
    closed_at       timestamptz,
    review_cursor   bigint NOT NULL DEFAULT 0,
    comment_cursor  bigint NOT NULL DEFAULT 0,
    line_cursor     bigint NOT NULL DEFAULT 0,
    concluded_at    timestamptz,
    last_polled_at  timestamptz,
    last_error      text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, pr_number)
);

-- Rodadas: o conjunto de comentários + o evento de fechamento. round_key = hash estável do conjunto.
CREATE TABLE pr_rounds (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    watch_id       uuid NOT NULL REFERENCES pr_watch (id) ON DELETE CASCADE,
    round_key      text NOT NULL,
    seq            integer NOT NULL,
    closing_kind   text NOT NULL CHECK (closing_kind IN ('review', 'command')),
    closing_id     bigint NOT NULL,
    closed_by      text NOT NULL DEFAULT '',
    head_sha       text NOT NULL DEFAULT '',
    status         text NOT NULL DEFAULT 'pending_issue' CHECK (status IN ('pending_issue', 'issue_created', 'ambiguous')),
    issue_remote_id text NOT NULL DEFAULT '',
    issue_key      text NOT NULL DEFAULT '',
    detail         text NOT NULL DEFAULT '',
    published_head text NOT NULL DEFAULT '',
    incorporated_at timestamptz,
    incorporation_proof jsonb,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (watch_id, round_key),
    UNIQUE (watch_id, seq)
);

-- Eventos e comentários observados. O primeiro snapshot do corpo é o auditável: edição ou reentrega
-- posterior não o altera (ON CONFLICT DO NOTHING); o corpo atual fica em current_body.
CREATE TABLE pr_events (
    id           bigserial PRIMARY KEY,
    watch_id     uuid NOT NULL REFERENCES pr_watch (id) ON DELETE CASCADE,
    kind         text NOT NULL CHECK (kind IN ('review', 'comment', 'line_comment', 'command')),
    remote_id    bigint NOT NULL,
    review_id    bigint,
    author       text NOT NULL DEFAULT '',
    state        text NOT NULL DEFAULT '',
    body         text NOT NULL DEFAULT '',
    current_body text NOT NULL DEFAULT '',
    url          text NOT NULL DEFAULT '',
    authorized   boolean NOT NULL DEFAULT false,
    remote_at    timestamptz NOT NULL,
    round_id     uuid REFERENCES pr_rounds (id),
    recorded_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (watch_id, kind, remote_id)
);
CREATE INDEX pr_events_open_idx ON pr_events (watch_id) WHERE round_id IS NULL;

-- Despachos de correção por rodada (preenchida em AJ09-03): cabeça publicada na PR original.
-- A cabeça publicada de cada correção fica em pr_rounds.published_head.

-- +goose Down
DROP TABLE pr_events;
DROP TABLE pr_rounds;
DROP TABLE pr_watch;
