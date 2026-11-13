# Entrega 2026-11-12 — E09: modelos e gateways de API (T09-02)

**Ticket:** [T09-02](../NEW%20MVP/03-BACKLOG.md#t09-02). Decisões: [ADR-0021](../adr/0021-modelos-e-gateways-de-api.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 14 (`models`).
- `internal/models`: catálogo, validação de parâmetros e `Check` (veredito com motivos).
- `internal/gateways`: cliente OpenAI-compatível para OpenRouter e Vercel AI Gateway, laço de agente com `Executor` controlado e `RegistryExecutor`.
- API: `GET/POST /api/models`, `POST /api/models/check`, `POST /api/models/{id}/state`; `model_check` no efetivo do nó; contrato em `contracts/openapi.yaml`.
- Front: **não** incluído.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| `go test -race ./internal/gateways` (cliente, erros sem vazamento, elegibilidade, laço, limites, edição + teste nos dois gateways com sandbox Docker real) | PASS |
| Integração com PostgreSQL real (catálogo, compatibilidade, sobreposição por nó, cofre sem vazamento) | PASS |
| OpenRouter / Vercel AI Gateway reais | **NOT_RUN** |
| Interface no painel / ligação ao worker | **não feitas** |

## Aceite → evidência
- **Modelo só é usado quando compatível:** `TestModelIsOnlyUsableWhenCompatible` (9 incompatibilidades com motivo), `TestEligibilityIsCheckedBeforeAnyRequest` (nenhum pedido sai).
- **Cada gateway roda edição e teste com modelo elegível:** `TestEachGatewayRunsEditAndTestWithAnEligibleModel` (subtestes `openrouter` e `vercel`: `write_file`, `run_tests` falha, `edit_file`, `run_tests` passa, tudo pela política).
- **Sobreposição modelo/agente/nó é verificável:** `TestModelOverridesAreVerifiablePerNode` (origem por campo, veredito e base do catálogo intacta).
- Ferramenta fora do agente / sem concessão não executa: `TestLoopOnlyRunsToolsTheAgentHasAndFeedsResultsBack`, `TestRegistryExecutorStillEnforcesGrantAndScope`.

## Rollback
`git revert`. A migration 14 tem `Down`.
