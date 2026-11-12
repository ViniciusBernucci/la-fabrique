# FAC-028 — Seta minimalista no menu
2026-10-06. IMPLEMENTADO / AWAITING_HUMAN. Baseline 73c23a4; código `efbcb86631da70e6afe40ce33265a97a69f62847`.
[Ticket READY](../../../08-desenvolvimento/tickets/controle/tickets/FAC-028.md). Branch fix/fac-028-minimal-menu-toggle/worktree isolada, mesmo writer/Codex da sessão, sem subagentes/handoff/cliente adicional; modelo/cota/custo não comprovados. Merge/limpeza previamente autorizados; sem push/deploy.

## Mudança e funcionamento
Responsável pediu somente seta, sem texto/contorno. Removido span visível do toggle; SVG de 16px, border zero e background transparente, inclusive hover. Seta aponta para recolher/expandir conforme estado. Área clicável, aria-label/title, aria-expanded, teclado e foco visível apenas na navegação por teclado preservados. Persistência/menu/template/backend permanecem com comportamento anterior, sem dependência nova.

## Checks/evidências
Typecheck web, build web, lint raiz e git diff --check PASS; aviso useOptionalChain preexistente da API mantido. Nenhum teste unitário novo por alteração visual reversível; reutilizado browser FAC-027 com checagem adicional de texto vazio, SVG, borderTopWidth 0px e fundo transparente em repouso/hover. Recolher/expandir por teclado, widths/margens, persistência navegação/reload, mobile/storage bloqueado, cinco larguras sem overflow e sem pageerror PASS. [Resultado](../evidencias/controle/evidencias/FAC-028/browser-results.txt), [harness](../evidencias/controle/evidencias/FAC-028/browser-check.mjs), [expandido](../evidencias/controle/evidencias/FAC-028/expanded.png), [recolhido](../evidencias/controle/evidencias/FAC-028/collapsed.png), [mobile](../evidencias/controle/evidencias/FAC-028/mobile.png), [typecheck](../evidencias/controle/evidencias/FAC-028/typecheck.txt), [build](../evidencias/controle/evidencias/FAC-028/build.txt), [lint](../evidencias/controle/evidencias/FAC-028/lint.txt), [patch sanitizado](../evidencias/controle/evidencias/FAC-028/implementation.patch). Captura inspecionada. Sem correção adicional, provider ou banco.

## Limites, rollback e docs
Nome aparece só como tooltip/descrição acessível; foco de teclado mantém indicação para acessibilidade, sem moldura padrão/hover. Reverter `efbcb86631da70e6afe40ce33265a97a69f62847` restaura texto e borda, sem dados/migration. Rollback não executado. README raiz/controle, INDEX, CHANGELOG, BACKLOG/ticket e lesson de template atualizados. AWAITING_HUMAN: ajuste feito conforme pedido, aceite visual exato ainda pendente.

## Integração e limpeza realizadas
Fast-forward local developer até 725852f sem conflitos. Build web no destino PASS; browser 5173 confirma seta única/sem moldura/fundo e os fluxos de recolhimento/persistência/mobile/storage. [Resultado developer](../evidencias/controle/evidencias/FAC-028/browser-results-developer.txt).
Worktree limpa e ancestralidade comprovadas; preview próprio PID 3104223 parado após conferir cwd. git worktree remove /home/vinicius/le-fabrique-fac-028 realizado sem force, diretório ausente e dependências principais preservadas; git worktree list contém somente developer, branch feature preservada. Sem push/deploy; aceite visual pendente.
