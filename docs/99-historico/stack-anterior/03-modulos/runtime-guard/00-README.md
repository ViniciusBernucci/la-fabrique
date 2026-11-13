> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../documentation-gate/00-README.md)

# Limites de execução

IMPLEMENTADO no código base `a2cc5e0`; verificação operacional NÃO VERIFICADA nesta migração. Responsabilidade lógica, não nova classe/microserviço.

## Finalidade e limites

Conter tempo, tentativas, falhas e handoffs com assinatura como modo elegível.

## Fluxo e comportamento

Validar política → autorizar chamada → observar término/falha → bloquear nova chamada se limite/cancel/unknown.

Exemplo didático: ticket com critério “validar um campo” só avança com a base/definição/checks aprovados; request inválido não vira comando autorizado. O exemplo não é execução desta migração.

## Estados, dados e regras

Orçamento API zero, extras/fallback false. RuntimeGuard limita workflow; cgroups/lifecycle impõem recursos. Não mede quota externa por suposição. Veja [estados por entidade](../../05-contratos/schemas/01-workflow.md) e [contratos canônicos](../../05-contratos/00-README.md).

## Falhas, recuperação e segurança

Payload/versão/base inválidos são recusados antes de efeito pertinente; auth/cota não autorizam fallback pago. Writer unknown permanece bloqueado; não repetir execução ou liberar lease por idade. Credenciais não entram em DTO/contexto/log/artefato; API não executa código do projeto. Limites de cada contrato e gates de provider continuam necessários.

## Código e evidência

[packages/runtime/src/runtime-guard.ts](../../../packages/runtime/src/runtime-guard.ts) e [apps/worker/src/lease-guard.ts](../../../apps/worker/src/lease-guard.ts). Testes versionados ao lado dessas implementações e evidências por FAC-006/012K/Y no [histórico](../../09-entregas/00-README.md). Não reexecutados aqui. [ADR-003 aceito](../../06-decisoes/ADR-003-stack-typescript.md) e [feature relacionada](../../04-features/05-provider-handoff.md).

## Limitações e direção posterior

[AS-IS versus alvo](../../02-arquitetura/01-as-is.md) separa controles atuais de tenancy/RLS/broker propostos. Presença de implementação/teste versionado não comprova instalação ou eficácia sob UID de serviço.
