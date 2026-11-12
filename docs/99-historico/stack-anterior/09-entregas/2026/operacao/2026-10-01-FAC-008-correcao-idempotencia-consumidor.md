# FAC-008 — Correcao de idempotencia e consumidor

Data: 2026-10-01
Estado: DONE
Revisao funcional: `5a766c4c928dda27c3b1417afbc4ec3e1de52d0d`

## Objetivo e resultado

A revisao de `dbdee37` encontrou dois bloqueios: uma reentrega entre checkpoint e conclusao criava outro attempt, e o processor sintetico nao estava conectado a fila publicada pela API. A correcao preserva o mesmo attempt nessa janela e liga o worker a `le-fabrique.execution`, sem executar provider, build, teste ou codigo do projeto.

## Funcionamento

- Se uma run ja possui attempt `STOPPED`, uma nova entrega devolve esse attempt com `replayed=true`; retry deliberado exigira contrato proprio em ticket futuro.
- O evento de outbox carrega `baseRevision` quando `Project.baseRef` e um SHA Git completo. Eventos legados ou referencias simbolicas sao normalizados para `null`.
- O worker valida o job e recusa uma revisao-base desconhecida antes do claim. Com SHA resolvido, executa apenas claim, checkpoint sintetico com parada confirmada e complete para `VALIDATING`.
- O worker fecha os consumidores de probe e orquestracao no mesmo shutdown. Nenhum adapter ou sandbox foi acoplado nesta correcao.

## Evidencias

- Teste unitario novo reproduz o estado `run=RUNNING` e `attempt=STOPPED` e comprova que `attempt.create` nao e chamado.
- Teste do processor comprova falha anterior ao claim quando `baseRevision=null`.
- A suite passou com 57 testes: contratos 8, runtime 27, API 15, worker 6 e web 1.
- `npm run lint`, `npm run typecheck` e `git diff --check` passaram na revisao funcional.
- O build final e a revisao do diff completo sao repetidos depois do commit documental; a evidencia final deve permanecer vinculada a revisao entregue.

## Limites

O consumer e deliberadamente uma fixture de protocolo. Ele nao cria worktree, nao chama RuntimeGuard, sandbox, adapter Codex ou checks; essa composicao continua no FAC-009. Uma referencia como `main` nao e resolvida pela API e faz o job falhar antes do claim; a resolucao segura de refs pertence ao supervisor confiavel.

Nao houve migration, provider, API de IA, credito, extra usage, merge ou deploy.

## Rollback

Reverter `5a766c4c928dda27c3b1417afbc4ec3e1de52d0d` remove o segundo consumidor e restaura o comportamento anterior. Antes do rollback, parar o worker e confirmar que nao ha job ativo; a reversao de codigo nao remove runs, attempts, checkpoints nem jobs persistidos. Nao ha migration nova para desfazer.

## Estado do aceite

O responsavel aceitou explicitamente a revisao `29061a911e0f6bc5122e9e53511f2f475caf8dce` em 2026-10-01. FAC-008 esta `DONE`; FAC-009 pode ser preparado em branch/worktree proprio.
