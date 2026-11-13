# FAC-036 — Instruções de agentes e skills em Markdown

Data: 2026-10-09, Europe/Berlin. Domínios: configuração, painel e contratos. IMPLEMENTADO / AWAITING_HUMAN; base 38998ebf313625d2202dcb51c574803d78a3a1ef, branch feat/fac-036-instrucoes-md, worktree isolada /home/vinicius/le-fabrique-fac-036. Sem commit/PR/integração/deploy nesta entrega.

## Objetivo, problema e critérios de aceite

Pedido explícito: permitir adicionar arquivo MD para instruções do agente e da skill, manter o texto do modal como descrição breve e incluir área de download. [Ticket autorizado](../../08-desenvolvimento/tickets/configuracao/FAC-036-instrucoes-markdown.md). Anteriormente agentes personalizados e skills exigiam texto no textarea; agentes predefinidos não tinham área de instruções própria.

Critérios implementados: upload/substituição/download, separação de descrição e instruções, manutenção de conteúdo/nome pela configuração versionada, leitura legada e validação de tipo/codificação/limites. Conteúdo e persistência ATENDIDOS por testes; fluxo completo selecionar/salvar/reabrir/baixar/cancelar em browser NÃO VALIDADO. Um writer nesta worktree; sessões abertas no workspace principal não foram alteradas.

## Solução e comportamento implementado

Modais de agentes personalizados, pré-configurados e skills apresentam Descrição breve e Arquivo de instruções. O arquivo deve ser .md UTF-8, não vazio, até 16.000 caracteres para agentes ou 64.000 para skills; nomes até 200, sem caminhos/controles ASCII. Leitura limitada antes da alocação; UTF-8 inválido, NUL, excesso e vazio geram erro sem substituir conteúdo anterior. Salvar aguarda a leitura; sucesso atualiza só o rascunho até salvar pela API. Cancelar descarta o rascunho.

Download gera Blob Markdown com o conteúdo atual e nome enviado, revogando URL ao mudar/fechar. Espaços, BOM e quebras de linha são preservados. Registros anteriores continuam acessíveis via download com AGENTS.md/SKILL.md quando não tinham nome. Descrição permanece resumo, sem conversão para instruções. Registrar arquivo não instala ou invoca automaticamente skills/perfis.

## Arquivos e decisões

MarkdownInstructions.tsx é a área compartilhada; markdown-instructions.ts valida e decodifica; SettingsPanel.tsx e TeamsPanel.tsx incorporam o componente; styles.css usa tokens HUD existentes. packages/contracts/src/index.ts define campos opcionais e preserva texto bruto. Specs web/contratos/settings verificam conteúdo/limites/persistência. Stack ADR-003 preservada, sem dependências novas ou decisão estrutural.

## Dados, API, eventos e configuração

FactorySettings.configuration JSON usa a rota versionada existente. instructionsFilename opcional para os três cadastros; assignments também ganham description/instructions opcionais. Sem nova rota, multipart, arquivo servidor, migration ou evento. Registros antigos válidos. Leitores strict antigos exigem rollout conjunto ou retirada dos novos campos; cliente antigo que sobrescreve JSON pode perder metadados. [Contrato](../../05-contratos/schemas/00-entidades.md#arquivos-markdown-fac-036).

## Diff e artefatos

Diff e novos arquivos permanecem na worktree isolada, revisáveis por git diff e git status. Evidências sem secrets em [leitura/checks](evidencias/FAC-036/leitura-checks.md), [settings](evidencias/FAC-036/settings-test.log), [build](evidencias/FAC-036/web-build.log), [tipos](evidencias/FAC-036/typecheck.log), [lint](evidencias/FAC-036/lint.log) e [documentação](evidencias/FAC-036/documentacao.log). Não há SHA de commit inventado. Hashes das políticas e omissões constam da leitura; patch adicional não é necessário enquanto worktree/diff estão preservados.

## Testes e evidências

Checks sobre diff local da base indicada:

| Critério | Comando | Resultado |
|---|---|---|
| Upload/UTF-8/vazio/NUL/nomes/limites e regressões web | npm run test -w @le-fabrique/web | PASS, 67 testes |
| Metadados, conteúdo bruto e leitura legada | npm run test -w @le-fabrique/contracts | PASS, 54 testes |
| Persistência versionada de instruções nos três cadastros | npm run test -w @le-fabrique/api -- src/settings/settings.service.spec.ts | PASS, 8 testes |
| Bundle/compilação web | npm run build -w @le-fabrique/web | PASS |
| Tipos de consumidores | npm run typecheck | PASS |
| Qualidade dos nove arquivos afetados | npx biome check (paths da entrega) | PASS |
| Integridade do diff | git diff --check | PASS |
| Gate documental | python3 scripts/validar-documentacao.py | PASS |
| Selecionar/salvar/reabrir/download/cancelar em browser | Fluxo manual | NOT_RUN: ferramenta de browser ausente |
| Revisão independente | Sessão independente | NOT_RUN |

Ambiente instalado com npm ci --ignore-scripts --offline; cliente Prisma gerado localmente por db:generate, sem DB/migration. Primeiro teste de settings não carregou por cliente ausente, corrigido pela geração. Primeiro lint pediu remoção de controles ASCII do regex; validação passou a refine com charCodeAt. Não houve regressão de produto observada. Testes PostgreSQL/provider/deploy não foram necessários nem executados.

## Segurança, riscos e limitações

Markdown é texto, sem renderização HTML/execução pelo upload. Nome não vira path servidor; URL Blob é temporária. Limites/codificação impedem alocação sem limite. Auth e execução do worker permanecem separadas do cadastro. Rollout de leitores strict e teste visual/browser pendentes. Autoavaliação de código e checks não são revisão/QA independentes.

## Rollback

Reverter somente os arquivos deste incremento. Antes de usar schema antigo com configuração contendo novos campos, preservar cópia privada e retirar instructionsFilename de todos os cadastros e description/instructions de assignments. Não reverter dados com reset destrutivo. Rollback não ensaiado; DB ativo não alterado.

## Documentação atualizada

[Módulo](../../03-modulos/configuracao/00-README.md#arquivos-de-instruções--fac-036), [feature](../../04-features/01-control-settings.md#instruções-em-markdown--fac-036), contrato, ticket, [backlog](../../08-desenvolvimento/09-backlog.md), [matriz](../../08-desenvolvimento/05-matriz-cobertura.md), [índice](../../02-INDEX.md), índice de entregas e [changelog](../../04-CHANGELOG.md). Nenhum capítulo didático novo, portanto mapa de ordem preservado; registros de ticket/entrega são consultas.

## Lessons

N/A: validação compartilhada e rascunho versionado seguem padrões existentes; nenhum conceito novo justifica capítulo adicional.

## Uso de IA e custos

Codex nesta sessão; modelo efetivo/versão/modo de cobrança/uso/cota não observados, unknown. Sem subagentes, handoff, API de IA adicional, login ou gastos habilitados. Custo fixo/total/incremental não medidos; nenhum valor inferido de tokens. Fonte: ferramentas locais e logs deste diff em 2026-10-09.

## Pendências, próximos passos e aceite

AWAITING_HUMAN. Revisão independente, browser, integração e aceite da revisão exata pendentes. Não houve merge, push, deploy ou promoção a DONE.

## Adendo 2026-10-24 — integração e verificação em browser

O registro acima descreve o estado em 2026-10-09 e permanece inalterado. Em 2026-10-24 o trabalho da worktree foi commitado em `40c9159` e integrado à developer local no merge `82bcfc7`, combinando os modais com o AgentModelSelector do FAC-037. Os logs de evidência desta entrega foram adicionados ao Git apesar do ignore de `*.log`, para manter os links.

Fluxo completo em browser, antes NOT_RUN, passou em fixture descartável: 7 verificações e zero exceções de runtime. Checks pós-integração PASS. Detalhes, achados da revisão e limites em [revisão e verificação](evidencias/FAC-036/revisao-integracao.md). Pendentes: revisão independente formal e aceite humano da revisão exata. Sem push, PR ou deploy.
