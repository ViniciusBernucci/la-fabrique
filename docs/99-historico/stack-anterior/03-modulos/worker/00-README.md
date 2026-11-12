> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../executor-manager/00-README.md)

# Worker

IMPLEMENTADO no código base `a2cc5e0`; verificação operacional NÃO VERIFICADA nesta migração. Responsabilidade lógica, não nova classe/microserviço.

## Finalidade e limites

Executar trabalho em processo separado do controle.

## Fluxo e comportamento

Registra identidade → heartbeat → consumers de integrações; consumer execution opt-in compõe checkout/perfil/workflow/journal/lease. Shutdown aborta e aguarda.

Exemplo didático: ticket com critério “validar um campo” só avança com a base/definição/checks aprovados; request inválido não vira comando autorizado. O exemplo não é execução desta migração.

## Estados, dados e regras

WorkerIdentity e jobs estritos. Concorrência de execução 1; heartbeat default 5s, lease default 90s. Três falhas de heartbeat encerram. Veja [estados por entidade](../../05-contratos/schemas/01-workflow.md) e [contratos canônicos](../../05-contratos/00-README.md).

## Falhas, recuperação e segurança

Payload/versão/base inválidos são recusados antes de efeito pertinente; auth/cota não autorizam fallback pago. Writer unknown permanece bloqueado; não repetir execução ou liberar lease por idade. Credenciais não entram em DTO/contexto/log/artefato; API não executa código do projeto. Limites de cada contrato e gates de provider continuam necessários.

## Código e evidência

[apps/worker/src/main.ts](../../../apps/worker/src/main.ts) e [apps/worker/src/control-client.ts](../../../apps/worker/src/control-client.ts). Testes versionados ao lado dessas implementações e evidências por FAC-004/012L/OPS-005 no [histórico](../../09-entregas/00-README.md). Não reexecutados aqui. [ADR-003 aceito](../../06-decisoes/ADR-003-stack-typescript.md) e [feature relacionada](../../04-features/02-ticket-execution.md).

## Limitações e direção posterior

[AS-IS versus alvo](../../02-arquitetura/01-as-is.md) separa controles atuais de tenancy/RLS/broker propostos. Presença de implementação/teste versionado não comprova instalação ou eficácia sob UID de serviço.
