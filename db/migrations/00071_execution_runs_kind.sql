-- +goose Up
-- Tipo da execução: "manual" (pedida pela pessoa) ou "test" (o botão Testar o fluxo: feature fictícia padronizada,
-- checks próprios e envio ao GitHub apenas simulado).
ALTER TABLE execution_runs ADD COLUMN kind text NOT NULL DEFAULT 'manual' CHECK (kind IN ('manual', 'test'));

-- +goose Down
ALTER TABLE execution_runs DROP COLUMN kind;
