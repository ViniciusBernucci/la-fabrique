-- +goose Up
-- T07-03: versões publicadas de workflow. Uma versão é IMUTÁVEL (gatilho): guarda
-- o grafo, o grafo expandido (subfluxos fixados por versão), o snapshot sanitizado
-- da configuração efetiva dos nós e o hash do conteúdo. Rascunho não executa:
-- uma execução só nasce de uma versão publicada e fica presa a ela.

CREATE TABLE workflow_versions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id   uuid NOT NULL REFERENCES workflows (id) ON DELETE RESTRICT,
    number        integer NOT NULL CHECK (number >= 1),
    draft_version integer NOT NULL CHECK (draft_version >= 1),
    name          text NOT NULL,
    kind          text NOT NULL CHECK (kind IN ('feature', 'subflow')),
    description   text NOT NULL DEFAULT '',
    graph         jsonb NOT NULL,
    flat_graph    jsonb NOT NULL,
    subflows      jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(subflows) = 'object'),
    effective     jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(effective) = 'array'),
    content_hash  text NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    published_by  uuid REFERENCES admins (id) ON DELETE SET NULL,
    published_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT workflow_versions_number_key UNIQUE (workflow_id, number)
);
CREATE INDEX workflow_versions_workflow_idx ON workflow_versions (workflow_id, number DESC);

-- Uma versão publicada nunca muda nem some (só o ponteiro do workflow se move).
-- +goose StatementBegin
CREATE FUNCTION workflow_versions_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'workflow_versions é imutável' USING ERRCODE = '23000';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER workflow_versions_no_update BEFORE UPDATE OR DELETE ON workflow_versions
    FOR EACH ROW EXECUTE FUNCTION workflow_versions_immutable();

ALTER TABLE workflows ADD COLUMN published_version_id uuid REFERENCES workflow_versions (id) ON DELETE RESTRICT;
ALTER TABLE executions ADD COLUMN workflow_version_id uuid REFERENCES workflow_versions (id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE executions DROP COLUMN workflow_version_id;
ALTER TABLE workflows DROP COLUMN published_version_id;
DROP TRIGGER workflow_versions_no_update ON workflow_versions;
DROP FUNCTION workflow_versions_immutable();
DROP TABLE workflow_versions;
