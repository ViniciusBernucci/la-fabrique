-- +goose Up
-- T14-01: checks no baseline e na revisão candidata. Cada execução fica gravada com
-- a revisão testada, o resultado, a assinatura da falha e um trecho da saída (já
-- sem segredos). A classificação compara candidata e baseline; a falha anterior
-- COMPROVADA (assinatura igual no baseline) é registrada em known_failures; a
-- origem desconhecida fica em investigação e nunca vira baseline por suposição.

CREATE TABLE check_runs (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    seq             bigserial NOT NULL,
    project_id      uuid NOT NULL,
    feature_id      uuid NOT NULL,
    role            text NOT NULL CHECK (role IN ('baseline', 'candidate')),
    revision        text NOT NULL CHECK (length(revision) BETWEEN 1 AND 100),
    check_name      text NOT NULL CHECK (length(check_name) BETWEEN 1 AND 60),
    command         text NOT NULL CHECK (length(command) <= 500),
    status          text NOT NULL CHECK (status IN ('pass', 'fail', 'error', 'timeout', 'not_viable')),
    exit_code       integer,
    signature       text NOT NULL DEFAULT '',
    output_excerpt  text NOT NULL DEFAULT '',
    output_hash     text NOT NULL DEFAULT '',
    truncated       boolean NOT NULL DEFAULT false,
    reason          text NOT NULL DEFAULT '' CHECK (length(reason) <= 500),
    duration_ms     integer NOT NULL DEFAULT 0,
    classification  text CHECK (classification IN ('pass', 'regression', 'preexisting', 'unknown', 'inconclusive')),
    baseline_run_id uuid REFERENCES check_runs (id) ON DELETE SET NULL,
    resolution      text NOT NULL DEFAULT '' CHECK (length(resolution) <= 1000),
    started_at      timestamptz NOT NULL,
    finished_at     timestamptz NOT NULL,
    CONSTRAINT check_runs_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT
);
CREATE INDEX check_runs_lookup_idx ON check_runs (feature_id, revision, role, check_name, seq DESC);

CREATE TABLE known_failures (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE RESTRICT,
    check_name  text NOT NULL,
    signature   text NOT NULL,
    run_id      uuid NOT NULL REFERENCES check_runs (id) ON DELETE RESTRICT, -- a execução de baseline que a comprova
    revision    text NOT NULL,
    note        text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT known_failures_key UNIQUE (project_id, check_name, signature)
);

-- +goose Down
DROP TABLE known_failures;
DROP TABLE check_runs;
