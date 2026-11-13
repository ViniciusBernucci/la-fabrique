# FAC-012S — Comandos administrativos de execução

Estado final: IMPLEMENTADO / AWAITING_HUMAN; não aceito. Baseline `25ab4a4` (READY sobre `4be88df`). Um writer na worktree isolada, providers sintéticos, nenhuma operação em serviço/banco/fila real. Correção documental de cabeçalho pelo FAC-012T; checkpoint histórico abaixo preservado.

## Checkpoint e diagnóstico após duas rodadas

Typecheck inicial passou. Primeiro formatter/lint detectou dependências de efeito React sem uso real; segunda rodada detectou captura de objeto `attempt` contra dependência específica `attempt.id`. Não é falha do baseline. Diagnóstico: registrar pedido em state e executar via effect com IDs/versões primitivas estáveis evita repetição pelo polling; confirmação ligada à versão. Verificação completa ainda pendente neste checkpoint. Nenhum sucesso declarado ou evidência operacional inventada.

## Entrega verificada

Código `be4e2ff75b0ea43ec3042e7c5b681bace152557e`, IMPLEMENTADO / AWAITING_HUMAN. POST `/api/runs/:id/control`, AdminAuthGuard e payload estrito expectedVersion/attemptId/PAUSE|CANCEL. Transação serializável exige run/attempt RUNNING atuais, fence e ausência de stop; apenas grava intenção/incrementa versão. Mesmo pedido idempotente; outro pedido/stale/terminal recusado. Detail expõe pedido, renew fenced comunica ao worker.

LeaseGuard aborta com motivo tipado, distinto de perda da lease; espera operação/stopWriter. Workflow captura snapshot/observações após stop, produz PAUSED/OPERATOR_PAUSED ou CANCELLED/OPERATOR_CANCELLED. Consumer entrega journal → relatório → bundle → checkpoint → complete. Complete limpa intenção; unknown/falha de evidência conserva fence. Corrida com conclusão natural preserva resultado verdadeiro. Reconcile cobre pausa manual inclusive setup sem relatório fictício.

UI exige confirmação por versão, desabilita pedidos duplicados e mostra intenção pendente sem afirmar parada. Effect usa IDs primitivos, não reenvia por polling; respostas abortadas descartadas. Atualização manual permite conferir antes de repetir pedido incerto. Migration 20261003040000_fac_012s_run_control adiciona PAUSED e runs.control_action nullable com constraint; apenas versionada/validada.

## Evidências e diff sanitizado

Checks completos da revisão final passaram: typecheck; 336 testes (launcher 2, contracts 32, runtime 44, API 113, worker 133, web 12); builds; lint 182 arquivos; git diff --check; Prisma validate com URL sintética sem conexão. Sem PostgreSQL/Redis/CLI real ou teste interativo de navegador. Testes comprovam intenção sem liberar writer, replay/stale/fence/terminal, renew, abort/stop/snapshot/journal e unknown sem conclusão; UI estática conservadora.

Após checkpoint, typecheck encontrou acesso undefined possível no fixture UI, corrigido por guarda. Primeira suíte passou testes/build mas lint encontrou formatação de teste adicionado; formatado e checks repetidos no código final. Diff revisável: `git diff 25ab4a4 be4e2ff -- apps packages`. Nenhum segredo/prompt real/piloto adicionado.

## Limites, rollback e aceite

Pedido aguarda próxima renew (lease/3), sem SLA instantâneo. Worker inacessível/unknown conserva pendência/fence, sem kill remoto. Abort pré-contexto, crash abrupto, retomada, identidades/handoff, docs técnicas e artefatos acima de 64 KiB pendentes. Gate real desligado; contas/modelos continuam na interface.

Rollback: reverter código com writer parado, preservar snapshots/journal/dados. Migration não aplicada; enums aditivos não são removidos automaticamente. Aceite humano exato pendente; merge local autorizado não é DONE.
