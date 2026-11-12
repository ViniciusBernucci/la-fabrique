> Leitura: [← Anterior](10-contexto-alvo.md) · [Índice didático](../02-INDEX.md) · [Próximo →](12-componentes-alvo.md)

> Proposta posterior do pacote, não estado implementado. Conciliação: [AS-IS](01-as-is.md). Sem liberação de provider/runtime por este diagrama.

# C4 nível 2 — containers

Revisão documental: 2026-10-06 (America/Sao_Paulo). Arquitetura **PLANEJADA**; implementação e eficácia **NÃO VERIFICADAS** na origem do pacote.

```mermaid
flowchart TB
  U[Navegador] -->|HTTPS| WEB[React Web]
  WEB -->|API autenticada| API[NestJS / controle modular]
  subgraph VPS[VPS Linux única planejada]
    WEB
    API -->|transações e outbox| PG[(PostgreSQL)]
    API -->|dispatch publicado e cache| REDIS[(Redis)]
    WORK[Worker Node/TypeScript] -->|claim / heartbeat / eventos| API
    WORK -->|lifecycle restrito| EX[Executor Manager]
    EX -->|sessão por tentativa| PR[Provider Runtime por tenant/provider]
    AUTH[Auth oficial segregada] -->|somente cliente oficial| PR
    PR -->|operações mediadas| BR[Tool Broker por tentativa]
    BR -->|tools autorizadas| SB[Sandbox efêmera do projeto]
    EX -->|preparar / parar / coletar| SB
    SB -->|resultados não confiáveis validados| EX
    EX -->|artefatos e eventos limitados| WORK
  end
  PR -->|inferência por destino autorizado| AI[Fornecedor externo]
  API -->|backup / artefatos por ownership| EXT[Storage externo]
```

Redis recebe publicação derivada da outbox; a outbox persiste no PostgreSQL. Nenhuma seta concede sandbox/broker acesso a PostgreSQL/Redis/API geral/auth/daemon. As unidades C4 podem corresponder a processos/serviços, não exigem microserviços. React pode ser servido pelo proxy; forma real de empacotamento permanece pendente.

Um runtime por tenant/provider e sessão por attempt é requisito planejado. Cliente incapaz de mediar tools fica DISABLED/UNSUPPORTED_ISOLATION. Backup externo não requer segunda VPS de execução.

## Proveniência

- [Base v3](07-fronteiras-e-invariantes.md)
- [Runtime](../03-modulos/runtime/00-README.md)
