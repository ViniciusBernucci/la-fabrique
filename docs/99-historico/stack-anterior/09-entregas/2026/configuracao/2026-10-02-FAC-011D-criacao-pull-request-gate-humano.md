# Entrega FAC-011D — Criacao de pull request sob gate humano

Data: 2026-10-02

Status: DONE

## Objetivo e revisoes

Permitir preparar e aprovar explicitamente um PR no Centro de Configuracoes, executando a escrita somente no worker depois de validar permissao e reconciliar duplicacao.

- Base aceita e registro do FAC-011C: `bf5ca83f7f3421767e2d4acdfe540957a4a84f18`.
- Ticket READY: `7dab2a83f76c25035578d2b4ef3b9310ea1ed104`.
- Codigo verificado: `29c4931f74eedb134150fbb12add2d520d68022d`.
- Branch/worktree: `feat/fac-011d-github-pull-request`, `/home/vinicius/le-fabrique-fac-011-github-pr`.

## Funcionamento implementado

- O painel prepara head, titulo, body e modo draft. Essa primeira acao persiste payload imutavel, versao e SHA-256, sem outbox ou worker.
- Um segundo ato administrativo mostra alvo/digest, pede confirmacao e envia `expectedVersion` + digest. A API reconfere configuracao `CONNECTED`, opt-in de PR e leitura FAC-011C correspondente.
- A aprovacao muda para `APPROVED` e cria `github.pull-request.requested.v1` atomicamente. Preparacao ainda nao aprovada pode ser cancelada sem side effect.
- Fila `le-fabrique.github-pull-request` usa ID estavel e concorrencia um. API nao executa CLI.
- O worker primeiro usa `gh api --method GET` e exige identidade normalizada e `permissions.push=true`.
- Antes de criar, `gh pr list` procura PR aberto no mesmo repositorio, head e base; resultado exato vira `EXISTING` sem nova escrita.
- Criacao usa `gh pr create --repo --base --head --title --body-file -`; body segue por stdin, e `--draft` e aplicado quando aprovado. `--head` explicito evita o fluxo interativo de fork/push.
- URL so passa se for HTTPS, host/repositorio exatos e `/pull/<numero>`. Resultado incerto executa uma reconciliacao final antes de falhar.
- Ambiente remove tokens GitHub herdados, prompt fica desligado, shell fica falso e processo tem 30 s/64 KiB. Shutdown mata processos de PR ativos.
- Mudanca do alvo, perda de conexao ou revogacao do opt-in durante o job invalida o callback. Nao existe comando de merge, push, ready, review, comment ou deploy.

## Evidencias e checks

- Contratos rejeitam campos extras, owner com `/`, branch invalida e conclusao parcial.
- Testes da API comprovam preparacao sem outbox, aprovacao separada com outbox e invalidacao de alvo alterado.
- Testes do worker cobrem permissao negada, PR existente, criacao draft com argv/stdin exatos e protocolo autenticado.
- `npm ci`: 211 pacotes, 0 vulnerabilidades.
- `npm run db:generate`: Prisma Client 6.12.0 gerado.
- `npm run lint`: passou, 131 arquivos.
- `npm run typecheck`: passou em todos os workspaces.
- `npm test`: passou, 143 testes em 36 arquivos (16 contracts, 38 runtime, 41 API, 46 worker e 2 web).
- `npm run build`: passou; Vite transformou 115 modulos.
- `prisma validate`: schema valido; `git diff --check`: passou.
- `command -v gh`: sem resultado; nenhuma chamada/escrita GitHub real ocorreu.

A primeira rodada corrigiu formatacao e nulabilidade tipada de streams do processo. A segunda endureceu owner/repository/branch antes de formar `--repo` e normalizou capitalizacao da identidade GitHub. Testes direcionados e suite integral passaram; nao houve terceira rodada.

## Fontes, limites e rollback

Fontes oficiais: [gh pr create](https://cli.github.com/manual/gh_pr_create), [gh pr list](https://cli.github.com/manual/gh_pr_list), [gh api](https://cli.github.com/manual/gh_api) e [ambiente](https://cli.github.com/manual/gh_help_environment).

Sem `gh`, keyring, branch remota e conta reais, permissao de escrita e criacao/reconciliacao nao foram comprovadas operacionalmente. Migration nao foi aplicada nesta worktree; PostgreSQL/Redis/BullMQ foram cobertos por doubles. Painel nao teve E2E visual. O fluxo nao cria branch/commit e nao mescla PR.

O diff funcional sanitizado e o commit `29c4931f74eedb134150fbb12add2d520d68022d`. Antes de aplicar migration, rollback e reverter os commits funcional/documental. Depois de aplicada, manter historico e usar migration compensatoria autorizada; reverter codigo nao fecha PR remoto eventualmente criado.

O responsavel aceitou explicitamente a revisao documental exata `76df60bbad4eac78b5d87fad8c2e79282355bf8c` em 2026-10-02. FAC-011D esta `DONE`; prova remota, merge e deploy continuam fora deste aceite.
