> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../controle/00-README.md)

# Aceite humano

IMPLEMENTADO no código base `a2cc5e0`; verificação operacional NÃO VERIFICADA nesta migração. Responsabilidade lógica, não nova classe/microserviço.

## Finalidade e limites

Concluir somente após confirmar a entrega exata.

## Fluxo e comportamento

GET delivery monta Markdown e digests; POST approve-delivery compara expectedVersion/attempt/deliveryDigest/result/bundle/stop/checks/docs; grava approval e DONE.

Exemplo didático: ticket com critério “validar um campo” só avança com a base/definição/checks aprovados; request inválido não vira comando autorizado. O exemplo não é execução desta migração.

## Estados, dados e regras

Run.approval JSON. Workflow AWAITING_HUMAN e Run VALIDATING são diferentes. Replay idêntico não duplica; stale/mismatch bloqueia. Veja [estados por entidade](../../05-contratos/schemas/01-workflow.md) e [contratos canônicos](../../05-contratos/00-README.md).

## Falhas, recuperação e segurança

Payload/versão/base inválidos são recusados antes de efeito pertinente; auth/cota não autorizam fallback pago. Writer unknown permanece bloqueado; não repetir execução ou liberar lease por idade. Credenciais não entram em DTO/contexto/log/artefato; API não executa código do projeto. Limites de cada contrato e gates de provider continuam necessários.

## Código e evidência

[apps/api/src/orchestration/run-delivery.service.ts](../../../apps/api/src/orchestration/run-delivery.service.ts) e [apps/api/src/orchestration/run-delivery.controller.ts](../../../apps/api/src/orchestration/run-delivery.controller.ts). Testes versionados ao lado dessas implementações e evidências por FAC-012Q/Z no [histórico](../../09-entregas/00-README.md). Não reexecutados aqui. [ADR-003 aceito](../../06-decisoes/ADR-003-stack-typescript.md) e [feature relacionada](../../04-features/03-documentation-approval.md).

## Limitações e direção posterior

[AS-IS versus alvo](../../02-arquitetura/01-as-is.md) separa controles atuais de tenancy/RLS/broker propostos. Presença de implementação/teste versionado não comprova instalação ou eficácia sob UID de serviço.
