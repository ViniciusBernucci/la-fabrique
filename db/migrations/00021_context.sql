-- +goose Up
-- T12-02: contexto seletivo e renovação de sessão. O pacote de contexto guarda o
-- texto já sem segredos e o manifesto de fontes (hashes, limites, omissões), cada
-- um com o seu hash. A sessão liga o pacote a um checkpoint íntegro da execução.

CREATE TABLE context_packs (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       uuid NOT NULL,
    execution_id     uuid NOT NULL,
    seq              integer NOT NULL CHECK (seq >= 1),
    manifest         jsonb NOT NULL,
    manifest_hash    text NOT NULL CHECK (manifest_hash ~ '^[0-9a-f]{64}$'),
    content          text NOT NULL,
    content_hash     text NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    estimated_tokens bigint NOT NULL CHECK (estimated_tokens >= 0),
    limit_tokens     bigint NOT NULL CHECK (limit_tokens > 0),
    created_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT context_packs_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT context_packs_seq_key UNIQUE (execution_id, seq),
    CONSTRAINT context_packs_id_exec_key UNIQUE (id, execution_id)
);

CREATE TABLE agent_sessions (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     uuid NOT NULL,
    execution_id   uuid NOT NULL,
    seq            integer NOT NULL CHECK (seq >= 1),
    milestone      text NOT NULL CHECK (milestone IN ('planning','implementation','tests','review','suspension','session_change','saturation')),
    checkpoint_id  uuid NOT NULL REFERENCES checkpoints (id) ON DELETE RESTRICT,
    pack_id        uuid NOT NULL,
    model          text NOT NULL DEFAULT '' CHECK (length(model) <= 200),
    previous_model text NOT NULL DEFAULT '' CHECK (length(previous_model) <= 200),
    usage_method   text NOT NULL CHECK (usage_method IN ('official','estimate','none')),
    usage_tokens   bigint NOT NULL DEFAULT 0 CHECK (usage_tokens >= 0),
    window_tokens  bigint NOT NULL DEFAULT 0 CHECK (window_tokens >= 0),
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT agent_sessions_execution_fk FOREIGN KEY (project_id, execution_id)
        REFERENCES executions (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT agent_sessions_pack_fk FOREIGN KEY (pack_id, execution_id)
        REFERENCES context_packs (id, execution_id) ON DELETE RESTRICT,
    CONSTRAINT agent_sessions_seq_key UNIQUE (execution_id, seq)
);

-- +goose Down
DROP TABLE agent_sessions;
DROP TABLE context_packs;
