# FAC-003A — Gate de revisao-base exata antes de READY

Status: DONE

## Objetivo

Impedir que um ticket seja promovido a `READY` quando o projeto ainda possui apenas um nome de branch, como `main`, em vez de um commit Git exato. Permitir que o operador atualize essa revisao-base pelo painel antes de criar a intencao de execucao.

## Atual e esperado

O controle aceita qualquer `baseRef` no projeto e, ao promover o ticket, grava `baseRevision: null` quando o valor nao e um SHA hexadecimal de 40 caracteres. O dispatcher publica o job, e o worker falha antes do claim com `Fixture base revision is unavailable`. Os seis IDs observados no ensaio eram jobs distintos, nao retries do mesmo job.

O esperado e falhar fechado no controle antes de alterar ticket ou outbox, exibir o bloqueio no painel e oferecer atualizacao otimista para um SHA exato. Jobs antigos permanecem no historico da fila.

## Escopo permitido e proibido

- Permitido: contrato compartilhado de SHA/base revision, endpoint administrativo para atualizar `baseRef`, gate transacional de READY, painel e testes.
- Permitido: manter nome de branch no cadastro inicial para planejamento, desde que READY fique bloqueado ate a revisao exata ser configurada.
- Proibido: resolver branch remotamente, executar `git fetch`, chamar GitHub, limpar Redis, remover jobs falhos ou reabrir tickets ja publicados.
- Proibido: migration, provider de IA, merge, deploy ou qualquer escrita em repositorio externo.

## Criterios de aceite

1. Um schema runtime compartilhado aceita somente SHA Git hexadecimal de 40 caracteres em minusculas para a revisao-base executavel.
2. Endpoint administrativo atualiza a revisao do projeto com `expectedBaseRef`, recusando concorrencia e SHA invalido sem escrita parcial.
3. Projeto inexistente retorna not found; conflito preserva o valor atual.
4. Promocao de `DRAFT` para `READY` valida a revisao exata antes de atualizar ticket ou criar outbox.
5. Evento `ticket.ready.v1` novo sempre contem `baseRevision` nao nula e validada.
6. Retry idempotente de um ticket ja `READY` com sua outbox existente continua retornando o estado atual.
7. O painel mostra a referencia atual, permite substitui-la por SHA exato e desabilita `Marcar READY` enquanto o projeto selecionado nao estiver resolvido.
8. O painel explica que jobs falhos historicos sao preservados e nao confunde seus IDs distintos com retry automatico.
9. Nenhum caminho consulta Git/GitHub, limpa fila ou altera os jobs historicos.
10. Lint, typecheck, testes, build, Prisma validate e `git diff --check` passam.

## Baseline, caminhos e limites

- Base aceita: `4e5b772a2dbeb1dd8868e3fe2bedd480c8e837c4`, registro do aceite OPS-001.
- Branch/worktree: `fix/ops-002-stale-orchestration-jobs`, `/home/vinicius/le-fabrique-ops-002`.
- Caminhos: `packages/contracts`, `apps/api/src/control`, `apps/web`, documentacao de controle/operacao, planejamento, indice, changelog, backlog e lesson pertinente.
- Sem migration; banco/Redis reais nao serao acessados; ate duas rodadas de correcao.
- Um writer, nenhuma IA do produto, custo adicional zero.

## Provider elegivel

Nenhum. A revisao e informada pelo operador e validada localmente; resolucao remota futura depende de um fluxo Git/GitHub explicitamente autorizado.

## Checks

`npm ci`, `npm run db:generate`, `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`, Prisma validate com URL sintetica e `git diff --check`.

## Entregaveis e aceite

Ticket READY: `dcc380c11775bba97c7020d5a22ca075f4feee05`. Codigo verificado: `e65c88fd8f3bd15e3aa65773fedc945af3709def`. Aceite do responsavel: revisao documental `4630ce0b9a613b937d40d550e4551fa1402d6c6a` em 2026-10-02.
