# Entrega 2026-11-12 — E06: sincronização manual e preflight (T06-04)

**Ticket:** [T06-04](../NEW%20MVP/03-BACKLOG.md#t06-04). Decisões: [ADR-0033](../adr/0033-sincronizacao-manual-e-preflight.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 23 (`source_items`, `source_syncs`, `source_changes`).
- `internal/syncsource`: normalização, `Syncer.Run`, `Preflight`, `SetLocalPriority`, `ResolveConflict`, `Last`, `Items`.
- **API** (`contracts/openapi.yaml`): `POST/GET /api/projects/{id}/sync`, `PUT …/sync/items/{remoteId}/priority`, `POST …/sync/items/{remoteId}/resolve`.
- Sem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race` (relógio ocioso sem leitura, mudança detectada uma vez por sync e preflight, deduplicação e paginação, saída e retorno, leitura falha sem efeito, conflito com edição humana, preflight bloqueando item inelegível, concorrência, API) | PASS |
| Jira/ClickUp reais; GitHub; ligação ao agendador; tela | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Relógio avança ocioso sem leitura periódica:** `TestIdleClockNeverReadsTheSource`.
- **Sync manual ou preflight detecta a alteração uma vez:** `TestAChangeIsDetectedExactlyOnce`, `TestConcurrentSyncsRecordTheChangeOnce`.
- **Dedup, edição humana, elegibilidade:** `TestImportIsDeduplicatedAndPaginated`, `TestRemotePriorityChangeNeverOverwritesALocalEdit`, `TestPreflightBlocksAnItemThatStoppedBeingEligible`, `TestAFailedReadChangesNothing`, `TestManualSyncThroughTheAPI`.

## Rollback
`git revert`. A migration 23 tem `Down`.
