# Entrega 2026-11-12 — E07: publicar e congelar configuração efetiva (T07-03)

**Ticket:** [T07-03](../NEW%20MVP/03-BACKLOG.md#t07-03). Decisões: [ADR-0027](../adr/0027-publicacao-e-congelamento-de-workflows.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 18 (`workflow_versions` imutável, `workflows.published_version_id`, `executions.workflow_version_id`).
- `internal/releases`: `Publish`, `Unpublish`, `Revalidate`, `StartExecution`, `ExecutionVersion`, `Get`, `List`.
- API: publish, unpublish, versions, workflow-versions e revalidate; contrato em `contracts/openapi.yaml`.
- Sem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL (validação de referências, snapshot sanitizado, idempotência, rascunho não executa, execução presa à versão, imutabilidade, subfluxos fixados, revalidação com conta revogada) | PASS |
| Motor/worker consumindo `StartExecution` e tela | **não feitos** |

## Aceite → evidência
- **Rascunho não executa:** `TestDraftDoesNotRunAndANewPublicationDoesNotChangeAnExistingExecution` (sem execução criada).
- **Publicação nova não muda execução existente:** mesmo teste (execução 1 segue na v1; a 2 usa a v2; mudar o agente no catálogo não altera o congelado).
- **Revalidação respeita revogação:** `TestRuntimeRevalidationHonorsRevocation` (`connection_revoked`; nenhuma execução nasce).
- Validação, imutabilidade e fixação de subfluxos: `TestPublishValidatesReferencesAndFreezesASanitizedEffectiveConfig`, `TestPublishedVersionsAreImmutableAndSubflowsArePinned`.

## Rollback
`git revert`. A migration 18 tem `Down`.
