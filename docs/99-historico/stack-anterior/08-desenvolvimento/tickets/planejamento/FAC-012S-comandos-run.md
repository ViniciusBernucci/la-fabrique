# FAC-012S — Pausa e cancelamento administrativos

Status: AWAITING_HUMAN. Código `be4e2ff75b0ea43ec3042e7c5b681bace152557e`; 336 testes/checks passaram. Data: 2026-10-03. Baseline: `4be88df`; 315 testes passaram no FAC-012R. Branch/worktree: `feat/fac-012s-run-control`, `/home/vinicius/le-fabrique-fac-012s`.

## Objetivo, caminhos e limites

Solicitar PAUSE/CANCEL pelo painel com versão otimista. A API apenas registra intenção; worker recebe na renovação de lease, aborta, comprova parada, captura snapshot e publica resultado antes de concluir. PAUSED manual difere de PAUSED_LIMIT. Nenhum writer novo, retomada, banco/fila/serviço real, provider, login, piloto, push ou deploy neste incremento. Provider: nenhum, fixtures sintéticas. Um writer isolado; duas rodadas de correção antes de diagnóstico.

Caminhos: contracts, API orchestration/schema/migration versionada, worker lease/workflow/processor/journal, painel e testes; docs controle/operação/infraestrutura e lessons.

## Critérios

1. Endpoint administrativo estrito/autenticado, run/attempt RUNNING atuais, expectedVersion, mesma intenção idempotente; intenção divergente/stale/terminal recusada sem liberar fence.
2. Renovação comunica intenção atual; abort não se confunde com perda de lease. Resultado exige stop/snapshot; unknown permanece bloqueado.
3. PAUSED/CANCELLED preservam report/bundle/journal e reconcile coerentes. Corrida com conclusão natural mantém evidência verdadeira, sem inventar cancelamento.
4. Painel distingue pedido pendente de parada confirmada; confirmação explícita, nenhuma retomada ou aprovação automática.
5. Testes e checks completos; relatório, docs atuais, índice/backlog/changelog/lesson e SHA exato; AWAITING_HUMAN.

## Rollback

Reverter código com writer parado, preservando evidências. Migration aditiva não aplicada; valores enum não são removidos automaticamente. Retomada/recovery seguem próximo incremento.
