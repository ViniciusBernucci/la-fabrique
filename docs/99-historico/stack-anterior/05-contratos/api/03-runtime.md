> Leitura: [← Anterior](02-worker.md) · [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../eventos/00-catalogo.md)

# Runtime implementado

Base a2cc5e0; conferência por leitura local, testes NOT_RUN nesta migração.

Interface RuntimeAdapter em packages/runtime/src/runtime-adapter.ts: execute(request,onEvent), getStatus, getCapabilities, getUsage e cancel. Resume não consta da interface atual; é intenção histórica, não método implementado. Consumers worker/router/workflow. runtimeExecutionRequestSchema v1 leva executionId/workspacePath/prompt/permissionMode/writablePaths/modelRequested nullable/limits. Result/status/events/usage validam Zod; modelo/uso podem ser null. CodexAdapter e ClaudeAdapter usam spawn/argv/stdin, com cancelamento de árvore e limites de output. Native resume não é exposto pela interface atual; handoff usa estado externo. Auth vem do cliente oficial e store privado por instalação, não do request. CONTEXT_TOO_LARGE/TOOL_DENIED/TIMEOUT/AUTH_REQUIRED/RATE_LIMITED e demais códigos efetivos estão no enum runtimeErrorCodeSchema. UNSUPPORTED_ISOLATION é direção posterior: não inventar enum físico. JSONL e finalMessage não comprovam broker. Testes codex/claude-adapter.spec.ts são fixtures e não preflight de assinatura nesta migração.

## Fonte, versão e compatibilidade

[Schemas Zod canônicos](../../../packages/contracts/src/index.ts), [AS-IS](../../02-arquitetura/01-as-is.md), [responsabilidades e código](../../03-modulos/00-README.md). Requests/responses completos pertencem ao schema, sem cópia concorrente. Alteração exige teste positivo/negativo, compatibilidade de consumidores e entrega em ticket próprio.
