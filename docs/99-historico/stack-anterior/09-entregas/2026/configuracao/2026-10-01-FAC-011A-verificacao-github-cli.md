# Entrega FAC-011A — Verificacao gerenciada do GitHub CLI

Data: 2026-10-01

Status: DONE

## Objetivo e revisoes

Adicionar ao Centro de Configuracoes uma verificacao somente-leitura da instalacao e autenticacao do GitHub CLI, executada exclusivamente pelo worker e sem transportar token pelo painel, API ou PostgreSQL.

- Base aceita e registro do FAC-010C: `bf8f45edd1143ffc21f0223214d5f0b782aabf90`.
- Ticket READY: `0d85f198be1bf32ea2dad156e1d0852e31f00597`.
- Codigo verificado: `6ce04c5f8c94779c4a110fde52ba77891df8c926`.
- Branch/worktree: `feat/fac-011a-github-verification`, `/home/vinicius/le-fabrique-fac-011-github`.

## Funcionamento implementado

- `packages/contracts` valida historico, job e conclusao; objetos sao estritos e nao possuem campo de credencial.
- `POST /api/settings/github/verifications` cria uma verificacao e `github.verification.requested.v1` atomicamente. `GET` lista as 50 mais recentes.
- O dispatcher publica `github.verify.v1` na fila dedicada `le-fabrique.github-verification`; o worker consome com concorrencia um.
- O worker usa somente `gh --version` e `gh auth status --hostname <host> --json hosts`, com `shell: false`, timeout de 15 segundos e limite combinado de 64 KiB.
- `GH_TOKEN`, `GITHUB_TOKEN`, `GH_ENTERPRISE_TOKEN` e `GITHUB_ENTERPRISE_TOKEN` sao removidas do ambiente; `GH_PROMPT_DISABLED=1` evita prompt interativo.
- O JSON bruto, login, origem da credencial, scopes e stderr nao sao devolvidos ou persistidos. O controle recebe somente estado, versao e mensagem fixa.
- Conta ativa com estado `success` produz `CONNECTED`; host sem conta ativa valida produz `AUTH_REQUIRED`; CLI ausente, timeout, versao/JSON invalidos produzem `FAILED`/`ERROR`.
- A conclusao autenticada pelo worker atualiza configuracao e historico numa transacao. Se o host mudou durante a execucao, o job antigo falha sem sobrescrever a configuracao nova.
- O painel mostra estado/versao/mensagem, desabilita novo pedido enquanto ha job ativo e atualiza o estado observado por polling.

## Criterios e evidencias

1. Pedido ativo e deduplicado pelo servico; criacao e outbox usam transacao serializavel.
2. API apenas persiste/consulta; nenhum `spawn` existe no controle.
3. Executavel e argv sao fixos no worker; host vem do singleton validado e fica no snapshot do job.
4. Contratos rejeitam campo extra como `token` e resultados inconsistentes (`FAILED` com `CONNECTED`).
5. Fixture com login/tokenSource sinteticos comprova que esses dados nao entram no resultado.
6. Fixtures cobrem conectado, auth requerida, output invalido, ambiente sanitizado, protocolo worker e host alterado durante a execucao.
7. O binario `gh` nao existe neste ambiente (`command -v gh` sem resultado); portanto nenhuma consulta/login GitHub real foi executada.

## Checks reais

Executados no worktree sobre o codigo `6ce04c5f8c94779c4a110fde52ba77891df8c926`:

- `npm ci`: 211 pacotes instalados; auditoria reportou 0 vulnerabilidades.
- `npm run db:generate`: Prisma Client 6.12.0 gerado.
- `npm run lint`: passou, 110 arquivos.
- `npm run typecheck`: passou em contracts, runtime, API, worker e web.
- `npm test`: passou, 109 testes em 27 arquivos (13 contracts, 38 runtime, 26 API, 30 worker e 2 web).
- `npm run build`: passou; Vite transformou 115 modulos e gerou bundle de producao.
- `DATABASE_URL=postgresql://fixture:fixture@127.0.0.1:5432/fixture npm exec -w @le-fabrique/api -- prisma validate --schema prisma/schema.prisma`: schema valido sem conexao ao banco.
- `git diff --check`: passou.

A primeira rodada corrigiu apenas parse/formatacao/imports apontados pelo lint. A segunda corrigiu o encerramento do job quando o host muda durante a execucao; os testes direcionados e depois a suite integral passaram. Nao houve terceira rodada.

## Seguranca, limites e fontes

A verificacao de status pode fazer uma consulta autenticada de leitura ao host quando `gh` estiver instalado, mas nunca usa `--show-token`, `--with-token` ou `--insecure-storage`. O status JSON oficial inclui metadados potencialmente sensiveis; por isso ele e classificado somente em memoria e descartado. As variaveis de token documentadas pelo GitHub tem precedencia sobre credenciais armazenadas e sao removidas do subprocesso gerenciado.

- [Manual oficial `gh auth status`](https://cli.github.com/manual/gh_auth_status)
- [Variaveis de ambiente oficiais do GitHub CLI](https://cli.github.com/manual/gh_help_environment)
- [Manual oficial `gh auth login`](https://cli.github.com/manual/gh_auth_login)

Nao foram executados login, instalacao do `gh`, chamada remota, PR, merge, deploy, migration real, provider de IA ou consumo de cota. Login oficial iniciado pelo painel e criacao de PR permanecem incrementos separados.

## Limitacoes conhecidas

- Sem `gh`, nao foi possivel validar neste ambiente a versao real, o credential store, a conectividade nem a forma exata de uma conta configurada.
- Persistencia/outbox/fila foram cobertas por contratos e testes com doubles; nao houve ensaio integrado com PostgreSQL/Redis reais nesta revisao.
- O painel passou por typecheck, testes de view model e build, mas nao por E2E/ensaio visual em navegador.
- A verificacao observa autenticacao do host; ela ainda nao comprova permissao no repositorio configurado nem capacidade de criar PR.

## Diff, rollback e aceite

O diff funcional sanitizado e revisavel e o commit `6ce04c5f8c94779c4a110fde52ba77891df8c926`; ele nao contem output real, conta, token ou host privado.

Antes de aplicar a migration, o rollback e reverter o commit funcional e este commit documental. Se a migration aditiva ja tiver sido aplicada em outro ambiente, a aplicacao anterior pode continuar com a tabela sem uso; qualquer migration compensatoria deve preservar historico e ser autorizada, sem apagar linhas manualmente. Nenhum rollback externo e necessario porque nenhuma configuracao GitHub real foi alterada.

O responsavel aceitou explicitamente a revisao documental exata `dce2e676a91b5ffaeb246afeea2cc699e60aff9d` em 2026-10-01. O FAC-011A esta `DONE`; a instalacao/login real do GitHub CLI continua fora deste aceite.
