> Leitura: [← Anterior](01-workflow.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](03-checkpoint.md)

# Artefatos e contexto

Base a2cc5e0; conferência por leitura local, testes NOT_RUN nesta migração.

workspaceSnapshotManifestSchema e executionArtifactSchema definem patch Git binário e untracked base64/modos/tamanhos/SHA-256; verifyExecutionArtifact confere bytes além de schema. Limites 8 MiB JSON/6 MiB raw, sem truncar. ArtifactReader faz leitura bounded e valida origem; transporte worker→API autenticado/fenced, leitura administrativa confere vínculo exato. ContextManifest em context-builder.ts ordena fontes e registra hash, bytes/omissões/truncated, sem persistir prompt automaticamente. Contexto não varre todo repo nem usa embeddings. Retention 30/90 dias e backup 7/4 são propostas, não cron automático comprovado. Ownership por tenant/caches segregados são alvo posterior.

## Fonte, versão e compatibilidade

[Schemas Zod canônicos](../../../packages/contracts/src/index.ts), [AS-IS](../../02-arquitetura/01-as-is.md), [responsabilidades e código](../../03-modulos/00-README.md). Requests/responses completos pertencem ao schema, sem cópia concorrente. Alteração exige teste positivo/negativo, compatibilidade de consumidores e entrega em ticket próprio.
