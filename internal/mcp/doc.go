// Package mcp implementa o cliente MCP (Streamable HTTP), o catálogo de servidores
// e ferramentas descobertas e a allowlist por agente (T08-04). Regras: ferramenta
// descoberta nasce SEM autorização; a aprovação fixa a revisão do esquema; cada
// chamada é revalidada (concessão, esquema, argumentos e política) e auditada;
// descrição, anotação e resultado de ferramenta são tratados como DADOS.
package mcp
