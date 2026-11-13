# ADR-0020 — MCP: cadastro, descoberta e allowlist por agente

- **Estado:** ACEITA · **Ticket:** [T08-04](../NEW%20MVP/03-BACKLOG.md#t08-04) · **Épico:** E08 · **Relacionadas:** [ADR-0005](0005-autenticacao-sessoes-e-cofre-de-conexoes.md), [ADR-0012](0012-politica-de-risco-no-executor.md), [ADR-0015](0015-catalogo-de-ferramentas.md)

## Decisão
1. **Transporte:** Streamable HTTP (resposta JSON ou SSE), versões 2025-11-25, 2025-06-18 e 2025-03-26. HTTPS obrigatório, sem credenciais na URL, sem seguir redirecionamento, destino interno recusado (`policy.IsInternalHost`). stdio não é suportado (UNSUPPORTED).
2. **Credencial só pelo cofre:** o servidor referencia uma conexão `mcp`; o token vai em `Authorization: Bearer`, nunca é devolvido pela API nem logado, e os erros do cliente têm mensagem fixa (não ecoam o corpo remoto).
3. **Descoberta é catálogo, não acesso:** `tools/list` (paginado) faz upsert. Ferramenta nova nasce **sem concessão**; ausente vira `removed`; descrições são dados (caracteres de controle removidos, tamanho limitado).
4. **Allowlist por agente com hash pinado:** a concessão guarda o hash do contrato (nome + `inputSchema` canônico) aprovado. Aprovar exige o hash **vigente** (`stale_approval` caso contrário). Descrição não faz parte do hash.
5. **Cada chamada revalida:** concessão → hash → argumentos contra o esquema (subconjunto de JSON Schema; propriedade extra recusada) → `initialize` + `tools/list` com o contrato vigente (se mudou: atualiza o catálogo e bloqueia com `schema_changed`) → política de risco → `tools/call` → trilha.
6. **Conceder não dispensa a política:** nomes que alteram estado (ex.: `delete_*`) continuam exigindo aprovação; anotações do servidor não concedem acesso.
7. **Resultado é dado:** texto limpo e limitado; blocos não textuais só contados. Trilha `mcp_calls` com argumentos redigidos (`Redactor`) e hash.

## Limites
- Servidor MCP real e stdio: NOT_RUN / UNSUPPORTED. Só o servidor falso de teste foi exercitado.
- A revalidação adiciona 2 round-trips por chamada (decisão: segurança sobre latência).
- Esquemas com `$ref`, `oneOf`/`anyOf` não são validados localmente; o servidor continua sendo a autoridade final.
