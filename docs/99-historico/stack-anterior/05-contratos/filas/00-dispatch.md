> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../../06-decisoes/00-README.md)

# Outbox e fila

Base a2cc5e0; conferência por leitura local, testes NOT_RUN nesta migração.

OutboxEvent no PostgreSQL grava intenção atomicamente; dispatcher Node publica BullMQ com jobId do eventId e preserva removeOnComplete/removeOnFail=false. execution fila le-fabrique.execution; probe e consumers provider/GitHub têm filas próprias no main.ts. orchestrationJobSchema v1 leva eventId/ticketId/projectId/ticketVersion/baseRevision/projectDefinitionVersion/executionSpecification/resumeFrom opcional. Campos nullable existem para leitura legada, mas dispatcher rejeita base/snapshot inadequados para execução. Consumer revalida snapshot; claim transacional/leasing/fence/global stop são independentes de lock BullMQ. Retentativas não oferecem exactly-once. Pausa default conservadora; 429 pré-claim conhecido adia, erro ambíguo propaga.

## Fonte, versão e compatibilidade

[Schemas Zod canônicos](../../../packages/contracts/src/index.ts), [AS-IS](../../02-arquitetura/01-as-is.md), [responsabilidades e código](../../03-modulos/00-README.md). Requests/responses completos pertencem ao schema, sem cópia concorrente. Alteração exige teste positivo/negativo, compatibilidade de consumidores e entrega em ticket próprio.
