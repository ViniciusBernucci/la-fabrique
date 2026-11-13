> Leitura: [← Anterior](08-lifecycle-processo-cli.md) · [Índice didático](../02-INDEX.md) · [Próximo →](10-worker-heartbeat-quiescencia.md)

# Lease, fencing e checkpoint cobrem falhas diferentes

FAC-012U torna retry de entrega diferente de retry de execução: outbox FINALIZATION_ONLY aponta job original; processFinalizationRecovery não recebe portas de checkout/workflow e nunca chama claim. Reconcile pode concluir blocked somente com prova íntegra; testes de journal ausente/worker divergente/stop unknown mostram que pedido/lease não provam parada.

FAC-012S separa intenção administrativa de parada física: RunControlService apenas grava pedido; LeaseGuard aborta/espera stop, workflow captura depois, consumer entrega antes de concluir. run-control.spec.ts prova que pedido não altera status/fence/stop; execution.processor.spec.ts prova que unknown não libera tentativa. PAUSED manual não é PAUSED_LIMIT e confirmação UI é vinculada à versão.

FAC-012O aplica write-ahead da finalização: `ResultJournal.save` publica intenção parada antes do HTTP. Depois de falha de upload, uma instância nova lê relatório/hash do job e usa mesmo fence sem claim/IA. fsync/hardlink exclusivo impede publicar arquivo parcial/sobrescrever evidência. Journal só existe após retorno/parada, nunca substitui prova de quiescência de execução interrompida. Teste `execution.processor.spec.ts` demonstra reinício sintético.

FAC-012N aplica reconciliação, não retry: replay chama `OrchestrationService.reconcile`, deriva outcome do checkpoint imutável e exige stop persistido, relatório/digest/snapshot compatíveis. `completeTransaction` recusa outro outcome terminal e estados humanos; a transação serializável preserva leitura e escrita juntas. Nenhum cliente de IA é reexecutado.

No FAC-012M, `ExecutionResultsService.report` só grava result/resultDigest, autenticando worker e fencing atual. Mesmo relatório retorna receipt; digest divergente falha. Isso preserva evidência, não quiescência: `stoppedConfirmed` e lease não são atualizados. `execution.processor` envia relatório antes de checkpoint/complete; indisponibilidade deixa reconciliação pendente em vez de declarar sucesso sem evidência.

Lease limita por quanto tempo o controle considera um writer vigente. Fencing token identifica a tentativa atual e faz a API recusar escrita atrasada. Nenhum dos dois prova que um processo local morreu: no FAC-008, lease vencido sem `stoppedConfirmed` leva run e ticket a `BLOCKED_RECOVERY`, sem entregar o workspace a outro writer.

Idempotencia precisa cobrir o ciclo inteiro. O dispatcher usa o ID da outbox como `jobId`; run e attempt possuem restricoes unicas; claim, checkpoint e complete devolvem o estado existente quando o mesmo request reaparece. O ensaio HTTP repetiu cada etapa e terminou com uma unica linha em cada tabela.

Reentrega e retry deliberado precisam ser comandos distintos. No FAC-008, um checkpoint com parada confirmada muda o attempt para `STOPPED` antes de `complete`; uma reentrega nessa janela deve devolver esse mesmo attempt. Criar outro fencing token apenas porque o writer parou transforma redelivery em retry e impede a conclusao pelo token anterior. O retry administrativo fica para um contrato explicito posterior.

Checkpoint liga recuperacao logica ao artefato do FAC-007. Ele registra base/code SHA, snapshot ou hash do patch e confirmacao de parada. A conclusao so avanca depois desse registro, evitando que uma mensagem de sucesso sem artefato recuperavel libere a etapa seguinte.

Identificadores de revisao nao podem ser inventados pelo consumidor. O evento propaga `baseRevision` apenas quando o projeto ja possui SHA Git completo; sem ele, o probe falha antes de adquirir claim e preserva o ticket para reconciliacao.

FAC-012AC demonstra que exclusão por run não implementa um writer global: claim consulta todas as tentativas sem stoppedConfirmed e a migration attempts_single_unconfirmed_writer usa índice parcial sobre constante. Mesmo se duas transações observarem ausência, só um insert vence no PostgreSQL. `scripts/global-writer.postgres.test.mjs` comprova corrida, lease expirada/status terminal e recusa de reabrir writer anterior; migrar dados inconsistentes recusa em vez de fabricar stop. [Evidências](../09-entregas/2026/controle/2026-10-03-FAC-012AC-writer-global.md).

FAC-012AG separa pedido global, entrega e stop: factory_operations FOR SHARE ordena claim com pausa; renew entrega PAUSE sem declarar quiescência; queue.pause conserva jobs. Gap de active antes do claim recebe 423 e moveToDelayed + DelayedError sem attemptsMade. Testes PostgreSQL observam lock e Redis/BullMQ reais confirmam estados; erros de autoridade/trabalho iniciado nunca entram nesse helper de retry. [Evidências](../09-entregas/2026/controle/2026-10-03-FAC-012AG-pausa-global.md).

FAC-012AH distingue retry antes de autoridade de repetir execução: claim 429 e erro WriterAdmissionDeferredError só nessa chamada levam job a delayed; renew 429 e falha ambígua não entram. execution.processor.spec.ts comprova zero checkout/IA/checkpoint na recusa e mesmo intent executado uma vez após liberar; Redis real comprova attemptsMade=0 enquanto espera. P2034 conhecido cancela admissão, transporte ambíguo propaga. [Evidências](../09-entregas/2026/controle/2026-10-03-FAC-012AH-admissao-capacidade.md).
