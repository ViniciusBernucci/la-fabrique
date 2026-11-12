> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../runtime/00-README.md)

# Instalações e roteamento

IMPLEMENTADO no código base `a2cc5e0`; verificação operacional NÃO VERIFICADA nesta migração. Responsabilidade lógica, não nova classe/microserviço.

## Finalidade e limites

Resolver conta/modelo/adapter explicitamente pela configuração atual.

## Fluxo e comportamento

GET worker-settings → rota por função → instalação habilitada/AVAILABLE/catalogada → adapter privado. Alternativas limitadas só após término e revalidação.

Exemplo didático: ticket com critério “validar um campo” só avança com a base/definição/checks aprovados; request inválido não vira comando autorizado. O exemplo não é execução desta migração.

## Estados, dados e regras

FactorySettings/ProviderVerification/Onboarding. Cadastro não promove elegibilidade; Antigravity tem login/verificação, sem adapter de execução. Veja [estados por entidade](../../05-contratos/schemas/01-workflow.md) e [contratos canônicos](../../05-contratos/00-README.md).

## Falhas, recuperação e segurança

Payload/versão/base inválidos são recusados antes de efeito pertinente; auth/cota não autorizam fallback pago. Writer unknown permanece bloqueado; não repetir execução ou liberar lease por idade. Credenciais não entram em DTO/contexto/log/artefato; API não executa código do projeto. Limites de cada contrato e gates de provider continuam necessários.

## Código e evidência

[apps/worker/src/configured-agent-router.ts](../../../apps/worker/src/configured-agent-router.ts) e [apps/worker/src/provider-identity.ts](../../../apps/worker/src/provider-identity.ts). Testes versionados ao lado dessas implementações e evidências por FAC-010A/B/C/012V/Y/015/018 no [histórico](../../09-entregas/00-README.md). Não reexecutados aqui. [ADR-003 aceito](../../06-decisoes/ADR-003-stack-typescript.md) e [feature relacionada](../../04-features/05-provider-handoff.md).

## Limitações e direção posterior

[AS-IS versus alvo](../../02-arquitetura/01-as-is.md) separa controles atuais de tenancy/RLS/broker propostos. Presença de implementação/teste versionado não comprova instalação ou eficácia sob UID de serviço.
