> Leitura: [← Anterior](04-componentes.md) · [Índice didático](../02-INDEX.md) · [Próximo →](06-dados.md)

# Fluxo implementado e handoff

Base a2cc5e0, leitura de ControlService/OrchestrationService/worker/workflow/RunDeliveryService; software NOT_RUN aqui.

```mermaid
sequenceDiagram
  participant U as Operador
  participant A as API NestJS
  participant DB as PostgreSQL / outbox
  participant Q as Redis / BullMQ
  participant W as Worker
  participant R as Cliente oficial
  U->>A: POST ticket/ready + expectedVersion
  A->>DB: ticket + ExecutionSpecification + outbox
  A->>Q: dispatcher publica job estável
  Q->>W: job
  W->>A: claim autenticado
  A->>DB: Run/Attempt, writer global e fence
  W->>W: checkout, contexto, perfil e lease
  W->>R: Developer, stop, checks, Reviewer separado
  W->>W: docs, snapshot, journal e hashes
  W->>A: result/artifact/checkpoint/complete
  A->>DB: VALIDATING preserva evidências
  U->>A: approve-delivery por digest/versão
  A->>DB: approval e DONE
```

Handoff Y exige stop conhecido/snapshot, cria outra worktree e revalida alternativa, conservando autoridade da Attempt/lease/fence durante o workflow. Resume administrativo é nova tentativa/fence a partir de snapshot parado. Recuperação de finalização reenvia journal/reconcile sem IA. [Estados](../05-contratos/schemas/01-workflow.md), [handoff](../07-operacao/04-handoff.md), [fluxos alvo](13-fluxos-alvo.md).
