-- +goose Up
-- LF-AJ-04 (AJ04-01): snapshots imutáveis de contrato de execução, versionados por escopo
-- (Issue inicial ou subtarefa) e encadeados pelo hash do anterior.

CREATE TABLE execution_contracts (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    scope_key   text NOT NULL,
    version     integer NOT NULL CHECK (version >= 1),
    hash        text NOT NULL CHECK (hash ~ '^[0-9a-f]{64}$'),
    parent_hash text NOT NULL DEFAULT '',
    body        jsonb NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, scope_key, version)
);

-- +goose StatementBegin
CREATE FUNCTION execution_contracts_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'execution_contracts é imutável' USING ERRCODE = '23000';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER execution_contracts_no_change BEFORE UPDATE OR DELETE ON execution_contracts
    FOR EACH ROW EXECUTE FUNCTION execution_contracts_immutable();

-- +goose Down
DROP TRIGGER execution_contracts_no_change ON execution_contracts;
DROP FUNCTION execution_contracts_immutable();
DROP TABLE execution_contracts;
