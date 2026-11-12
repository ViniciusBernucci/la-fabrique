# Entrega 2026-11-12 — E13: pedidos e decisões humanas (T13-02)

**Ticket:** [T13-02](../NEW%20MVP/03-BACKLOG.md#t13-02). Decisões: [ADR-0017](../adr/0017-pedidos-e-decisoes-humanas.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 12 (`decision_requests`).
- **Backend:** `internal/decisions` (criação com deduplicação e evidência redigida, listagem, `Resolve` transacional) e 3 operações de API.
- **Front:** a Central de Decisões deixou de ser um painel vazio (abas, formulário de decisão, histórico).

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| `make test-integration` (criação e regras; evidência redigida; resolução que só afeta a execução do pedido; repetição idempotente e conflito; 8 resoluções concorrentes; execução obsoleta por cancelamento e por versão; validação; reautenticação em operação sensível; CSRF; rollback de 12 migrations) | PASS |
| vitest (72 testes; 5 novos da Central), `tsc`, Biome | PASS (cópia no disco interno) |
| Pedidos criados por um fluxo real de execução; consumo do sinal pelo escalonador | **não implementado** (E10/E12) |

## Aceite → evidência
- "Resolver pendência por UI altera só execução afetada": `TestResolvingChangesOnlyTheAffectedExecution` (A vira `queued` com 2 eventos; B continua `waiting`, mesma versão, sem eventos, pedido B intacto) e os testes de tela da Central.
- "Decisão repetida ... tratada de forma consistente": `TestRepeatedDecisionIsIdempotentAndAConflictingOneIsRefused`, `TestConcurrentResolutionsProduceExactlyOneResumeSignal`.
- "... ou sobre versão obsoleta ...": `TestDecisionOnAnObsoleteExecutionIsRefusedAndMarkedStale` (cancelada e versão alterada → 409 `decision_stale`, pedido `stale`, nada sinalizado, repetição dá o mesmo resultado).

## Rollback
`git revert`. A migration 12 tem `Down` (descarta os pedidos e o histórico).
