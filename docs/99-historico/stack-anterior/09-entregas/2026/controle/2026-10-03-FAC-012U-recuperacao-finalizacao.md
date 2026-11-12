# FAC-012U — Recuperar finalização, não executar novamente

Estado: IMPLEMENTADO / AWAITING_HUMAN. Data: 2026-10-03. Baseline `82764f9` (READY sobre `7c3ca19`). Código `9d4a4f1cdf7f9fa7935928b30abb02e501d27cac`. Writer único na worktree isolada. Sem provider/serviço/banco/fila real, migration aplicada, piloto/login/push/deploy.

## Funcionamento

POST `/api/runs/:id/recover-finalization`, AdminAuthGuard/payload estrito expectedVersion/attemptId. Transação serializável exige run RUNNING/BLOCKED_RECOVERY e tentativa/fence atuais; outbox congela job original/run/attempt/worker/fence no modo FINALIZATION_ONLY. Pedido incrementa apenas versão, não status/fence/stop. Deduplicação por versão/origem; replay idêntico retorna REQUESTED sem mutação, não afirma recuperação.

Dispatcher publica job separado com ID estável sem mudar original FAILED no BullMQ. Consumer identifica modo e chama processFinalizationRecovery; sua porta não recebe checkout/workflow. Exige identidade original do worker; journal parado precisa casar run/attempt/fence/job. Reenvia result/bundle/checkpoint e reconcilia. Sem journal, apenas reconcile; sem prova retorna EVIDENCE_REQUIRED, nunca claim/restore/IA/writer/fence novo.

Reconcile admite BLOCKED_RECOVERY SOMENTE após stop/checkpoint/relatório íntegros e, se snapshot existe, bundle/digest/metadados verificados. Complete direto continua recusando esse estado; estados humanos/terminais não regridem. Unknown sem prova permanece bloqueado mesmo com lease expirada. Finalização recuperada pode levar a VALIDATING, não DONE; aceite Q permanece explícito.

Painel exige confirmação ligada à versão, explica recuperação apenas de entrega e ausência de prova, descarta resposta abortada. Recebimento REQUESTED não é conclusão; status do run observado pelo polling é a evidência de conclusão. Resultado EVIDENCE_REQUIRED fica no retorno sanitizado do job, não em nova tabela administrativa; painel continua mostrando estado pendente/bloqueado. Não há scanner/cleanup/reexecução automática.

## Evidências e diff sanitizado

Checks finais passaram: typecheck, builds completos, 370 testes (launcher 2, contracts 32, runtime 46, API 135, worker 140, web 15), lint 192 arquivos, git diff --check. Primeiro pipeline passou; após adicionar cenário positivo de blocked+bundle exato, suíte completa/lint repetidos. API/Prisma/filas/adapters usam fixtures, sem operação real; UI por renderização estática, não ensaio interativo.

Testes comprovam outbox/dispatch recuperação separado e replay; stale/terminal/foreign/fence/origem recusados; journal após upload falho sem segundo checkout/claim/IA; identidade/journal divergentes recusados; ausência de prova sem mutação; blocked recuperável apenas com stop/artifact exatos; UI desabilitada até confirmação. npm ci sem vulnerabilidades. Diff revisável sanitizado: `git diff 82764f9 9d4a4f1 -- apps packages`. Nenhum segredo/piloto/provider hardcoded.

## Limites, rollback e aceite

Só executor original com journal/snapshot locais pode recuperar; worker desligado/gate false aguarda. Job é consumido em concurrency 1; não interrompe execução ativa. Recovery não comprova hard crash/reboot/morte de processo e não torna unknown retomável; operador precisa procedimento de infraestrutura com prova, nunca botão de liberação cega. Artifact até 64 KiB, identidade/provider/handoff/docs técnicas/transporte maior ainda pendentes.

Rollback: com writer parado, reverter código e preservar journal/snapshot/outbox; eventos novos não podem ser entregues ao consumer anterior como ticket.execute. Nenhuma migration nova ou remoção de dados. Aceite humano da revisão exata pendente; merge local autorizado não significa DONE.
