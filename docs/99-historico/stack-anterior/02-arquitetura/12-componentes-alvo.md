> Leitura: [← Anterior](11-containers-alvo.md) · [Índice didático](../02-INDEX.md) · [Próximo →](13-fluxos-alvo.md)

> Proposta posterior do pacote, não estado implementado. Conciliação: [AS-IS](01-as-is.md). Sem liberação de provider/runtime por este diagrama.

# C4 nível 3 — componentes

Revisão documental: 2026-10-06 (America/Sao_Paulo). Arquitetura **PLANEJADA**; implementação e eficácia **NÃO VERIFICADAS** na origem do pacote.

```mermaid
flowchart TD
  subgraph API[NestJS — plano de controle planejado]
    AUTH[Auth / Tenant / Project grants] --> PROJ[Projects / Tickets]
    PROJ --> RUN[Runs / Orchestrator]
    RUN --> CTX[Context Builder]
    RUN --> PM[Provider Manager / Router / RuntimeGuard]
    RUN --> OUT[Outbox / Worker Protocol]
    RUN --> DOC[Documentation Gate]
    DOC --> APP[Approvals por revisão]
    RUN --> OBS[Usage / Ledger / Auditoria]
  end
  subgraph EXEC[Zona de execução planejada]
    OUT --> WRK[Worker coordinator]
    WRK --> EX[Executor / Sandbox Driver / Artifact collector]
    EX --> RT[Provider Adapter / Runtime isolado]
    RT --> BROKER[Tool Broker / capability validator]
    BROKER --> S[Sandbox / checks / outputs]
  end
```

Auth resolve autoridade antes de qualquer recurso; Orchestrator coordena transições e não executa shell. Context Builder seleciona fontes. Provider Manager escolhe instalação elegível, sem troca entre tenants. Executor controla recursos e término. Broker possui apenas manifesto restrito da tentativa, sem credenciais administrativas. Gate verifica docs e evidência; Approval registra ação humana.

Nomes representam responsabilidades lógicas. Caminhos de código, classes e injeção NestJS devem ser preenchidos somente ao inspecionar o repo real. [Módulos](../03-modulos/00-README.md) detalham cada responsabilidade.
