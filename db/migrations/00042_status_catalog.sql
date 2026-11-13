-- +goose Up
-- LF-AJ-02: catálogo de status com IDs estáveis, tabela de transições e mapa por projeto
-- (Jira/ClickUp). Aditiva: nenhuma tabela existente é alterada.

CREATE TABLE status_catalog (
    id      text PRIMARY KEY,
    code    text NOT NULL UNIQUE,
    label   text NOT NULL,
    ordinal integer NOT NULL
);

CREATE TABLE status_aliases (
    alias     text PRIMARY KEY,
    status_id text NOT NULL REFERENCES status_catalog (id)
);

CREATE TABLE status_transitions (
    from_id text NOT NULL REFERENCES status_catalog (id),
    to_id   text NOT NULL REFERENCES status_catalog (id),
    PRIMARY KEY (from_id, to_id)
);

CREATE TABLE project_status_maps (
    project_id uuid PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    source     text NOT NULL CHECK (source IN ('jira', 'clickup')),
    remote     jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(remote) = 'object'),
    moves      jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(moves) = 'object'),
    version    integer NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE project_status_maps;
DROP TABLE status_transitions;
DROP TABLE status_aliases;
DROP TABLE status_catalog;
