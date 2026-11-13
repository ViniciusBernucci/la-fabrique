> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../runs/00-README.md)

# Tickets

IMPLEMENTADO no código base `a2cc5e0`; verificação operacional NÃO VERIFICADA nesta migração. Responsabilidade lógica, não nova classe/microserviço.

## Finalidade e limites

Transformar objetivo em critérios verificáveis e intenção executável imutável.

## Fluxo e comportamento

POST projeto/tickets cria DRAFT; POST tickets/:id/ready exige expectedVersion, definição e base SHA. Ticket e ticket.ready.v1 são atômicos.

Exemplo didático: ticket com critério “validar um campo” só avança com a base/definição/checks aprovados; request inválido não vira comando autorizado. O exemplo não é execução desta migração.

## Estados, dados e regras

Ticket; chave ticket:{id}:ready deduplica evento. Critérios não podem ser substituídos por output de modelo. Veja [estados por entidade](../../05-contratos/schemas/01-workflow.md) e [contratos canônicos](../../05-contratos/00-README.md).

## Falhas, recuperação e segurança

Payload/versão/base inválidos são recusados antes de efeito pertinente; auth/cota não autorizam fallback pago. Writer unknown permanece bloqueado; não repetir execução ou liberar lease por idade. Credenciais não entram em DTO/contexto/log/artefato; API não executa código do projeto. Limites de cada contrato e gates de provider continuam necessários.

## Código e evidência

[apps/api/src/control/control.controller.ts](../../../apps/api/src/control/control.controller.ts) e [apps/api/src/control/control.service.ts](../../../apps/api/src/control/control.service.ts). Testes versionados ao lado dessas implementações e evidências por FAC-003/003A/012A no [histórico](../../09-entregas/00-README.md). Não reexecutados aqui. [ADR-003 aceito](../../06-decisoes/ADR-003-stack-typescript.md) e [feature relacionada](../../04-features/02-ticket-execution.md).

## Limitações e direção posterior

[AS-IS versus alvo](../../02-arquitetura/01-as-is.md) separa controles atuais de tenancy/RLS/broker propostos. Presença de implementação/teste versionado não comprova instalação ou eficácia sob UID de serviço.

## Painel demonstrativo de tarefas

FAC-033 acrescenta interface Kanban/lista com edição manual e vínculo por ID com agentes do projeto. O código [tasks-model.ts](../../../apps/web/src/tasks-model.ts) valida responsável habilitado no mesmo projeto, critérios e dependências sem ciclos. [TasksPanel.tsx](../../../apps/web/src/TasksPanel.tsx) mantém demonstração em armazenamento local versionado e consulta agentes cadastrados somente com sessão administrativa.

Essa interface não altera TicketStatus, o snapshot executável, READY/outbox ou aceites reais. A sequência planejada do onboarding é projeto → agentes → tarefas → atribuição. A simulação implementa essa ordem com exemplos identificados; persistência compartilhada e integração com execução são pendências explícitas. [Comportamento](../../04-features/01-control-settings.md#painel-tarefas--demonstração-fac-033).
