# Entrega 2026-11-12 — E15: memória técnica pendente e oficial (T15-05)

**Ticket:** [T15-05](../NEW%20MVP/03-BACKLOG.md#t15-05). Decisões: [ADR-0053](../adr/0053-memoria-tecnica-pendente-e-oficial.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 40 (`memory_records`, com unicidade de pendente por branch e de oficial por projeto e chave).
- `internal/memory`: `Store` (`Record`, `Get`, `Relevant`, `Invalidate`, `Promote`, `ExtractFromDelivery`).
- Sem rotas nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race` (deduplicação e evidências, redação, isolamento entre projetos, pendente só na própria branch, PR aberta/fechada não promove, merge promove só o válido, duplicata e substituição, invalidação, validador, extração de lições) | PASS |
| Ligação ao contexto/worker; GitHub real; rotas/tela | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Projeto A não consome memória de B:** `TestProjectAnNeverConsumesProjectBMemory`.
- **PR fechada não promove; merge promove só registros válidos para a revisão integrada:** `TestOnlyAMergeConfirmedByTheOriginPromotesValidRecords`, `TestPromotionDeduplicatesAgainstOfficialAndSupersedesChangedFacts`.
- **Extração, deduplicação e evidência vinculada:** `TestPendingMemoryIsDeduplicatedLinkedToEvidenceAndRedacted`, `TestLessonsAreExtractedFromFixedReviewFindingsWithoutDuplicating`.

## Rollback
`git revert`. A migration 40 tem `Down`.
