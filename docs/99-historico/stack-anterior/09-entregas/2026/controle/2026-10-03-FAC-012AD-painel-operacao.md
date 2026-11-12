# FAC-012AD — Estado operacional da fábrica

Data: 2026-10-03. IMPLEMENTADO / AWAITING_HUMAN. Código `6277a81`; baseline `8a47bbf`, READY `735537a`. Branch `feat/fac-012ad-operation-panel`, worktree exclusiva `/home/vinicius/le-fabrique-fac-012ad`. Sem outro writer. Não DONE.

## Funcionamento

`GET /api/operation` exige AdminAuthGuard e usa transação RepeatableRead para observar workers e tentativas sem parada confirmada. DTO Zod estrito/minimizado, sem credenciais/modelos/comandos. Heartbeat ONLINE até 180 s é RECENT (cobre três intervalos máximos de 60 s); mais antigo STALE, futuro CLOCK_SKEW e registro OFFLINE preservado. Isso não comprova disponibilidade/financeiro/isolamento de IA. Amostra workers até 100 com truncamento explícito, tentativas até 10 com total completo. Lease vencida mantém tentativa sem stop visível e bloqueada.

Consulta verifica índice global AC por nome, tabela/schema, unicidade/validade e expressão/predicado reais. Guard compartilhado também impede novo claim sem índice, evitando operar silenciosamente apenas com consulta vulnerável a corrida. Replay/checkpoint/recovery já existentes continuam disponíveis para preservar evidências; não instala índice nem inventa parada.

Painel Estado da fábrica funciona sem projeto/piloto selecionado. Consulta sequencial a cada 10 s, AbortSignal no cleanup, timeout 8 s. Início de nova consulta/falha limpa estado antigo; observação nova identifica horário. React escapa nomes. Falha de banco vira 503 genérico sem detalhes privados; nenhuma observação saudável é cacheada. API não chama clientes ou executa código de projeto.

## Checks e evidências

- Baseline: npm ci --ignore-scripts, db:generate e build passaram, sem migration em serviços existentes.
- `npm run typecheck`, `npm run build`, `npm test`, `npm run lint`, `git diff --check`: passaram. Build final também verificou tipos após ajuste de timeout/limpeza visual.
- Suíte: 461 passaram (scripts 5, contracts 34, runtime 50, API 159, worker 190, web 23). API +11, web +5 desde AC. HTTP real em porta efêmera localhost confirmou 401 antes da leitura e 200 autenticado sem projeto; banco nessa prova é porta mockada.
- PostgreSQL real efêmero: `npm run test:postgres`, 4 passaram/0 skipped; todas migrations aplicadas só no container isolado. Query do guard real distingue índice AC de índice comum com mesmo nome. Container removido; serviços anteriores intactos.
- Lint 217 arquivos, um warning anterior optional chaining em run-delivery.service.ts, nenhum novo. Uma rodada de formatação/ajustes preventivos; nenhum check funcional falhou.

Logs efêmeros `/tmp/fac-012ad-{tests,postgres-final,typecheck,lint,build}.log`. Diff sanitizado `git show 6277a81 -- apps packages/contracts/src/index.ts scripts/global-writer.postgres.test.mjs`: 14 arquivos, 510 inserções; somente fixtures/identidades sintéticas. Prova reproduzível nos testes versionados.

## Limitações, rollback e aceite

Observação do controle não é inspeção física da árvore do worker; não consulta filesystem/credenciais do serviço. Não afirma cliente elegível nem concede stop/claim pelo painel. Sem ativação/migration de ambiente existente, login, gasto, provider/modelo reais, push/merge/deploy. Piloto não bloqueia software. Próxima lacuna interna identificada: transporte SSE autenticado/retomável previsto no ADR-003, ainda com fallback HTTP existente.

Rollback reverte feature do painel/endpoint; preservar proteção AC e dados/evidências. Relaxar admissão sem índice não é rollback seguro com execução ativa. Aceite humano da revisão exata pendente.

## Docs/lessons

README e docs controle/operação/planejamento/arquitetura/handoff, índices, backlog/changelog/CONTROLE-MVP e lesson heartbeat/quiescência atualizados. Conceito aplicado: liveness do registro não prova prontidão operacional ou parada física; guard consultado precisa corresponder à garantia estrutural real.
