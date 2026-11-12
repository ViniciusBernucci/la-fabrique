> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo →](01-controle.md)

# Rotas físicas observadas

Extraídas de todos os controllers NestJS da base a2cc5e0. Prefixo global /api em main.ts. Guard de classe/método deve ser lido no controller; extração não gera OpenAPI nem prova HTTP em serviço. Headers Bearer não são exemplos de credenciais reais.

| Método | Caminho | Guard observado | Schema de entrada/saída referenciado | Fonte |
|---|---|---|---|---|
| GET | `/api/auth/session` | AdminAuthGuard | `adminSessionSchema` | [control.controller.ts](../../../apps/api/src/control/control.controller.ts) |
| POST | `/api/projects` | AdminAuthGuard | `projectSchema` | [control.controller.ts](../../../apps/api/src/control/control.controller.ts) |
| GET | `/api/projects` | AdminAuthGuard | `projectListSchema` | [control.controller.ts](../../../apps/api/src/control/control.controller.ts) |
| PATCH | `/api/projects/:projectId/base-revision` | AdminAuthGuard | `projectSchema` | [control.controller.ts](../../../apps/api/src/control/control.controller.ts) |
| GET | `/api/projects/:projectId/definition` | AdminAuthGuard | `projectDefinitionStateSchema` | [control.controller.ts](../../../apps/api/src/control/control.controller.ts) |
| PUT | `/api/projects/:projectId/definition` | AdminAuthGuard | `projectDefinitionSchema` | [control.controller.ts](../../../apps/api/src/control/control.controller.ts) |
| POST | `/api/projects/:projectId/tickets` | AdminAuthGuard | `ticketSchema` | [control.controller.ts](../../../apps/api/src/control/control.controller.ts) |
| GET | `/api/projects/:projectId/tickets` | AdminAuthGuard | `ticketListSchema` | [control.controller.ts](../../../apps/api/src/control/control.controller.ts) |
| POST | `/api/tickets/:ticketId/ready` | AdminAuthGuard | `ticketSchema` | [control.controller.ts](../../../apps/api/src/control/control.controller.ts) |
| GET | `/api/health/live` | pública (health) | ler método/serviço | [health.controller.ts](../../../apps/api/src/health/health.controller.ts) |
| GET | `/api/health/ready` | pública (health) | ler método/serviço | [health.controller.ts](../../../apps/api/src/health/health.controller.ts) |
| GET | `/api/runs` | AdminAuthGuard | ler método/serviço | [execution-results.controller.ts](../../../apps/api/src/orchestration/execution-results.controller.ts) |
| GET | `/api/runs/:runId` | AdminAuthGuard | ler método/serviço | [execution-results.controller.ts](../../../apps/api/src/orchestration/execution-results.controller.ts) |
| GET | `/api/runs/:runId/attempts/:attemptId/artifact` | AdminAuthGuard | ler método/serviço | [execution-results.controller.ts](../../../apps/api/src/orchestration/execution-results.controller.ts) |
| POST | `/api/internal/orchestration/attempts/:attemptId/result` | WorkerAuthGuard | `reportExecutionResultSchema` | [execution-results.controller.ts](../../../apps/api/src/orchestration/execution-results.controller.ts) |
| POST | `/api/internal/orchestration/attempts/:attemptId/artifact` | WorkerAuthGuard | `reportExecutionArtifactSchema` | [execution-results.controller.ts](../../../apps/api/src/orchestration/execution-results.controller.ts) |
| PUT | `/api/operation/scheduling` | AdminAuthGuard | `updateFactorySchedulingSchema` | [factory-scheduling.controller.ts](../../../apps/api/src/orchestration/factory-scheduling.controller.ts) |
| GET | `/api/operation` | AdminAuthGuard | ler método/serviço | [operation-status.controller.ts](../../../apps/api/src/orchestration/operation-status.controller.ts) |
| POST | `/api/internal/orchestration/claims` | WorkerAuthGuard | `orchestrationClaimSchema` | [orchestration.controller.ts](../../../apps/api/src/orchestration/orchestration.controller.ts) |
| POST | `/api/internal/orchestration/attempts/:attemptId/lease` | WorkerAuthGuard | `orchestrationClaimSchema` | [orchestration.controller.ts](../../../apps/api/src/orchestration/orchestration.controller.ts) |
| POST | `/api/internal/orchestration/attempts/:attemptId/checkpoint` | WorkerAuthGuard | `orchestrationStateSchema` | [orchestration.controller.ts](../../../apps/api/src/orchestration/orchestration.controller.ts) |
| POST | `/api/internal/orchestration/attempts/:attemptId/complete` | WorkerAuthGuard | `orchestrationStateSchema` | [orchestration.controller.ts](../../../apps/api/src/orchestration/orchestration.controller.ts) |
| POST | `/api/internal/orchestration/attempts/:attemptId/reconcile` | WorkerAuthGuard | `orchestrationReconcileResultSchema` | [orchestration.controller.ts](../../../apps/api/src/orchestration/orchestration.controller.ts) |
| GET | `/api/projects/:projectId/events` | AdminAuthGuard | ler método/serviço | [project-events.controller.ts](../../../apps/api/src/orchestration/project-events.controller.ts) |
| GET (SSE) | `/api/projects/:projectId/events/stream` | AdminAuthGuard | ler método/serviço | [project-events.controller.ts](../../../apps/api/src/orchestration/project-events.controller.ts) |
| POST | `/api/runs/:runId/resume` | AdminAuthGuard | `requestRunResumeSchema` | [run-control.controller.ts](../../../apps/api/src/orchestration/run-control.controller.ts) |
| POST | `/api/runs/:runId/control` | AdminAuthGuard | `requestRunControlSchema` | [run-control.controller.ts](../../../apps/api/src/orchestration/run-control.controller.ts) |
| GET | `/api/runs/:runId/delivery` | AdminAuthGuard | ler método/serviço | [run-delivery.controller.ts](../../../apps/api/src/orchestration/run-delivery.controller.ts) |
| POST | `/api/runs/:runId/approve-delivery` | AdminAuthGuard | `approveRunDeliverySchema` | [run-delivery.controller.ts](../../../apps/api/src/orchestration/run-delivery.controller.ts) |
| POST | `/api/runs/:runId/recover-finalization` | AdminAuthGuard | `requestRunRecoverySchema` | [run-recovery.controller.ts](../../../apps/api/src/orchestration/run-recovery.controller.ts) |
| POST | `/api/settings/github/onboarding` | AdminAuthGuard | `githubOnboardingSessionSchema` | [github-onboarding.controller.ts](../../../apps/api/src/settings/github-onboarding.controller.ts) |
| GET | `/api/settings/github/onboarding` | AdminAuthGuard | `githubOnboardingSessionListSchema` | [github-onboarding.controller.ts](../../../apps/api/src/settings/github-onboarding.controller.ts) |
| GET | `/api/settings/github/onboarding/:id/challenge` | AdminAuthGuard | `githubOnboardingChallengeSchema` | [github-onboarding.controller.ts](../../../apps/api/src/settings/github-onboarding.controller.ts) |
| POST | `/api/internal/github-onboarding/:id/start` | WorkerAuthGuard | `githubOnboardingSessionSchema`, `startGithubOnboardingSchema` | [github-onboarding.controller.ts](../../../apps/api/src/settings/github-onboarding.controller.ts) |
| POST | `/api/internal/github-onboarding/:id/challenge` | WorkerAuthGuard | `githubOnboardingSessionSchema`, `publishGithubOnboardingChallengeSchema` | [github-onboarding.controller.ts](../../../apps/api/src/settings/github-onboarding.controller.ts) |
| POST | `/api/internal/github-onboarding/:id/complete` | WorkerAuthGuard | `completeGithubOnboardingSchema`, `githubOnboardingSessionSchema` | [github-onboarding.controller.ts](../../../apps/api/src/settings/github-onboarding.controller.ts) |
| POST | `/api/settings/github/pull-requests` | AdminAuthGuard | `githubPullRequestSchema`, `prepareGithubPullRequestSchema` | [github-pull-request.controller.ts](../../../apps/api/src/settings/github-pull-request.controller.ts) |
| GET | `/api/settings/github/pull-requests` | AdminAuthGuard | `githubPullRequestListSchema` | [github-pull-request.controller.ts](../../../apps/api/src/settings/github-pull-request.controller.ts) |
| POST | `/api/settings/github/pull-requests/:id/approve` | AdminAuthGuard | `approveGithubPullRequestSchema`, `githubPullRequestSchema` | [github-pull-request.controller.ts](../../../apps/api/src/settings/github-pull-request.controller.ts) |
| POST | `/api/settings/github/pull-requests/:id/cancel` | AdminAuthGuard | `cancelGithubPullRequestSchema`, `githubPullRequestSchema` | [github-pull-request.controller.ts](../../../apps/api/src/settings/github-pull-request.controller.ts) |
| POST | `/api/internal/github-pull-requests/:id/start` | WorkerAuthGuard | `githubPullRequestSchema`, `startGithubPullRequestSchema` | [github-pull-request.controller.ts](../../../apps/api/src/settings/github-pull-request.controller.ts) |
| POST | `/api/internal/github-pull-requests/:id/complete` | WorkerAuthGuard | `completeGithubPullRequestSchema`, `githubPullRequestSchema` | [github-pull-request.controller.ts](../../../apps/api/src/settings/github-pull-request.controller.ts) |
| POST | `/api/settings/github/repository-verifications` | AdminAuthGuard | `githubRepositoryVerificationSchema` | [github-repository-verification.controller.ts](../../../apps/api/src/settings/github-repository-verification.controller.ts) |
| GET | `/api/settings/github/repository-verifications` | AdminAuthGuard | `githubRepositoryVerificationListSchema` | [github-repository-verification.controller.ts](../../../apps/api/src/settings/github-repository-verification.controller.ts) |
| POST | `/api/internal/github-repository-verifications/:id/start` | WorkerAuthGuard | `githubRepositoryVerificationSchema`, `startGithubRepositoryVerificationSchema` | [github-repository-verification.controller.ts](../../../apps/api/src/settings/github-repository-verification.controller.ts) |
| POST | `/api/internal/github-repository-verifications/:id/complete` | WorkerAuthGuard | `completeGithubRepositoryVerificationSchema`, `githubRepositoryVerificationSchema` | [github-repository-verification.controller.ts](../../../apps/api/src/settings/github-repository-verification.controller.ts) |
| POST | `/api/settings/github/verifications` | AdminAuthGuard | `githubVerificationSchema` | [github-verification.controller.ts](../../../apps/api/src/settings/github-verification.controller.ts) |
| GET | `/api/settings/github/verifications` | AdminAuthGuard | `githubVerificationListSchema` | [github-verification.controller.ts](../../../apps/api/src/settings/github-verification.controller.ts) |
| POST | `/api/internal/github-verifications/:id/start` | WorkerAuthGuard | `githubVerificationSchema`, `startGithubVerificationSchema` | [github-verification.controller.ts](../../../apps/api/src/settings/github-verification.controller.ts) |
| POST | `/api/internal/github-verifications/:id/complete` | WorkerAuthGuard | `completeGithubVerificationSchema`, `githubVerificationSchema` | [github-verification.controller.ts](../../../apps/api/src/settings/github-verification.controller.ts) |
| POST | `/api/settings/installations/:installationId/onboarding` | AdminAuthGuard | `providerOnboardingSessionSchema` | [provider-onboarding.controller.ts](../../../apps/api/src/settings/provider-onboarding.controller.ts) |
| POST | `/api/settings/onboarding/:id/authorization-code` | AdminAuthGuard | `providerAuthorizationCodeSchema` | [provider-onboarding.controller.ts](../../../apps/api/src/settings/provider-onboarding.controller.ts) |
| GET | `/api/settings/onboarding` | AdminAuthGuard | `providerOnboardingSessionListSchema` | [provider-onboarding.controller.ts](../../../apps/api/src/settings/provider-onboarding.controller.ts) |
| GET | `/api/settings/onboarding/:id/challenge` | AdminAuthGuard | `providerOnboardingChallengeSchema` | [provider-onboarding.controller.ts](../../../apps/api/src/settings/provider-onboarding.controller.ts) |
| POST | `/api/internal/provider-onboarding/:id/start` | WorkerAuthGuard | `providerOnboardingSessionSchema`, `startProviderOnboardingSchema` | [provider-onboarding.controller.ts](../../../apps/api/src/settings/provider-onboarding.controller.ts) |
| POST | `/api/internal/provider-onboarding/:id/challenge` | WorkerAuthGuard | `providerOnboardingSessionSchema`, `publishProviderOnboardingChallengeSchema` | [provider-onboarding.controller.ts](../../../apps/api/src/settings/provider-onboarding.controller.ts) |
| POST | `/api/internal/provider-onboarding/:id/authorization-code` | WorkerAuthGuard | `providerAuthorizationCodeResultSchema`, `startProviderOnboardingSchema` | [provider-onboarding.controller.ts](../../../apps/api/src/settings/provider-onboarding.controller.ts) |
| POST | `/api/internal/provider-onboarding/:id/complete` | WorkerAuthGuard | `completeProviderOnboardingSchema`, `providerOnboardingSessionSchema` | [provider-onboarding.controller.ts](../../../apps/api/src/settings/provider-onboarding.controller.ts) |
| POST | `/api/settings/installations/:installationId/verifications` | AdminAuthGuard | `providerVerificationSchema` | [provider-verification.controller.ts](../../../apps/api/src/settings/provider-verification.controller.ts) |
| GET | `/api/settings/verifications` | AdminAuthGuard | `providerVerificationListSchema` | [provider-verification.controller.ts](../../../apps/api/src/settings/provider-verification.controller.ts) |
| POST | `/api/internal/provider-verifications/:id/start` | WorkerAuthGuard | `providerVerificationSchema`, `startProviderVerificationSchema` | [provider-verification.controller.ts](../../../apps/api/src/settings/provider-verification.controller.ts) |
| POST | `/api/internal/provider-verifications/:id/complete` | WorkerAuthGuard | `completeProviderVerificationSchema`, `providerVerificationSchema` | [provider-verification.controller.ts](../../../apps/api/src/settings/provider-verification.controller.ts) |
| GET | `/api/settings` | AdminAuthGuard | `factorySettingsSchema` | [settings.controller.ts](../../../apps/api/src/settings/settings.controller.ts) |
| PUT | `/api/settings` | AdminAuthGuard | `factorySettingsSchema`, `updateFactorySettingsSchema` | [settings.controller.ts](../../../apps/api/src/settings/settings.controller.ts) |
| GET | `/api/internal/worker-settings` | WorkerAuthGuard | `workerConfigurationSnapshotSchema` | [worker-settings.controller.ts](../../../apps/api/src/settings/worker-settings.controller.ts) |
| POST | `/api/internal/workers/register` | WorkerAuthGuard | ler método/serviço | [worker-identity.controller.ts](../../../apps/api/src/worker-identity/worker-identity.controller.ts) |
| POST | `/api/internal/workers/:workerId/heartbeat` | WorkerAuthGuard | ler método/serviço | [worker-identity.controller.ts](../../../apps/api/src/worker-identity/worker-identity.controller.ts) |
| GET | `/api/workers` | AdminAuthGuard | `workerListSchema` | [worker-identity.controller.ts](../../../apps/api/src/worker-identity/worker-identity.controller.ts) |

Fonte runtime de todos os DTOs: [Zod compartilhado](../../../packages/contracts/src/index.ts). Auth inválida retorna 401; payload/UUID/cursor inválido 400; objeto ausente 404; conflito de versão/estado/evidência 409; pausa global de claim 423; capacidade conhecida 429 WRITER_BUSY. Não impor 422/403/404 de proposta ao código. Exemplos/efeitos e concorrência dos principais fluxos em [controle](01-controle.md) e [worker](02-worker.md).

## Rotas da conta — FAC-038

[API da conta do usuário](01-controle.md#api-da-conta-do-usuário--fac-038) cataloga /api/account, com cookie humano/AccountGuard e contratos próprios. Não substitui guards administrativos/worker anteriores. Catálogo machine-readable atualizado em [ROTAS-OBSERVADAS.json](ROTAS-OBSERVADAS.json).

FAC-039–041 ampliam a mesma camada com [rotas complementares](01-controle.md#recursos-complementares-da-conta--fac-039041), mantendo separado o canal interno do worker e o webhook assinado da cobrança.
