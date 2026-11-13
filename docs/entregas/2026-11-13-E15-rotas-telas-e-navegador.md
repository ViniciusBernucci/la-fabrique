# Entrega 2026-11-13 — Rotas, telas e navegador real (E15/E16)

**Decisões:** [ADR-0057](../adr/0057-rotas-telas-e-ensaio-com-navegador.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 41 (`delivery_settings`).
- **API:** `internal/httpapi/operations.go` (entregas, comentários, ações, memória, controles, relatório da feature), `internal/memory.List`, mapeamento de erros; contrato atualizado.
- **Painel:** `DeliveryPanel`, `MemoryPanel`, controles no `RunPanel`; correções de classes de estilo na Central (`secondary-action`, `item-list`); `useResource.reload` mantém os dados.
- **Navegador real:** `cmd/e2e-stack`, `apps/web/e2e/` (Playwright), `make e2e`; erros de tipo antigos corrigidos.

## Verificação (2026-11-13)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race`: observar o merge, promover memória, comentários autorizados, ações da PR fechada, controles, invalidação, relatório, isolamento entre projetos, falha fechada sem conexão do repositório, sessão e CSRF | PASS |
| Contrato OpenAPI | PASS |
| Vitest (93) e `npm run typecheck`, `npm run build` | PASS |
| **Navegador real** (`make e2e`, Chromium, 9 cenários) | PASS |
| Ambiente e projeto reais (GitHub, Jira, ClickUp, VPS) | **NOT_RUN** |

## Rollback
`git revert`. A migration 41 tem `Down`.
