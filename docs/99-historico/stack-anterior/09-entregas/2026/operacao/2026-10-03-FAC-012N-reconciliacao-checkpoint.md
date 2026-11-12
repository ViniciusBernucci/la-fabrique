# FAC-012N — Reconciliação após checkpoint

Data: 2026-10-03. Status: AWAITING_HUMAN. Baseline `f3b9b058c58ffbee09950f27a9147fd93e583521`; ticket READY `c3496fa`; código `94b5d641030072ef1f06e2b815eec0bec7f7880d`.

## Objetivo e funcionamento

Fecha a janela de interrupção entre checkpoint parado e complete. Endpoint interno POST `/api/internal/orchestration/attempts/:attemptId/reconcile` exige WorkerAuthGuard e schema estrito com workerId/fencingToken; não aceita outcome nem confirmação de parada do caller. Resposta `{state: null}` indica evidência insuficiente e não faz mutações.

API confirma owner/fence atuais, attempt parado e checkpoint persistido parado/não PROGRESS. Outcome deriva do checkpoint. COMPLETED/PAUSED exigem relatório validado, digest, identidade e status compatíveis; snapshot/base/code/hash precisam coincidir quando presentes. Não gera DONE. Falha/setup sem relatório só recupera FAILED/CANCELLED. Transação serializável reutiliza a rotina de conclusão; repetição idêntica não incrementa versões e outro outcome/estado humano não pode ser reescrito.

Consumer chama reconcile no replay, sem checkout/workflow/IA. Erro de rede ou contrato propaga. Não trata redelivery como retry de implementação e não cria fencing token/writer novo.

## Checks e diff sanitizado

npm ci/db:generate locais sem banco; typecheck, todos os testes, build, lint e git diff --check passaram. 257 testes: launcher 2, contracts 24, runtime 44, API 76, worker 103, web 8. Novos testes exercitam quiescência ausente, digest/snapshot/owner/fence divergentes, resultado ausente, conclusão idempotente, regressão de estados humanos, guard/payload e replay sem IA/checkout. Lint: 163 arquivos. Sem rodadas de correção funcional.

Diff `git diff c3496fa 94b5d641030072ef1f06e2b815eec0bec7f7880d`: 7 arquivos, 341 inserções/37 remoções; contracts, controller/service/test API e control-client/processor/test worker. Sem auth/credenciais, migration ou serviços alterados. Docs/lessons em commit separado.

## Limitações, rollback e aceite

Não recupera relatório ainda não persistido, crash com writer desconhecido, snapshots de interrupção nem execução de IA. Mantém esses casos pendentes/bloqueados; journal é próximo incremento. Fixtures não provam DB/HTTP/fila reais ou serviço/CLI autenticado. Gate false, sem piloto/provider/login/push/deploy. Serviços existentes ficaram intocados.

Rollback por revert do código com writer parado; nenhum schema de banco mudou. Integração local autorizada pelo responsável após checks/árvore limpa, branch preservada na limpeza. Lessons de leases/fencing atualizada para distinguir reconciliação de retry. Aceite humano do SHA exato pendente; MVP ainda incompleto.
