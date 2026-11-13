# FAC-012U — Recuperação administrativa apenas de finalização

Status: AWAITING_HUMAN. Código `9d4a4f1cdf7f9fa7935928b30abb02e501d27cac`; 370 testes/checks passaram. Data: 2026-10-03. Baseline `7c3ca19`, 355 testes/checks. Branch/worktree `feat/fac-012u-finalization-recovery`, `/home/vinicius/le-fabrique-fac-012u`.

Objetivo: solicitar no painel recuperação de entrega pendente/failed job por outbox separada. Worker só lê journal parado/reenvia evidências ou reconcilia checkpoint existente; nunca claim, checkout, restore, Developer ou novo fence. Intenção autenticada/otimista/idempotente ligada ao run/attempt/worker/fence/job original. Unknown sem prova continua bloqueado. Reconcile pode concluir BLOCKED_RECOVERY apenas com stop/evidência persistidos/coerentes.

Caminhos: contracts, API orchestration/outbox, worker processor/consumer e UI/testes; docs controle/operação/planejamento/lessons. Sem serviços/bancos/filas/provider real, migrations aplicadas, piloto/login/push/deploy. Um writer isolado; fixtures; até duas correções antes de checkpoint. Provider nenhum, configuração UI/financeira inalterada.

Critérios: outbox atômica e versão/idempotência, terminal/foreign/stale recusados; consumer recuperação separado sem porta de execução/claim, worker atual/fence/origem/journal coerentes; sem journal/prova retorna não recuperado; dados íntegros recuperam sem repetir IA, unknown não libera writer; botão explica limites, confirmação explícita; checks/docs exatas, AWAITING_HUMAN. Rollback código com worker parado, preservar outbox/journal/evidências.
