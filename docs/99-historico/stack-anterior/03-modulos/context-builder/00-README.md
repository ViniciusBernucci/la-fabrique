> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo →](01-referencia-v2.3.md)

# Context Builder

IMPLEMENTADO no código base `a2cc5e0`; verificação operacional NÃO VERIFICADA nesta migração. Responsabilidade lógica, não nova classe/microserviço.

## Finalidade e limites

Enviar apenas fontes selecionadas com hashes e omissões explícitas.

## Fluxo e comportamento

Ordena paths → lê regulares dentro do workspace → exclui symlinks/segredos/binários/dependências → manifesto e hash → prompt por papel.

Exemplo didático: ticket com critério “validar um campo” só avança com a base/definição/checks aprovados; request inválido não vira comando autorizado. O exemplo não é execução desta migração.

## Estados, dados e regras

Não indexa todo repo/embeddings; limite e omissão produzem truncated. Hash prova bytes selecionados, não suficiência semântica. Veja [estados por entidade](../../05-contratos/schemas/01-workflow.md) e [contratos canônicos](../../05-contratos/00-README.md).

## Falhas, recuperação e segurança

Payload/versão/base inválidos são recusados antes de efeito pertinente; auth/cota não autorizam fallback pago. Writer unknown permanece bloqueado; não repetir execução ou liberar lease por idade. Credenciais não entram em DTO/contexto/log/artefato; API não executa código do projeto. Limites de cada contrato e gates de provider continuam necessários.

## Código e evidência

[packages/runtime/src/context-builder.ts](../../../packages/runtime/src/context-builder.ts) e [apps/worker/src/execution-profile.ts](../../../apps/worker/src/execution-profile.ts). Testes versionados ao lado dessas implementações e evidências por FAC-006/009/012I no [histórico](../../09-entregas/00-README.md). Não reexecutados aqui. [ADR-003 aceito](../../06-decisoes/ADR-003-stack-typescript.md) e [feature relacionada](../../04-features/02-ticket-execution.md).

## Limitações e direção posterior

[AS-IS versus alvo](../../02-arquitetura/01-as-is.md) separa controles atuais de tenancy/RLS/broker propostos. Presença de implementação/teste versionado não comprova instalação ou eficácia sob UID de serviço.

## Referência integral local

[Detalhes por incremento com nota editorial](01-referencia-v2.3.md). Evidências/contratos históricos permanecem completos; afirmações de estado vencidas são delimitadas pela auditoria.

## Sequência de leitura deste capítulo

1. [referencia-v2.3](01-referencia-v2.3.md).

## Evolução planejada em 2026-10-09

[Guia didático de Context Packs](../../02-arquitetura/memoria-e-decisoes/04-context-packs.md) e [plano](../../08-desenvolvimento/planos-memoria-e-decisoes/04-context-packs.md) propõem seleção, obrigatórios, expansão e cache. A descrição implementada acima permanece; não atribuir índice/grafo ao builder atual.
