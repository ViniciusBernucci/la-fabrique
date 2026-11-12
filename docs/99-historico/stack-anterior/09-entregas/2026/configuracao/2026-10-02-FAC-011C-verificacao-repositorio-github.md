# Entrega FAC-011C — Verificacao somente-leitura do repositorio GitHub

Data: 2026-10-02

Status: DONE

## Objetivo e revisoes

Comprovar pelo worker que a conta GitHub conectada consegue ler o repositorio e a branch base salvos no Centro de Configuracoes, sem clone, escrita ou PR.

- Base aceita e registro do FAC-011B: `4e3f01d85e8ed386bc7dc6476f5a386d02d894a0`.
- Ticket READY: `9ef339c8aad58aaa546ef24d52189ad5b304d84e`.
- Codigo verificado: `53d605a53b4762bc1055046345aab7fdbeb31b34`.
- Branch/worktree: `feat/fac-011c-github-repository-verification`, `/home/vinicius/le-fabrique-fac-011-github-repo`.

## Funcionamento implementado

- `packages/contracts` valida snapshot, job e conclusao. Sucesso exige todas as observacoes; falha rejeita metadado parcial.
- `POST /api/settings/github/repository-verifications` so aceita configuracao persistida `CONNECTED`, com owner/repository. Pedido e `github.repository-verification.requested.v1` nascem na mesma transacao serializavel.
- A fila `le-fabrique.github-repository-verification` usa job ID estavel e concorrencia um. A API nunca executa `gh`.
- O worker executa duas chamadas fixas: `gh api --hostname <host> --method GET repos/<owner>/<repository>` e `GET` da branch base. Segmentos sao codificados; metodo, filtros e endpoints nao vêm do browser.
- A primeira resposta e reduzida em memoria a identidade canonica, branch padrao, visibilidade, arquivamento e `permissions.pull`; a segunda precisa devolver a branch base exata.
- `READABLE` exige `permissions.pull=true`, identidade normalizada igual e branch exata. Exit code nao zero, ausencia de permissao, divergencia, timeout, excesso de output ou JSON invalido retornam somente `UNAVAILABLE` e mensagem controlada.
- O subprocesso reutiliza o limite de 15 segundos/64 KiB, `shell: false`, stdin fechado, `GH_PROMPT_DISABLED=1` e ambiente sem `GH_TOKEN`, `GITHUB_TOKEN`, `GH_ENTERPRISE_TOKEN` ou `GITHUB_ENTERPRISE_TOKEN`.
- Alteracao do alvo ou perda de `CONNECTED` durante o job invalida a conclusao. Retransmissao do callback original devolve a falha normalizada de forma idempotente.
- O painel lista o resultado sanitizado e habilita a acao apenas para a configuracao salva elegivel; nao oferece clone, escrita, PR ou permissao de escrita.

## Criterios e evidencias

1. Fixtures da API rejeitam estado nao conectado, criam snapshot/outbox sem segredo e descartam observacoes quando a configuracao muda.
2. Contratos estritos rejeitam metodo extra no job, sucesso incompleto e falha com observacao parcial.
3. Fixtures do worker conferem binario `gh`, metodo `GET`, ausencia de campos de escrita e codificacao de branch com `/`.
4. Acesso negado, identidade divergente e branch indisponivel falham sem devolver stderr nem metadado parcial.
5. Suite comprova protocolo autenticado do worker, dispatcher com ID estavel e repeticao idempotente apos invalidacao.
6. `command -v gh` permaneceu sem resultado; nenhuma requisicao remota foi realizada.

## Checks reais

Executados no worktree sobre o codigo `53d605a53b4762bc1055046345aab7fdbeb31b34`:

- `npm ci`: 211 pacotes instalados; auditoria reportou 0 vulnerabilidades.
- `npm run db:generate`: Prisma Client 6.12.0 gerado.
- `npm run lint`: passou, 124 arquivos.
- `npm run typecheck`: passou em contracts, runtime, API, worker e web.
- `npm test`: passou, 134 testes em 33 arquivos (15 contracts, 38 runtime, 37 API, 42 worker e 2 web).
- `npm run build`: passou; Vite transformou 115 modulos e gerou bundle de producao.
- `DATABASE_URL=postgresql://fixture:fixture@127.0.0.1:5432/fixture npm exec -w @le-fabrique/api -- prisma validate --schema prisma/schema.prisma`: schema valido sem conexao ao banco.
- `git diff --check`: passou.

A primeira rodada real encontrou apenas formatacao/imports e foi corrigida pelo Biome. A segunda revisao fortaleceu a idempotencia da conclusao invalidada e adicionou teste; checks direcionados e suite integral passaram. Nao houve terceira rodada.

## Seguranca e fontes oficiais

O manual do `gh api` documenta requisicao autenticada, host e metodo explicitos. A API REST oficial documenta os metadados/permissoes do repositorio e que a leitura da branch exige permissao `Contents: read` para recurso privado. O worker fixa `GET`, seleciona poucos campos e descarta todo output bruto.

- [Manual oficial `gh api`](https://cli.github.com/manual/gh_api)
- [REST: obter um repositorio](https://docs.github.com/en/rest/repos/repos#get-a-repository)
- [REST: obter uma branch](https://docs.github.com/en/rest/branches/branches#get-a-branch)
- [Variaveis de ambiente do GitHub CLI](https://cli.github.com/manual/gh_help_environment)

Nenhum token, login, scope, header, URL privada ou output bruto entra em contrato/persistencia. Nenhuma API de IA, extra usage, credito, autorecharge ou fallback pago foi habilitado.

## Limitacoes conhecidas

- `gh`, keyring e conta reais continuam ausentes; rede, GitHub/GHES, credencial e permissoes nao foram comprovados operacionalmente.
- Fixtures cobrem PostgreSQL/outbox/BullMQ por doubles; migration nao foi aplicada e nao houve ensaio integrado com servicos reais.
- O painel passou por tipos, teste de view model e build, mas nao por E2E/ensaio visual em navegador.
- O resultado comprova apenas leitura do repositorio e branch naquele instante. Nao comprova clone, push, criacao de PR ou permissao futura.
- Repositorios publicos podem ser legiveis sem autenticacao segundo a API; o gate local exige `CONNECTED`, mas a prova real da identidade armazenada depende do preflight operacional FAC-011A/B.

## Diff, rollback e aceite

O diff funcional sanitizado e revisavel e o commit `53d605a53b4762bc1055046345aab7fdbeb31b34`; ele contem apenas alvos e respostas sinteticos, sem conta, repositorio privado, credencial ou resposta real.

Antes de aplicar a migration, o rollback e reverter o commit funcional e o commit documental. Se a migration aditiva ja tiver sido aplicada, manter a tabela/historico sem uso e criar migration compensatoria somente com autorizacao; nao apagar verificacoes manualmente. Nenhuma configuracao externa precisa de rollback.

O responsavel aceitou explicitamente a revisao documental exata `8fd38781652fac187d0f5a419eb336c15e94b816` em 2026-10-02. O FAC-011C esta `DONE`; instalacao, credencial e verificacao remota reais continuam fora deste aceite.
