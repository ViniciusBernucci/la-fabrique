# Entrega 2026-11-12 — E17: ensaios e relatório de aceite (T17-02, T17-03, T17-04)

**Tickets:** [T17-02](../NEW%20MVP/03-BACKLOG.md#t17-02), [T17-03](../NEW%20MVP/03-BACKLOG.md#t17-03), [T17-04](../NEW%20MVP/03-BACKLOG.md#t17-04). Decisões: [ADR-0055](../adr/0055-ensaios-e-relatorio-de-aceite.md). Revisão independente: NOT_RUN.

## O que mudou
- `tests/integration/e2e_rehearsal_test.go`: feature da Draft PR ao pós-merge nos dobles.
- `internal/acceptance/rehearsal.go`, `cmd/ensaios`, `make ensaios`: catálogo, execução e relatório [03-ensaios](../aceite/03-ensaios-de-falha-e-seguranca.md).
- `internal/acceptance/report.go`: relatório [04-relatorio-de-aceite](../aceite/04-relatorio-de-aceite.md), gerado por `make matriz`.
- Sem migration, rotas nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| `make ensaios` (15 cenários, PostgreSQL real) | PASS (ver o relatório gerado) |
| Ensaio ponta a ponta nos dobles com `-race` | PASS |
| Catálogo conferido contra o código; deriva do relatório e da matriz | PASS |
| Ambiente, projeto, GitHub, Jira, ClickUp e IA reais; VPS; revisão independente | **NOT_RUN** |

## Aceite → evidência
- **T17-02 (PR, commits, checks, subtarefas, memória e dependência ligados à revisão):** `TestFeatureRehearsalFromDraftPRToPostMergeCycleLinkedToTheIntegratedRevision` (no ambiente real: NOT_RUN).
- **T17-03 (esperado, observado, evidência e revisão por cenário):** relatório gerado por `make ensaios`; `TestRehearsalCatalogMatchesTheCode`.
- **T17-04 (critérios, dependências e decisão):** `TestTheCommittedAcceptanceReportIsTheGeneratedOne`, `TestTheReportListsEveryOpenTicketAndLeavesTheDecisionBlank`. O aceite integral **não** está fechado.

## Rollback
`git revert`.
