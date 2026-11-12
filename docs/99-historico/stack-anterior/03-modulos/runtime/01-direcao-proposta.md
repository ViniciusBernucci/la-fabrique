> Leitura: [← Anterior](00-README.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](02-referencia-v2.3.md)

> Direção posterior do pacote, PLANEJADO. Código atual no [README do módulo](00-README.md); nenhum gate operacional liberado por este texto.

# Provider Runtime e credenciais por tenant

Data: 2026-10-06 (America/Sao_Paulo). Status da arquitetura e controles: **PLANEJADO**. Implementação e eficácia: **NÃO VERIFICADAS**. Este documento especifica trabalho futuro; não comprova instalação, configuração ou execução.

## Contrato de segurança

Cada combinação tenant/provider possui instalação e armazenamento oficial próprios. Auth do Codex, Claude e Antigravity nunca é montada no sandbox, enviada em JobPackage, copiada em checkpoint, herdada por shell/teste ou publicada em logs. Control plane armazena somente referências e metadados; não recebe token OAuth/auth.json. CredentialStore resolve somente a referência vinculada à instalação autorizada, com segredo entregue ao cliente oficial no mecanismo suportado.

Diretório HOME diferente sozinho não isola usuários. Serviço por tenant/provider usa UID/GID distintos, sem grupos compartilhados de leitura, home 0700/segredos 0600, namespaces/LSM e mounts próprios. Exemplo conceitual: `/var/lib/lefabrique/provider-auth/<tenant>/<provider>/`; nenhum diretório pai montado em processos de clientes. A localização real é validada com a versão oficial, não presumida por nome `.codex` ou `.claude`. Sessões e histórico da tentativa ficam em área efêmera por projeto/attempt; se cliente mistura sessões no home de auth, separar pelo mecanismo suportado ou impedir execução até solução comprovada.

## Modelo não confiável em runtime administrado

O cliente autenticado inevitavelmente lê sua autenticação. Isso NÃO autoriza shell, leitura genérica ou extensões locais controladas pelo modelo nessa identidade. Binário/configuração fixos e administrados; desabilitar descoberta de MCP/config/hooks/plugins do repo e do home pessoal, terminals locais, navegação genérica no host e ferramentas embutidas que escapem do broker. A configuração deve impor isso tecnicamente; permissões descritas só no prompt não bastam.

Todas as ações de projeto devem ser traduzidas para o broker de ferramentas, que envia operações ao sandbox. A arquitetura exige ponte oficialmente suportada e verificável; não pressupõe que JSONL ou flag de sandbox redireciona automaticamente as ferramentas de um CLI. Se o cliente oficial só funciona com shell no próprio processo que lê credenciais, a instalação é UNSUPPORTED_ISOLATION e fica DISABLED. Não remediar montando auth no sandbox, adulterando cliente, extraindo OAuth para HTTP/SDK ou ligando API paga.

## Interface planejada

`execute`, `getStatus`, `getCapabilities`, `getUsage`, `cancel`, `resume` mantêm contratos v2.1. Request: schema_version, tenant/project/run/step/attempt IDs, installation_id, workspace_id, revision, context hashes, instruction hashes, policy_version, fencing_token, deadline, command_profile e limites. Ref de credential é resolvida internamente, nunca passada ao modelo. Eventos versionados com sequence/event_id, payload limitado e sanitizado; nenhum evento pode trocar tenant, política, segredo, mount ou cobrança.

Result contém status normalizado, exit code, versão real, modelo efetivo/uso/session ID quando disponíveis, diff/hashes, checks e timestamps. Não coletar raciocínio privado. Erros: AUTH_REQUIRED, RATE_LIMITED, TOOL_DENIED, TRANSIENT, TIMEOUT, RESULT_UNKNOWN, UNSUPPORTED_ISOLATION. Registrar catálogo DISABLED/UNCONFIGURED/AUTH_REQUIRED/AVAILABLE/BUSY/RATE_LIMITED/COOLDOWN/ERROR sem inventar quota.

## Preflight por instalação

1. Registrar origem/versão/binário, identidade Linux, armazenamento oficial e isolamento de leitura por UID real.
2. Login humano pelo fluxo oficial em ambiente administrativo isolado; callback temporário só se suportado, sem portas públicas permanentes. Nunca solicitar senha/token por formulário geral da fábrica.
3. Validar plano/modo/termos para uso por tenant e modo automatizado; guardar evidência sem segredo. Multi-tenant técnico não prova elegibilidade comercial.
4. Bloquear env de API key herdada e comprovar extras/autorecharge/fallback pagos desligados; fornecedor não é controlado só pela política da fábrica.
5. Provar ferramentas mediadas, nenhuma execução local e nenhum acesso do modelo à auth, inclusive tentativa por config/hook malicioso.
6. Validar eventos/schema, cancelamento, sessão separada, leitura das regras, modelo/uso observáveis e egress necessário. Unsupported é registrado honestamente.

Registrar capabilities com verified_at, cli_version, evidence_id, config hash e policy hash. Alteração de versão/config invalida preflight antes de novo job. Primeiro provider só é habilitado depois dos gates; os outros permanecem bloqueados até validação própria.

## Revogação e renovação

Credential tem generation; resolução sempre verifica status atual. Revogar instalação bloqueia claims, invalida capabilities e cancela árvore da tentativa; confirmar quiescência. Renovação por login oficial, sem copiar sessão entre tenants/máquinas. Backup da autenticação só quando mecanismo oficial e política aprovarem; padrão de recovery é reautenticar. Segredo de worker é distinto de segredo de provider e nunca fornecido ao runtime do modelo.

Handoff troca instalação apenas dentro do tenant e com grant de projeto. É nova sessão/attempt/sandbox a partir de checkpoint sanitizado. Nunca transferir sessão privada, home ou token de fornecedor.


## Origem desta edição

[Versão original preservada](../../99-historico/originais/output/le-fabrique-multitenant/documentacoes/runtime/README.md). Migração editorial de paths em 2026-10-06; conteúdo de engenharia continua proposto.

## Localização, relações e validação

Na origem do pacote o código estava ausente; AS-IS atual no README; nenhuma classe/entrypoint ou teste foi comprovado. Consulte [ADRs](../../06-decisoes/00-README.md), [features](../../04-features/00-README.md) e [SEC-01–14](../../08-desenvolvimento/12-testes-seguranca.md). Estados/contratos citados são planejados; registrar código/revisão/checks por controle quando implementado.
