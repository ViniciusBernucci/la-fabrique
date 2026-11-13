-- +goose Up
-- T08-04: servidores MCP, ferramentas descobertas, concessões por agente e trilha
-- de chamadas. Ferramenta descoberta nasce SEM concessão; a concessão fixa o hash
-- do esquema aprovado; chamadas são auditadas sem segredos.

CREATE TABLE mcp_servers (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name            text NOT NULL CHECK (name ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    url             text NOT NULL CHECK (length(url) BETWEEN 8 AND 300),
    connection_id   uuid REFERENCES connections (id) ON DELETE RESTRICT,
    protocol        text NOT NULL DEFAULT '',
    server_name     text NOT NULL DEFAULT '',
    server_version  text NOT NULL DEFAULT '',
    discovered_at   timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT mcp_servers_name_key UNIQUE (name)
);

CREATE TABLE mcp_tools (
    server_id    uuid NOT NULL REFERENCES mcp_servers (id) ON DELETE CASCADE,
    name         text NOT NULL,
    description  text NOT NULL DEFAULT '',
    input_schema jsonb NOT NULL,
    schema_hash  text NOT NULL CHECK (length(schema_hash) = 64),
    status       text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'removed')),
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (server_id, name)
);

CREATE TABLE mcp_grants (
    agent_id    uuid NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
    server_id   uuid NOT NULL,
    tool_name   text NOT NULL,
    schema_hash text NOT NULL CHECK (length(schema_hash) = 64),
    granted_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id, server_id, tool_name),
    FOREIGN KEY (server_id, tool_name) REFERENCES mcp_tools (server_id, name) ON DELETE CASCADE
);

CREATE TABLE mcp_calls (
    id            bigserial PRIMARY KEY,
    agent_id      uuid REFERENCES agents (id) ON DELETE SET NULL,
    server_id     uuid REFERENCES mcp_servers (id) ON DELETE SET NULL,
    tool_name     text NOT NULL,
    args_hash     text NOT NULL DEFAULT '',
    args_redacted jsonb NOT NULL DEFAULT '{}'::jsonb,
    outcome       text NOT NULL CHECK (outcome IN ('ok', 'tool_error', 'denied', 'changed', 'invalid', 'blocked', 'failed')),
    reason        text NOT NULL DEFAULT '' CHECK (length(reason) <= 300),
    duration_ms   integer NOT NULL DEFAULT 0,
    result_bytes  integer NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX mcp_calls_agent_idx ON mcp_calls (agent_id, id DESC);

-- +goose Down
DROP TABLE mcp_calls;
DROP TABLE mcp_grants;
DROP TABLE mcp_tools;
DROP TABLE mcp_servers;
