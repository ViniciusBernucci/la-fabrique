# FAC-030 — Menu recolhido ao abrir
2026-10-06. IMPLEMENTADO / AWAITING_HUMAN. Baseline 3906b77, código `3608f1766c50ee49071a6430d1087fa2fd538749`. [Ticket READY](../../../08-desenvolvimento/tickets/controle/tickets/FAC-030.md).
Pedido recebido durante FAC-029; incremento separado na mesma branch/worktree exclusiva. Mesmo writer/Codex da sessão, sem subagentes/handoff/provider adicional; modelo/cota/custo não comprovados. Merge/limpeza já autorizados; sem push/deploy.

## Funcionamento
DashboardLayout inicia sidebarCollapsed em true, inclusive renderização inicial. Ao abrir ou recarregar o software, sidebar fica em 66px/ícones; preferência antiga de menu aberto não prevalece. Toggle usa atualização funcional de estado; escolha manual permanece durante navegação pelo template, até nova abertura/reload. Removidas leitura/escrita de localStorage para esse estado. Chave anterior pode permanecer inerte no browser, sem excluir dados automaticamente. Mobile mantém recolhimento ao cruzar breakpoint e expansão manual sobre conteúdo. Seta minimalista continua na borda direita (FAC-029), sem texto/borda/fundo.

## Checks e evidências
46 testes web/18 arquivos, build web (inclui TypeScript), lint raiz e git diff --check PASS; aviso preexistente useOptionalChain da API mantido. Browser testa key antiga false, storage indisponível, abertura em 66px, expansão por teclado, navegação mantendo aberto, reload voltando recolhido, SVG à direita/borda zero/texto vazio e cinco larguras/mobile sem overflow/pageerror. [Resultado](../evidencias/controle/evidencias/FAC-030/browser-results.txt), [harness](../evidencias/controle/evidencias/FAC-030/browser-check.mjs), [abertura](../evidencias/controle/evidencias/FAC-030/default.png), [mobile](../evidencias/controle/evidencias/FAC-030/mobile.png), [testes](../evidencias/controle/evidencias/FAC-030/tests.txt), [build](../evidencias/controle/evidencias/FAC-030/build.txt), [lint](../evidencias/controle/evidencias/FAC-030/lint.txt), [patch sanitizado](../evidencias/controle/evidencias/FAC-030/implementation.patch). Sem correção adicional, cliente/provider/banco.

## Limites, rollback e docs
Expandir é escolha da navegação atual, sem persistência entre aberturas. Esse comportamento substitui a preferência após reload do FAC-027 conforme novo pedido. Sem mudança de backend/contrato/migration/dependência. Reverter `3608f1766c50ee49071a6430d1087fa2fd538749` restaura a inicialização/preferência anterior mantendo seta à direita; rollback não executado.
README raiz/controle, INDEX, CHANGELOG, BACKLOG/ticket e lesson de template atualizados; relatório FAC-029 conserva evidência de alinhamento e histórico da preferência antiga. AWAITING_HUMAN; DONE apenas com aceite visual exato.

## Integração e limpeza realizadas
Fast-forward developer até f4fabe7, build e browser 5173 PASS: recolhido ao abrir/recarregar, expansão durante navegação e seta à direita. Captura de abertura inspecionada. [Resultado developer](../evidencias/controle/evidencias/FAC-030/browser-results-developer.txt).
Worktree limpa e feature ancestral comprovadas; preview próprio PID 3109036 parado após conferir cwd. git worktree remove /home/vinicius/le-fabrique-fac-029 executado sem force; diretório ausente, dependências principais preservadas, somente developer em git worktree list. Branch feature preservada, sem push/deploy. Aceite visual permanece pendente.
