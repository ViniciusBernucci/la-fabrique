> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../tickets/00-README.md)

# Projetos

IMPLEMENTADO no código base `a2cc5e0`; verificação operacional NÃO VERIFICADA nesta migração. Responsabilidade lógica, não nova classe/microserviço.

## Finalidade e limites

Configurar o repositório externo e a definição versionada antes de um READY.

## Fluxo e comportamento

Cadastro → base SHA → definição de paths/contexto/checks/documentação → snapshot READY. Definição pode estar ausente/null em rascunho; READY exige perfil aprovado.

Exemplo didático: ticket com critério “validar um campo” só avança com a base/definição/checks aprovados; request inválido não vira comando autorizado. O exemplo não é execução desta migração.

## Estados, dados e regras

Project e ProjectDefinition; edição usa expectedVersion; alterar comandos invalida aprovações. Não escolhe conta/modelo e não muda stack externa. Veja [estados por entidade](../../05-contratos/schemas/01-workflow.md) e [contratos canônicos](../../05-contratos/00-README.md).

## Falhas, recuperação e segurança

Payload/versão/base inválidos são recusados antes de efeito pertinente; auth/cota não autorizam fallback pago. Writer unknown permanece bloqueado; não repetir execução ou liberar lease por idade. Credenciais não entram em DTO/contexto/log/artefato; API não executa código do projeto. Limites de cada contrato e gates de provider continuam necessários.

## Código e evidência

[apps/api/src/control/control.service.ts](../../../apps/api/src/control/control.service.ts) e [apps/web/src/ProjectDefinitionPanel.tsx](../../../apps/web/src/ProjectDefinitionPanel.tsx). Testes versionados ao lado dessas implementações e evidências por FAC-001A/003A/012I no [histórico](../../09-entregas/00-README.md). Não reexecutados aqui. [ADR-003 aceito](../../06-decisoes/ADR-003-stack-typescript.md) e [feature relacionada](../../04-features/02-ticket-execution.md).

## Limitações e direção posterior

[AS-IS versus alvo](../../02-arquitetura/01-as-is.md) separa controles atuais de tenancy/RLS/broker propostos. Presença de implementação/teste versionado não comprova instalação ou eficácia sob UID de serviço.
