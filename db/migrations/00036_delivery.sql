-- +goose Up
-- T15-02: entrega da feature. Os commits por tarefa ficam registrados (para consolidar o
-- histórico da feature) e cada tentativa de entrega guarda o resultado dos gates, o
-- documento vivo e a Draft PR (idempotente: uma por feature e tentativa).

CREATE TABLE feature_commits (
    id         bigserial PRIMARY KEY,
    project_id uuid NOT NULL,
    feature_id uuid NOT NULL,
    task_key   text NOT NULL,
    sha        text NOT NULL CHECK (sha ~ '^[0-9a-f]{40}$'),
    branch     text NOT NULL,
    subject    text NOT NULL DEFAULT '' CHECK (length(subject) <= 300),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT feature_commits_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT feature_commits_key UNIQUE (feature_id, sha)
);

CREATE TABLE deliveries (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL,
    feature_id  uuid NOT NULL,
    attempt     integer NOT NULL CHECK (attempt >= 1),
    branch      text NOT NULL,
    revision    text NOT NULL,
    state       text NOT NULL CHECK (state IN ('blocked', 'draft_open')),
    reasons     jsonb NOT NULL DEFAULT '[]'::jsonb,
    doc_path    text NOT NULL DEFAULT '',
    doc_hash    text NOT NULL DEFAULT '',
    pr_number   integer,
    pr_url      text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT deliveries_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT deliveries_attempt_key UNIQUE (feature_id, attempt)
);

-- +goose Down
DROP TABLE deliveries;
DROP TABLE feature_commits;
