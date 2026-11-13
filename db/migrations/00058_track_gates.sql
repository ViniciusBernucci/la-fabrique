-- +goose Up
-- LF-AJ-08 (AJ08-05): resultados por trilha e testes integrados ligados à revisão do código.
CREATE TABLE track_results (
    feature_id  uuid NOT NULL,
    track       text NOT NULL CHECK (track IN ('FRONTEND', 'BACKEND')),
    revision    text NOT NULL,
    requirement jsonb NOT NULL,
    evidence    jsonb NOT NULL,
    artifacts   jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (feature_id, track, revision)
);

CREATE TABLE integrated_results (
    feature_id uuid NOT NULL,
    head_sha   text NOT NULL,
    tests      jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (feature_id, head_sha)
);

-- +goose Down
DROP TABLE integrated_results;
DROP TABLE track_results;
