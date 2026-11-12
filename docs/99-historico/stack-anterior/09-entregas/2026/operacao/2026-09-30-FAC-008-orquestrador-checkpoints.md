# FAC-008 — Orquestrador, leases e checkpoints

Data: 2026-09-30
Estado: AWAITING_HUMAN
Revisao funcional: `ddb8937a6bcdf59ee5a270142710c1df445d76bd`

## Resultado

A API NestJS agora publica `ticket.ready.v1` no BullMQ com identidade estavel, cria runs e attempts transacionais e controla o writer por lease e fencing token monotono. Checkpoint e conclusao exigem a identidade do worker e o token atuais. Um writer cujo encerramento nao foi confirmado bloqueia a recuperacao.

## Funcionamento

- O dispatcher le a outbox, valida o payload Zod, publica `ticket.execute.v1` com `jobId` igual ao ID do evento e so entao marca o evento como publicado.
- A compatibilidade de leitura aceita o campo legado `version`; eventos novos persistem `ticketVersion`.
- O claim confere worker online, ticket, projeto, versao e evento. Uma reentrega devolve o mesmo attempt; outro worker nao toma um lease vigente.
- Cada nova tentativa recebe `fencingToken` maior. Renovacao, checkpoint e conclusao recusam worker estrangeiro ou token antigo.
- O checkpoint persiste revisoes base/code, snapshot, hash do patch, motivo e confirmacao de parada. Conteudo repetido e idempotente; conteudo divergente e recusado.
- A conclusao exige checkpoint com `stoppedConfirmed=true` e move run/ticket para `VALIDATING`, `PAUSED_LIMIT`, `FAILED` ou `CANCELLED`.
- Lease vencido sem confirmacao de parada move run e ticket para `BLOCKED_RECOVERY`, sem criar outro writer.
- O worker ganhou cliente e processor de fixture para o protocolo. O loop principal permanece sem o consumidor de execucao ate o FAC-009.

## Evidencias

A migration `20260930000300_fac_008_orchestration` foi aplicada ao PostgreSQL local com `prisma migrate deploy`. PostgreSQL e Redis responderam pela composicao de desenvolvimento.

No fluxo HTTP real, a promocao do ticket publicou a outbox. Claim e reentrega conservaram o mesmo attempt e fencing token 1. Checkpoint, checkpoint repetido, conclusao, conclusao repetida e nova entrega terminal produziram um unico efeito. A consulta final encontrou 1 run, 1 attempt e 1 checkpoint, com estado `VALIDATING`.

Em um segundo fluxo, o lease foi vencido deliberadamente sem checkpoint de parada. O novo claim respondeu HTTP 409, run e ticket ficaram em `BLOCKED_RECOVERY` e a contagem permaneceu em 1 attempt.

## Checks

- `npm run lint`: 75 arquivos, passou.
- `npm run typecheck`: contratos, runtime, API, worker e web, passou.
- `npm test`: 55 testes em 16 arquivos, passou; API tem 14 e worker tem 5.
- `npm run build`: contratos, runtime, API, worker e web, passou.
- `npm run db:deploy`: quatro migrations reconhecidas; FAC-008 aplicada com sucesso.
- `git diff --check`: passou.
- Diff sanitizado: tokens do ensaio foram sinteticos e nao foram persistidos; nenhum segredo ou payload de provider entrou no Git.

Houve duas correcoes orientadas por evidencia. A primeira tornou o dispatcher compativel com eventos anteriores e conteve payload invalido sem derrubar a API. A segunda tornou idempotentes claim e complete depois do termino e passou a conferir a relacao ticket/projeto.

## Limites

O processor de fixture comprova o contrato, mas ainda nao esta conectado ao loop BullMQ principal. O FAC-009 deve compor contexto, guard, sandbox, adapter, checks e revisao antes de habilitar execucao. Pausa e cancelamento possuem estados de conclusao; retomada administrativa e retry deliberado continuam futuros.

O dispatcher usa polling por instancia. O `jobId` do BullMQ e as restricoes unicas do PostgreSQL impedem efeitos duplicados, mas o ensaio de reinicio concorrente da composicao completa pertence a validacao operacional. Nao ha painel de runs, SSE, merge, deploy ou piloto real nesta entrega.

Nenhum provider, API, credito ou extra usage foi utilizado.

## Correcao posterior

Em 01/10/2026, a revisao identificou uma janela em que a reentrega depois do checkpoint parado e antes da conclusao criava outro attempt. A revisao funcional `5a766c4c928dda27c3b1417afbc4ec3e1de52d0d` passa a reutilizar o attempt existente, conecta o probe sintetico ao consumidor BullMQ principal e recusa claim quando a revisao-base nao foi resolvida. Evidencias e rollback adicionais estao em `2026-10-01-FAC-008-correcao-idempotencia-consumidor.md`.

## Rollback

Parar API e worker que usem o protocolo e reverter o commit funcional `ddb8937a6bcdf59ee5a270142710c1df445d76bd`. Para preservar dados, manter as tabelas sem uso. Se os dados FAC-008 forem comprovadamente descartaveis, uma migration revisada pode remover, nesta ordem, `checkpoints`, `attempts`, `runs`, `AttemptStatus` e `RunStatus`. Nao remover volumes nem dados existentes como rollback padrao.
