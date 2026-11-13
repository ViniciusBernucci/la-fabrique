# Entrega 2026-11-12 — E05: perfil técnico e recursos do projeto (T05-03)

**Ticket:** [T05-03](../NEW%20MVP/03-BACKLOG.md#t05-03). Decisões: [ADR-0034](../adr/0034-perfil-tecnico-e-recursos-do-projeto.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 24 (`project_profiles`, `project_resources`).
- `internal/projectres` (descoberta, perfil, recursos, validador, `Ready`); `projects.AddValidator`; `selection.Load`.
- **API** (`contracts/openapi.yaml`): `POST /profile/discover`, `GET/PUT /profile`, `GET/PUT /resources`, `GET /readiness`.
- Sem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race` (descoberta com origem e sem vazamento, perfil com origem e segredo só por referência, override sem alterar o global, rascunho e conta incompatível bloqueando ativação, projeto apto iniciando execução) | PASS |
| Leitura de repositório real no GitHub; tela do assistente | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Overrides preservam recurso global:** `TestProjectOverridesNeverChangeTheGlobalResource`.
- **Rascunho de workflow e conta incompatível bloqueiam ativação:** `TestADraftWorkflowBlocksActivationUntilPublished`, `TestAnIncompatibleAccountBlocksActivation`.
- **Projeto apto pode iniciar execução:** `TestAnAptProjectCanStartAnExecutionOnlyFromAPublishedDefaultWorkflow`.
- **Descoberta e segredos:** `TestDiscoverySuggestsCommandsWithTheirSourceAndNeverReadsSecrets`, `TestProfileKeepsTheSourceOfEachCommandAndStoresSecretsOnlyByReference`.

## Rollback
`git revert`. A migration 24 tem `Down`.
