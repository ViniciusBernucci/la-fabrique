# FAC-009 — Developer, checks e revisao

Data: 2026-10-01
Estado: DONE
Revisao funcional: `7ba457470171d80571c0ce8650ed2b26f3f197ef`

## Resultado

Foram adicionados contratos Zod e um `DeveloperWorkflow` no worker. O fluxo compoe worktree detached, contexto deterministico, RuntimeGuard, Developer com escrita, baseline/checks isolados, snapshots e Reviewer separado somente leitura. A saida normalizada distingue aprovacao tecnica pendente de humano, pausa por limite e falha fechada.

## Funcionamento

1. Valida request, revisao, comandos, limites e politica subscription-only.
2. Cria workspace na revisao exata e constroi manifesto de contexto.
3. Executa baseline com argv confiavel; termino nao confirmado bloqueia o fluxo.
4. Autoriza Developer no RuntimeGuard e chama o adapter com `WORKSPACE_WRITE`.
5. Repete checks, marca falha preexistente por nome e trata somente falha nova como regressao.
6. Captura snapshot verificavel antes de review ou nova correcao.
7. Autoriza nova execucao do adapter em `READ_ONLY`; somente JSON valido `APPROVE|REQUEST_CHANGES` e aceito.
8. `APPROVE` leva a `AWAITING_HUMAN`. Regressao/rejeicao pode gerar no maximo duas correcoes, sujeitas ao limite menor do RuntimeGuard; falha repetida pausa.

## Evidencias

- Contratos rejeitam mais de duas correcoes.
- Testes do worker cobrem baseline preexistente, uma regressao corrigida, review nao estruturado, budget esgotado, rejeicao repetida e processo baseline sem quiescencia.
- Testes usam mocks de adapter/sandbox/workspace/snapshot e nao chamam Codex nem API.
- `npm ci` instalou 211 pacotes com zero vulnerabilidades reportadas, e `npm run db:generate` preparou o Prisma Client do worktree; lockfile e schema nao mudaram.
- `npm run lint`: 77 arquivos, passou.
- `npm run typecheck`: contratos, runtime, API, worker e web, passou.
- `npm test`: 64 testes em 17 arquivos, passou; contratos 9, runtime 27, API 15, worker 12 e web 1.
- `npm run build`: contratos, runtime, API, worker e web, passou.
- `git diff --check`: passou.

Uma revisao interna encontrou que falha baseline sem parada confirmada poderia ser tratada como preexistente. A correcao adicionou `CHECK_UNQUIESCED` antes de qualquer writer ou snapshot e teste dedicado. Nao houve segunda rodada de correcao funcional.

## Limites

O coordenador e executavel por chamada direta, mas ainda nao esta ligado ao consumer `le-fabrique.execution`. O controle nao possui um perfil confiavel que associe `projectId` a clone local e allowlist de comandos; aceitar esses valores do output de IA ou de caminho arbitrario violaria o limite operacional. Persistir resultados do coordenador no checkpoint FAC-008 e configurar o perfil pertencem a integracao seguinte.

O Codex CLI elegivel continua sendo `0.159.2` conforme evidencia anterior, mas nao foi chamado nesta entrega. Modelo efetivo, uso e cota permanecem nao observados. Nao houve API, credito, extra usage, migration, merge ou deploy.

## Rollback

Reverter `7ba457470171d80571c0ce8650ed2b26f3f197ef` remove os contratos e o coordenador. Como o loop BullMQ nao foi alterado, nao ha job real novo para drenar, migration para desfazer nem artefato persistente criado pelos testes.

## Estado do aceite

O responsavel aceitou explicitamente a revisao `02819fb847e303f1a823cc2784a7326ec5e696f9` em 2026-10-01. FAC-009 esta `DONE`; FAC-010 pode ser preparado em branch/worktree proprio.
