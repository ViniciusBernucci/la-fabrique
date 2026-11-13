-- +goose Up
-- T16-04: retenção com integridade. O resumo permanente de cada execução (contagens, tipos,
-- primeiro/último horário e o último seq) é gravado ANTES de os logs detalhados serem
-- removidos, e o último seq nunca é reaproveitado. Checkpoints protegidos (bloqueio explícito
-- ou vínculo a incidente) não são removidos. Toda limpeza é auditada, com a decisão por item.

CREATE TABLE execution_summaries (
    execution_id      uuid PRIMARY KEY,
    project_id        uuid NOT NULL,
    last_seq          bigint NOT NULL DEFAULT 0 CHECK (last_seq >= 0),
    compacted_through bigint NOT NULL DEFAULT 0 CHECK (compacted_through >= 0),
    compacted_total   integer NOT NULL DEFAULT 0 CHECK (compacted_total >= 0),
    compacted_by_type jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(compacted_by_type) = 'object'),
    first_at          timestamptz,
    last_at           timestamptz,
    compacted_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT execution_summaries_execution_fk FOREIGN KEY (project_id, execution_id) REFERENCES executions (project_id, id) ON DELETE RESTRICT
);

CREATE TABLE checkpoint_protections (
    checkpoint_id uuid PRIMARY KEY REFERENCES checkpoints (id) ON DELETE CASCADE,
    reason        text NOT NULL CHECK (length(btrim(reason)) BETWEEN 1 AND 500),
    incident_ref  text NOT NULL DEFAULT '' CHECK (length(incident_ref) <= 200),
    created_by    uuid,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE retention_runs (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    dry_run             boolean NOT NULL,
    now_ref             timestamptz NOT NULL,
    log_days            integer NOT NULL,
    grace_days          integer NOT NULL,
    logs_deleted        integer NOT NULL DEFAULT 0,
    executions_summarized integer NOT NULL DEFAULT 0,
    checkpoints_deleted integer NOT NULL DEFAULT 0,
    checkpoints_kept    integer NOT NULL DEFAULT 0,
    objects_deleted     integer NOT NULL DEFAULT 0,
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE retention_decisions (
    id         bigserial PRIMARY KEY,
    run_id     uuid NOT NULL REFERENCES retention_runs (id) ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('logs', 'checkpoint')),
    ref        text NOT NULL,
    action     text NOT NULL CHECK (action IN ('deleted', 'would_delete', 'kept')),
    reason     text NOT NULL DEFAULT '' CHECK (length(reason) <= 500),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX retention_decisions_run_idx ON retention_decisions (run_id, id);

-- +goose Down
DROP TABLE retention_decisions;
DROP TABLE retention_runs;
DROP TABLE checkpoint_protections;
DROP TABLE execution_summaries;
