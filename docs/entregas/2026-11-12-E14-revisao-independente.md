# Entrega 2026-11-12 — E14: revisão independente e correções (T14-03)

**Ticket:** [T14-03](../NEW%20MVP/03-BACKLOG.md#t14-03). Decisões: [ADR-0047](../adr/0047-revisao-independente-e-correcoes.md). Revisão independente (deste código): NOT_RUN.

## O que mudou
- **Banco:** migration 35 (`review_configs`, `reviews`, `review_findings`).
- `internal/review`: `Store` (`SetConfig`, `GetConfig`, `Pick`, `Start`, `Submit`, `BeginFix`, `CompleteFix`, `Gate`, `History`, `Get`).
- `queue.SuspendOne`.
- Sem rotas, tela nem agente revisor.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race` (independência da primeira rodada com restrição no banco, fallback do revisor, crítico/relevante bloqueiam e baixo fica registrado sem segredo, correção só com checks verdes e rodada de revalidação, aprovação presa à revisão, esgotamento suspende a tarefa com checkpoint seguro e abre decisão, sem checkpoint seguro não suspende) | PASS |
| Agente revisor real; checkpointer real; rotas/tela | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Crítico/relevante bloqueiam; baixa severidade fica registrada:** `TestCriticalAndRelevantFindingsBlockWhileLowSeverityIsOnlyRecorded`.
- **Correção do revisor roda checks:** `TestAFixIsOnlyAcceptedWithGreenChecksAndTriggersARevalidationRound`.
- **Esgotamento suspende com checkpoint:** `TestExhaustingTheRoundsSuspendsTheTaskWithASafeCheckpoint`.
- **Independência, fallback e limite padrão de duas rodadas:** `TestTheInitialReviewMustBeIndependentAndTheReviewerComesFromAConfiguredChain`.

## Rollback
`git revert`. A migration 35 tem `Down`.
