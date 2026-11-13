# FAC-031 — Posição da seta por estado do menu
2026-10-06. IMPLEMENTADO / AWAITING_HUMAN. Baseline f0271e1; código `3b7e02067884378b7775ebf48af81d61c82187a8`. [Ticket READY](../../../08-desenvolvimento/tickets/controle/tickets/FAC-031.md).
Branch/worktree fix/fac-031-menu-arrow-spacing exclusiva, mesmo writer/Codex da sessão; sem subagentes/handoff/provider adicional, modelo/cota/custo não comprovados. Merge/limpeza autorizados, sem push/deploy.

## Mudança e funcionamento
Pedido: seta centralizada quando recolhido, com mais espaço da borda quando expandido. Toggle expandido mantém flex-end e passa a padding horizontal de 12px. Regra específica recolhida usa center/padding horizontal zero. Continua apenas SVG sem texto/borda/fundo, menu inicia recolhido e estado manual permanece na navegação conforme FAC-030. Backend, dados e dependências preservados.

## Checks e evidências
Build web (inclui TypeScript), lint raiz e git diff --check PASS; aviso preexistente useOptionalChain da API mantido. Nenhum teste unitário novo para CSS reversível; browser FAC-030 reutilizado com medidas adicionais: desvio do centro externo até 0,5px (borda direita de 1px) no recolhido, distância de 13px da borda externa no expandido (12px internos + 1px borda). Inicialização/reload recolhidos, expansão por teclado, navegação, mobile/storage antigo/indisponível e cinco larguras sem overflow/pageerror PASS.
[Resultado](../evidencias/controle/evidencias/FAC-031/browser-results.txt), [harness](../evidencias/controle/evidencias/FAC-031/browser-check.mjs), [recolhido](../evidencias/controle/evidencias/FAC-031/default.png), [expandido](../evidencias/controle/evidencias/FAC-031/expanded.png), [mobile](../evidencias/controle/evidencias/FAC-031/mobile.png), [build](../evidencias/controle/evidencias/FAC-031/build.txt), [lint](../evidencias/controle/evidencias/FAC-031/lint.txt), [patch sanitizado](../evidencias/controle/evidencias/FAC-031/implementation.patch). Sem correção adicional/provider/banco.

## Limites, rollback e docs
Somente alinhamento visual; foco e nomes acessíveis preservados. Reverter `3b7e02067884378b7775ebf48af81d61c82187a8` retorna seta à borda direita em ambos estados sem afetar inicialização/dados; rollback não executado. README raiz/controle, INDEX, CHANGELOG, BACKLOG/ticket e lesson de template atualizados. AWAITING_HUMAN até aceite visual exato.

## Integração e limpeza
Merge fast-forward em developer até `5e52ecc`, autorizado pela sessão. Build web e [browser em developer](../evidencias/controle/evidencias/FAC-031/browser-results-developer.txt) PASS; screenshots recolhido/expandido inspecionados visualmente. Preview exclusivo encerrado após conferir PID/cwd; worktree limpa removida sem force, branch preservada e node_modules raiz intacto. Apenas worktree developer restante; sem push/deploy. Aceite humano da revisão visual permanece pendente.
