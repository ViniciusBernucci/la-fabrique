> Leitura: [← Anterior](01-objetivos.md) · [Índice didático](../02-INDEX.md) · [Próximo →](03-glossario.md)

> Integração DOC-MV-001: texto de direção/planejamento do pacote. O [AS-IS atual](../02-arquitetura/01-as-is.md) prevalece para implementação. Não confundir proposta com controle instalado.

# Escopo e evolução

Revisão documental: 2026-10-06 (America/Sao_Paulo). Arquitetura **PLANEJADA**; implementação e eficácia **NÃO VERIFICADAS** na origem do pacote.

| Dimensão | Direção documentada | Limite |
|---|---|---|
| Stack da fábrica | React + NestJS + worker Node/TypeScript; PostgreSQL/Redis | Versões fixadas em manifests/lockfile; ambiente efetivo registrado na validação |
| Topologia | VPS Linux única, inferência externa, um executor | Não implica HA nem contratação |
| Autenticação IA | Clientes oficiais, assinaturas primeiro | Preflight e elegibilidade por instalação |
| Tenancy | Cadeia tenant/projeto/run/attempt/instalação planejada | Piloto próprio/sintético; terceiros hostis bloqueados até gates |
| Entrega | código/checks/review/docs/aceite por revisão | Sem merge/deploy automáticos |
| Evolução | terceiro adapter/browser, mais workers, Temporal, API, perfil forte | Decisão posterior, sem ativação automática |

Uso pessoal exclusivo e Laravel/Angular eram limites históricos do PDF. A proposta v3 amplia desenho técnico para multi-tenant; isso não prova suporte dos fornecedores ao modelo comercial. O software piloto mantém sua própria stack. R0/R1 no primeiro experimento; R3/R4 excluídos.

## Proveniência

- [Direção v3](../99-historico/originais/output/le-fabrique-multitenant/README.md)
- [Arquitetura histórica](../99-historico/pdf-v2.1/recuperados/documentacoes/arquitetura/ARQUITETURA.md)
