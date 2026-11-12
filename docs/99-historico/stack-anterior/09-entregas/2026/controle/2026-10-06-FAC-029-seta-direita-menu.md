# FAC-029 — Seta na borda direita do menu
2026-10-06. IMPLEMENTADO / AWAITING_HUMAN. Baseline b930802; código `ec98b7dc2f2ea700857810fe5df708bae91a720a`. [Ticket READY](../../../08-desenvolvimento/tickets/controle/tickets/FAC-029.md).
Mesmo writer/Codex desta sessão em branch fix/fac-029-menu-arrow-right/worktree isolada. Sem subagentes/handoff/provider adicional; modelo/cota/custo não comprovados. Merge/limpeza autorizados, sem push/deploy.

## Mudança e checks
Toggle usa justify-content flex-end, margens laterais zero e padding horizontal zero. SVG termina na borda interna direita do sidebar expandido/recolhido; visual sem texto/borda/fundo e comportamento/persistência/acessibilidade preservados. Apenas três declarações CSS mudaram.
Build web (inclui TypeScript) e lint raiz PASS, aviso preexistente useOptionalChain da API mantido. Browser reutilizado FAC-028 comprova distância de 1px entre borda externa do menu e SVG, correspondente à borda do sidebar, em ambos estados. Fluxo mouse/teclado, preferências/reload, mobile, storage bloqueado e cinco larguras sem overflow/pageerror PASS. [Resultados](../evidencias/controle/evidencias/FAC-029/browser-results.txt), [harness](../evidencias/controle/evidencias/FAC-029/browser-check.mjs), [expandido](../evidencias/controle/evidencias/FAC-029/expanded.png), [recolhido](../evidencias/controle/evidencias/FAC-029/collapsed.png), [mobile](../evidencias/controle/evidencias/FAC-029/mobile.png), [build](../evidencias/controle/evidencias/FAC-029/build.txt), [lint](../evidencias/controle/evidencias/FAC-029/lint.txt), [patch sanitizado](../evidencias/controle/evidencias/FAC-029/implementation.patch). Nenhum teste unitário novo para ajuste CSS reversível; nenhuma correção adicional.

## Limites, rollback e docs
Sem mudança de backend/contrato/serviço/dado ou dependência. Foco acessível preservado; menu mantém limites atuais de preferência local/mobile. Reverter `ec98b7dc2f2ea700857810fe5df708bae91a720a` volta a centralizar a seta sem afetar dados; rollback não executado. README raiz/controle, INDEX, CHANGELOG, BACKLOG/ticket e lesson de template atualizados. AWAITING_HUMAN; aceite visual exato pendente.

Atualização posterior FAC-030: padrão de abertura/reload passa a recolhido, sem alterar alinhamento à direita; [estado atual](2026-10-06-FAC-030-menu-inicial-recolhido.md). Evidências FAC-029 preservam o comportamento anterior à mudança de inicialização.

## Integração e limpeza realizadas
Fast-forward developer até 3906b77, build e browser 5173 PASS antes do FAC-030. Seta à direita comprovada no destino; mudança posterior de inicialização registrada separadamente. [Resultado developer](../evidencias/controle/evidencias/FAC-029/browser-results-developer.txt).
Worktree limpa e feature ancestral comprovadas; preview próprio PID 3109036 parado após conferir cwd. git worktree remove /home/vinicius/le-fabrique-fac-029 executado sem force; diretório ausente, dependências principais preservadas, somente developer em git worktree list. Branch feature preservada, sem push/deploy. Aceite visual permanece pendente.
