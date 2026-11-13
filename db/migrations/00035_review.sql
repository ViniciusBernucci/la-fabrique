-- +goose Up
-- T14-03: revisão independente e correções. Cada rodada é uma revisão de UMA revisão do
-- código (hash/SHA) por um revisor diferente de quem implementou; os achados têm severidade
-- (crítico e relevante bloqueiam, baixo fica registrado); a correção só segue com os checks
-- aprovados e abre a rodada seguinte; o limite de rodadas (padrão 2) esgota e suspende.

CREATE TABLE review_configs (
    project_id    uuid PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    reviewer      text NOT NULL CHECK (length(reviewer) BETWEEN 1 AND 100),
    fallback      jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(fallback) = 'array'),
    max_rounds    integer NOT NULL DEFAULT 2 CHECK (max_rounds BETWEEN 1 AND 10),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE reviews (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    seq            bigserial NOT NULL,
    project_id     uuid NOT NULL,
    feature_id     uuid NOT NULL,
    task_key       text NOT NULL,
    round          integer NOT NULL CHECK (round >= 1),
    revision       text NOT NULL CHECK (length(revision) BETWEEN 1 AND 100),
    implementer    text NOT NULL CHECK (length(implementer) BETWEEN 1 AND 100),
    reviewer       text NOT NULL CHECK (length(reviewer) BETWEEN 1 AND 100),
    fixer          text NOT NULL DEFAULT '',
    status         text NOT NULL CHECK (status IN ('reviewing', 'changes_requested', 'fixing', 'approved', 'exhausted')),
    fix_revision   text NOT NULL DEFAULT '',
    checkpoint_id  uuid REFERENCES checkpoints (id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    finished_at    timestamptz,
    CONSTRAINT reviews_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT reviews_independent_ck CHECK (reviewer <> implementer OR round > 1)
);
CREATE INDEX reviews_task_idx ON reviews (feature_id, task_key, seq DESC);
-- Uma revisão em andamento por tarefa.
CREATE UNIQUE INDEX reviews_active_key ON reviews (feature_id, task_key) WHERE status IN ('reviewing', 'changes_requested', 'fixing');

CREATE TABLE review_findings (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id    uuid NOT NULL REFERENCES reviews (id) ON DELETE CASCADE,
    severity     text NOT NULL CHECK (severity IN ('critical', 'relevant', 'low')),
    title        text NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    body         text NOT NULL DEFAULT '' CHECK (length(body) <= 4000),
    file         text NOT NULL DEFAULT '' CHECK (length(file) <= 300),
    line         integer CHECK (line IS NULL OR line >= 1),
    state        text NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'fixed', 'noted')),
    fix_revision text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX review_findings_review_idx ON review_findings (review_id, created_at);

-- +goose Down
DROP TABLE review_findings;
DROP TABLE reviews;
DROP TABLE review_configs;
