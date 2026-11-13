> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../sandbox/00-README.md)

# Executor e checkout

IMPLEMENTADO no código base `a2cc5e0`; verificação operacional NÃO VERIFICADA nesta migração. Responsabilidade lógica, não nova classe/microserviço.

## Finalidade e limites

Preparar workspace confiável, impor lifecycle e preservar bytes antes de liberar autoridade.

## Fluxo e comportamento

SHA/URL HTTPS/host allowlisted → checkout detached → worktree → execução/checks → stop → snapshot → coleta validada.

Exemplo didático: ticket com critério “validar um campo” só avança com a base/definição/checks aprovados; request inválido não vira comando autorizado. O exemplo não é execução desta migração.

## Estados, dados e regras

Não há serviço chamado ExecutorManager: responsabilidades distribuídas no worker/runtime. Keyring oficial para checkout, sem credencial em job ou artefato. Veja [estados por entidade](../../05-contratos/schemas/01-workflow.md) e [contratos canônicos](../../05-contratos/00-README.md).

## Falhas, recuperação e segurança

Payload/versão/base inválidos são recusados antes de efeito pertinente; auth/cota não autorizam fallback pago. Writer unknown permanece bloqueado; não repetir execução ou liberar lease por idade. Credenciais não entram em DTO/contexto/log/artefato; API não executa código do projeto. Limites de cada contrato e gates de provider continuam necessários.

## Código e evidência

[apps/worker/src/repository-checkout.ts](../../../apps/worker/src/repository-checkout.ts) e [packages/runtime/src/workspace-manager.ts](../../../packages/runtime/src/workspace-manager.ts). Testes versionados ao lado dessas implementações e evidências por FAC-007/012J/L no [histórico](../../09-entregas/00-README.md). Não reexecutados aqui. [ADR-003 aceito](../../06-decisoes/ADR-003-stack-typescript.md) e [feature relacionada](../../04-features/04-recovery-operation.md).

## Limitações e direção posterior

[AS-IS versus alvo](../../02-arquitetura/01-as-is.md) separa controles atuais de tenancy/RLS/broker propostos. Presença de implementação/teste versionado não comprova instalação ou eficácia sob UID de serviço.
