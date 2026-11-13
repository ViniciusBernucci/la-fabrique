# Entrega 2026-11-12 — E10: plano versionado da feature (T10-01)

**Ticket:** [T10-01](../NEW%20MVP/03-BACKLOG.md#t10-01). Decisões: [ADR-0035](../adr/0035-plano-versionado-da-feature.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 25 (`plans`, `plan_tasks`).
- `internal/planning`: contrato, `Validate`, `Order`, hashes, `Store` (`Propose`, `Current`, `Versions`, `Tasks`, `SetTaskState`, `Publish`).
- **API** (`contracts/openapi.yaml`): `PUT/GET /api/projects/{id}/features/{featureId}/plan`.
- Sem tela, sem agente planejador.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Unitários (ciclo, referência ausente, critérios inverificáveis, ordem topológica, hash independente da ordem) | PASS |
| Integração PostgreSQL com `-race` (plano aceito, rejeições preservam a versão, conflito com tarefa em execução, mesmo plano sem versão nova e concorrência, publicação sem duplicar após resposta perdida e no retry, replanejamento só cria o novo, API) | PASS |
| Agente planejador; Jira/ClickUp reais; scheduler; tela | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Feature vira tarefas verificáveis, sem ciclo:** `TestFeatureBecomesVerifiableTasksInAnAcyclicPlan`, testes unitários de validação.
- **Sem efeito duplicado no retry:** `TestPublishingThePlanToTheOriginNeverDuplicatesOnRetry`, `TestPublishFailureIsReportedAndCompletedOnTheNextRun`.
- **Plano conflitante preserva a versão anterior:** `TestInvalidOrConflictingPlansPreserveThePreviousVersion`, `TestPlanThroughTheAPI`.

## Rollback
`git revert`. A migration 25 tem `Down`.
