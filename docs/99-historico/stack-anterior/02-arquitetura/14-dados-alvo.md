> Leitura: [← Anterior](13-fluxos-alvo.md) · [Índice didático](../02-INDEX.md) · [Próximo →](15-referencia-v2.3.md)

> Proposta posterior do pacote, não estado implementado. Conciliação: [AS-IS](01-as-is.md). Sem liberação de provider/runtime por este diagrama.

# Arquitetura de dados

Revisão documental: 2026-10-06 (America/Sao_Paulo). Arquitetura **PLANEJADA**; implementação e eficácia **NÃO VERIFICADAS** na origem do pacote.

PostgreSQL é a fonte de workflow/identidade, com outbox transacional, optimistic version e eventos deduplicados. Redis serve fila/cache e não oferece RLS. Artefatos imutáveis por hash têm ownership tenant/projeto/run/attempt e storage próprio.

```mermaid
erDiagram
  Tenant ||--o{ Project : possui
  Tenant ||--o{ Membership : autoriza
  Project ||--o{ Ticket : organiza
  Ticket ||--o{ Run : executa
  Run ||--o{ Step : ordena
  Step ||--o{ Attempt : tenta
  Tenant ||--o{ ProviderInstallation : possui
  ProviderInstallation ||--o{ Attempt : atende
  Attempt ||--o{ Artifact : produz
  Attempt ||--o{ Checkpoint : preserva
  Run ||--o{ Approval : recebe
```

Diagrama conceitual, sem cardinalidades físicas verificadas nem migrations executadas. [Entidades canônicas](../05-contratos/schemas/00-entidades.md) descrevem campos; [isolamento](../03-modulos/tenant-isolation/00-README.md) descreve integridade e RLS. Dinheiro decimal; uso/custo ausentes ficam unknown. Não migrar órfãos para tenant arbitrário.
