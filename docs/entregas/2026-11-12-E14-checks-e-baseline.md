# Entrega 2026-11-12 — E14: checks no baseline e classificação de falhas (T14-01)

**Ticket:** [T14-01](../NEW%20MVP/03-BACKLOG.md#t14-01). Decisões: [ADR-0040](../adr/0040-checks-baseline-e-classificacao-de-falhas.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 29 (`check_runs`, `known_failures`).
- `internal/checks`: `Signature`, `Store` (`RunSet`, `RecordNotViable`, `Evaluate`, `Gate`, `Resolve`, `KnownFailures`, `Runs`), `SandboxExecer`.
- Sem rotas nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Unitário (assinatura estável a ruído, sensível à falha) | PASS |
| Integração PostgreSQL com `-race` (regressão bloqueia, falha anterior comprovada registrada e reconhecida depois, origem desconhecida em investigação sem virar baseline, resolução com prova, evidência por revisão, erro de infraestrutura inconclusivo, redação/truncamento, validações) | PASS |
| Docker real: checks no sandbox (sucesso, falha com código 3, tempo limite) | PASS |
| Preparo automático do baseline; comandos vindos do perfil; rotas/tela | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Regressão bloqueia:** `TestARegressionBlocksTheRevision`.
- **Falha anterior comprovada é registrada:** `TestAProvenPreexistingFailureIsRecordedAndDoesNotBlock`.
- **Origem desconhecida fica em investigação e não vira baseline por suposição:** `TestAnUnknownOriginStaysUnderInvestigationAndNeverBecomesBaseline`.
- **Evidência por revisão e infraestrutura:** `TestEvidenceBelongsToTheTestedRevisionAndInfrastructureErrorsAreInconclusive`; execução real: `TestChecksRunInTheRealSandbox`.

## Rollback
`git revert`. A migration 29 tem `Down`.
