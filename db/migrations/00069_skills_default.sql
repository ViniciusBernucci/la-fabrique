-- +goose Up
-- ADR-0066: skills padrão da fábrica. Vêm embutidas no binário e são instaladas no catálogo ao subir a API;
-- toda skill marcada como padrão é associada a todos os projetos (os existentes, uma única vez por skill,
-- e os novos, na criação). A marca não impede editar nem substituir a skill pela tela.
ALTER TABLE skills ADD COLUMN is_default boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE skills DROP COLUMN is_default;
