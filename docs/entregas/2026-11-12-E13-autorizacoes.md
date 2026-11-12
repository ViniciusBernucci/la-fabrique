# Entrega 2026-11-12 — E13: escopo, expiração e revogação (T13-03)

**Ticket:** [T13-03](../NEW%20MVP/03-BACKLOG.md#t13-03). Decisões: [ADR-0026](../adr/0026-autorizacoes-com-escopo-uso-e-revogacao.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 17 (`authorizations`, `authorization_uses`, `authorization_events`).
- `internal/authorizations`: criação (com vínculo à decisão), `Consume` atômico, `Do`/`Finish`, `Revoke` com `Halter`, trilha.
- Sem rotas nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race` (corrida pelo último uso, divergências, prazo e revogação, revogação concorrente, parada segura com sandbox e checkpoint reais, falha da parada, vínculo com decisão) | PASS |
| Corridas repetidas 6× com `-race` | PASS |
| Ligação ao executor/worker, API e tela | **não feitas** |

## Aceite → evidência
- **Corrida de último uso autoriza um:** `TestRaceForTheLastUseAuthorizesExactlyOne` (30 disputando 1 uso: 1 vence, 29 `ErrExhausted`; 50 disputando 5: 5).
- **Argumento ou destino divergente impedem a operação:** `TestDivergentActionIsRefusedWithoutSpendingAUse` (9 variações: SHA, force, parâmetro a mais, aninhado, destino, destino interno, recurso, operação, ferramenta — nenhuma gasta uso; a forma canônica equivalente passa).
- **Prazo expirado e autorização revogada impedem a operação:** `TestExpiredAndRevokedAuthorizationsBlockTheOperation`, `TestRevocationRacingWithConsumersNeverLetsAUseThrough` (nenhum uso iniciado depois da revogação passa).
- **Revogar dispara parada segura/checkpoint:** `TestRevokingWithAUseInFlightTriggersSafeStopAndCheckpoint` (processo cancelado, checkpoint com o progresso, sandbox liberado, trilha `created,consumed,revoked,halted,refused`), `TestRevokingWithoutAUseInFlightDoesNotHaltAndAHaltFailureIsExplicit`.

## Rollback
`git revert`. A migration 17 tem `Down`.
