-- +goose Up
-- ADR-0064: o nome de uma conexão só precisa ser único entre as ATIVAS. Revogar libera o nome para
-- uma conexão nova (que substitui a antiga na prática); a revogada fica no histórico até ser apagada.
ALTER TABLE connections DROP CONSTRAINT connections_name_key;
CREATE UNIQUE INDEX connections_name_active_key ON connections (name) WHERE state = 'active';

-- +goose Down
DROP INDEX connections_name_active_key;
ALTER TABLE connections ADD CONSTRAINT connections_name_key UNIQUE (name);
