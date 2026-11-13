# FAC-034 — Centro de Comando em todo apps/web

2026-10-07, Europe/Berlin. IMPLEMENTADO no diff local da `developer`; AWAITING_HUMAN. Base e origin/main: `de5dbfac0b00f3e1699b999f987856f592f65899`. Sem commit/run/PR novo: índice Git somente leitura. Comparação por `git log --left-right developer...origin/main` e `git diff origin/main developer` antes da edição mostrou igualdade, inclusive DashboardLayout, home-dashboard.css, App e specs. Handoff recebido era o único diretório não rastreado; preservado integralmente com hashes.

## Objetivo, problema e aceite

Substituir a cena cartoon da home e unificar a apresentação das áreas com o tema HUD do [handoff recebido](evidencias/design_handoff_centro_comando/README.md), [dashboard hi-fi](<evidencias/design_handoff_centro_comando/Fabrica Dashboard v3.dc.html>) e [kit de UI](<evidencias/design_handoff_centro_comando/HUD UI Kit.dc.html>). Aplicação em tokens → frame/home → login → Projetos/Operação → Tarefas → Configurações/modais, com documentação e etapas revisáveis. [Ticket READY autorizado](../../08-desenvolvimento/tickets/controle/FAC-034-centro-comando.md).

Critérios: preservar props/HomeDestination, navegação, aria e textos; manter sidebar recolhível, busca ⌘K/Ctrl+K, prévia, skip link e aviso mock; núcleo SVG/CSS com movimento reduzido; todos os estados reais coloridos; API e regras preservadas; lint/typecheck/testes/build e documentação verificáveis. Aceite humano da revisão exata e QA visual continuam pendentes.

## Solução e funcionamento

`theme-hud.css` é importado antes de `styles.css`: grade/fundo escuro, ciano/teal/âmbar/rosa/violeta, Rajdhani/Orbitron/JetBrains Mono e tokens para superfície, foco, raios, glow e espaçamento. Cores fixas permanecem apenas no arquivo de tokens; referências recebidas mantêm sua paleta original íntegra.

Frame usa topbar com logo, status explicitamente demonstrativo, relógio local atualizado a cada segundo com cleanup, busca, notificações/ajuda/perfil; sidebar 226px expandida e 66px recolhida; trilho de configurações 78px, convertido em faixa inferior no mobile. Props, destinos e handlers existentes permanecem. Novos itens Núcleo IA/Pipeline/Revisões/Repositórios/Histórico abrem prévias; o badge Revisões diz Prévia, sem inventar contagem de revisões. Trilho abre a mesma área settings. Sidebar inicia recolhida e preserva a expansão durante navegação; media query do comportamento anterior permanece.

Home organiza status/diagnóstico, núcleo central e notificações acima; agentes/projetos/indicadores abaixo; atalhos e mensagens continuam disponíveis. Arrays/textos atuais foram preservados. Diagnóstico usa atividade mock dos agentes e declara que não é cota/consumo; projetos recentes não foram transformados em runs fictícias. ENVIAR é atalho para Tarefas, sem enviar/persistir diretivas.

`OrchestrationCore` substitui a imagem: 120 ticks, anel âmbar, arcos, esfera/elipses, rede de 34 pontos com seed fixa, órbitas/ondas, partículas e JARVIS no centro. Os quatro outros agentes atuais aparecem nos nós, em vez dos seis agentes ilustrativos do HTML. Gradientes/filtros usam IDs de useId; animações são CSS, sem SMIL/animateMotion. `prefers-reduced-motion: reduce` desativa animações e transições, inclusive durações inline. O SVG é identificado por role img e aria-label Núcleo de orquestração JARVIS.

Login tem card HUD, anel decorativo e campo mono. Mensagens mantêm o texto original/role status e recebem tom visual OK/INFO/ATENÇÃO/ERRO; a marca adicional usa aria-hidden. Projetos/Operação, definição, runs/controle/recovery/resume, entrega/artefatos e handoff herdam inputs/cards/botões/logs/tabelas do tema. Pausar/cancelar usam apresentação de perigo sem alterar elegibilidade/confirmação. Os 17 estados reais do contrato têm mapeamento explícito, sem introduzir IN_REVIEW/BLOCKED do kit.

Tarefas conserva filtros, armazenamento, validação, atribuição, quadro/lista e mocks; nove etapas ganham cores em colunas/cards/badges, mantendo fase distinta de etapa. Configurações/contas/modelos/equipes/skills/agentes predefinidos preservam abas, rascunhos, switches e diálogos; modais de todas as áreas compartilham superfície/backdrop/foco HUD.

## Arquivos, decisões, dados e API

Mudanças concentradas em `apps/web/src`: theme-hud.css/main, DashboardLayout/home-dashboard.css, OrchestrationCore, App/HudSystemMessage, styles.css, TasksPanel/tasks.css, RunPanel/RunControlPanel/OperationPanel e specs. Demais componentes são alcançados pelos estilos compartilhados; não foi necessário editar seus handlers.

Sem mudança de API, schemas, modelos persistidos, banco, eventos, permissões, stack, dependências npm ou configuração de providers. Cliente React/Vite e ADR-003 mantidos; sem nova decisão arquitetural/C4. Nenhuma credencial consultada. Bundle não inclui o handoff/prompt de referência nem os patches de evidência.

## Diff, revisão e artefatos

[Manifesto](evidencias/FAC-034/manifest.json) registra base, SHA-256 das cinco fontes recebidas, patches e hashes por arquivo/etapa. [Verificação de reconstrução](evidencias/FAC-034/patch-verification.txt): seis patches foram aplicados em sequência a arquivos limpos da base numa pasta temporária, sem índice Git, e reconstruíram integralmente os fontes finais de apps/web.

[Comandos de commits separados](evidencias/FAC-034/commits-comentados.sh) usam os patches apenas no index, permitindo commits por etapa sobre o diff já existente, e último commit documental. Todos os comandos estão comentados. `git add` foi tentado e rejeitado: Unable to create .git/index.lock: Read-only file system. Não houve contorno de permissões, push ou merge. O pedido atual de não fazer merge prevalece sobre o comando do prompt histórico recebido.

Autorrevisão do diff confirma alterações de apresentação nos componentes administrativos e manutenção dos arrays/contratos/handlers. Não equivale a revisão independente; esta e o aceite visual são NÃO VERIFICADOS. Referência HTML contém support.js/DCLogic ausentes; seus arquivos foram lidos como especificação, não executados ou copiados para a aplicação.

## Checks reais

Todos os checks de software se referem ao diff sobre de5dbfa, não a um novo SHA de commit. Logs finais: [lint](evidencias/FAC-034/lint.txt), [typecheck](evidencias/FAC-034/typecheck.txt), [testes](evidencias/FAC-034/tests.txt), [build](evidencias/FAC-034/build.txt), [resumo dos checks](evidencias/FAC-034/checks.json); nenhum teste de produção, provider, Redis ou PostgreSQL foi iniciado.

| Critério | Comando/procedimento | Resultado |
|---|---|---|
| Baseline | npm run test -w @le-fabrique/web, antes de editar | PASS, 20 arquivos/53 testes |
| Lint | npm run lint | PASS, um warning preexistente useOptionalChain em apps/api/src/orchestration/run-delivery.service.ts:209 |
| Tipagem web | npm run typecheck -w @le-fabrique/web | PASS |
| Specs web | npm run test -w @le-fabrique/web | PASS, 22 arquivos/62 testes |
| Build web | npm run build -w @le-fabrique/web | PASS |
| Documentação | python3 scripts/validar-documentacao.py | PASS, 385 Markdown correntes/601 snapshots; [log](evidencias/FAC-034/validation-docs.txt) |
| Diff | git diff --check | PASS |
| Patches | Aplicação sequencial em base limpa e comparação byte a byte | PASS |
| Roteiro de browser | node --check evidencias/FAC-034/browser-check.mjs | PASS somente sintaxe; execução funcional NOT_RUN |
| Vite local | npm run dev -w @le-fabrique/web -- --host 127.0.0.1 --port 5185 --strictPort | BLOQUEADO: listen EPERM |
| Chromium | Playwright com chromiumSandbox:true e bibliotecas já disponíveis | BLOQUEADO: sandbox_host_linux.cc shutdown Operation not permitted |
| Fidelidade hi-fi/interações/responsividade em browser | Conferência visual/cenários abaixo | NÃO VERIFICADO |
| Revisão independente/aceite | Sessão separada e aceite da revisão exata | NÃO VERIFICADO |

O novo teste CSS inicialmente falhou: import ?raw era stubado pelo Vitest. Corrigido para leitura direta com tipos Node restritos ao spec, sem mudar config ou instalar dependência. Typecheck havia identificado a falta desses tipos no primeiro teste; configuração da aplicação foi preservada. Falhas intermediárias não são resultado final.

## Verificação visual reproduzível

Em ambiente de desenvolvimento autorizado que permita porta local e browser, executar Vite pelo comando do projeto e abrir a home. [Roteiro automatizado](evidencias/FAC-034/browser-check.mjs) usa browser com sandbox e checa somente home/login/Tarefas demonstrativos, prévia, foco da busca, movimento reduzido e larguras 1536/1024/768/390/320. Requer Playwright já disponível; não instalar nem reduzir sandbox para contornar restrições. `PREVIEW_URL` é local, `PLAYWRIGHT_MODULE` pode apontar a instalação existente. O roteiro não autentica nem chama API administrativa; quando passar, gera screenshots/results em sua pasta browser/.

Conferência humana complementar: comparar home com o hi-fi (core/glow, títulos, spacing, relógio, quatro agentes reais); testar ⌘K/Ctrl+K, navegação com teclado, expandir/recolher, resize e reload; verificar modal/fechar/Escape, aviso mock e meters; em ambiente de teste com sessão administrativa autorizada, percorrer Projetos/Operação/Run/artefato/recovery/resume e todas as abas/modais de Configurações, mantendo os gates originais. Verificar especialmente renderização das partículas por offset-path, fontes offline, zoom e overflow mobile. Não iniciar execução, login de provider ou produção para validar CSS.

## Riscos, limitações e rollback

Fidelidade hi-fi e comportamento real de layout/animação/foco são pendentes por indisponibilidade do browser. Google Fonts segue o pacote e possui fallbacks; fontes externas podem não carregar offline. Estados/tokens/testes estruturais não comprovam contraste ou usabilidade por si só. Dados da home/Tarefas continuam demonstrativos; status OPERACIONAL da topbar é rotulado demonstração e não afirma saúde operacional.

Rollback: após commits, reverter os seis commits de apresentação em ordem inversa, seguido de nova validação; preservar evidências históricas. Antes de commitar, aplicar os patches em ordem inversa com opção reversa somente após confirmar que seus hashes ainda correspondem ao diff. Não descartar trabalho posterior ou diretório recebido. Rollback de software não ensaiado; reconstrução dos patches foi verificada. Não há migration ou reversão de dados.

## Documentação e lessons

Atualizados [controle](../../03-modulos/controle/00-README.md), [configuração](../../03-modulos/configuracao/00-README.md), [feature](../../04-features/01-control-settings.md), [matriz](../../08-desenvolvimento/05-matriz-cobertura.md), [backlog](../../08-desenvolvimento/09-backlog.md), [changelog](../../04-CHANGELOG.md), [índice](../../02-INDEX.md) e catálogo de entregas. Mapa de leitura ganha exceção justificada para README do handoff como evidência íntegra; não é novo capítulo corrente. Nenhuma alteração de AGENTS/CLAUDE/políticas necessária: o fluxo documental já é o corrente. Sem nova lesson: aplicação do tema não exige criar conceito independente/duplicado.

## Uso de IA, custos e pendências

Execução por Codex nesta sessão; versão/modelo efetivo/cota/custo não informados no ambiente, não estimados como evidência. Sem subagentes, handoffs de provider, instalação ou chamadas de IA pelo software. Limite de dois ciclos de correção observado nos checks da implementação; falha de browser é restrição do ambiente.

Próximos passos: revisão visual/independente pelo proprietário em ambiente permitido; criar commits por etapa quando Git for gravável; aceitar a revisão exata. AWAITING_HUMAN; nenhum DONE de software, push, merge, deploy ou publicação declarado.
