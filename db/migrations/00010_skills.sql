-- +goose Up
-- T08-02: catálogo de skills. Uma skill vem de cadastro manual, arquivo, pacote
-- ou Git; guarda origem, revisão, hash do conteúdo, manifesto, permissões,
-- recursos e o tipo de executor. Pacotes são DADOS: nada executa na importação.
-- A trilha `skill_imports` registra toda tentativa (inclusive as rejeitadas).

CREATE TABLE skills (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name           text NOT NULL CHECK (name ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    description    text NOT NULL DEFAULT '' CHECK (length(description) <= 500),
    executor       text NOT NULL CHECK (executor IN ('ai', 'script', 'deterministic', 'hybrid')),
    source_kind    text NOT NULL CHECK (source_kind IN ('manual', 'file', 'package', 'git')),
    source_ref     text NOT NULL DEFAULT '' CHECK (length(source_ref) <= 300),
    revision       text NOT NULL DEFAULT '' CHECK (length(revision) <= 80),
    content_hash   text NOT NULL CHECK (length(content_hash) = 64),
    body           text NOT NULL DEFAULT '' CHECK (length(body) <= 60000),
    manifest       jsonb NOT NULL,
    needs_approval boolean NOT NULL DEFAULT false,
    version        integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT skills_name_key UNIQUE (name)
);

CREATE TABLE skill_files (
    skill_id uuid NOT NULL REFERENCES skills (id) ON DELETE CASCADE,
    path     text NOT NULL CHECK (length(path) BETWEEN 1 AND 300),
    sha256   text NOT NULL CHECK (length(sha256) = 64),
    size     integer NOT NULL CHECK (size >= 0 AND size <= 1048576),
    content  bytea NOT NULL,
    PRIMARY KEY (skill_id, path)
);

CREATE TABLE skill_imports (
    id           bigserial PRIMARY KEY,
    skill_id     uuid REFERENCES skills (id) ON DELETE SET NULL,
    name         text NOT NULL DEFAULT '' CHECK (length(name) <= 100),
    source_kind  text NOT NULL,
    source_ref   text NOT NULL DEFAULT '' CHECK (length(source_ref) <= 300),
    outcome      text NOT NULL CHECK (outcome IN ('imported', 'unchanged', 'replaced', 'rejected')),
    reason       text NOT NULL DEFAULT '' CHECK (length(reason) <= 500),
    content_hash text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX skill_imports_skill_idx ON skill_imports (skill_id, id DESC);

-- +goose Down
DROP TABLE skill_imports;
DROP TABLE skill_files;
DROP TABLE skills;
