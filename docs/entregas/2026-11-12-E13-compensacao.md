# Entrega 2026-11-12 — E13: compensação de efeitos parciais por risco (T13-05)

**Ticket:** [T13-05](../NEW%20MVP/03-BACKLOG.md#t13-05). Decisões: [ADR-0029](../adr/0029-compensacao-de-efeitos-por-risco.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 20 (`compensations`, `compensation_events`, `compensation_blocks`).
- `internal/compensation`: `Compensator` (`Compensate`, `CheckAggravating`, `ResolveBlock`, `Events`), classificação de risco e `ActionFor`.
- Correção durante o teste: `compensate:<op>` não era reconhecida como operação que não agrava.
- Sem rotas nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race` (baixo risco idempotente e com prova, resposta perdida, sensível com/sem autorização, irreversível preservado, falha com bloqueio, retomada, resolução humana) | PASS |
| Adaptadores reais de desfazer; ligação ao executor | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Compensação executada ou recusada fica auditável:** `TestLowRiskCompensationIsIdempotentAndProvenAndAudited`, `TestSensitiveCompensationNeedsAMatchingAuthorization`, `TestIrreversibleEffectsArePreservedAndTheRefusalIsAudited` (trilha de eventos).
- **Falha conserva trabalho e bloqueia nova operação agravante:** `TestFailedCompensationKeepsTheWorkAndBlocksAggravatingOperations` (o efeito segue no destino; `push`, `create_subtask`, `transition`, `merge` bloqueadas; leitura permitida; outra execução livre; retomada e resolução humana).

## Rollback
`git revert`. A migration 20 tem `Down`.
