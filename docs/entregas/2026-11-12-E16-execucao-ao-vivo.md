# Entrega 2026-11-12 — E16: status, controles e logs em tempo real (T16-01)

**Ticket:** [T16-01](../NEW%20MVP/03-BACKLOG.md#t16-01). Decisões: [ADR-0048](../adr/0048-painel-de-execucao-ao-vivo.md). Revisão independente: NOT_RUN.

## O que mudou
- **API** (`contracts/openapi.yaml`): `GET /api/projects/{id}/overview`, `GET …/executions/{executionId}/events` (histórico pesquisável) e `GET …/executions/{executionId}/stream` (SSE com cursor).
- `internal/httpapi/live.go`; invólucros do middleware repassam `Flush`/`Unwrap`; `Queue.ControlOf`.
- **Web:** `RunPanel` no detalhe do projeto, hook `useEventStream`, tipos e chamadas em `api.ts`, estilos com tokens `--hud-*`.
- Sem migration.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race`: SSE real (eventos ao vivo; desconectar e reconectar por `Last-Event-ID` e por `?after=` sem perder nem repetir; heartbeat; encerramento por tempo máximo; segredo redigido; escopo por projeto; cursor inválido), histórico pesquisável e paginado, resumo com controles persistidos e a causa de cada bloqueio | PASS |
| vitest (80 testes; novos: controles e causas, reconexão pelo cursor sem repetir, busca no histórico) | PASS |
| Biome, tsc, contrato OpenAPI | PASS |
| Navegador real com a API ao vivo; executores gravando eventos | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Desconexão/reconexão recupera eventos pelo cursor:** `TestSSEReconnectionRecoversEventsByTheCursor`; no painel, vitest "recupera os eventos pelo cursor ao reconectar e ignora os já vistos".
- **Estado bloqueado explica causa e painel exibe controles persistidos:** `TestOverviewExplainsBlockedStatesAndShowsThePersistedControls`; vitest "mostra os controles persistidos e a causa de cada bloqueio".

## Rollback
`git revert` (sem migration).
