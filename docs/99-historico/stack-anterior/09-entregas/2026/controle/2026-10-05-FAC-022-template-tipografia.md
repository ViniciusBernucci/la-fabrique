# FAC-022 — Template persistente e tipografia compacta
Data: 2026-10-05. Status: IMPLEMENTADO / AWAITING_HUMAN.
Domínio: controle web/configurações.
Baseline: `cde0cc6`; revisão de código: `4e042a1232c628f1291d13696984dcd8da05bc92`; branch de preparação feat/fac-020-home-dashboard, integrada em developer.
## Objetivo e critérios
O responsável pediu menu lateral e superior persistentes, como template padrão, mudando apenas o conteúdo central; pediu também reduzir fontes anteriores para seguir o padrão da home. [Ticket READY FAC-022](../../../08-desenvolvimento/tickets/controle/tickets/FAC-022.md). Integração em developer e remoção da worktree já autorizadas.
## Implementação
`DashboardLayout.tsx` é o template único, com sidebar, topbar sticky, atalhos/busca, perfil e prévias. Recebe `children`, `activeDestination`, `onNavigate` e `onHome`. Sem children, apresenta a home existente; com children, apresenta o conteúdo administrativo em `workspace-content`. `HomeDashboard.tsx` mantém export compatível para a home.

`App.tsx` mantém o mesmo DashboardLayout na raiz para home, login, controle e configurações; somente o conteúdo central muda. A autenticação continua governando a montagem das telas administrativas. Marca/Painel retornam à home; menu destaca Projetos ou Configurações conforme tela ativa. A busca filtra atalhos na home e apresenta resultados sobre o conteúdo nas áreas internas. Removida a navegação administrativa duplicada do miolo.

CSS escopado ao conteúdo interno: h1 de página 24px, h2/cartões/modal 16px, h3 13px, texto/labels 12px e campos/botões 13px; métricas 22px. Cartões/listas permanecem escuros com ações primárias azuis. Topbar permanece no topo ao rolar; sidebar continua fixa. Breakpoints consideram o espaço ocupado pelos menus. Nome La fabrique e cena desktop de 60% do FAC-021 preservados.
## Funcionamento e limites
Menu/topbar mantêm os mesmos nós DOM e estado da busca nas trocas; os painéis administrativos são montados conforme a área escolhida, mantendo o comportamento anterior de seus rascunhos/salvamento. Header/workspace continuam demonstrativos. Login 401 não abre áreas protegidas. Trocar conteúdo não executa providers, não aprova tickets nem altera dados sozinho. Nenhum novo contrato/backend/migration ou dependência.
## Verificação e evidências
- Typecheck web: PASS; build web em worktree e developer: PASS, Vite 8.3.1/132 módulos.
- 43 testes web/17 arquivos PASS. Novo teste do template verifica slot central/menus/área selecionada; teste da home continua passando pelo export compatível.
- Lint raiz: PASS, 252 arquivos, mesmo aviso preexistente useOptionalChain da API.
- Browser Chromium 153.0.8010.12/Playwright 1.63.0: PASS na worktree/5174 e developer/5173. Identidade dos nós sidebar/topbar preservada home→login→configurações→controle→home; menu ativo, busca global e header sticky verificados. Fontes medidas: título 24px, card/modal 16px, labels do modal 12px. Modal GitHub aberto e fechado sem salvar.
- Home sem overflow em seis larguras 1536/1280/1024/768/390/320px; controle/configurações sem overflow em 1536/1024/768/390/320px. Capturas inspecionadas.
- `git diff --check`: PASS. O browser usa API fixtures somente GET: auth sintética, lista vazia, configuração com todas as travas desativadas e operação 503. Nenhuma chamada real de login/provider ou alteração de configuração. O estado indisponível na captura controle é intencional, não evidência de falha operacional real.

[Resultado worktree](../evidencias/controle/evidencias/FAC-022/browser-results.json), [resultado developer](../evidencias/controle/evidencias/FAC-022/browser-results-developer.json), [script](../evidencias/controle/evidencias/FAC-022/browser-check.mjs), [home](../evidencias/controle/evidencias/FAC-022/desktop.png), [configurações](../evidencias/controle/evidencias/FAC-022/settings-desktop.png), [controle](../evidencias/controle/evidencias/FAC-022/control-desktop.png), [configurações mobile](../evidencias/controle/evidencias/FAC-022/settings-mobile.png). Browser com `PREVIEW_URL=http://127.0.0.1:5173`, módulo Playwright temporário e bibliotecas extraídas em /tmp. Uma correção de semântica do link da marca e ajuste visual da lista; falha inicial do harness ao importar contrato TypeScript corrigida com fixture JSON validada pelo próprio frontend.
## Diff e revisão
[Patch sanitizado com contexto zero](../evidencias/controle/evidencias/FAC-022/implementation.patch), cde0cc6→4e042a1. Fast-forward em developer sem conflitos. Fonte de implementação igual entre testes e integração.
## Rollback
Reverter 4e042a1 desfaz template/fontes e conserva nome/imagem reduzida do cde0cc6; sem migrations ou dados a restaurar. Não realizar rollback destrutivo. Sem push/deploy/ativação de serviços nesta entrega.
## Documentação e lessons
README raiz/controle/configuração, INDEX, CHANGELOG, BACKLOG, ticket, relatório e [lesson de modais/estado](../../../10-lessons/13-modal-edicao-configuracao.md). Reutilizar o mesmo template na raiz preserva chrome/DOM; trocar apenas conteúdo não implica preservar rascunho de um painel desmontado. [Baseline/review](../../../10-lessons/11-baseline-regressao-review.md) distingue renderização estática, browser fixture e operação real.
## Uso de IA e aceite
Codex da sessão, mesmo writer, zero subagentes/handoffs/providers adicionais e nenhuma leitura/exportação de credenciais. Modelo efetivo/cota/custos não comprovados. AWAITING_HUMAN para revisão visual exata. Merge/remoção autorizados não representam aceite visual/DONE.

## Conclusão da integração e limpeza
Em 2026-10-05, merge final fast-forward até 776a742, ancestry da feature comprovada, worktree limpa. Preview próprio PID 2345979 encerrado após conferir cwd; `git worktree remove /home/vinicius/le-fabrique-fac-020` executado sem force. `git worktree list` passou a conter somente `/home/vinicius/le-fabrique [developer]`; diretório antigo ausente e node_modules principal preservado. Branch feature preservada, sem push/deploy. Registro administrativo de encerramento feito em developer após remover a worktree, sem alterar código/evidências.
