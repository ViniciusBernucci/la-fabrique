> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../../04-features/00-README.md)

# Broker, ferramentas e capacidades temporárias

Data: 2026-10-06 (America/Sao_Paulo). Status da arquitetura e controles: **PLANEJADO**. Implementação e eficácia: **NÃO VERIFICADAS**. Este documento especifica trabalho futuro; não comprova instalação, configuração ou execução.

## Autoridade fora do modelo

Broker recebe contexto de identidade do canal autenticado e envelope emitido pelo Executor, não de tenant_id alegado no corpo. Não acessa banco/Redis ou CredentialStore. Mantém apenas manifesto imutável de uma tentativa e mecanismo de revogação. Ferramentas não resolvem recurso global por ID e não fazem chamadas à API geral. Model output é pedido de operação, nunca comando administrativo confiável.

| Ferramenta | Escopo |
|---|---|
| read_project_file / search_project_code | path relativo, allowlist e limites de tamanho, só projeto da tentativa |
| apply_project_patch | developer, paths permitidos, limites; sem alterar política montada RO |
| git_diff | snapshot local, sem hooks/remote/credential helpers |
| run_registered_check | profile ID cadastrado; argv fixo, timeout; roda no sandbox |
| collect_registered_artifact | output autorizado e validado, sem listing externo |

Shell arbitrário no host/runtime, SQL genérico, HTTP genérico, filesystem global, criação de container, alteração de tenant, secret_read e gestão de providers não existem no catálogo. Um comando cadastrado pode executar código hostil do repo; seu isolamento é garantido pelo sandbox, não pelo nome “teste”. Argumentos dinâmicos aceitos apenas via schema limitado; usar spawn/execFile e stdin, nunca concatenar prompt em shell.

## Capability planejada

Campos: schema_version, issuer, audience específica, tenant/project/run/attempt, installation_id quando pertinente, workspace_id, policy_version/hash, credential_generation quando vinculada, fencing_token, operações/path/profile IDs, nbf/exp, nonce/jti e quotas. Assinatura/MAC do emissor confiável; verificador fora da IA. Prazo nunca ultrapassa lease/deadline da tentativa (proposta inicial até 60s por capability, renovar apenas com lease vivo). Chave do emissor e revocation channel nunca são montados no sandbox. Preferir handle opaco ligado ao canal por tentativa, não bearer serializado no prompt.

Validar audience, assinatura, tenant/projeto, deadline, status revogado e fencing atual a CADA operação. JTI de operações não idempotentes tem dedupe; operações repetíveis respeitam quota/lease. Vincular capability ao canal/processo da tentativa impede uso em outro sandbox mesmo se handle vazar. Token expirado, versão de política antiga ou revogação indisponível gera negação. Cache de autorização não prolonga lease.

Capabilities não são credenciais de provider. Git/upload externo são preferencialmente feitos por integração confiável após validação, com token temporário limitado a repo/branch/objeto exato, fora do sandbox. Nunca emitir token global Git/S3/controle a código. ACL externa precisa impor escopo real; nome de prefixo não é restrição suficiente. Se armazenamento não permite restrição real, executar upload no broker de integração, sem entregar token.

## Erros e auditoria

DENIED_TENANT, DENIED_PROJECT, PATH_OUTSIDE_WORKSPACE, CAPABILITY_EXPIRED, CAPABILITY_REVOKED, STALE_FENCE, UNSUPPORTED_TOOL e POLICY_UNAVAILABLE são códigos internos sanitizados. UI não revela existência de objeto de outro cliente. Auditar operação/IDs de escopo/revisão/resultado sem token ou conteúdo integral. Evento “finished” do modelo não autoriza DONE nem novas capacidades.


## Origem desta edição

[Versão original preservada](../../99-historico/originais/output/le-fabrique-multitenant/documentacoes/seguranca/CAPABILITIES-E-TOOLS.md). Migração editorial de paths em 2026-10-06; conteúdo de engenharia continua proposto.

## Localização, relações e validação

Código da fábrica existe; não foi encontrada implementação de tool-broker em a2cc5e0. Nenhum teste operacional desse controle foi executado nesta migração. Consulte [ADRs](../../06-decisoes/00-README.md), [features](../../04-features/00-README.md) e [SEC-01–14](../../08-desenvolvimento/12-testes-seguranca.md). Estados/contratos citados são planejados; registrar código/revisão/checks por controle quando implementado.

## Conciliação no repo real

[AS-IS conferido](../../02-arquitetura/01-as-is.md). Este capítulo preserva integralmente a direção proposta posterior; não redefine o software entregue nem promove LF-MT a DONE.
