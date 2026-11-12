# Entrega 2026-11-12 — E07: motor de workflows (T07-02)

**Ticket:** [T07-02](../NEW%20MVP/03-BACKLOG.md#t07-02). Decisões: [ADR-0010](../adr/0010-motor-de-workflows.md). Revisão independente: NOT_RUN.

## O que mudou
`internal/workflows`: `expr.go` (expressões), `flatten.go` (subfluxos em linha), `engine.go` (motor e estado), nó `fork` no catálogo e validação de expressões. Sem migration nem mudança de contrato HTTP (o catálogo ganhou o tipo `fork`).

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| `go test -race ./internal/workflows` (motor, expressões, subfluxos, validador) | PASS |
| `make lint`, testes Go do repositório, `make test-integration`, `make check-secrets` | PASS (ver commit) |
| Execução com executores reais (agente, skill, script, Git) | **NOT_RUN** (executores são de E09/E11/E13/E15; os testes usam um executor de mentira) |

## Aceite → evidência
- "Percorrem caminhos e convergência": `TestConditionTakesEachBranchAndConverges`, `TestForkRunsBranchesSequentiallyAndConvergeAllWaitsForActivePaths`, `TestBoundedLoopStopsAtLimitOrWhenConditionClears`, `TestErrorRouteHandlesFailureOtherwiseRunFails`.
- "Retomam espera persistida": `TestDurableWaitResumesFromPersistedState` (JSON ida e volta idêntico, antes do prazo continua aguardando, no prazo conclui, concluída não roda de novo) e `TestHumanDecisionPausesAndResumesOnce`.
- "Bloqueiam loop sem limite": `TestLoopWithoutLimitIsBlockedBeforeRunning`. "…ou subfluxo recursivo": `TestRecursiveAndTooDeepSubflowsAreBlocked` (A→B→A, auto-chamada, profundidade, inexistente, inválido).

## Rollback
`git revert`; sem migration.
