# Entrega 2026-11-12 — E09: fallback e disponibilidade (T09-04)

**Ticket:** [T09-04](../NEW%20MVP/03-BACKLOG.md#t09-04). Decisões: [ADR-0031](../adr/0031-fallback-e-disponibilidade-de-contas.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 22 (`provider_health`, `provider_events`, `fallback_events`, `waiting_tasks`).
- `internal/availability`: `Tracker` (`Record`, `Success`, `Clear`, `Get`, `Backoff`) e `Resolver` (`Choose`, `Due`, `Events`).
- `ckEnv` dos testes passou a guardar o cliente HTTP (usado para criar contas).
- Sem rotas nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race` (cota → só a alternativa autorizada, conta fora da cadeia ignorada, troca exige checkpoint e quiescência, esgotamento com espera/hold e outra execução avançando, sem rodízio, classes com regras próprias, cota observada, compatibilidade e orçamento) | PASS |
| Provedores reais; ligação ao worker; rotas/tela | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Cota no principal leva só à alternativa autorizada:** `TestQuotaOnThePrimaryLeadsOnlyToTheAuthorizedAlternative`, `TestFallbackSkipsIncompatibleAndOverBudgetAlternatives`, `TestSwitchingRequiresAnIntactCheckpointAndAQuietSandbox`.
- **Esgotamento preserva tarefa e outra avança:** `TestExhaustionPreservesTheTaskWhileAnotherAdvances`, `TestWaitBacksOffExponentiallyWhenNothingIsOfficial`.
- **Disponibilidade volta sem rodízio:** `TestAvailabilityReturnsWithoutAutomaticRotation`; sinais: `TestFailureClassesHaveDistinctRecoveryRules`, `TestObservedSubscriptionQuotaIsAnOfficialSignal`.

## Rollback
`git revert`. A migration 22 tem `Down`.
