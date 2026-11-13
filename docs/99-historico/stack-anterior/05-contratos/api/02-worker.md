> Leitura: [← Anterior](01-controle.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](03-runtime.md)

# Protocolo interno implementado

Base a2cc5e0. Consumers: ControlClient do worker e API NestJS. Bearer WORKER_API_TOKEN por WorkerAuthGuard; AdminAuthGuard protege apenas projeções administrativas. Nginx público bloqueia internal; runtime/sandbox/broker não recebem esse token. Protocolo pertence à fábrica, não é API de fornecedor.

| Rota POST (prefixo /api) | Request → response | Efeito / invariantes |
|---|---|---|
| /internal/workers/register | workerRegistrationSchema → workerSchema | Registra identidade com capacidades declaradas; não prova login/provider |
| /internal/workers/:id/heartbeat | sem body → workerHeartbeatSchema | Atualiza observação de saúde; não renova lease da Attempt |
| /internal/orchestration/claims | orchestrationClaimRequestSchema → orchestrationClaimSchema | Snapshot/worker/event/IDs/base/versões/lease 15–300s; transação/exclusão global/fence |
| /internal/orchestration/attempts/:id/lease | orchestrationLeaseRequestSchema → orchestrationClaimSchema | workerId/fencingToken/leaseDurationMs; ownership atual, controlAction nullable |
| /internal/orchestration/attempts/:id/checkpoint | orchestrationCheckpointRequestSchema → orchestrationStateSchema | base/code SHA, snapshotId/patchHash nullable, reason, stoppedConfirmed; valida fence |
| /internal/orchestration/attempts/:id/result | reportExecutionResultSchema | Resultado estrito/digest imutável por tentativa, não libera writer |
| /internal/orchestration/attempts/:id/artifact | reportExecutionArtifactSchema | Bundle/hash/bytes exatos, vínculo ao relatório; não confundir schema com integridade |
| /internal/orchestration/attempts/:id/complete | orchestrationCompleteRequestSchema → orchestrationStateSchema | outcome fenced; stop conhecido necessário; não é aceite humano |
| /internal/orchestration/attempts/:id/reconcile | orchestrationReconcileRequestSchema → orchestrationReconcileResultSchema | Recupera finalização conhecida; estado nullable, sem repetir cliente |

GET /api/internal/worker-settings retorna workerConfigurationSnapshotSchema sem segredos. Login/verificação/GitHub possuem start/challenge/complete internos próprios no [catálogo](00-catalogo-rotas.md). Nenhum endpoint genérico de shell ou events proposto foi inventado.

## Erros, efeitos e idempotência

400 contrato inválido; 401 auth; 404 recurso ausente; 409 lease/fence/estado; 423 pausa; 429 WRITER_BUSY só em recusa pré-autoridade conhecida. Worker pode adiar essa admissão sem consumir attemptsMade; falha de lease, execução ou commit ambíguo propaga, não vira retry. Replay de job/claim precisa corresponder ao mesmo evento/intenção; terminal divergente não pode ser sobrescrito.

Registro inicial default 30 tentativas com 1s; só rede/5xx têm retry. Heartbeat configurável 1–60s, default 5s; lease default 90s. Três falhas seguidas de heartbeat encerram worker. Lease expirada não prova stop; stoppedConfirmed externo e exclusão global permanecem requisitos.

## Exemplo sintético de response de claim

```json
{"runId":"11111111-1111-4111-8111-111111111111","attemptId":"22222222-2222-4222-8222-222222222222","workerId":"33333333-3333-4333-8333-333333333333","fencingToken":1,"leaseExpiresAt":"2026-10-07T15:00:00.000Z","replayed":false}
```

Ilustra orchestrationClaimSchema; não é lease real nem prova de execução. Request completo precisa do snapshot do evento e IDs/revisão reais, nunca campos inventados por modelo.

## Versão, limites e evidência

schemaVersion=1 nos jobs/relatórios; versão otimista e fence são distintos. Bundle até 8 MiB JSON/6 MiB raw, sem truncar; API body até 8 MiB+4096 bytes para envelope. Testes locais versionados em orchestration/execution-results/artifacts specs e execution.processor/control-client/lease-guard specs; não reexecutados aqui. Backup/restore devem preservar hashes/journal/enum/índice, sem copiar auth informalmente.

[Controllers](00-catalogo-rotas.md), [schemas reais](../../../packages/contracts/src/index.ts), [ControlClient](../../../apps/worker/src/control-client.ts), [config](../../../apps/worker/src/config.ts).
