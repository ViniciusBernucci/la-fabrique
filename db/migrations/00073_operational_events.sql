-- +goose Up
-- T16-05: eventos operacionais da fábrica (jobs e rotinas de fundo do worker), fora do escopo por
-- projeto já coberto por execution_events. Aditiva.
CREATE TABLE operational_events (
    id         bigserial PRIMARY KEY,
    level      text NOT NULL CHECK (level IN ('info', 'warn', 'error')),
    source     text NOT NULL CHECK (source IN ('sentinela', 'admissao', 'retencao', 'sincronizacao', 'clilogin', 'fabrica')),
    message    text NOT NULL,
    payload    jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX operational_events_created_at ON operational_events (created_at);
CREATE INDEX operational_events_source ON operational_events (source);

-- A retenção (T16-04) também expira operational_events, reaproveitando retention_runs/retention_decisions
-- como resumo de auditoria permanente (sem tabela de resumo por item, diferente de execution_events).
ALTER TABLE retention_runs ADD COLUMN operational_events_deleted integer NOT NULL DEFAULT 0;
ALTER TABLE retention_decisions DROP CONSTRAINT retention_decisions_kind_check;
ALTER TABLE retention_decisions ADD CONSTRAINT retention_decisions_kind_check CHECK (kind IN ('logs', 'checkpoint', 'operational_events'));

-- +goose Down
ALTER TABLE retention_decisions DROP CONSTRAINT retention_decisions_kind_check;
ALTER TABLE retention_decisions ADD CONSTRAINT retention_decisions_kind_check CHECK (kind IN ('logs', 'checkpoint'));
ALTER TABLE retention_runs DROP COLUMN operational_events_deleted;
DROP TABLE operational_events;
