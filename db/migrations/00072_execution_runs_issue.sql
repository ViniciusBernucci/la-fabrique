-- +goose Up
-- Execução iniciada a partir de uma Issue da origem (Jira/ClickUp): guarda a chave para a ponte com a Sentinela
-- (transições de status com evidência) e com o ciclo da PR.
ALTER TABLE execution_runs ADD COLUMN issue_key text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE execution_runs DROP COLUMN issue_key;
