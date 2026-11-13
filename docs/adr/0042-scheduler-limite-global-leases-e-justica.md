# ADR-0042 — Scheduler: limite global, leases com geração e justiça

- **Estado:** ACEITA · **Ticket:** [T10-03](../NEW%20MVP/03-BACKLOG.md#t10-03) · **Épico:** E10 · **Relacionadas:** [ADR-0039](0039-fila-local-por-projeto.md), [ADR-0023](0023-checkpoints-integros.md), [ADR-0016](0016-supervisao-de-processos.md)

## Decisão
1. **Reclamar vaga é uma transação serializada** (`Queue.Claim`, bloqueio consultivo global): devolve à fila o que está com lease vencida, confere o **limite global** (itens em `running`), escolhe o candidato e inicia o item com **lease** e **geração nova** — tudo ou nada. Uma tarefa por projeto continua garantida (índice único + elegibilidade da fila); o total em execução nunca passa do limite.
2. **Limite e envelhecimento persistentes** (`scheduler_settings`, 1 a 64 execuções; 1 minuto a 7 dias): `SetSettings` vale para as **próximas** reclamações. **Nada é preemptado:** baixar o limite abaixo do que já roda só bloqueia novas reclamações; mudar a prioridade (`SetPriority`) só vale para item ainda na fila.
3. **Justiça determinística:** ordem = correção de PR; **prioridade efetiva** do item (a informada, reduzida em 1 a cada período de espera, mínimo 0); prioridade do projeto (menor primeiro); chegada. Um item antigo e elegível, com o tempo, alcança prioridade 0 e, por ser mais antigo, ganha a vaga mesmo com urgentes chegando depois.
4. **Heartbeat e vencimento:** o dono renova a lease (`Heartbeat`, com o token `owner + geração`); `ExpireLeases` (e cada `Claim`) devolve à fila o item cuja lease venceu, libera a vaga e recoloca a tarefa em `pending`. Token antigo, lease vencida ou item fora de execução: `ErrStale`.
5. **Escritor obsoleto não grava:** a geração sobe a cada reclamação. Para item reclamado pelo scheduler, `Complete` e `Block` exigem o token do dono atual (`WithFence`) e a lease viva; sem token ou com token antigo → `ErrStale`. A gravação de checkpoint é protegida por `checkpoints.Store.Guard` (`Queue.Guard`), que roda **dentro da transação do `Save`** com o item travado: confere estado, dono, geração, lease e que o checkpoint é da execução do item; recusado, **nada é gravado**.
6. **Trilha:** `queue_events` registra reclamação, vencimento, heartbeat recusado e escrita recusada, com dono e geração.

## Limites
- Não há worker que chame `Claim`, renove a lease e use o token: este ticket entrega o mecanismo e a prova; a ligação ao `cmd/worker` e ao executor é posterior. Controles de ativar/pausar/suspender (T10-04) e recuperação após reboot (T12-03) ainda não existem; item com lease vencida volta à fila sem diagnóstico da tentativa interrompida.
- Itens iniciados por `Start` (sem scheduler, geração 0) seguem sem token: só `Claim` impõe a cerca.
- A cerca cobre `Complete`, `Block` e checkpoints; outras gravações dos executores precisam passar pelo `Guard` ou pelo token quando forem ligadas.
- Prioridade do item continua sendo a informada ao enfileirar (não deriva sozinha da origem).
