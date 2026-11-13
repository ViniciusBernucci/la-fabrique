# Entrega 2026-11-12 — E12: recuperação após falha ou reboot (T12-03)

**Ticket:** [T12-03](../NEW%20MVP/03-BACKLOG.md#t12-03). Decisões: [ADR-0045](../adr/0045-recuperacao-de-tentativas-interrompidas.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 33 (`recovery_attempts`, `recovery_events`).
- `internal/recovery`: `Recoverer` (`Stale`, `Recover`, `RecoverAll`), `Inspector`, `Limits`/`DefaultLimits`.
- `internal/queue`: `ListStale`, `RecoverRequeue`, `RecoverPark`, `RequireRecovery` (Claim deixa de devolver às cegas).
- Sem `Inspector` real, laço no worker, rotas ou tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race` (crash seguro retomado sozinho com o checkpoint, recomeço sem checkpoint; incerteza abre decisão e as outras tarefas seguem; quietude não provada segura a vaga; checkpoint corrompido, problema de Git e efeitos sem checkpoint pedem decisão; limites separados por classe; inspeção indisponível adia sem consumir) | PASS |
| Suíte do scheduler (T10-03) com o novo `RequireRecovery` desligado | PASS |
| Inspetor real (Docker/Git); varredura no worker; rotas/tela | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Reboot e crash retomam automaticamente quando seguros:** `TestACrashIsRecoveredAutomaticallyWhenItIsSafe`.
- **Incerteza abre decisão e outras tarefas elegíveis seguem:** `TestUncertaintyOpensADecisionAndOtherEligibleTasksKeepGoing`, `TestCorruptCheckpointAndVerifierProblemsRequireADecision`.
- **Limites separados:** `TestRecoveryLimitsAreSeparatedPerFailureClass`; inspeção indisponível: `TestAnUnavailableInspectionDefersWithoutChangingAnything`.

## Rollback
`git revert`. A migration 33 tem `Down`.
