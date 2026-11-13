> Leitura: [← Anterior](03-containers.md) · [Índice didático](../02-INDEX.md) · [Próximo →](05-fluxos.md)

# C4 nível 3 — componentes atuais

Base a2cc5e0. Monólito modular na API; responsabilidades do worker/runtime em processo separado. Caminhos reais, sem src/modules inventado.

```mermaid
flowchart TB
  subgraph API[apps/api/src]
    C[ControlModule / ControlService] -->|ticket READY transacional| O[OutboxDispatcher]
    G[AdminAuthGuard / WorkerAuthGuard] --> C
    G --> OR[OrchestrationService / writer e fence]
    S[SettingsModule] -->|config e integrações| O
    WI[WorkerIdentityModule] -->|heartbeat| OR
    OR --> DR[RunDeliveryService / aceite]
    EV[ProjectEvents / OperationStatus / Scheduling] --> OR
    O --> Q[(BullMQ)]
  end
  subgraph WK[apps/worker/src]
    Q --> M[main / execution.processor]
    M --> LC[LeaseGuard / ControlClient / ResultJournal]
    M --> CO[RepositoryCheckout / workflow compiler]
    CO --> DW[DeveloperWorkflow]
    DW --> AR[ConfiguredAgentRouter / ProviderIdentity]
    DW --> DG[DocumentationGate]
  end
  subgraph RT[packages/runtime/src]
    DW --> CB[ContextBuilder / RuntimeGuard]
    AR --> CA[CodexAdapter / ClaudeAdapter]
    DW --> SS[SandboxRunner / WorkspaceManager / SnapshotManager]
  end
```

Context Builder e execução do gate são no worker/runtime, não shell na API. API valida evidências/hashes para aceite. PostgreSQL fecha exclusão global; BullMQ lock não a substitui. Antigravity possui integração de login/status, sem adapter de execução. [Componentes alvo](12-componentes-alvo.md) acrescentam tenancy/broker/capabilities propostos.

## Componente de conta — FAC-038

AccountModule na API contém AccountAuthController/AccountController, AccountGuard e AccountService. Guarda identidade humana/configuração pessoal, membership/convites/sessões e auditoria no PostgreSQL; Redis é usado para limitar autenticação. React AccountPanel consome somente DTOs Zod sem credencial administrativa. AccountWorkService reutiliza serviços de domínio dentro da mesma transação do check de autorização; AccountDataService, AccountIdentityService, AccountEmailService e AccountBillingService cuidam dos recursos privados. AccountWorkerController entrega credencial GitHub apenas ao supervisor com lease/fence válidos; worker consulta configuração do projeto e revalida autoridade. Broker/isolamento hostil permanecem propostos. [Decisão](../06-decisoes/ADR-007-conta-usuario-e-sessoes.md).
