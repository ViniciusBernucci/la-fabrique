# FAC-011C — Verificacao somente-leitura do repositorio GitHub

Status: DONE

## Objetivo

Permitir que o operador solicite no Centro de Configuracoes uma verificacao autenticada e somente-leitura do repositorio e da branch base configurados. A API persiste apenas a intencao e o resultado sanitizado; o worker consulta metadados pelo GitHub CLI oficial sem clonar, escrever ou criar PR.

## Escopo

- Contratos Zod para verificacao, job e conclusao do acesso ao repositorio.
- Persistencia PostgreSQL e outbox transacional/idempotente.
- Fila BullMQ `le-fabrique.github-repository-verification`, com concorrencia um.
- Worker com executavel fixo `gh`, argv allowlisted, `shell: false`, timeout e saida limitados.
- Consulta autenticada `GET` via `gh api` ao repositorio e a branch base configurados.
- Remocao das variaveis de token herdadas e bloqueio de prompt interativo.
- Historico sanitizado no painel com identidade, branch observada, visibilidade e arquivamento.
- Fixtures sinteticas para acesso confirmado, repositorio/branch divergente, negado, timeout e resposta invalida.

## Fora do escopo

- Instalar ou autenticar o `gh`, receber PAT/token/cookie/chave pelo controle ou ler credencial.
- Clonar/fetch/pull, ler arquivos, configurar remote ou gravar no workspace.
- Criar branch, commit, issue, release ou PR; fazer merge, deploy ou qualquer chamada HTTP mutavel.
- Inferir permissao de escrita ou autorizar publicacao com base nesta verificacao.
- Executar provider de IA, habilitar API paga, extra usage, creditos, autorecharge ou fallback.

## Criterios de aceite

1. Pedido so e aceito com GitHub `CONNECTED`, owner/repository preenchidos e branch base valida; verificacao ativa equivalente e deduplicada.
2. Verificacao e evento `github.repository-verification.requested.v1` sao gravados atomicamente com snapshot de host, owner, repositorio e branch.
3. API nao executa CLI; worker usa apenas `gh api --hostname <host> --method GET` com endpoints derivados do snapshot validado.
4. Owner, repositorio e branch sao codificados como segmentos; browser nao envia argv, endpoint, metodo nem expressao de filtro.
5. Variaveis de token herdadas sao removidas; output bruto, URL, login, token, scopes e headers nunca sao persistidos ou retornados.
6. `READABLE` exige identidade exata, permissao de leitura declarada e branch base exata; divergencia, ausencia/negacao, timeout e JSON invalido falham fechado com mensagem controlada.
7. Alteracao de host/owner/repository/branch ou perda do estado `CONNECTED` durante o job invalida o resultado sem sobrescrever a configuracao nova.
8. Resultado exige worker autenticado e transicao valida; repeticao identica e idempotente.
9. Painel solicita e acompanha o historico, desabilita pedido inelegivel/ativo e nao oferece clone, escrita ou PR.
10. Lint, typecheck, testes, build, validacao Prisma e `git diff --check` passam.

## Baseline, caminhos e limites

- Base aceita e registro do FAC-011B: `4e3f01d85e8ed386bc7dc6476f5a386d02d894a0`.
- Branch/worktree: `feat/fac-011c-github-repository-verification`, `/home/vinicius/le-fabrique-fac-011-github-repo`.
- Caminhos: `packages/contracts`, `apps/api`, `apps/worker`, `apps/web`, migration Prisma, configuracao/operacao e lesson pertinente.
- GitHub CLI observado: ausente; nenhuma instalacao, autenticacao ou chamada GitHub real sera feita.
- Timeout: 15 segundos por consulta, no maximo duas consultas; saida combinada de cada processo: 64 KiB; fila com concorrencia um.
- Um writer, ate duas rodadas de correcao, nenhuma IA, budget adicional zero, sem merge/deploy/migration aplicada.

## Provider elegivel e fontes oficiais

O provider e GitHub CLI em modo `GH_CLI`. O pedido so e elegivel apos evidencia local `CONNECTED` com armazenamento seguro. `gh api` faz requisicoes autenticadas e oferece `--hostname`, `--method` e `--jq`; este ticket fixa `GET` e descarta a resposta bruta apos validacao em memoria.

- <https://cli.github.com/manual/gh_api>
- <https://docs.github.com/en/rest/repos/repos#get-a-repository>
- <https://docs.github.com/en/rest/branches/branches#get-a-branch>
- <https://cli.github.com/manual/gh_help_environment>

## Sequenciamento

FAC-011C comprova somente leitura do alvo configurado. Criacao de PR, permissao de escrita e gate humano pertencem a um ticket posterior; merge continua manual.

## Revisao entregue

- Ticket READY: `9ef339c8aad58aaa546ef24d52189ad5b304d84e`.
- Codigo verificado: `53d605a53b4762bc1055046345aab7fdbeb31b34`.
- Relatorio: `documentacoes/configuracao/2026-10-02-FAC-011C-verificacao-repositorio-github.md`.
- Estado real: fixtures aprovadas; `gh` ausente, migration nao aplicada e GitHub remoto nao consultado.

## Aceite

Entrega aceita explicitamente pelo responsavel em 2026-10-02 na revisao documental exata `8fd38781652fac187d0f5a419eb336c15e94b816`.
