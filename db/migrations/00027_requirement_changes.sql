-- +goose Up
-- T06-05: mudança de requisitos e troca de plataforma. Cada revisão do requisito
-- de uma feature é gravada (histórico nunca é apagado) com o impacto classificado
-- e o destino (aplicada, adiada para um marco seguro, precisa de replanejamento ou
-- em conflito com uma decisão). A troca de origem guarda o mapeamento de
-- identidades e preserva os vínculos antigos.

CREATE TABLE requirement_revisions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL,
    feature_id   uuid NOT NULL,
    seq          integer NOT NULL CHECK (seq >= 1),
    content      jsonb NOT NULL,
    content_hash text NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    impact       text NOT NULL CHECK (impact IN ('initial', 'metadata', 'content')),
    action       text NOT NULL CHECK (action IN ('record', 'update', 'defer', 'replan', 'decision')),
    state        text NOT NULL CHECK (state IN ('applied', 'deferred', 'needs_replan', 'conflict', 'rejected', 'superseded')),
    diff         jsonb NOT NULL DEFAULT '[]'::jsonb,
    decision_id  uuid,
    created_at   timestamptz NOT NULL DEFAULT now(),
    resolved_at  timestamptz,
    CONSTRAINT requirement_revisions_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT requirement_revisions_seq_key UNIQUE (feature_id, seq)
);
-- No máximo uma revisão vigente (aplicada) por feature.
CREATE UNIQUE INDEX requirement_revisions_applied_idx ON requirement_revisions (feature_id) WHERE state = 'applied';

CREATE TABLE source_switches (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    uuid NOT NULL REFERENCES projects (id) ON DELETE RESTRICT,
    from_source   jsonb NOT NULL,
    to_source     jsonb NOT NULL,
    mapping       jsonb NOT NULL,
    state         text NOT NULL CHECK (state IN ('validated', 'rejected', 'activated')),
    problems      jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now(),
    activated_at  timestamptz
);
CREATE INDEX source_switches_project_idx ON source_switches (project_id, created_at DESC);

-- Vínculos históricos: a identidade remota antiga de cada feature é conservada.
CREATE TABLE feature_identity_history (
    id               bigserial PRIMARY KEY,
    project_id       uuid NOT NULL,
    feature_id       uuid NOT NULL,
    source_kind      text NOT NULL,
    remote_id        text NOT NULL,
    superseded_by    text NOT NULL,
    switch_id        uuid NOT NULL REFERENCES source_switches (id) ON DELETE RESTRICT,
    created_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT feature_identity_history_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT
);
CREATE INDEX feature_identity_history_idx ON feature_identity_history (feature_id, id);

-- +goose Down
DROP TABLE feature_identity_history;
DROP TABLE source_switches;
DROP TABLE requirement_revisions;
