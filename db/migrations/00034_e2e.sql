-- +goose Up
-- T14-04: evidências de testes E2E e justificativas de inaplicabilidade, sempre presas à
-- REVISÃO testada (uma alteração depois exige novas evidências). A execução em si fica em
-- check_runs (T14-01); aqui ficam a jornada, as fixtures, o navegador isolado e as capturas.

CREATE TABLE e2e_evidence (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL,
    feature_id   uuid NOT NULL,
    task_key     text NOT NULL,
    revision     text NOT NULL CHECK (length(revision) BETWEEN 1 AND 100),
    journey      text NOT NULL CHECK (journey ~ '^[a-z0-9][a-z0-9_-]{0,59}$'),
    check_run_id uuid NOT NULL REFERENCES check_runs (id) ON DELETE RESTRICT,
    status       text NOT NULL CHECK (status IN ('pass', 'fail', 'error')),
    browser      text NOT NULL DEFAULT '' CHECK (length(browser) <= 100),
    isolated     boolean NOT NULL,
    fixtures     jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT e2e_evidence_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT
);
CREATE INDEX e2e_evidence_lookup_idx ON e2e_evidence (feature_id, task_key, revision, journey, created_at DESC);

CREATE TABLE e2e_captures (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    evidence_id uuid NOT NULL REFERENCES e2e_evidence (id) ON DELETE CASCADE,
    path        text NOT NULL CHECK (length(path) BETWEEN 1 AND 300),
    sha256      text NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    size        integer NOT NULL CHECK (size >= 0),
    content     bytea NOT NULL,
    CONSTRAINT e2e_captures_size_ck CHECK (octet_length(content) = size)
);

CREATE TABLE e2e_waivers (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL,
    feature_id   uuid NOT NULL,
    task_key     text NOT NULL,
    revision     text NOT NULL CHECK (length(revision) BETWEEN 1 AND 100),
    subject      text NOT NULL CHECK (subject = 'tests' OR subject ~ '^e2e:[a-z0-9][a-z0-9_-]{0,59}$'),
    reason       text NOT NULL CHECK (length(reason) BETWEEN 20 AND 1000),
    alt_kind     text NOT NULL CHECK (alt_kind IN ('check_run', 'capture', 'review')),
    alt_ref      text NOT NULL CHECK (length(alt_ref) BETWEEN 1 AND 1000),
    admin_id     uuid,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT e2e_waivers_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT
);
CREATE INDEX e2e_waivers_lookup_idx ON e2e_waivers (feature_id, task_key, revision, subject);

-- +goose Down
DROP TABLE e2e_waivers;
DROP TABLE e2e_captures;
DROP TABLE e2e_evidence;
