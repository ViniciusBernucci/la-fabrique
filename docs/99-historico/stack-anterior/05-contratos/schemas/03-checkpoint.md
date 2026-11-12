> Leitura: [← Anterior](02-artifact-context.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](04-job-package.md)

# Checkpoint e snapshot

Base a2cc5e0; conferência por leitura local, testes NOT_RUN nesta migração.

orchestrationCheckpointRequestSchema: workerId/fencingToken/baseRevision/codeRevision nullable/snapshotId nullable/patchHash nullable/reason/stoppedConfirmed. Checkpoint físico único por Attempt. Motivos PROGRESS, WAITING_PROVIDER, PAUSED, OPERATOR_PAUSED, CANCELLED, FAILED, COMPLETED. WorkspaceSnapshotManifest separado preserva base/head, patch/untracked/modos/tamanhos/hashes; journal guarda resultado parado antes da API. Sem auth/home/session/transcript. Restore valida base limpa exata/paths/hashes e não aceita symlink; quiescência externa é obrigatória, não consequência do hash. Handoff humano usa template; não confundir seu texto com schema executável. Testes snapshot/workspace/lease/journal/reconcile versionados; NOT_RUN aqui.

## Fonte, versão e compatibilidade

[Schemas Zod canônicos](../../../packages/contracts/src/index.ts), [AS-IS](../../02-arquitetura/01-as-is.md), [responsabilidades e código](../../03-modulos/00-README.md). Requests/responses completos pertencem ao schema, sem cópia concorrente. Alteração exige teste positivo/negativo, compatibilidade de consumidores e entrega em ticket próprio.
