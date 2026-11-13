> Leitura: [Índice didático](../02-INDEX.md) · [Próximo →](01-especificacao-v2.3.md)

# Contratos: implementação e proposta

Fonte única executável: [Zod compartilhado](../../packages/contracts/src/index.ts) e [Prisma/migrations](../../apps/api/prisma/schema.prisma). Código prevalece para implementação; intenção aprovada/proposta é registrada separadamente. Base a2cc5e0, sem teste HTTP/DB/provider nesta migração.

- [Controle](api/01-controle.md), [catálogo completo de rotas físicas](api/00-catalogo-rotas.md), [worker](api/02-worker.md), [runtime](api/03-runtime.md).
- [Entidades](schemas/00-entidades.md), [estados por entidade](schemas/01-workflow.md), [checkpoint](schemas/03-checkpoint.md), [artefato/contexto](schemas/02-artifact-context.md).
- [Outbox/fila](filas/00-dispatch.md), [eventos](eventos/00-catalogo.md).
- [JobPackage v3 PROPOSTO](schemas/04-job-package.md), [ESPEC local integral com errata](01-especificacao-v2.3.md).

Cada contrato identifica auth/produtor/consumidores/request/response/erros/efeitos/concorrência/versão/exemplo/teste ou lacuna explícita. Não há OpenAPI completo nem schema do erro HTTP versionado; os controllers são evidência da rota real. Tenancy/broker/capabilities v3 não possuem schemas executáveis nesta base.

## Sequência de leitura deste capítulo

1. [especificacao-v2.3](01-especificacao-v2.3.md).

## Contratos propostos de memória e decisões

[Semântica compartilhada e gates](memoria-e-decisoes/00-contratos-propostos.md). Planejamento; não schemas executáveis ou APIs implementadas.
