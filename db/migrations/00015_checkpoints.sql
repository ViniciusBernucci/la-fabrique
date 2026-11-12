-- +goose Up
-- T12-01: checkpoints íntegros. O conteúdo do workspace fica em objetos
-- endereçados por hash (por projeto, sem compartilhar entre projetos); o
-- checkpoint guarda o manifesto, o estado dos nós e as referências, com o hash
-- de cada um. O checkpoint e seus objetos entram na MESMA transação: ou o novo
-- existe por inteiro, ou nada muda e o anterior continua válido.

CREATE TABLE checkpoint_objects (
    project_id uuid NOT NULL REFERENCES projects (id) ON DELETE RESTRICT,
    hash       text NOT NULL CHECK (hash ~ '^[0-9a-f]{64}$'),
    size       integer NOT NULL CHECK (size >= 0),
    data       bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, hash),
    CONSTRAINT checkpoint_objects_size_ck CHECK (octet_length(data) = size)
);

CREATE TABLE checkpoints (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     uuid NOT NULL,
    execution_id   uuid NOT NULL,
    seq            integer NOT NULL CHECK (seq >= 1),
    label          text NOT NULL DEFAULT '' CHECK (length(label) <= 120),
    manifest       jsonb NOT NULL,
    manifest_hash  text NOT NULL CHECK (manifest_hash ~ '^[0-9a-f]{64}$'),
    tree_hash      text NOT NULL CHECK (tree_hash ~ '^[0-9a-f]{64}$'),
    state          jsonb NOT NULL DEFAULT '{}'::jsonb,
    state_hash     text NOT NULL CHECK (state_hash ~ '^[0-9a-f]{64}$'),
    refs           jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(refs) = 'object'),
    total_bytes    bigint NOT NULL CHECK (total_bytes >= 0),
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT checkpoints_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT checkpoints_seq_key UNIQUE (execution_id, seq)
);
CREATE INDEX checkpoints_execution_idx ON checkpoints (execution_id, seq DESC);

-- +goose Down
DROP TABLE checkpoints;
DROP TABLE checkpoint_objects;
