-- +goose Up
-- T14-02: política de risco por projeto (pisos determinísticos por caminho e limiares
-- de tamanho) e histórico das classificações (planejamento e reavaliação do diff), cada
-- uma com a justificativa do nível final.

CREATE TABLE risk_policies (
    project_id uuid PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    policy     jsonb NOT NULL,
    version    integer NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE risk_assessments (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    seq        bigserial NOT NULL,
    project_id uuid NOT NULL,
    feature_id uuid NOT NULL,
    task_key   text NOT NULL,
    phase      text NOT NULL CHECK (phase IN ('planning', 'diff', 'fix')),
    revision   text NOT NULL DEFAULT '' CHECK (length(revision) <= 100),
    suggested  text NOT NULL CHECK (suggested IN ('low', 'medium', 'high')),
    final      text NOT NULL CHECK (final IN ('low', 'medium', 'high')),
    raised     boolean NOT NULL,
    reasons    jsonb NOT NULL,
    files      jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT risk_assessments_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT
);
CREATE INDEX risk_assessments_task_idx ON risk_assessments (feature_id, task_key, seq DESC);

-- +goose Down
DROP TABLE risk_assessments;
DROP TABLE risk_policies;
