# FAC-011D — Criacao de pull request sob gate humano

Status: DONE

## Objetivo

Permitir preparar, revisar e aprovar explicitamente no Centro de Configuracoes a criacao de um pull request no repositorio GitHub configurado. A aprovacao fica vinculada ao payload exato; somente depois dela a API grava a intencao na outbox e o worker comprova permissao de escrita antes de executar o GitHub CLI oficial.

## Escopo

- Contratos Zod para pedido preparado, aprovacao exata, cancelamento, job e conclusao.
- Persistencia PostgreSQL com versao, digest de aprovacao, snapshot do alvo e historico.
- Preparacao sem side effect; aprovacao administrativa separada cria `github.pull-request.requested.v1` atomicamente.
- Fila BullMQ `le-fabrique.github-pull-request`, com concorrencia um.
- Worker primeiro consulta `permissions.push` por `gh api --method GET`.
- Reconciliacao de PR aberto por head/base antes de criar, reduzindo duplicacao em reentrega.
- Criacao nao interativa por `gh pr create`, com repo/base/head/title fixados e body por stdin.
- PR draft configuravel por pedido, com default seguro `true`.
- Painel para preparar, conferir digest/payload, aprovar em segundo ato e cancelar antes da aprovacao.
- Resultado sanitizado com numero, URL e disposicao `CREATED` ou `EXISTING`.

## Fora do escopo

- Criar ou enviar branch/commit, executar clone/fetch/push ou escolher repositorio diferente do salvo.
- Editar, fechar, reabrir, marcar ready, revisar, comentar ou mesclar PR.
- Auto-merge, merge queue, deploy, release, issue, label, reviewer, milestone ou projeto.
- Receber token/PAT/cookie/chave pelo controle, usar variavel de token ou expor output bruto.
- Executar provider de IA, habilitar API paga, extra usage, creditos, autorecharge ou fallback.

## Criterios de aceite

1. Preparacao exige GitHub `CONNECTED`, `pullRequestCreationEnabled=true`, alvo completo e verificacao FAC-011C `READABLE` correspondente.
2. Preparar nao cria outbox nem chama worker; payload imutavel recebe versao e SHA-256 para revisao.
3. Aprovacao e uma segunda requisicao autenticada com `expectedVersion` e digest exato; somente ela muda para `APPROVED` e cria outbox na mesma transacao.
4. Pedido `PREPARED` pode ser cancelado sem side effect; pedido aprovado/executado nao pode ser cancelado por este fluxo.
5. Worker recebe somente snapshot validado; browser nao envia binario, argv, metodo, endpoint ou flags livres.
6. Antes da escrita, `gh api --method GET` exige identidade exata e `permissions.push=true`; falha encerra sem `gh pr create`.
7. Worker procura PR aberto com repo/head/base exatos; se existir, registra `EXISTING` sem criar outro.
8. Criacao usa `gh pr create --repo --base --head --title --body-file -`, flags explicitas, shell desligado, prompt desabilitado, tokens herdados removidos e limites de tempo/output.
9. URL retornada precisa ser HTTPS no host e path exato do repositorio `/pull/<numero>`; output bruto/stderr nao e persistido.
10. Mudanca de configuracao, perda de conexao ou revogacao da opcao de PR antes da conclusao invalida o resultado.
11. Nenhum caminho executa `gh pr merge`, auto-merge, push ou comando Git.
12. Lint, typecheck, testes, build, validacao Prisma e `git diff --check` passam.

## Baseline, caminhos e limites

- Base aceita e registro do FAC-011C: `bf5ca83f7f3421767e2d4acdfe540957a4a84f18`.
- Branch/worktree: `feat/fac-011d-github-pull-request`, `/home/vinicius/le-fabrique-fac-011-github-pr`.
- Caminhos: `packages/contracts`, `apps/api`, `apps/worker`, `apps/web`, migration Prisma, configuracao/controle/operacao e lesson pertinente.
- GitHub CLI observado: ausente; nenhuma instalacao, autenticacao, escrita ou chamada GitHub real sera feita.
- Titulo: 1–200 caracteres; body: ate 10 KiB; branch: ate 200 caracteres; subprocesso: 30 segundos e 64 KiB.
- Um job por vez, um writer, ate duas rodadas de correcao, nenhuma IA, budget adicional zero, sem merge/deploy/migration aplicada.

## Provider elegivel e fontes oficiais

O provider e GitHub CLI em modo `GH_CLI`, elegivel apenas apos FAC-011C legivel e opcao administrativa de PR habilitada. O manual oficial documenta `--head` para evitar prompt de fork/push, `--body-file -` para stdin e URL impressa no sucesso. `gh pr list` oferece filtros head/base e JSON para reconciliacao.

- <https://cli.github.com/manual/gh_pr_create>
- <https://cli.github.com/manual/gh_pr_list>
- <https://cli.github.com/manual/gh_api>
- <https://cli.github.com/manual/gh_help_environment>

## Sequenciamento

FAC-011D cria ou reconcilia somente o PR aprovado. Tornar draft pronto, revisar, comentar, mesclar e fazer deploy permanecem fora do MVP atual e exigem tickets/gates proprios.

## Revisao entregue

- Ticket READY: `7dab2a83f76c25035578d2b4ef3b9310ea1ed104`.
- Codigo verificado: `29c4931f74eedb134150fbb12add2d520d68022d`.
- Relatorio: `documentacoes/configuracao/2026-10-02-FAC-011D-criacao-pull-request-gate-humano.md`.
- Estado real: fixtures aprovadas; `gh` ausente, migration nao aplicada e nenhum PR remoto criado.

## Aceite

Entrega aceita explicitamente pelo responsavel em 2026-10-02 na revisao documental exata `76df60bbad4eac78b5d87fad8c2e79282355bf8c`.
