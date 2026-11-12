> Leitura: [← Manual Vivo](01-README.md) · [Roteiro](00-LEIA-ME-PRIMEIRO.md) · [Glossário →](03-GLOSSARIO.md)

# Índice didático do Manual Vivo

[Comece pelo roteiro](00-LEIA-ME-PRIMEIRO.md). Capítulos numerados ensinam; apêndices preservam IDs/datas para consulta.

## Entrada do manual

1. [Como ler este sistema](00-LEIA-ME-PRIMEIRO.md).
2. [Manual Vivo da Fábrica de Software](01-README.md).
3. [Índice didático do Manual Vivo](02-INDEX.md).
4. [Atalho do glossário](03-GLOSSARIO.md).
5. [Changelog](04-CHANGELOG.md).

## Visão geral

1. [Visão geral](01-visao-geral/00-README.md).
2. [Objetivos e avaliação](01-visao-geral/01-objetivos.md).
3. [Escopo e evolução](01-visao-geral/02-escopo.md).
4. [Glossário](01-visao-geral/03-glossario.md).
5. [Mapa didático do sistema atual](01-visao-geral/04-mapa-do-sistema.md).

## Arquitetura

1. [Arquitetura](02-arquitetura/00-README.md).
2. [Estado observado versus direção proposta](02-arquitetura/01-as-is.md).
3. [C4 nível 1 — contexto atual](02-arquitetura/02-contexto.md).
4. [C4 nível 2 — containers atuais](02-arquitetura/03-containers.md).
5. [C4 nível 3 — componentes atuais](02-arquitetura/04-componentes.md).
6. [Fluxo implementado e handoff](02-arquitetura/05-fluxos.md).
7. [Dados — estrutura observada](02-arquitetura/06-dados.md).
8. [Arquitetura multi-tenant na VPS única](02-arquitetura/07-fronteiras-e-invariantes.md).
9. [Segurança e fronteiras](02-arquitetura/08-seguranca.md).
10. [Infraestrutura e implantação](02-arquitetura/09-infraestrutura.md).
11. [Direção proposta — C4 nível 1 — contexto](02-arquitetura/10-contexto-alvo.md).
12. [Direção proposta — C4 nível 2 — containers](02-arquitetura/11-containers-alvo.md).
13. [Direção proposta — C4 nível 3 — componentes](02-arquitetura/12-componentes-alvo.md).
14. [Direção proposta — Fluxos principais](02-arquitetura/13-fluxos-alvo.md).
15. [Direção proposta — Arquitetura de dados](02-arquitetura/14-dados-alvo.md).
16. [Software Factory - Arquitetura e operação](02-arquitetura/15-referencia-v2.3.md).

## Threat model e risco residual

1. [Threat model e risco residual](02-arquitetura/seguranca/00-threat-model.md).
2. [Rede deny-by-default e saída controlada](02-arquitetura/seguranca/01-rede-e-egress.md).

## Módulos e responsabilidades

1. [Módulos e responsabilidades](03-modulos/00-README.md).

## Projetos

1. [Projetos](03-modulos/projects/00-README.md).

## Tickets

1. [Tickets](03-modulos/tickets/00-README.md).

## Runs e tentativas

1. [Runs e tentativas](03-modulos/runs/00-README.md).

## Orquestração e outbox

1. [Orquestração e outbox](03-modulos/orchestrator/00-README.md).

## Context Builder

1. [Context Builder](03-modulos/context-builder/00-README.md).
2. [Contexto de execucao](03-modulos/context-builder/01-referencia-v2.3.md).

## Instalações e roteamento

1. [Instalações e roteamento](03-modulos/providers/00-README.md).

## Runtime de clientes oficiais

1. [Runtime de clientes oficiais](03-modulos/runtime/00-README.md).
2. [Direção proposta — Provider Runtime e credenciais por tenant](03-modulos/runtime/01-direcao-proposta.md).
3. [Agent Runtime Gateway e Provider Manager](03-modulos/runtime/02-referencia-v2.3.md).

## Worker

1. [Worker](03-modulos/worker/00-README.md).

## Executor e checkout

1. [Executor e checkout](03-modulos/executor-manager/00-README.md).

## Sandbox, worktrees e snapshots

1. [Sandbox, worktrees e snapshots](03-modulos/sandbox/00-README.md).
2. [Direção proposta — Sandboxes efêmeros e isolamento do host](03-modulos/sandbox/01-direcao-proposta.md).

## Limites de execução

1. [Limites de execução](03-modulos/runtime-guard/00-README.md).

## Gate documental

1. [Gate documental](03-modulos/documentation-gate/00-README.md).

## Aceite humano

1. [Aceite humano](03-modulos/approvals/00-README.md).

## Painel de controle

1. [Painel de controle](03-modulos/controle/00-README.md).
2. [Controle administrativo](03-modulos/controle/01-referencia-v2.3.md).

## Configuração e integrações

1. [Configuração e integrações](03-modulos/configuracao/00-README.md).
2. [Centro de configuracoes](03-modulos/configuracao/01-referencia-v2.3.md).

## Observabilidade e economia

1. [Observabilidade e economia](03-modulos/observabilidade-economia/00-README.md).

## Identidade, autorização e segregação de dados

1. [Identidade, autorização e segregação de dados](03-modulos/tenant-isolation/00-README.md).

## Broker, ferramentas e capacidades temporárias

1. [Broker, ferramentas e capacidades temporárias](03-modulos/tool-broker/00-README.md).

## Comportamentos de produto

1. [Comportamentos de produto](04-features/00-README.md).
2. [Navegar e configurar a fábrica](04-features/01-control-settings.md).
3. [Executar ticket configurado](04-features/02-ticket-execution.md).
4. [Documentar e aceitar revisão exata](04-features/03-documentation-approval.md).
5. [Pausar e recuperar execução](04-features/04-recovery-operation.md).
6. [Selecionar cliente e transferir progresso](04-features/05-provider-handoff.md).
7. [Observar estado e uso](04-features/06-usage-dashboard.md).
8. [Isolar tenants e projetos](04-features/07-tenant-isolation.md).

## Contratos: implementação e proposta

1. [Contratos: implementação e proposta](05-contratos/00-README.md).
2. [Especificação implementável v2](05-contratos/01-especificacao-v2.3.md).

## Dados implementados e entidades propostas

1. [Dados implementados e entidades propostas](05-contratos/schemas/00-entidades.md).
2. [Estados por entidade — código como fonte](05-contratos/schemas/01-workflow.md).
3. [Artefatos e contexto](05-contratos/schemas/02-artifact-context.md).
4. [Checkpoint e snapshot](05-contratos/schemas/03-checkpoint.md).
5. [Contratos implementáveis v3 propostos](05-contratos/schemas/04-job-package.md).

## Rotas físicas observadas

1. [Rotas físicas observadas](05-contratos/api/00-catalogo-rotas.md).
2. [API de controle implementada](05-contratos/api/01-controle.md).
3. [Protocolo interno implementado](05-contratos/api/02-worker.md).
4. [Runtime implementado](05-contratos/api/03-runtime.md).

## Eventos observados

1. [Eventos observados](05-contratos/eventos/00-catalogo.md).

## Outbox e fila

1. [Outbox e fila](05-contratos/filas/00-dispatch.md).

## Decisões e propostas

1. [Decisões e propostas](06-decisoes/00-README.md).
2. [ADR-007 — Conta do usuário e sessões individuais](06-decisoes/ADR-007-conta-usuario-e-sessoes.md).

## Operar e recuperar

1. [Operar e recuperar](07-operacao/00-README.md).
2. [Runbook de deploy](07-operacao/01-deploy.md).
3. [Execução, handoff e recuperação](07-operacao/02-execucao-e-recuperacao.md).
4. [Observabilidade e evidências](07-operacao/03-observabilidade.md).
5. [Handoff e troca de provider](07-operacao/04-handoff.md).
6. [Backup offline de evidências](07-operacao/05-backup.md).
7. [Restaurar em staging novo](07-operacao/06-restore.md).
8. [Diagnóstico e recuperação](07-operacao/07-troubleshooting.md).
9. [Resposta a incidente](07-operacao/08-incidentes.md).
10. [Economia, quotas e custos](07-operacao/09-economia.md).

## Política de economia

1. [Política de economia](07-operacao/economia/00-referencia-v2.3.md).

## Checkpoint e troca de provider

1. [Checkpoint e troca de provider](07-operacao/handoff/00-referencia-v2.3.md).

## Dimensionamento da VPS única - MVP v2.3

1. [Dimensionamento da VPS única - MVP v2.3](07-operacao/infraestrutura/00-dimensionamento-vps.md).
2. [Infraestrutura atual](07-operacao/infraestrutura/01-referencia-v2.3.md).

## Operação da fábrica e worker

1. [Operação da fábrica e worker](07-operacao/operacao/00-referencia-v2.3.md).

## Desenvolver e verificar

1. [Desenvolver e verificar](08-desenvolvimento/00-README.md).
2. [Ambiente local real](08-desenvolvimento/01-ambiente-local.md).
3. [Convenções de engenharia e documentação](08-desenvolvimento/02-convencoes.md).
4. [Como contribuir](08-desenvolvimento/03-como-contribuir.md).
5. [Checks reais e evidência](08-desenvolvimento/04-testes.md).
6. [Cobertura e migração v1 -> v2](08-desenvolvimento/05-matriz-cobertura.md).
7. [Controle do MVP — La fabrique](08-desenvolvimento/06-controle-mvp.md).
8. [Contrato do piloto](08-desenvolvimento/07-piloto.md).
9. [Handoff para Tech Lead](08-desenvolvimento/08-tech-lead-handoff.md).
10. [Backlog v2](08-desenvolvimento/09-backlog.md).
11. [Plano do MVP v2](08-desenvolvimento/10-plano-mvp.md).
12. [Plano incremental v3 — segurança antes da execução](08-desenvolvimento/11-plano-lf-mt-proposto.md).
13. [Plano de testes negativos e aceite](08-desenvolvimento/12-testes-seguranca.md).
14. [Catálogo funcional FAC preservado](08-desenvolvimento/13-catalogo-fac.md).

## Lessons

1. [Lessons](10-lessons/00-README.md).
2. [Definicao de projeto e uma fronteira configuravel](10-lessons/01-definicao-projeto-configuravel.md).
3. [Contratos de runtime no monorepo TypeScript](10-lessons/02-contratos-runtime-monorepo.md).
4. [Outbox idempotente e versão otimista](10-lessons/03-outbox-idempotencia.md).
5. [Contexto deterministico e limites conservadores](10-lessons/04-contexto-deterministico-limites.md).
6. [Checkout confiável mantém credenciais fora do projeto](10-lessons/05-checkout-confiavel.md).
7. [Sandbox, worktree e snapshot sao limites diferentes](10-lessons/06-sandbox-worktree-snapshot.md).
8. [Perfis de permissao para clientes CLI](10-lessons/07-perfis-permissao-cliente-cli.md).
9. [Lifecycle seguro de um cliente CLI](10-lessons/08-lifecycle-processo-cli.md).
10. [Lease, fencing e checkpoint cobrem falhas diferentes](10-lessons/09-leases-fencing-checkpoints.md).
11. [Heartbeat e parada conservadora](10-lessons/10-worker-heartbeat-quiescencia.md).
12. [Baseline, regressao e review sao sinais diferentes](10-lessons/11-baseline-regressao-review.md).
13. [Proveniência preservada antes da conciliação](10-lessons/12-documentacao-proveniencia.md).
14. [Edição em modal com rascunho separado](10-lessons/13-modal-edicao-configuracao.md).
15. [Vídeo decorativo com fronteira contínua](10-lessons/14-video-decorativo-loop.md).

## Governança canônica

1. [Governança canônica](00-governanca/00-README.md).
2. [Política comum de engenharia e agentes](00-governanca/01-POLITICA-IA.md).
3. [Política do Manual Vivo e docs-as-code](00-governanca/02-POLITICA-DOCUMENTACAO.md).
4. [Integração do Manual Vivo neste repositório](00-governanca/03-GUIA-DE-INTEGRACAO.md).
5. [Checklist de entrega](00-governanca/04-CHECKLIST-ENTREGA.md).
6. [Carregamento das instruções dos agentes](00-governanca/05-fontes-carregamento.md).
7. [Fontes e evidências](00-governanca/06-fontes-tecnicas.md).
8. [Pendências após conciliação](00-governanca/07-PENDENCIAS.md).
9. [Auditoria da integração no repo real](00-governanca/08-AUDITORIA.md).
10. [Inventário completo da integração](00-governanca/09-INVENTARIO.md).
11. [Mapa da migração real](00-governanca/10-MAPA-MIGRACAO.md).
12. [Verificação da migração real — DOC-MV-001](00-governanca/11-VALIDACAO.md).

## Templates de autoria

1. [Templates de autoria](00-governanca/templates/00-README.md).
2. [ID — título](00-governanca/templates/01-TICKET.md).
3. [Módulo — título](00-governanca/templates/02-MODULO.md).
4. [Feature — título](00-governanca/templates/03-FEATURE.md).
5. [Contrato — título](00-governanca/templates/04-CONTRATO.md).
6. [ADR-ID — decisão](00-governanca/templates/05-ADR.md).
7. [Procedimento operacional](00-governanca/templates/06-RUNBOOK.md).
8. [HANDOFF — TICKET / RUN](00-governanca/templates/07-HANDOFF.md).
9. [TICKET — título da entrega](00-governanca/templates/08-ENTREGA.md).
10. [Conceito aplicado](00-governanca/templates/09-LESSON.md).

## Planejamento atual

1. [Planejamento atual](08-desenvolvimento/tickets/planejamento/00-README.md).

## Histórico de entregas

- [FAC-038 — Ticket do painel da conta do usuário](08-desenvolvimento/tickets/FAC-038.md).

- [FAC-038 — Painel da conta do usuário](09-entregas/2026/2026-10-09-FAC-038-painel-conta-usuario.md).

- [PLAN-ADM-001 — Proposta do painel administrativo da conta](09-entregas/2026/2026-10-09-PLAN-ADM-001-painel-administrativo-conta.md).

1. [Histórico de entregas](09-entregas/00-README.md).

## Guias, prompts e pontes de consulta

1. [Ponte do projeto](../.agents/rules/documentacao.md).
2. [Guia do projeto — AGENTS.md](../AGENTS.md).
3. [Guia do projeto — ANTIGRAVITY.md](../ANTIGRAVITY.md).
4. [Guia do projeto — CLAUDE.md](../CLAUDE.md).
5. [La fabrique — Fábrica de Software](../README.md).
6. [Ponte de leitura da política](00-governanca/POLITICA-DOCUMENTACAO.md).
7. [Ponte de leitura da política](00-governanca/POLITICA-IA.md).
8. [Regras da fixture FAC-002](../fixtures/codex-preflight/AGENTS.md).
9. [Fixture sintetica do Codex](../fixtures/codex-preflight/README.md).
10. [Teste de isolamento FAC-002](../fixtures/codex-preflight-isolation/AGENTS.md).
11. [Regras da fixture de escrita FAC-002](../fixtures/codex-preflight-write/AGENTS.md).
12. [Atalho de compatibilidade](../lessons/INDEX.md).
13. [Prompt para entregas futuras](../prompts/PROMPT-ENTREGA.md).
14. [Prompt de início no repo real](../prompts/PROMPT-INICIAL.md).
15. [Prompt mestre — migração inicial do Manual Vivo](../prompts/PROMPT-MESTRE-MIGRACAO.md).
16. [Prompt zero — preparar/refatorar sem habilitar a nova arquitetura](../prompts/implementacao/PROMPT-00-PREPARACAO.md).
17. [Prompts de implementação — 11 etapas](../prompts/implementacao/PROMPTS-IMPLEMENTACAO.md).

## Decisões com IDs preservados

1. [ADR-003 - Stack TypeScript da fábrica](06-decisoes/ADR-003-stack-typescript.md).
2. [ADR-002 - Controle e execução na mesma VPS](06-decisoes/ADR-002-vps-unica.md).
3. [DOC-ADR-001 — Manual Vivo e fonte canônica](06-decisoes/DOC-ADR-001-manual-vivo.md).
4. [ADR-003 — multi tenant e stack](06-decisoes/ADR-003-multi-tenant-e-stack.md).
5. [ADR-004 — provider runtime sem segredos no sandbox](06-decisoes/ADR-004-provider-runtime-sem-segredos-no-sandbox.md).
6. [ADR-005 — sandbox efemero e perfil de isolamento](06-decisoes/ADR-005-sandbox-efemero-e-perfil-de-isolamento.md).
7. [ADR-006 — rls capabilities e egress](06-decisoes/ADR-006-rls-capabilities-e-egress.md).

## Tickets identificados

1. [DOC-MV-001 — Migração inicial do Manual Vivo](08-desenvolvimento/tickets/DOC-MV-001.md).
2. [DOC-MV-002 — Organização da raiz e retirada de documentacoes](08-desenvolvimento/tickets/DOC-MV-002.md).
3. [DOC-MV-003 — Ordem didática obrigatória](08-desenvolvimento/tickets/DOC-MV-003.md).
4. [FAC-023 — Agentes e skills por projeto](08-desenvolvimento/tickets/configuracao/FAC-023-agentes-skills.md).
5. [FAC-026 — Agentes pré-configurados na lista](08-desenvolvimento/tickets/configuracao/FAC-026-agentes-preconfigurados.md).
6. [FAC-020 — Central de controle estática](08-desenvolvimento/tickets/controle/tickets/FAC-020.md).
7. [FAC-020A — Ativar home no aplicativo em desenvolvimento](08-desenvolvimento/tickets/controle/tickets/FAC-020A.md).
8. [FAC-021 — Nome correto La fabrique](08-desenvolvimento/tickets/controle/tickets/FAC-021.md).
9. [FAC-022 — Template persistente e tipografia das telas internas](08-desenvolvimento/tickets/controle/tickets/FAC-022.md).
10. [FAC-024 — Vídeo decorativo no dashboard](08-desenvolvimento/tickets/controle/tickets/FAC-024.md).
11. [FAC-025 — Retirar vídeo da home](08-desenvolvimento/tickets/controle/tickets/FAC-025.md).
12. [FAC-027 — Menu lateral recolhível](08-desenvolvimento/tickets/controle/tickets/FAC-027.md).
13. [FAC-028 — Seta minimalista para recolher menu](08-desenvolvimento/tickets/controle/tickets/FAC-028.md).
14. [FAC-029 — Seta na borda direita do menu](08-desenvolvimento/tickets/controle/tickets/FAC-029.md).
15. [FAC-030 — Abrir com menu recolhido](08-desenvolvimento/tickets/controle/tickets/FAC-030.md).
16. [FAC-031 — Posição da seta conforme menu](08-desenvolvimento/tickets/controle/tickets/FAC-031.md).
17. [PLAN-001 — adiamento do piloto externo](08-desenvolvimento/tickets/planejamento/2026-09-30-PLAN-001-adiamento-piloto.md).
18. [FAC-000 - Bootstrap TypeScript da fábrica](08-desenvolvimento/tickets/planejamento/FAC-000-bootstrap-typescript.md).
19. [FAC-001A — Definicao de projeto configuravel](08-desenvolvimento/tickets/planejamento/FAC-001A-definicao-projeto-configuravel.md).
20. [FAC-002 — Preflight do Codex oficial](08-desenvolvimento/tickets/planejamento/FAC-002-preflight-codex.md).
21. [FAC-003 — controle web e persistência](08-desenvolvimento/tickets/planejamento/FAC-003-controle-web-persistencia.md).
22. [FAC-003A — Gate de revisao-base exata antes de READY](08-desenvolvimento/tickets/planejamento/FAC-003A-gate-revisao-base-ready.md).
23. [FAC-004 — worker interno e identidade de serviço](08-desenvolvimento/tickets/planejamento/FAC-004-worker-identidade.md).
24. [FAC-005 — Runtime Gateway e primeiro adapter Codex](08-desenvolvimento/tickets/planejamento/FAC-005-runtime-gateway-codex.md).
25. [FAC-006 — Context Builder e RuntimeGuard](08-desenvolvimento/tickets/planejamento/FAC-006-context-builder-runtime-guard.md).
26. [FAC-007 — Sandbox e snapshots recuperaveis](08-desenvolvimento/tickets/planejamento/FAC-007-sandbox-snapshots.md).
27. [FAC-008 — Orquestrador, leases e checkpoints](08-desenvolvimento/tickets/planejamento/FAC-008-orquestrador-checkpoints.md).
28. [FAC-009 — Developer, checks e revisao](08-desenvolvimento/tickets/planejamento/FAC-009-developer-checks-review.md).
29. [FAC-010 — Segundo provider e handoff automatico](08-desenvolvimento/tickets/planejamento/FAC-010-segundo-provider-handoff.md).
30. [FAC-010A — Verificacao gerenciada de instalacoes](08-desenvolvimento/tickets/planejamento/FAC-010A-verificacao-instalacoes.md).
31. [FAC-010B — Login efemero do Codex](08-desenvolvimento/tickets/planejamento/FAC-010B-login-efemero-codex.md).
32. [FAC-010C — Adapter Claude e handoff seguro](08-desenvolvimento/tickets/planejamento/FAC-010C-adapter-claude-handoff.md).
33. [FAC-011 — Centro de configuracoes e atribuicao de agentes](08-desenvolvimento/tickets/planejamento/FAC-011-centro-configuracoes.md).
34. [FAC-011A — Verificacao gerenciada do GitHub CLI](08-desenvolvimento/tickets/planejamento/FAC-011A-verificacao-github-cli.md).
35. [FAC-011B — Login efemero do GitHub CLI](08-desenvolvimento/tickets/planejamento/FAC-011B-login-efemero-github.md).
36. [FAC-011C — Verificacao somente-leitura do repositorio GitHub](08-desenvolvimento/tickets/planejamento/FAC-011C-verificacao-repositorio-github.md).
37. [FAC-011D — Criacao de pull request sob gate humano](08-desenvolvimento/tickets/planejamento/FAC-011D-criacao-pull-request-gate-humano.md).
38. [FAC-012A — Especificacao imutavel de execucao](08-desenvolvimento/tickets/planejamento/FAC-012A-especificacao-execucao-imutavel.md).
39. [FAC-012AA — Perfil granular e preflight Claude](08-desenvolvimento/tickets/planejamento/FAC-012AA-confinamento-claude.md).
40. [FAC-012AB — Ensaio integrado interno do MVP](08-desenvolvimento/tickets/planejamento/FAC-012AB-ensaio-integrado-mvp.md).
41. [FAC-012AC — Exclusão global do writer](08-desenvolvimento/tickets/planejamento/FAC-012AC-writer-global.md).
42. [FAC-012AD — Estado operacional da fábrica no painel](08-desenvolvimento/tickets/planejamento/FAC-012AD-painel-operacao.md).
43. [FAC-012AE — Eventos autenticados e retomáveis de projeto](08-desenvolvimento/tickets/planejamento/FAC-012AE-eventos-projeto.md).
44. [FAC-012AF — Backup criptografado e restauração isolada](08-desenvolvimento/tickets/planejamento/FAC-012AF-backup-restauracao.md).
45. [FAC-012AG — Pausa global persistida da fábrica](08-desenvolvimento/tickets/planejamento/FAC-012AG-pausa-global.md).
46. [FAC-012AH — Preservar jobs enquanto capacidade global está ocupada](08-desenvolvimento/tickets/planejamento/FAC-012AH-admissao-capacidade.md).
47. [FAC-012B — Compilador da especificação para workflow](08-desenvolvimento/tickets/planejamento/FAC-012B-compilador-workflow-configuravel.md).
48. [FAC-012C — Política de escrita do SandboxRunner](08-desenvolvimento/tickets/planejamento/FAC-012C-politica-escrita-sandbox.md).
49. [FAC-012D — Restringir escrita do CLI aos caminhos do projeto](08-desenvolvimento/tickets/planejamento/FAC-012D-caminhos-escrita-cli.md).
50. [FAC-012E — Leitura interna da configuração de runtime pelo worker](08-desenvolvimento/tickets/planejamento/FAC-012E-configuracao-runtime-worker.md).
51. [FAC-012F — Resolver rota do agente pela configuração atual](08-desenvolvimento/tickets/planejamento/FAC-012F-rota-agente-configurada.md).
52. [FAC-012G — Workflow com rota configurada por função](08-desenvolvimento/tickets/planejamento/FAC-012G-workflow-rotas-por-funcao.md).
53. [FAC-012H — Isolamento da raiz do SandboxRunner](08-desenvolvimento/tickets/planejamento/FAC-012H-isolamento-raiz-sandbox.md).
54. [FAC-012I — Perfil de execução configurado por projeto](08-desenvolvimento/tickets/planejamento/FAC-012I-perfil-execucao-projeto.md).
55. [FAC-012J — Checkout confiável no worker](08-desenvolvimento/tickets/planejamento/FAC-012J-checkout-confiavel-worker.md).
56. [FAC-012K - Lease vivo e cancelamento conservador](08-desenvolvimento/tickets/planejamento/FAC-012K-lease-vivo-cancelamento.md).
57. [FAC-012L - Consumer de execução autônoma](08-desenvolvimento/tickets/planejamento/FAC-012L-consumer-execucao-real.md).
58. [FAC-012M — Resultados de execução no painel](08-desenvolvimento/tickets/planejamento/FAC-012M-resultados-execucao-painel.md).
59. [FAC-012N — Reconciliação após checkpoint](08-desenvolvimento/tickets/planejamento/FAC-012N-reconciliacao-checkpoint.md).
60. [FAC-012O — Journal de resultados antes da API](08-desenvolvimento/tickets/planejamento/FAC-012O-journal-resultados.md).
61. [FAC-012P — Diff e artefatos no painel](08-desenvolvimento/tickets/planejamento/FAC-012P-artefatos-painel.md).
62. [FAC-012Q — Documentação e aceite exato da entrega](08-desenvolvimento/tickets/planejamento/FAC-012Q-documentacao-aceite-entrega.md).
63. [FAC-012R — Snapshot de interrupção comprovada](08-desenvolvimento/tickets/planejamento/FAC-012R-snapshot-interrupcao.md).
64. [FAC-012S — Pausa e cancelamento administrativos](08-desenvolvimento/tickets/planejamento/FAC-012S-comandos-run.md).
65. [FAC-012T — Retomada explícita do snapshot](08-desenvolvimento/tickets/planejamento/FAC-012T-retomada-snapshot.md).
66. [FAC-012U — Recuperação administrativa apenas de finalização](08-desenvolvimento/tickets/planejamento/FAC-012U-recuperacao-finalizacao.md).
67. [FAC-012V — Identidade privada por instalação](08-desenvolvimento/tickets/planejamento/FAC-012V-identidades-instalacoes.md).
68. [FAC-012W — Transporte ampliado e limitado de artefatos](08-desenvolvimento/tickets/planejamento/FAC-012W-artefatos-ampliados.md).
69. [FAC-012X — Espera por provider com preservação e retomada](08-desenvolvimento/tickets/planejamento/FAC-012X-espera-provider.md).
70. [FAC-012Y — Handoff automático configurado](08-desenvolvimento/tickets/planejamento/FAC-012Y-handoff-configurado.md).
71. [FAC-012Z — Gate de documentação técnica](08-desenvolvimento/tickets/planejamento/FAC-012Z-gate-documentacao-tecnica.md).
72. [FAC-013 — Lista de contas IA com configuração em modal](08-desenvolvimento/tickets/planejamento/FAC-013-modal-contas-ia.md).
73. [FAC-014 — Configurar funcionários e GitHub em modais](08-desenvolvimento/tickets/planejamento/FAC-014-modais-funcionarios-github.md).
74. [FAC-015 — Contas oficiais autenticadas e modelos acessíveis](08-desenvolvimento/tickets/planejamento/FAC-015-contas-autenticadas-modelos.md).
75. [FAC-016 — Resultado visível da verificação Codex](08-desenvolvimento/tickets/planejamento/FAC-016-feedback-verificacao-codex.md).
76. [FAC-017 — Login device Codex e aba de autorização](08-desenvolvimento/tickets/planejamento/FAC-017-login-device-codex.md).
77. [FAC-018 — Autenticação de Claude e Antigravity](08-desenvolvimento/tickets/planejamento/FAC-018-login-claude-antigravity.md).
78. [FAC-019 — Abas de configurações](08-desenvolvimento/tickets/planejamento/FAC-019-abas-configuracoes.md).
79. [OPS-001 — Bootstrap confiavel do desenvolvimento local](08-desenvolvimento/tickets/planejamento/OPS-001-bootstrap-desenvolvimento-local.md).
80. [OPS-003 — Consolidar revisões aceitas e deltas locais](08-desenvolvimento/tickets/planejamento/OPS-003-consolidacao-revisoes-aceitas.md).
81. [OPS-004 — Consolidar worktrees locais autorizados](08-desenvolvimento/tickets/planejamento/OPS-004-consolidar-worktrees-locais.md).
82. [OPS-005 — Worker confiável no host da VPS](08-desenvolvimento/tickets/planejamento/OPS-005-worker-host-sandbox.md).
83. [OPS-006 — Integração local e controle do MVP](08-desenvolvimento/tickets/planejamento/OPS-006-integracao-local-controle-mvp.md).
84. [OPS-007 — Consolidar FAC-012Z–AH e remover worktrees incorporadas](08-desenvolvimento/tickets/planejamento/OPS-007-consolidacao-worktrees.md).
85. [OPS-008 — Diagnóstico das telas e comandos Prisma com ambiente raiz](08-desenvolvimento/tickets/planejamento/OPS-008-bootstrap-migrations-telas.md).
86. [OPS-009 — Reconciliar migration histórica no banco de desenvolvimento](08-desenvolvimento/tickets/planejamento/OPS-009-reconciliar-migration-historica.md).

## Histórico de entregas

1. [DOC-MIGRACAO — pacote Manual Vivo](09-entregas/2026/2026-10-06-DOC-MIGRACAO-manual-vivo.md).
2. [DOC-MV-001 — Migração inicial do Manual Vivo](09-entregas/2026/2026-10-07-DOC-MV-001-migracao-manual-vivo.md).
3. [DOC-MV-002 — Organização da raiz e retirada de documentacoes](09-entregas/2026/2026-10-07-DOC-MV-002-organizacao-raiz.md).
4. [DOC-MV-003 — Manual em ordem didática](09-entregas/2026/2026-10-07-DOC-MV-003-ordem-didatica.md).
5. [Entrega: correção consistente da stack](09-entregas/2026/arquitetura/2026-09-30-correcao-stack.md).
6. [Entrega FAC-010A — Verificacao gerenciada de instalacoes](09-entregas/2026/configuracao/2026-10-01-FAC-010A-verificacao-instalacoes.md).
7. [Entrega FAC-010B — Login efemero do Codex](09-entregas/2026/configuracao/2026-10-01-FAC-010B-login-efemero-codex.md).
8. [Entrega FAC-011 — Centro de configuracoes](09-entregas/2026/configuracao/2026-10-01-FAC-011-centro-configuracoes.md).
9. [Entrega FAC-011A — Verificacao gerenciada do GitHub CLI](09-entregas/2026/configuracao/2026-10-01-FAC-011A-verificacao-github-cli.md).
10. [Entrega FAC-011B — Login efemero do GitHub CLI](09-entregas/2026/configuracao/2026-10-01-FAC-011B-login-efemero-github.md).
11. [Entrega FAC-011C — Verificacao somente-leitura do repositorio GitHub](09-entregas/2026/configuracao/2026-10-02-FAC-011C-verificacao-repositorio-github.md).
12. [Entrega FAC-011D — Criacao de pull request sob gate humano](09-entregas/2026/configuracao/2026-10-02-FAC-011D-criacao-pull-request-gate-humano.md).
13. [FAC-013 — Lista de contas IA e configuração em modal](09-entregas/2026/configuracao/2026-10-05-FAC-013-modal-contas-ia.md).
14. [FAC-014 — Modais para funcionários digitais e GitHub](09-entregas/2026/configuracao/2026-10-05-FAC-014-modais-funcionarios-github.md).
15. [FAC-015 — Assinaturas autenticadas e catálogo compartilhado](09-entregas/2026/configuracao/2026-10-05-FAC-015-contas-autenticadas-modelos.md).
16. [FAC-016 — Retorno da verificação Codex no modal](09-entregas/2026/configuracao/2026-10-05-FAC-016-feedback-verificacao-codex.md).
17. [FAC-017 — Autorização Codex em nova aba](09-entregas/2026/configuracao/2026-10-05-FAC-017-login-device-codex.md).
18. [FAC-018 — Login oficial Claude e Antigravity](09-entregas/2026/configuracao/2026-10-05-FAC-018-login-claude-antigravity.md).
19. [FAC-019 — Abas de configurações](09-entregas/2026/configuracao/2026-10-05-FAC-019-abas-configuracoes.md).
20. [FAC-023 — Agentes e skills por projeto](09-entregas/2026/configuracao/2026-10-05-FAC-023-agentes-skills-projeto.md).
21. [FAC-026 — Agentes pré-configurados na lista](09-entregas/2026/configuracao/2026-10-06-FAC-026-agentes-preconfigurados.md).
22. [FAC-006 — Context Builder e RuntimeGuard](09-entregas/2026/contexto/2026-09-30-FAC-006-context-builder-runtime-guard.md).
23. [Entrega FAC-003 — controle web e persistência](09-entregas/2026/controle/2026-09-30-FAC-003-controle-web-persistencia.md).
24. [FAC-001A — Definicao de projeto configuravel](09-entregas/2026/controle/2026-10-02-FAC-001A-definicao-projeto-configuravel.md).
25. [FAC-003A — Gate de revisao-base exata antes de READY](09-entregas/2026/controle/2026-10-02-FAC-003A-gate-revisao-base-ready.md).
26. [FAC-012I — Perfil de execução do projeto](09-entregas/2026/controle/2026-10-02-FAC-012I-perfil-execucao-projeto.md).
27. [FAC-012AC — Exclusão global do writer e independência do piloto](09-entregas/2026/controle/2026-10-03-FAC-012AC-writer-global.md).
28. [FAC-012AD — Estado operacional da fábrica](09-entregas/2026/controle/2026-10-03-FAC-012AD-painel-operacao.md).
29. [FAC-012AE — Eventos autenticados e retomáveis de projeto](09-entregas/2026/controle/2026-10-03-FAC-012AE-eventos-projeto.md).
30. [FAC-012AG — Pausa global persistida da fábrica](09-entregas/2026/controle/2026-10-03-FAC-012AG-pausa-global.md).
31. [FAC-012AH — Jobs preservados na admissão e conclusão interna do MVP](09-entregas/2026/controle/2026-10-03-FAC-012AH-admissao-capacidade.md).
32. [FAC-012M — Resultados de execução no painel](09-entregas/2026/controle/2026-10-03-FAC-012M-resultados-execucao-painel.md).
33. [FAC-012P — Diff e artefatos no painel](09-entregas/2026/controle/2026-10-03-FAC-012P-artefatos-painel.md).
34. [FAC-012Q — Documentação e aceite da entrega](09-entregas/2026/controle/2026-10-03-FAC-012Q-documentacao-aceite-entrega.md).
35. [FAC-012S — Comandos administrativos de execução](09-entregas/2026/controle/2026-10-03-FAC-012S-comandos-run.md).
36. [FAC-012T — Retomada segura do trabalho preservado](09-entregas/2026/controle/2026-10-03-FAC-012T-retomada-snapshot.md).
37. [FAC-012U — Recuperar finalização, não executar novamente](09-entregas/2026/controle/2026-10-03-FAC-012U-recuperacao-finalizacao.md).
38. [FAC-012W — Transporte ampliado de artefatos](09-entregas/2026/controle/2026-10-03-FAC-012W-artefatos-ampliados.md).
39. [FAC-012Z — Gate de documentação técnica](09-entregas/2026/controle/2026-10-03-FAC-012Z-gate-documentacao-tecnica.md).
40. [FAC-020 — Central de controle estática](09-entregas/2026/controle/2026-10-05-FAC-020-central-controle-estatica.md).
41. [FAC-020A — Home no aplicativo habitual](09-entregas/2026/controle/2026-10-05-FAC-020A-home-em-developer.md).
42. [FAC-021 — La fabrique e escala da home](09-entregas/2026/controle/2026-10-05-FAC-021-nome-escala-home.md).
43. [FAC-022 — Template persistente e tipografia compacta](09-entregas/2026/controle/2026-10-05-FAC-022-template-tipografia.md).
44. [FAC-023 — Agentes e skills: controle](09-entregas/2026/controle/2026-10-05-FAC-023-agentes-skills.md).
45. [FAC-024 — Vídeo decorativo e área compacta na home](09-entregas/2026/controle/2026-10-05-FAC-024-video-dashboard.md).
46. [FAC-025 — Retirar vídeo do dashboard](09-entregas/2026/controle/2026-10-06-FAC-025-retirar-video-dashboard.md).
47. [FAC-027 — Menu lateral recolhível](09-entregas/2026/controle/2026-10-06-FAC-027-menu-recolhivel.md).
48. [FAC-028 — Seta minimalista no menu](09-entregas/2026/controle/2026-10-06-FAC-028-seta-menu-minimalista.md).
49. [FAC-029 — Seta na borda direita do menu](09-entregas/2026/controle/2026-10-06-FAC-029-seta-direita-menu.md).
50. [FAC-030 — Menu recolhido ao abrir](09-entregas/2026/controle/2026-10-06-FAC-030-menu-inicial-recolhido.md).
51. [FAC-031 — Posição da seta por estado do menu](09-entregas/2026/controle/2026-10-06-FAC-031-posicao-seta-menu.md).
52. [FAC-012Y — Handoff automático configurado](09-entregas/2026/handoff/2026-10-03-FAC-012Y-handoff-configurado.md).
53. [Aceite do FAC-000 — bootstrap TypeScript](09-entregas/2026/infraestrutura/2026-09-30-FAC-000-aceite.md).
54. [Entrega FAC-000 — bootstrap TypeScript](09-entregas/2026/infraestrutura/2026-09-30-FAC-000-bootstrap-typescript.md).
55. [FAC-007 — Sandbox e snapshots recuperaveis](09-entregas/2026/infraestrutura/2026-09-30-FAC-007-sandbox-snapshots.md).
56. [FAC-012J — Checkout confiável no worker](09-entregas/2026/infraestrutura/2026-10-02-FAC-012J-checkout-confiavel-worker.md).
57. [OPS-001 — Bootstrap confiavel do desenvolvimento local](09-entregas/2026/infraestrutura/2026-10-02-OPS-001-bootstrap-desenvolvimento-local.md).
58. [OPS-005 — Worker host para o sandbox da VPS](09-entregas/2026/infraestrutura/2026-10-03-OPS-005-worker-host-sandbox.md).
59. [FAC-018 — Keyring privado Antigravity](09-entregas/2026/infraestrutura/2026-10-05-FAC-018-keyring-privado.md).
60. [OPS-008 — Comandos Prisma e diagnóstico das telas](09-entregas/2026/infraestrutura/2026-10-05-OPS-008-bootstrap-migrations-telas.md).
61. [OPS-009 — Reconciliação da migration histórica no banco dev](09-entregas/2026/infraestrutura/2026-10-05-OPS-009-reconciliar-migration-historica.md).
62. [Entrega FAC-004 — identidade e heartbeat do worker](09-entregas/2026/operacao/2026-09-30-FAC-004-worker-identidade.md).
63. [FAC-008 — Orquestrador, leases e checkpoints](09-entregas/2026/operacao/2026-09-30-FAC-008-orquestrador-checkpoints.md).
64. [FAC-008 — Correcao de idempotencia e consumidor](09-entregas/2026/operacao/2026-10-01-FAC-008-correcao-idempotencia-consumidor.md).
65. [FAC-009 — Developer, checks e revisao](09-entregas/2026/operacao/2026-10-01-FAC-009-developer-checks-review.md).
66. [FAC-012A — Especificacao imutavel de execucao](09-entregas/2026/operacao/2026-10-02-FAC-012A-especificacao-execucao-imutavel.md).
67. [FAC-012B — Compilador da especificação para workflow](09-entregas/2026/operacao/2026-10-02-FAC-012B-compilador-workflow-configuravel.md).
68. [FAC-012C — Política de escrita do SandboxRunner](09-entregas/2026/operacao/2026-10-02-FAC-012C-politica-escrita-sandbox.md).
69. [FAC-012D — Escrita do Codex limitada aos caminhos do projeto](09-entregas/2026/operacao/2026-10-02-FAC-012D-caminhos-escrita-cli.md).
70. [FAC-012E — Configuração versionada disponível ao worker](09-entregas/2026/operacao/2026-10-02-FAC-012E-configuracao-runtime-worker.md).
71. [FAC-012F — Rota de agente por configuração](09-entregas/2026/operacao/2026-10-02-FAC-012F-rota-agente-configurada.md).
72. [FAC-012G — Workflow com rota por função](09-entregas/2026/operacao/2026-10-02-FAC-012G-workflow-rotas-por-funcao.md).
73. [FAC-012H — Isolamento da raiz de checks](09-entregas/2026/operacao/2026-10-02-FAC-012H-isolamento-raiz-sandbox.md).
74. [FAC-012I — Perfil de execução por projeto](09-entregas/2026/operacao/2026-10-02-FAC-012I-perfil-execucao-projeto.md).
75. [OPS-003 — Aceites e reconciliação dos deltas locais](09-entregas/2026/operacao/2026-10-02-OPS-003-aceites-e-deltas-locais.md).
76. [FAC-012AB — Ensaio integrado interno do MVP](09-entregas/2026/operacao/2026-10-03-FAC-012AB-ensaio-integrado-mvp.md).
77. [FAC-012AF — Backup criptografado e restauração isolada](09-entregas/2026/operacao/2026-10-03-FAC-012AF-backup-restauracao.md).
78. [FAC-012K — lease vivo e cancelamento conservador](09-entregas/2026/operacao/2026-10-03-FAC-012K-lease-vivo-cancelamento.md).
79. [FAC-012L — Consumer de execução real com ativação explícita](09-entregas/2026/operacao/2026-10-03-FAC-012L-consumer-execucao-real.md).
80. [FAC-012N — Reconciliação após checkpoint](09-entregas/2026/operacao/2026-10-03-FAC-012N-reconciliacao-checkpoint.md).
81. [FAC-012O — Journal de resultados](09-entregas/2026/operacao/2026-10-03-FAC-012O-journal-resultados.md).
82. [FAC-012R — Snapshot de interrupção comprovada](09-entregas/2026/operacao/2026-10-03-FAC-012R-snapshot-interrupcao.md).
83. [FAC-012X — Espera por provider com trabalho preservado](09-entregas/2026/operacao/2026-10-03-FAC-012X-espera-provider.md).
84. [OPS-004 — Consolidação local das worktrees](09-entregas/2026/operacao/2026-10-03-OPS-004-consolidacao-worktrees.md).
85. [OPS-006 — Integração local e controle do MVP](09-entregas/2026/operacao/2026-10-03-OPS-006-integracao-local-controle-mvp.md).
86. [FAC-015 — Preparação das assinaturas na VPS](09-entregas/2026/operacao/2026-10-05-FAC-015-preparacao-assinaturas.md).
87. [FAC-016 — .env e processo dev ativo](09-entregas/2026/operacao/2026-10-05-FAC-016-env-processo-dev.md).
88. [FAC-017 — Confirmar a assinatura Codex](09-entregas/2026/operacao/2026-10-05-FAC-017-confirmacao-login-codex.md).
89. [FAC-018 — Testar autenticação](09-entregas/2026/operacao/2026-10-05-FAC-018-keyring-autenticacao.md).
90. [OPS-007 — Integração de FAC-012Z–AH na developer](09-entregas/2026/operacao/2026-10-05-OPS-007-consolidacao-worktrees.md).
91. [FAC-002 — Preflight do Codex oficial](09-entregas/2026/runtime/2026-09-30-FAC-002-preflight-codex.md).
92. [FAC-005 — Runtime Gateway e adapter Codex](09-entregas/2026/runtime/2026-09-30-FAC-005-runtime-gateway-codex.md).
93. [FAC-010 — Preflight do segundo provider](09-entregas/2026/runtime/2026-10-01-FAC-010-preflight-segundo-provider.md).
94. [Entrega FAC-010C — Adapter Claude e handoff seguro](09-entregas/2026/runtime/2026-10-01-FAC-010C-adapter-claude-handoff.md).
95. [FAC-012AA — Perfil granular e preflight Claude](09-entregas/2026/runtime/2026-10-03-FAC-012AA-confinamento-claude.md).
96. [FAC-012V — Identidades privadas por instalação](09-entregas/2026/runtime/2026-10-03-FAC-012V-identidades-instalacoes.md).
97. [FAC-015 — Descoberta de modelos no cliente oficial](09-entregas/2026/runtime/2026-10-05-FAC-015-catalogo-clientes-oficiais.md).
98. [FAC-016 — Diagnóstico sanitizado da identidade](09-entregas/2026/runtime/2026-10-05-FAC-016-diagnostico-raiz-privada.md).
99. [FAC-017 — Desafio oficial colorido](09-entregas/2026/runtime/2026-10-05-FAC-017-desafio-ansi-codex.md).
100. [FAC-018 — Autorização dos clientes oficiais](09-entregas/2026/runtime/2026-10-05-FAC-018-autorizacao-clientes.md).
101. [FAC-023 — Agentes e skills: runtime](09-entregas/2026/runtime/2026-10-05-FAC-023-agentes-skills.md).


## Skills correntes e registro da adaptação

1. [Perfil das skills — La fabrique](../skills/00-perfil-la-fabrique.md).
2. [Arquiteto de software](../skills/arquiteto-de-software/SKILL.md).
3. [Roteiro arquitetural da fábrica](../skills/arquiteto-de-software/references/roteiro-arquitetural.md).
4. [Fechar entrega](../skills/fechar-entrega/SKILL.md).
5. [Registro de entrega local](../skills/fechar-entrega/references/registro-local.md).
6. [Implementador de ticket](../skills/implementador-de-ticket/SKILL.md).
7. [Orquestrador local](../skills/orquestrador-fluxo-ia/SKILL.md).
8. [Estado e gates locais](../skills/orquestrador-fluxo-ia/references/estado-e-gates.md).
9. [Fila e correções de review](../skills/review-loop-driver/SKILL.md).
10. [Revisor de código](../skills/revisor-de-codigo/SKILL.md).
11. [Correção e performance — TypeScript/PostgreSQL/Redis](../skills/revisor-de-codigo/references/correcao-performance.md).
12. [Qualidade — convenções da La fabrique](../skills/revisor-de-codigo/references/qualidade.md).
13. [AppSec do diff — fronteiras da fábrica](../skills/revisor-de-codigo/references/seguranca.md).
14. [Revisor de QA](../skills/revisor-de-qa/SKILL.md).
15. [Auditoria de segurança](../skills/security-audit/SKILL.md).
16. [Checks do monorepo e VPS](../skills/security-audit/references/ecosystem-checks.md).
17. [Superfícies de segurança da La fabrique](../skills/security-audit/references/security-checklist.md).
18. [Simplificação de código](../skills/simplify/SKILL.md).
19. [Tech Lead](../skills/tech-lead/SKILL.md).
20. [Planejador de worktrees](../skills/worktree-planner/SKILL.md).
21. [Plano de isolamento local](../skills/worktree-planner/references/plano-local.md).
22. [DOC-MV-004 — Adaptar skills importadas](08-desenvolvimento/tickets/DOC-MV-004.md).
23. [DOC-MV-004 — Skills adaptadas para a La fabrique](09-entregas/2026/2026-10-07-DOC-MV-004-skills-la-fabrique.md).

## Checkpoint de integração e limpeza

1. [DOC-MV-005 — commit bloqueado pelo ambiente](09-entregas/2026/2026-10-07-DOC-MV-005-checkpoint-integracao.md).

## Evolução do menu principal

1. [FAC-032 — Menu inicial reduzido](09-entregas/2026/2026-10-07-FAC-032-menu-inicial-reduzido.md).

## Incremento de tarefas

- [Painel Tarefas demonstrativo](04-features/01-control-settings.md#painel-tarefas--demonstração-fac-033).
- [Ticket FAC-033](08-desenvolvimento/tickets/FAC-033.md).
- [Entrega FAC-033](09-entregas/2026/2026-10-07-FAC-033-painel-tarefas.md).

## Centro de Comando — FAC-034 (consulta)

- [Ticket autorizado](08-desenvolvimento/tickets/controle/FAC-034-centro-comando.md).
- [Entrega, checks e limites](09-entregas/2026/2026-10-07-FAC-034-centro-comando.md).
- [Handoff original de design](09-entregas/2026/evidencias/design_handoff_centro_comando/README.md).
- [Prompt original recebido](09-entregas/2026/evidencias/design_handoff_centro_comando/PROMPT-CLAUDE-CODE.md).

## Nome dos agentes — FAC-035 (consulta)

- [Ticket](08-desenvolvimento/tickets/configuracao/FAC-035-nomes-agentes.md).
- [Entrega e checks](09-entregas/2026/2026-10-08-FAC-035-nomes-agentes.md).

- [FAC-035: correção da inicialização para nomes](09-entregas/2026/2026-10-09-FAC-035-correcao-salvamento-nomes.md).

### Incremento FAC-037 — cadastro API/CLI

1. [Ticket FAC-037](08-desenvolvimento/tickets/configuracao/FAC-037-api-cli-agentes.md).
2. [Entrega FAC-037](09-entregas/2026/2026-10-09-FAC-037-api-cli-agentes.md).

## Complemento do painel da conta — consulta FAC-039–041

1. [Projetos/grants/execução](08-desenvolvimento/tickets/FAC-039.md).
2. [Integrações/uso/dados](08-desenvolvimento/tickets/FAC-040.md).
3. [Segurança/email/planos](08-desenvolvimento/tickets/FAC-041.md).
4. [Entrega e evidências complementares](09-entregas/2026/2026-10-09-FAC-041-complemento-painel-conta.md).

## Instruções Markdown — FAC-036 (consulta)

- [Ticket](08-desenvolvimento/tickets/configuracao/FAC-036-instrucoes-markdown.md).
- [Entrega e checks](09-entregas/2026/2026-10-09-FAC-036-instrucoes-markdown.md).
- [Leitura obrigatória e evidências da sessão FAC-036](09-entregas/2026/evidencias/FAC-036/leitura-checks.md).
- [Revisão e verificação em browser após integração](09-entregas/2026/evidencias/FAC-036/revisao-integracao.md).

## Escritório virtual — FAC-042 (consulta)

- [Ticket](08-desenvolvimento/tickets/controle/FAC-042-escritorio-virtual.md).
- [Entrega e checks](09-entregas/2026/2026-10-24-FAC-042-escritorio-virtual.md).

## Visual sóbrio — FAC-043 (consulta)

- [Ticket](08-desenvolvimento/tickets/controle/FAC-043-visual-sobrio.md).
- [Entrega e checks](09-entregas/2026/2026-10-25-FAC-043-visual-sobrio.md).

## Memória e decisões — guias, planos e contratos

1. [Aprender a arquitetura de memória e decisões](02-arquitetura/memoria-e-decisoes/00-README.md).
2. [Visão integrada: memória, contexto e decisão](02-arquitetura/memoria-e-decisoes/01-visao-integrada.md).
3. [Memória persistente: o que a fábrica precisa lembrar](02-arquitetura/memoria-e-decisoes/02-memoria-persistente.md).
4. [Índices e grafo: como descobrir relações](02-arquitetura/memoria-e-decisoes/03-indexacao-e-grafo.md).
5. [Context Packs: como entregar conhecimento a uma tarefa](02-arquitetura/memoria-e-decisoes/04-context-packs.md).
6. [Decision Engine: quem decide o quê](02-arquitetura/memoria-e-decisoes/05-decision-engine.md).
7. [Modelo local: encontrar semelhanças sem gerar código](02-arquitetura/memoria-e-decisoes/06-modelo-local.md).
8. [JEV: julgamento entre alternativas permitidas](02-arquitetura/memoria-e-decisoes/07-jev.md).
9. [Aprendizado: transformar resultados em exemplos confiáveis](02-arquitetura/memoria-e-decisoes/08-aprendizado.md).
10. [Fluxo completo: como os componentes trabalham juntos](02-arquitetura/memoria-e-decisoes/09-fluxo-completo.md).
11. [Planos arquiteturais de memória e decisões](08-desenvolvimento/planos-memoria-e-decisoes/00-README.md).
12. [Decisões confirmadas, propostas e evidências](08-desenvolvimento/planos-memoria-e-decisoes/01-decisoes-e-dependencias.md).
13. [Plano — Memória persistente](08-desenvolvimento/planos-memoria-e-decisoes/02-memoria-persistente.md).
14. [Plano — Indexação, grafo e análise de impacto](08-desenvolvimento/planos-memoria-e-decisoes/03-indexacao-e-grafo.md).
15. [Plano — Context Packs e continuidade](08-desenvolvimento/planos-memoria-e-decisoes/04-context-packs.md).
16. [Plano — Decision Engine](08-desenvolvimento/planos-memoria-e-decisoes/05-decision-engine.md).
17. [Plano futuro — Modelo local e recuperação semântica](08-desenvolvimento/planos-memoria-e-decisoes/06-modelo-local.md).
18. [Plano futuro — Integração JEV preparada para configuração no sistema](08-desenvolvimento/planos-memoria-e-decisoes/07-jev.md).
19. [Plano futuro — Dataset e classificador próprio](08-desenvolvimento/planos-memoria-e-decisoes/08-aprendizado.md).
20. [Plano — Fluxo completo e matriz de decisões](08-desenvolvimento/planos-memoria-e-decisoes/09-fluxo-completo.md).
21. [Matriz de rastreabilidade do fluxo discutido](08-desenvolvimento/planos-memoria-e-decisoes/10-matriz-fluxo.md).
22. [Contratos comuns propostos — memória e decisões](05-contratos/memoria-e-decisoes/00-contratos-propostos.md).
23. [ADR-007 — Memória e decisões versionadas](06-decisoes/ADR-007-memoria-e-decisoes-versionadas.md).
24. [ADR-008 — Exceção específica para API JEV](06-decisoes/ADR-008-excecao-jev.md).

## Entrega arquitetural de memória e decisões

1. [ARQ-MEM-001 — planos, guias e evidências](09-entregas/2026/2026-10-09-ARQ-MEM-001-planos-memoria-decisoes.md).
