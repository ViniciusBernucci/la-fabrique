# Entrega 2026-11-12 — E08: catálogo de ferramentas (T08-03)

**Ticket:** [T08-03](../NEW%20MVP/03-BACKLOG.md#t08-03). Decisões: [ADR-0015](../adr/0015-catalogo-de-ferramentas.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 11 (`tools`).
- **Backend:** `internal/tools` (esquema e validação, 8 nativas, `Registry` com `Check`/`Call`, execução no sandbox, ferramentas de script e de integração, saída tipada); regras `fs.*` na política; 3 operações de API (listar, cadastrar, validar chamada).
- **Front:** menu **Ferramentas** (lista com assinatura, permissões e limites).

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| `go test -race ./internal/tools` (catálogo, confinamento de caminho, validação, ordem existe→concedida→escopo→argumentos, registro, saída tipada; **no sandbox Docker**: leitura/escrita/edição/lista/busca/comando, política, script por ambiente e sem injeção, limites de saída e de tempo, integração) | PASS |
| `make test-integration` (catálogo, cadastro com política, `check` com 404/403/400, ferramenta nova sem acesso, CSRF; rollback de 11 migrations) | PASS |
| vitest (67 testes), `tsc`, Biome | PASS (cópia no disco interno) |
| `git_status`/`git_diff` executando num repositório | **NOT_RUN** (sem git nas imagens) |
| Links simbólicos e socket | **NOT_RUN** (T11-02) |

## Aceite → evidência
- "Chamada válida devolve saída tipada": `TestNativeToolsReturnTypedOutputInTheSandbox` (ex.: `read_file` → `{content, truncated, bytes}` com tipos conferidos; `edit_file` exige trecho único), `TestCustomScriptToolsPassParametersByEnvironmentAndTypeTheirOutput` (JSON tipado; tipo errado e não-JSON falham), `TestIntegrationToolsUseTheConnectorRunner`.
- "Parâmetro ou escopo indevido falha antes de qualquer efeito": `TestInvalidScopeOrArgsFailBeforeAnyEffect` (caminho que escapa, escopo insuficiente e ferramenta não concedida: nada gravado e **zero** consultas à política), `TestRunCommandIsSubjectToPolicy`, `TestToolCheckFailsBeforeAnyEffectAndNewToolsInheritNoAccess`.

## Rollback
`git revert`. A migration 11 tem `Down` (descarta as ferramentas personalizadas).
