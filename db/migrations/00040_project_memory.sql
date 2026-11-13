-- +goose Up
-- T15-05: memória técnica do projeto. Registros nascem PENDENTES, presos à branch da feature, com
-- evidência vinculada e chave semântica normalizada (deduplicação). Só o merge confirmado promove
-- a OFICIAL, e só o que vale para a revisão integrada; o resto é invalidado com o motivo.

CREATE TABLE memory_records (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id        uuid NOT NULL,
    feature_id        uuid NOT NULL,
    branch            text NOT NULL CHECK (length(branch) BETWEEN 1 AND 200),
    pr_url            text NOT NULL DEFAULT '' CHECK (length(pr_url) <= 300),
    key               text NOT NULL CHECK (length(key) BETWEEN 1 AND 120),
    kind              text NOT NULL CHECK (kind IN ('fact', 'decision', 'constraint', 'lesson')),
    content           text NOT NULL CHECK (length(content) BETWEEN 1 AND 2000),
    content_hash      text NOT NULL,
    source            text NOT NULL CHECK (length(source) BETWEEN 1 AND 200),
    evidence          jsonb NOT NULL CHECK (jsonb_typeof(evidence) = 'array' AND jsonb_array_length(evidence) >= 1),
    revision          text NOT NULL CHECK (length(revision) BETWEEN 1 AND 100),
    confidence        text NOT NULL DEFAULT 'medium' CHECK (confidence IN ('low', 'medium', 'high')),
    state             text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'official', 'invalidated')),
    promoted_revision text NOT NULL DEFAULT '',
    promoted_at       timestamptz,
    invalid_reason    text NOT NULL DEFAULT '' CHECK (length(invalid_reason) <= 500),
    version           integer NOT NULL DEFAULT 1,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT memory_records_feature_fk FOREIGN KEY (project_id, feature_id) REFERENCES features (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT memory_official_ck CHECK (state <> 'official' OR promoted_revision <> ''),
    CONSTRAINT memory_invalid_ck CHECK (state <> 'invalidated' OR invalid_reason <> '')
);
CREATE UNIQUE INDEX memory_pending_key ON memory_records (project_id, branch, key) WHERE state = 'pending';
CREATE UNIQUE INDEX memory_official_key ON memory_records (project_id, key) WHERE state = 'official';
CREATE INDEX memory_project_idx ON memory_records (project_id, state);

-- +goose Down
DROP TABLE memory_records;
