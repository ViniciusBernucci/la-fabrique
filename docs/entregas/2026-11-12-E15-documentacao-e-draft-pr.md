# Entrega 2026-11-12 — E15: documentação viva e Draft PR (T15-02)

**Ticket:** [T15-02](../NEW%20MVP/03-BACKLOG.md#t15-02). Decisões: [ADR-0049](../adr/0049-documentacao-viva-e-draft-pr.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 36 (`feature_commits`, `deliveries`).
- `internal/delivery`: `DocsGaps`/`DefaultDocRules`, `FeatureDocPath`, `BuildReport`/`Render`, `Deliverer` (`RecordCommit`, `Document`, `Deliver`), gates (`ChecksGate`, `ReviewGate`, `GateFunc`).
- `internal/gitops/pr.go`: `GitHubAPI.FindPR`/`CreateDraftPR`, `PRPublisher.OpenDraft` (pelo ledger).
- Sem rotas nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Unitários (lacunas de documentação por módulo, caminho do histórico) | PASS |
| Integração PostgreSQL com `-race` (ausência de documentação bloqueia sem abrir PR e fica registrada; histórico desatualizado bloqueia; Draft PR com corpo e commits; repetir encontra a mesma PR; resposta perdida e "já existe" não duplicam; gates de checks e E2E bloqueiam; branch que não é da feature é recusada; origem recebe "Aguardando merge") | PASS |
| GitHub real; `git` real junto da PR; rotas/tela | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Repo real contém docs e testes coerentes:** `TestMissingDocumentationBlocksTheDelivery` (gate por módulo + histórico) e a orientação de testes de T14-04.
- **Repetição encontra a mesma Draft PR:** `TestRepeatingTheDeliveryFindsTheSameDraftPR`.
- **Ausência de documentação bloqueia entrega:** `TestMissingDocumentationBlocksTheDelivery`, `TestOtherGatesAndTheBranchGuardAlsoBlock`.

## Rollback
`git revert`. A migration 36 tem `Down`.
