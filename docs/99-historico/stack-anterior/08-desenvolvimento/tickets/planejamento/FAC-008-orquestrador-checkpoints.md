# FAC-008 — Orquestrador, leases e checkpoints

Status: DONE

Revisao aceita: `29061a911e0f6bc5122e9e53511f2f475caf8dce`, em 2026-10-01. O commit documental posterior apenas registra o aceite da revisao apresentada.

## Objetivo

Transformar eventos `ticket.ready.v1` em runs recuperaveis, com dispatcher idempotente, claim autenticado, um writer por run, lease, fencing monotono e checkpoint antes de liberar nova tentativa.

## Escopo

- Contratos Zod de job, claim, renovacao, checkpoint, conclusao e estado de run.
- Migration aditiva para runs, attempts e checkpoints no PostgreSQL.
- Dispatcher NestJS da outbox para BullMQ com `jobId` estavel e marcacao posterior de publicacao.
- Protocolo interno autenticado para claim, renovacao de lease, checkpoint e conclusao.
- Consumidor worker que valida o job e exercita o protocolo de orquestracao sem chamar provider.
- Testes unitarios e integracao local com PostgreSQL/Redis.

Ficam fora: execucao do Codex, aplicacao automatica de patch, reviewer/checks FAC-009, segundo provider, SSE/painel de runs, merge, deploy e piloto real.

## Criterios de aceite

1. Evento outbox usa ID estavel no BullMQ; republicacao do mesmo evento nao cria segundo efeito.
2. Claim cria no maximo um run por evento/ticket e um attempt ativo, com fencing token monotono.
3. Lease vigente recusa segundo writer; lease expirado sem `stoppedConfirmed` move run/ticket para `BLOCKED_RECOVERY` e recusa novo claim.
4. Renovacao, checkpoint e conclusao exigem attempt atual, worker dono e fencing token exato; token antigo nao altera estado.
5. Checkpoint persiste base/code SHA, snapshot/patch hash e confirmacao de parada; nova tentativa so pode nascer apos checkpoint com quiescencia comprovada.
6. Reentrega do job e repeticao de requests retornam o mesmo estado sem duplicar attempt/checkpoint.
7. Worker executa somente probe de orquestracao sintetico; provider, API, credito e extra usage permanecem desligados.

## Baseline e limites

- Base: `developer` em `2a4137549682e5f1f14427baa9a9ab81a2313e97`.
- Branch/worktree: `feat/fac-008-orchestrator-checkpoints` em `/home/vinicius/le-fabrique-fac-000-accepted`.
- PostgreSQL e Redis locais da composicao de desenvolvimento; migration aditiva e reversao documentada.
- Um worker/um writer; lease inicial 90 segundos, heartbeat operacional 15 segundos e fencing inteiro crescente.
- Provider: nenhum nesta entrega.

## Rollback

Parar dispatcher/worker novos, reverter os commits e, somente se dados FAC-008 forem descartaveis, remover tabelas/checkpoints e colunas pela migration reversa revisada. Nao apagar volumes nem dados existentes como rollback padrao.
