# FAC-012AH — Jobs preservados na admissão e conclusão interna do MVP

Data: 2026-10-03. IMPLEMENTADO / AWAITING_HUMAN. Código `f2d80f1671ac8b0c76a5c76f35be4ce626fcab6d`; baseline `2eced85`, READY `965ad17`. Branch `fix/fac-012ah-admission-capacity`, worktree exclusiva `/home/vinicius/le-fabrique-fac-012ah`; um writer nela. Z–AG por ancestry. Nenhum piloto necessário; nenhuma implantação/aceite inferidos.

## Problema e comportamento

Com um writer global sem stop, o próximo ticket recebia 409 antes de adquirir autoridade e o job podia ficar FAILED sem trabalho iniciado. Claim agora retorna HTTP 429/code WRITER_BUSY nessa ocupação, P2002 no insert da tentativa e P2034 conhecido da transação Serializable. O erro cancela a transação: não cria tentativa/fence nem muda ticket para RUNNING. Índice global, guard de instalação, pausa, stop e fencing continuam obrigatórios. Expiração/status FAILED/CANCELLED do writer anterior não liberam capacidade.

P2034 é write conflict/deadlock no [código primário Prisma Engines 6.12.0](https://raw.githubusercontent.com/prisma/prisma-engines/6.12.0/libs/user-facing-errors/src/query_engine/mod.rs), correspondente à versão instalada. Só esse rollback conhecido é convertido; erro de transporte/resposta de commit ambígua propaga. Teste P2034 é sintético, não afirma corrida real do Prisma. PostgreSQL real comprova exclusão e locks nos testes existentes.

ControlClient converte 429 apenas na chamada claim para WriterAdmissionDeferredError; FactorySchedulingPausedError (423) deriva dessa classe. deferWriterAdmission moveToDelayed +2 s/token e lança DelayedError, preservando job/intent sem declarar sucesso/falha ou consumir tentativa. Alias antigo preservado. Renew/report/complete não convertem 429; 409 stale/foreign/stop desconhecido da própria run, falhas de setup/execução/evidência e deferral Redis falho propagam. Finalization-only/journal não ganham novo claim. Após capacidade livre, o mesmo intent passa pelo claim e preparação normais, que leem configuração atual; não há reexecução de trabalho admitido por essa recusa.

`npm test` passou a compilar contracts/runtime em pretest. Checkout novo antes falhava nos testes CLI por MODULE_NOT_FOUND; não se removeu asserção nem se mascarou exit code.

## Evidências da revisão exata

- npm ci --ignore-scripts, db:generate, typecheck (inclui scripts/integração), lint, build e git diff --check passaram. Lint tem somente warning optional-chain preexistente em run-delivery.service.ts:209.
- Suíte completa: **528 passou/0 falhou** — scripts 6, contracts 45, runtime 65, API 177, worker 202, web 33. Logs /tmp/fac-012ah-test-complete.log e /tmp/fac-012ah-{typecheck,lint-final,build}.log.
- API testa quatro estados de outro writer, falta de índice, unique race, Serializable rollback e falha ambígua. Processor real com portas externas sintéticas: capacidade recusada não chama checkout/profile/workflow/renew/result/checkpoint/complete; liberação executa mesmo intent uma vez. Contratos Zod clonam entrada; teste compara conteúdo exato, não identidade JS.
- npm run test:postgres: **10 passou/0 skipped**, todas migrations reais em container próprio network none/tmpfs/512 MiB/1 CPU; exclusão entre runs, stop, índice, cursor, restore e pausa/lock permanecem comprovados. /tmp/fac-012ah-postgres.log.
- npm run test:redis: **3 passou/0 skipped**, Redis 8.4.0 e BullMQ 6.3.10 reais, container exclusivo RAM128 MiB/0,5 CPU/porta aleatória loopback/sem volume. Queue pause mantém intents; ambos erros tipados levam active→delayed com attemptsMade=0/failed=0, depois completed com exatamente uma execução sintética. /tmp/fac-012ah-redis.log.
- Duas rodadas de correção: bootstrap da suíte nova sem dist (pretest); teste novo comparava identidade referencial após parse Zod (corrigido para conteúdo). Falhas iniciais registradas em /tmp/fac-012ah-test.log e /tmp/fac-012ah-worker-final.log. Nenhuma regressão anterior usada para ocultar failure.
- Diff sanitizado recuperável: `git show f2d80f1671ac8b0c76a5c76f35be4ce626fcab6d -- apps package.json scripts` (10 arquivos, 323 inserções/173 remoções; indentação de promise encadeada explica boa parte do diff API). Dados/URLs/IDs sintéticos; sem auth/token no patch. Containers efêmeros removidos; serviços existentes preservados.

## Estado do MVP, limitações e rollback

As lacunas internas auditadas foram implementadas: documentação técnica/revisão, perfis oficiais configurados, integração real do worker com portas externas sintéticas, writer global, painel de operação, SSE, backup/restauração, pausa e espera por capacidade. O software está IMPLEMENTADO/VERIFICADO no escopo atual de administrador único/um writer, disponível nesta branch. Aceite humano exato permanece pendente; não marcar DONE automaticamente.

Não houve execução com conta IA real, prova Claude nativa real, serviço dedicado instalado, migration em banco existente, push/merge/deploy, compra/gasto, nem restauração externa/reboot. Elegibilidade financeira/auth sob UID do serviço, integração/implantação e ensaio operacional continuam etapas próprias. Essas etapas e o experimento externo opcional não são dependências do código do MVP. Writer sem parada pode manter outros jobs aguardando indefinidamente, deliberadamente; não há timeout que fabrique stop. Deferral de 2 s não é SLA de execução e PostgreSQL indisponível não vira sucesso.

Rollback só com writers parados e queue pausada: reverter código deste incremento não remove índice/factory_operations/eventos/evidências nem confirma stop. Reversão do worker pode fazer recusa pré-claim voltar a FAILED; preservar outbox/jobs para diagnóstico. Não reverter schema/dados automaticamente. Root developer permaneceu limpo em `745ce2d`; outros processos no workspace raiz impedem comprovar writer parado para integração local segura.

## Documentação e lessons

README atual de controle/operação/planejamento, índice, backlog/changelog, CONTROLE-MVP, plano, handoff, ADR-003 e lesson leases/fencing atualizados. Conceito aplicado: retry de admissão anterior à autoridade é distinto de repetir execução; erro tipado só nessa fase preserva fila sem enfraquecer exclusão ou evidência. Aceite da revisão exata pendente.
