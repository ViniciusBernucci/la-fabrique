-- +goose Up
-- LF-AJ-03 (AJ03-05): qual despachante admite trabalho no projeto (legado ou Sentinela), a prévia da migração
-- e a reversão operacional. Aditiva: nada do legado é alterado ou apagado.
CREATE TABLE flow_activation (
    project_id   uuid PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    state        text NOT NULL DEFAULT 'legacy' CHECK (state IN ('legacy', 'active', 'draining', 'rolled_back')),
    dispatcher   text NOT NULL DEFAULT 'legacy' CHECK (dispatcher IN ('legacy', 'sentinel', 'none')),
    workflow_id  uuid,
    preview      jsonb,
    activated_at timestamptz,
    rollback_at  timestamptz,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE flow_activation;
