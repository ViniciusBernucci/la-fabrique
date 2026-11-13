> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../context-builder/00-README.md)

# Orquestração e outbox

IMPLEMENTADO no código base `a2cc5e0`; verificação operacional NÃO VERIFICADA nesta migração. Responsabilidade lógica, não nova classe/microserviço.

## Finalidade e limites

Coordenar persistência/dispatch sem executar cliente ou shell na API.

## Fluxo e comportamento

READY persiste snapshot; dispatcher publica jobId estável; claim transacional valida autoridade; mutações fenced; replay reconcilia.

Exemplo didático: ticket com critério “validar um campo” só avança com a base/definição/checks aprovados; request inválido não vira comando autorizado. O exemplo não é execução desta migração.

## Estados, dados e regras

OutboxEvent no PostgreSQL, BullMQ no Redis. Pausa global default true; recusa de capacidade pré-claim pode adiar job; falha ambígua não vira retry. Veja [estados por entidade](../../05-contratos/schemas/01-workflow.md) e [contratos canônicos](../../05-contratos/00-README.md).

## Falhas, recuperação e segurança

Payload/versão/base inválidos são recusados antes de efeito pertinente; auth/cota não autorizam fallback pago. Writer unknown permanece bloqueado; não repetir execução ou liberar lease por idade. Credenciais não entram em DTO/contexto/log/artefato; API não executa código do projeto. Limites de cada contrato e gates de provider continuam necessários.

## Código e evidência

[apps/api/src/orchestration/outbox-dispatcher.ts](../../../apps/api/src/orchestration/outbox-dispatcher.ts) e [apps/api/src/orchestration/global-writer-guard.ts](../../../apps/api/src/orchestration/global-writer-guard.ts). Testes versionados ao lado dessas implementações e evidências por FAC-008/012AC–AH no [histórico](../../09-entregas/00-README.md). Não reexecutados aqui. [ADR-003 aceito](../../06-decisoes/ADR-003-stack-typescript.md) e [feature relacionada](../../04-features/02-ticket-execution.md).

## Limitações e direção posterior

[AS-IS versus alvo](../../02-arquitetura/01-as-is.md) separa controles atuais de tenancy/RLS/broker propostos. Presença de implementação/teste versionado não comprova instalação ou eficácia sob UID de serviço.
