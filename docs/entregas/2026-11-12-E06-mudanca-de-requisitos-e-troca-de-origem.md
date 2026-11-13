# Entrega 2026-11-12 — E06: mudança de requisitos e troca de plataforma (T06-05)

**Ticket:** [T06-05](../NEW%20MVP/03-BACKLOG.md#t06-05). Decisões: [ADR-0038](../adr/0038-mudanca-de-requisitos-e-troca-de-origem.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 27 (`requirement_revisions`, `source_switches`, `feature_identity_history`).
- `internal/reqchange`: `Classify`, `Handler` (`Handle`, `AtMilestone`, `Settle`, `CompleteReplan`, `Current`, `History`) e `Switcher` (`Plan`, `Activate`, `Switches`, `IdentityHistory`).
- `projects.RecordChange`/`SourceDetail` exportados; `planning.TasksOf`; `planning.BusinessKey` passa a incluir a identidade remota da feature.
- Sem rotas nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Unitário (classificação de impacto, hash independente da ordem) | PASS |
| Integração PostgreSQL com `-race` (compatível mantém histórico, adiamento até marco seguro, replanejamento, conflito com decisão sem alterar nada, resolução `keep_work`/`replan`, troca validada/rejeitada/ativada com histórico de identidades, troca obsoleta recusada, publicação após troca na nova origem) | PASS |
| Jira/ClickUp reais; rotas/tela; ligação ao sync e ao planejador | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Mudança compatível mantém histórico:** `TestACompatibleChangeKeepsTheHistory`, `TestMetadataChangesWaitForASafeMilestone`.
- **Conflito preserva alterações e cria decisão:** `TestAConflictWithWorkInProgressPreservesEverythingAndOpensADecision`, `TestAConflictResolvedWithReplanBecomesPendingReplan`; sem trabalho iniciado: `TestContentChangeBeforeAnyWorkAsksForANewPlan`.
- **Troca mantém vínculos históricos e uma origem ativa:** `TestSwitchingThePlatformKeepsHistoricalLinksAndASingleActiveOrigin`, `TestAStaleValidatedSwitchIsRefused`, `TestPublishingAfterAnOriginSwitchCreatesTheItemsInTheNewOrigin`.

## Rollback
`git revert`. A migration 27 tem `Down`.
