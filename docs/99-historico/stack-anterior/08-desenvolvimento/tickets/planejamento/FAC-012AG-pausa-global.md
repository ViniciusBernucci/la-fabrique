# FAC-012AG — Pausa global persistida da fábrica

Status: IMPLEMENTADO / AWAITING_HUMAN. Data: 2026-10-03. Baseline `41ecfa2` (501 testes internos + 8 PostgreSQL). Branch `feat/fac-012ag-factory-pause`, worktree exclusiva `/home/vinicius/le-fabrique-fac-012ag`, SHA/patch/untracked limpos, nenhum outro writer nela.

Objetivo: implementar controle global de agendamento/kill switch conservador pela interface, independente de projeto/piloto, mantendo parada/evidências/fencing. Defaults pausados; operação real continua exigindo env gate/auth/financeiro próprios.

Escopo: singleton PostgreSQL de agendamento com versão otimista, API administrativa/read-only worker quando pertinente, guard de claim e renew PAUSE, dispatcher sincroniza pause/resume BullMQ sequencialmente e preserva outbox PENDING, worker adia claim recusado por pausa (não FAILED/não gasta), UI operacional, contratos/tests e docs. Critérios: pause bloqueia novos writers mesmo se Redis falhar; req em corrida serialize com claim; lease atual pede PAUSE preservando CANCEL mais forte; resume não libera unknown/retoma run terminal implicitamente; jobs não perdidos no gap de queue pause; defaults/migration ausente fail-closed; HTTP admin guard/version, Redis real efêmero e PostgreSQL real testados sem dados/serviços anteriores. Um writer, até duas rodadas de correção. Sem provider/login/gasto/piloto/deploy/migration de produção. Rollback conserva pausa/índice/evidências até quiescência; aceite humano exato pendente.

Resultado: 881ce3d, 519 testes + 10 PostgreSQL + 2 Redis passaram, duas correções documentadas. [Evidências](../../../09-entregas/2026/controle/2026-10-03-FAC-012AG-pausa-global.md). Aceite pendente.
