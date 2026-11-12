# FAC-012AG — Pausa global persistida da fábrica

Data: 2026-10-03. IMPLEMENTADO / AWAITING_HUMAN. Código `881ce3d`; baseline `41ecfa2`, READY `d35f62c`. Branch `feat/fac-012ag-factory-pause`, worktree exclusiva `/home/vinicius/le-fabrique-fac-012ag`. Um writer, nenhum piloto/provider exigido; não DONE automático.

## Funcionamento

Singleton PostgreSQL factory_operations inicia paused=true/version=1. PUT /api/operation/scheduling administrativo exige boolean paused e expectedVersion; atualização Serializable com compare-and-increment, sem upsert/mudar planos/login/gates financeiros. Estado entra no DTO GET /api/operation. Ausência de linha/tabela não habilita execução.

Claim novo lê linha FOR SHARE mantida até commit: pausa administrativa precisa esperar claim já admitido; claim posterior observa pausa ou falha conservadoramente na serialização. Paused retorna 423 antes de criar writer; índice AC continua obrigatório. Replay/journal/recovery de execução já conhecida não concede writer novo. Renew fenced retorna PAUSE enquanto flag global pausada, preservando CANCEL/PAUSE já pedido por run. Pedido não afirma parada: worker usa LeaseGuard/stop/snapshot/journal/checkpoint/protocolo existentes; só stop real libera exclusão.

Dispatcher sequencial sincroniza pause/resume da queue BullMQ conforme DB. Pausado/uninitialized não publica nem consome budget/edita outbox PENDING. Erro de sincronização preserva intents, PostgreSQL ainda barra admissão. Queue pausada conserva jobs; também adia pedidos de recovery enfileirados até liberação de agendamento, mas finalização direta de tentativas ativas/paradas permanece disponível. Retomar não dá resume automático de runs terminais, nem ultrapassa UNKNOWN/índice/auth/gate env.

Gap de pausa: job já ativo recebe 423 no claim; ControlClient produz FactorySchedulingPausedError somente nessa chamada. Helper move job a delayed +2 s usando token e lança DelayedError reconhecido pela versão instalada BullMQ 6.3.10, sem marcar COMPLETED/FAILED/consumir tentativa. Falha no defer propaga; outros erros de execução/lease/unknown não viram retry cego. Main usa esse helper só ao redor do processor, sem fixtures na fila real.

Painel sem projeto mostra pausa e botão Pausar fábrica/Retomar agendamento; resume indisponível sem índice válido. Mudança usa versão atual, limpa observação antiga e consulta novamente; epoch/versão descartam resposta antiga/de outra sessão e cleanup evita atualizar view abandonada. Parada ativa permanece mostrada por contagem de tentativas sem stop. Novos installs precisam de liberação administrativa explícita; WORKER_EXECUTION_ENABLED continua default false e não é modificado pela UI.

## Verificação e diff

- ci --ignore-scripts, db:generate, npm run typecheck, npm test, npm run lint, npm run build, git diff --check passaram. Sem migration/serviço existente alterado.
- Suíte 519 passou: scripts 6, contracts 45, runtime 65, API 175, worker 195, web 33. Testes novos cobrem guard 423/defaults/versão/auth HTTP, pause/renew CANCEL, dispatcher falta de estado/Redis/concorrência, defer sem writer e view conservadora.
- npm run test:postgres: 10 passou/0 skipped, todas migrations reais no container exclusivo network none/tmpfs/512 MiB/1 CPU. Default pause true; versão antiga não clobbera; pg_stat_activity mostrou pausa bloqueada por lock real de claim em curso, ambas concluíram e contagem de writer permaneceu 1. Provas de índice/cursor/backup continuam passando.
- npm run test:redis: 2 passou/0 skipped. Redis 8.4.0/BullMQ 6.3.10 reais, container exclusivo RAM128 MiB/0,5 CPU e porta aleatória só loopback, sem volume persistente. Queue pausada reteve job; unpause despachou 2 intents sem duplicar. Gap moveu active→delayed com attemptsMade=0/failed=0, depois completou exatamente 1 execução sintética ao liberar. Builds API/worker anteriores ao teste passaram. Estado de scheduling no teste Redis é mock; prova de persistência/lock é PostgreSQL independente.
- Lint 234 files, warning optional-chain preexistente único. Duas rodadas de correção: mocks antigos de guard/dispatcher com nova porta; observador SQL buscava marker de BEGIN em vez de query pg_sleep atual (harness), corrigido para observar lock real. Ajuste de import/config da nova fixture ControlClient antes dos checks finais. Nenhum failure ocultado como baseline.

Logs efêmeros /tmp/fac-012ag-{tests,typecheck-final,lint,build,postgres-final,redis}.log. Diff sanitizado: git show 881ce3d -- apps package.json packages/contracts/src/index.ts scripts (27 files, 975 inserções/92 remoções). Dados/contas/chaves sintéticos; containers removidos e serviços antigos preservados.

## Limites, rollback e aceite

Pausa é pedido assíncrono recebido na renovação (até lease/3), não prova física de stop nem desligamento do host. Falha Redis pode deixar queue momentaneamente fora de sync, mas DB não concede writer pausado; ciclo seguinte reconcilia. Queue compartilhada pausada aguarda também finalization-only enfileirado. Não há autorização de cliente/provisionamento pela flag. Sem login/preflight real/gasto/API de IA/piloto/push/merge/deploy.

Rollback exige writer parado, mantendo factory_operations pausada, índice e evidências; reverter UI não deve reativar queue por fora. Não remover tabela/seed/rows automaticamente. Aceite humano exato pendente. Última revisão de fluxo identificou followup: capacidade global ocupada em outro ticket ainda retorna 409 antes do claim e pode transformar o job não iniciado em FAILED; implementar deferral tipado só nessa recusa conhecida de admissão, sem repetir execução já iniciada.

## Docs/lessons

Controle/operação/infraestrutura/planejamento, README/índices/backlog/changelog/CONTROLE/handoff/ADR-003 e lesson leases/fencing atualizados. Conceito aplicado: estado PostgreSQL pausa a autoridade, pause Redis controla entrega e ambos não substituem stop físico; DelayedError preserva somente recusa pré-admissão conhecida.
