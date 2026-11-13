# FAC-012AC — Exclusão global do writer

Status: IMPLEMENTADO / AWAITING_HUMAN. Data: 2026-10-03. Baseline `a54876f` (438 testes, typecheck/lint/build concluídos no AB). Branch `fix/fac-012ac-global-writer`; worktree exclusiva `/home/vinicius/le-fabrique-fac-012ac`, nenhum outro writer nessa worktree, SHA/patch/untracked conferidos.

Objetivo: garantir um writer global independentemente de run/projeto/worker, incluindo lease vencida sem prova de parada. Lacuna observada: claim atual consulta somente a última tentativa da mesma run, BullMQ concurrency=1 não é exclusão persistida global.

Escopo: orchestration.service/specs, migration PostgreSQL aditiva de índice único parcial sobre tentativa sem stoppedConfirmed, teste isolado de integração PostgreSQL, scripts de verificação e documentação controle/infra/operação/planejamento/ADRs/lessons. Corrigir pontos de entrada atuais que tratam piloto externo como bloqueio da conclusão do software. Preservar histórico de ensaios externos como etapa opcional separada.

Critérios: segundo ticket não obtém novo writer enquanto outra tentativa não confirma parada; expiração/status/worker diferente não liberam; replay e recovery existentes preservados; prova de stop permite próxima tentativa; PostgreSQL real recusa insert simultâneo e atualização que reabre writer. Migração recusa inconsistência prévia, não altera dados. Testes com banco efêmero exclusivo, sem tocar serviço/volume existente. Um writer, até duas rodadas de correção, sem modelo/cliente oficial/API/gasto/login/piloto/produção/merge/deploy. Checks baseline e revisão final; documentação completa e aceite humano exato, não DONE automático. Rollback preserva índice enquanto execução ativa e evidências; remoção só após quiescência e ticket autorizado.

Resultado: código `143f3d6`, 445 testes + 3 PostgreSQL passaram, typecheck/lint/build concluídos. Aceite pendente. [Evidências](../../../09-entregas/2026/controle/2026-10-03-FAC-012AC-writer-global.md).
