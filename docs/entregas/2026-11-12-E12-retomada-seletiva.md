# Entrega 2026-11-12 — E12: retomada seletiva e reconciliação (T12-04)

**Ticket:** [T12-04](../NEW%20MVP/03-BACKLOG.md#t12-04). Decisões: [ADR-0036](../adr/0036-retomada-seletiva-e-reconciliacao.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 26 (`resume_baselines`, `resume_events`).
- `internal/resume`: `Select`, `Rerun`, `Reconcile`, `Resumer` (`RecordBaseline`, `Resume`).
- Sem rotas nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Unitários (conjunto afetado, reexecução com saída diferente e igual, falha, classificação de insumos) | PASS |
| Integração PostgreSQL com `-race` (reexecuta só o afetado em workflow republicado, compatível incorporado, conflito abre decisão sem alterar nada, efeito incerto bloqueia, config revogada bloqueia, checkpoint corrompido) | PASS |
| Ligação ao worker; Git real (fast-forward); rotas/tela | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Reexecutar nó altera só o conjunto afetado:** `TestRerunChangesOnlyTheAffectedSetWhenTheOutputChanges`, `TestRerunWithTheSameOutputKeepsTheDependents`, `TestResumingANodeRerunsOnlyTheAffectedSetOnTheOriginalWorkflow`.
- **Conflito preserva edições humanas:** `TestConflictingInputChangesOpenADecisionAndChangeNothing`.
- **Workflow original continua fixado:** `TestResumingANodeRerunsOnlyTheAffectedSetOnTheOriginalWorkflow` (republicação durante a retomada).
- **Compatível incorporado, efeitos e validações:** `TestCompatibleInputChangesAreIncorporatedIntoTheBaseline`, `TestAnUncertainEffectInTheAffectedSetBlocksTheResume`, `TestResumeValidatesNodeCheckpointAndPublishedConfig`.

## Rollback
`git revert`. A migration 26 tem `Down`.
