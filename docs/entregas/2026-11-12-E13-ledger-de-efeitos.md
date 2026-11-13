# Entrega 2026-11-12 — E13: ledger de efeitos externos (T13-04)

**Ticket:** [T13-04](../NEW%20MVP/03-BACKLOG.md#t13-04). Decisões: [ADR-0018](../adr/0018-ledger-de-efeitos-externos.md). Revisão independente: NOT_RUN.

## O que mudou
`internal/effects` (ledger sobre a tabela `outbox` existente: `Run`, reconciliação, `Resolve`, `DecisionEscalator`, `ResolveFromDecision`), `persistence.GetOutbox` e o gancho da Central (resolver um pedido de efeito aplica a decisão ao ledger). **Sem migration** (o `outbox` já tinha os estados `intent`/`sending`/`confirmed`/`failed`/`uncertain`, a chave de negócio e o campo de prova).

## Verificação (2026-11-12, PostgreSQL real)
| Verificação | Resultado |
|---|---|
| `make test-integration` (10 testes do ledger com um destino de teste sem idempotência) | PASS |
| `go test -race -count=5` nos testes de concorrência, timeout e queda | PASS |
| Adaptadores reais (Jira, ClickUp, GitHub) | **não implementados** (T06-03, E15) |
| Compensação de efeitos | **não implementada** |

## Aceite → evidência
- "Timeout após efeito confirmado não duplica": `TestTimeoutAfterTheEffectLandedDoesNotDuplicate` (o efeito aconteceu, a resposta se perdeu: a prova confirma, **1 efeito, 1 envio**; repetir a operação devolve `already_confirmed` sem reenviar).
- "Ausência segura permite repetir": `TestProvenAbsenceAllowsASafeRetry` (ausência comprovada → reenvia e aplica uma vez; ausência persistente falha após 3 tentativas sem efeito e continua repetível) e `TestDefinitiveRefusalFailsWithoutVerifying`.
- "Estado inconclusivo abre bloqueio com evidência": `TestInconclusiveStateBlocksAndOpensADecisionWithEvidence` (bloqueia, três repetições não enviam nada, **um** pedido na Central com a evidência; "o efeito existe" confirma sem reenviar), `TestHumanSaysTheEffectDoesNotExistAndTheRetryIsAllowed`, `TestVerificationErrorCountsAsInconclusiveNotAsAbsent`.
- Concorrência e queda: `TestConcurrentRunsOfTheSameOperationSendOnce` (8 chamadas, 1 envio), `TestAbandonedSendingIsReconciledBeforeAnyRetry` (dentro da concessão não toca; vencida: confirma, refaz ou bloqueia conforme a prova), `TestIntentIsRecordedBeforeSendingAndRequestsAreRedacted`.

## Rollback
`git revert`; sem migration.
