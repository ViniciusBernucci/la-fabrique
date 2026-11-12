> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo →](01-referencia-v2.3.md)

# Configuração e integrações

IMPLEMENTADO no código base `a2cc5e0`; verificação operacional NÃO VERIFICADA nesta migração. Responsabilidade lógica, não nova classe/microserviço.

## Finalidade e limites

Persistir contas/instalações, modelos, agentes, skills de projeto e GitHub sem segredos.

## Fluxo e comportamento

Contas/Integrações IA/Equipes → modal com rascunho → salvar versionado; verificar/login/PR são pedidos explícitos persistidos com outbox.

Exemplo didático: ticket com critério “validar um campo” só avança com a base/definição/checks aprovados; request inválido não vira comando autorizado. O exemplo não é execução desta migração.

## Apresentação comum — FAC-034

Configurações, contas, modelos, equipes/skills, agentes predefinidos e modais compartilham os tokens HUD do [painel de controle](../controle/00-README.md#apresentação-centro-de-comando--fac-034). Abas têm estado ativo ciano, cards/badges refletem estados reais do provider, switches mantêm role/aria, dialogs usam superfície sólida e backdrop escuro. A camada visual não altera rascunhos, salvar/cancelar, aprovação de revisão nem login oficial. [Entrega e limites de verificação](../../09-entregas/2026/2026-10-07-FAC-034-centro-comando.md).

## Estados, dados e regras

FactorySettings singleton JSON; skills não são instaladas/executadas pelo cadastro. PR exige preparar/aprovar payload exato; código não faz merge. Veja [estados por entidade](../../05-contratos/schemas/01-workflow.md) e [contratos canônicos](../../05-contratos/00-README.md).

## Falhas, recuperação e segurança

Payload/versão/base inválidos são recusados antes de efeito pertinente; auth/cota não autorizam fallback pago. Writer unknown permanece bloqueado; não repetir execução ou liberar lease por idade. Credenciais não entram em DTO/contexto/log/artefato; API não executa código do projeto. Limites de cada contrato e gates de provider continuam necessários.

## Código e evidência

[apps/api/src/settings/settings.service.ts](../../../apps/api/src/settings/settings.service.ts) e [apps/web/src/SettingsPanel.tsx](../../../apps/web/src/SettingsPanel.tsx). Testes versionados ao lado dessas implementações e evidências por FAC-011/013–019/023/026 no [histórico](../../09-entregas/00-README.md). Não reexecutados aqui. [ADR-003 aceito](../../06-decisoes/ADR-003-stack-typescript.md) e [feature relacionada](../../04-features/01-control-settings.md).

## Limitações e direção posterior

[AS-IS versus alvo](../../02-arquitetura/01-as-is.md) separa controles atuais de tenancy/RLS/broker propostos. Presença de implementação/teste versionado não comprova instalação ou eficácia sob UID de serviço.

## Referência integral local

[Detalhes por incremento com nota editorial](01-referencia-v2.3.md). Evidências/contratos históricos permanecem completos; afirmações de estado vencidas são delimitadas pela auditoria.

## Sequência de leitura deste capítulo

1. [referencia-v2.3](01-referencia-v2.3.md).

## Nome e cargo dos agentes — FAC-035

Configurações → Equipes → Agentes permite atribuir um nome próprio aos agentes pré-configurados e personalizados. No resumo e nas informações do agente, o nome aparece em destaque e o cargo em uma linha separada. Exemplo didático: Alex, cargo Developer. O campo Nome do agente está no modal de configuração; salvar persiste o rascunho pela API versionada, cancelar o descarta.

O cargo pré-configurado é fixado pelo papel operacional. Para agentes personalizados, a identificação anterior passa a ser o campo Cargo na empresa, editável. Nome opcional vazio volta a exibir o cargo. A configuração existente continua válida; veja [contrato de dados](../../05-contratos/schemas/00-entidades.md#nomes-de-agentes-fac-035) e [entrega](../../09-entregas/2026/2026-10-08-FAC-035-nomes-agentes.md).

O ambiente de desenvolvimento recompila os contratos antes de iniciar os serviços. API e painel precisam reconhecer o campo `nickname` na mesma revisão; um contrato compilado anterior rejeita esse campo. [Diagnóstico e correção de inicialização](../../09-entregas/2026/2026-10-09-FAC-035-correcao-salvamento-nomes.md).

## Cadastro API ou assinatura CLI — FAC-037

O primeiro passo de Adicionar conta ou Cadastrar integração de IA é escolher chave de API ou plano de assinatura (CLI). O modo escolhido é imutável nessa conta; para outro modo, criar outra integração. API aceita OpenAI e Claude; Antigravity permanece CLI. Assinaturas mantêm login e verificação oficiais existentes.

Chaves são enviadas somente no salvamento autenticado e guardadas com AES-256-GCM em tabela separada. Configuração pública e snapshot do worker contêm somente modo, conta e modelos, sem segredo. Editar conta API com campo de chave vazio preserva a chave existente; remover conta e salvar remove o segredo na mesma transação. Configurações antigas CLI seguem válidas.

Agentes predefinidos e personalizados têm dois seletores: IA cadastrada/habilitada e modelo dessa IA. Trocar IA seleciona seu modelo padrão ou primeiro modelo; integrações sem modelos exigem configurar o catálogo antes de salvar a atribuição. Nenhum catálogo global inventado aparece no formulário.

Este incremento implementa cadastro/atribuição, não inferência por API. API continua sem adapter agentivo, sem verificação de chave por chamada externa e sem execução paga. Conta API não pode usar login, evidência ou adapter CLI. [Entrega e requisitos operacionais](../../09-entregas/2026/2026-10-09-FAC-037-api-cli-agentes.md).

## Configuração pessoal e da conta — FAC-038

A camada AccountModule persiste perfil/preferências em AccountUser e nome em AccountOrganization, com atualização otimista. AccountMembership autoriza equipe; AccountSession autentica pessoas. Essas configurações são independentes de FactorySettings global, integrações e agentes anteriores. [Fonte](../../../apps/api/src/account/account.service.ts), [contratos](../../05-contratos/schemas/00-entidades.md#conta-do-usuário--fac-038) e [feature](../../04-features/01-control-settings.md#painel-da-conta-do-usuário--fac-038). Nenhum segredo/senha/session token aparece no DTO. Convite entrega segredo apenas uma vez ao criador, fora de auditoria/logs. Novos controles não prometem tenant-isolation dos projetos/runtime existentes.

## Recursos próprios da organização — FAC-039–041

[Feature da conta](../../04-features/01-control-settings.md#painel-da-conta-do-usuário--fac-038) descreve papéis/grants, projetos/execuções, integração privada/roteamento, MFA/email, plano, notificações e dados. AccountModule importa serviços de onboarding/verificação existentes; nenhuma CLI/build/check é executado pela API. FactorySettings continua registro operacional de instalações, mas novos runtimeIds pertencem à organização; o worker recebe configuração filtrada pelo projeto. Conta humana nunca recebe o token global. SMTP/Stripe exigem configuração privada explícita e não foram ativados.

## Arquivos de instruções — FAC-036

Nos modais de agentes pré-configurados, personalizados e skills, Descrição breve contém somente o resumo. A área Arquivo de instruções permite adicionar ou substituir um .md UTF-8 e baixar o conteúdo registrado. Seleção altera o rascunho; Salvar configuração grava conteúdo e nome no JSON versionado; cancelar descarta a substituição. Instruções antigas sem nome podem ser baixadas como AGENTS.md ou SKILL.md.

Arquivos vazios, binários com NUL, UTF-8 inválido, nomes com caminhos e conteúdo acima do limite são recusados, preservando o rascunho anterior. Durante leitura, o salvamento aguarda. Limites: 16.000 caracteres para agentes, 64.000 para skills. O cadastro não instala skills nem adiciona invocação automática de perfis; a descrição nunca é convertida em instrução. [Contrato](../../05-contratos/schemas/00-entidades.md#arquivos-markdown-fac-036) · [Entrega](../../09-entregas/2026/2026-10-09-FAC-036-instrucoes-markdown.md).
