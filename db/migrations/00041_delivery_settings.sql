-- +goose Up
-- Configuração de entrega do projeto: quem pode pedir correção na PR (logins do GitHub) e como os
-- estados da fábrica se chamam na origem (Jira/ClickUp), para a conclusão pós-merge.

CREATE TABLE delivery_settings (
    project_id     uuid PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    review_authors jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(review_authors) = 'array' AND jsonb_array_length(review_authors) <= 20),
    status_map     jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(status_map) = 'object'),
    version        integer NOT NULL DEFAULT 1,
    updated_at     timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE delivery_settings;
