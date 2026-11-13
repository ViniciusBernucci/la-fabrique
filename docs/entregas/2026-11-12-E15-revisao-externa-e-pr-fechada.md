# Entrega 2026-11-12 — E15: revisão externa e PR fechada (T15-04)

**Ticket:** [T15-04](../NEW%20MVP/03-BACKLOG.md#t15-04). Decisões: [ADR-0052](../adr/0052-revisao-externa-e-pr-fechada.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 39 (`deliveries` ganha o estado `cancelled`; `pr_comments`; `delivery_actions`).
- `internal/gitops`: `GitHubAPI.ListComments`, `ReopenPR`, `PRClient`.
- `internal/delivery`: `External` (`Discover`, `MarkFixed`, `Offer`, `Decide`, `List`, `History`); `Deliver` passa a respeitar tentativas encerradas e a decisão humana.
- Sem rotas nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race` (comentários duplicados, editados, de não autorizados e vazios; redação; leitura que falha; um só item de correção; prova de docs e checks; PR fechada com artefatos; oferta única; reabrir recusado e confirmado; cancelar; nova tentativa; limite) | PASS |
| GitHub real; execução da correção por agente; rotas/tela | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Mesmo comentário não duplica trabalho:** `TestTheSameExternalCommentNeverDuplicatesWork`, `TestExternalCommentsAreRedactedAndAFailedReadChangesNothing`.
- **PR fechada conserva artefatos; cancelar, reabrir e nova tentativa preservam histórico e respeitam decisão:** `TestAClosedPRKeepsArtifactsAndOffersThreeControlledActions`.

## Rollback
`git revert`. A migration 39 tem `Down`.
