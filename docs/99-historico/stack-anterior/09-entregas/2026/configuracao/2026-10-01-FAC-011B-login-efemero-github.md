# Entrega FAC-011B — Login efemero do GitHub CLI

Data: 2026-10-01

Status: DONE

## Objetivo e revisoes

Permitir login web/device oficial do GitHub CLI iniciado no Centro de Configuracoes, sem PAT no controle e sem persistir URL/codigo temporarios no PostgreSQL.

- Base aceita e registro do FAC-011A: `eed7f85f564387ce6ed006e7cac4e46cd874ac30`.
- Ticket READY: `da02b574ba60a03e136da18848b259f9c5207d15`.
- Codigo verificado: `c2f353db0befbaa5b8bbddc69430f739206c0b7a`.
- Branch/worktree: `feat/fac-011b-github-login`, `/home/vinicius/le-fabrique-fac-011-github-login`.

## Funcionamento implementado

- `packages/contracts` valida sessao, job, desafio, armazenamento observado e conclusao. Objetos sao estritos e nao possuem PAT/token/cookie/senha.
- `POST /api/settings/github/onboarding` cria sessao e `github.onboarding.requested.v1` atomicamente quando o estado observado e `AUTH_REQUIRED`; `GET` lista historico e entrega desafio efemero por ID.
- A fila `le-fabrique.github-onboarding` tem concorrencia um. Verificacao e login GitHub ativos se excluem para evitar que resultados concorrentes sobrescrevam o estado.
- O worker executa `gh auth login --hostname <host> --git-protocol https --web --skip-ssh-key --clipboard=false`, com binario fixo, `shell: false`, stdin ignorado, dez minutos e 64 KiB.
- `GH_TOKEN`, `GITHUB_TOKEN`, `GH_ENTERPRISE_TOKEN` e `GITHUB_ENTERPRISE_TOKEN` sao removidas; `GH_PROMPT_DISABLED=1` impede prompts alternativos.
- O parser extrai somente codigo limitado e URL HTTPS `/login/device`. A API confere que o hostname coincide com o snapshot/configuracao antes de guardar o desafio no Redis privado por no maximo dez minutos.
- URL/codigo vivem apenas no Redis e no estado em memoria da tela; falha de Redis/publicacao mata o processo. Conclusao tenta remover a chave, e TTL continua sendo o limite se Redis estiver indisponivel.
- Exit code zero e insuficiente. O worker consulta `gh auth status --hostname <host> --json hosts`; apenas conta ativa `success` com `tokenSource=keyring` produz `CONNECTED/SECURE_STORE`.
- `hosts.yml` produz `FAILED/ERROR/PLAINTEXT_FILE`; origem desconhecida tambem falha. A mesma regra passou a valer na verificacao FAC-011A, impedindo promocao indireta do fallback inseguro.
- SIGINT, SIGTERM ou perda do controle cancelam os processos GitHub ativos antes de fechar a fila.
- O painel inicia o login, acompanha sessao/armazenamento/mensagem e mostra link/codigo somente enquanto o desafio existe.

## Criterios e evidencias

1. Sessao so nasce em `AUTH_REQUIRED`; sessao ativa e deduplicada e verificacao concorrente bloqueia o pedido.
2. Outbox contem somente `sessionId`, host e expiracao; teste rejeita desafio, token e dados de credencial.
3. Teste do argv comprova fluxo web HTTPS, sem `--with-token`, `--insecure-storage`, `--show-token` ou clipboard.
4. Contratos rejeitam path diferente de `/login/device`, campo extra e `COMPLETED` sem `CONNECTED/SECURE_STORE`.
5. Servico rejeita URL de outro host e limita o TTL Redis.
6. Fixtures classificam keyring, `hosts.yml`, origem desconhecida, JSON invalido, ausencia de desafio e confirmacao completa sem retornar login/path/output bruto.
7. Configuracao e historico sao atualizados na mesma transacao; host alterado invalida o resultado antigo.
8. UI e API nao possuem campo de PAT e o worker registra a capacidade `github-device-onboarding` sem segredo.
9. `command -v gh` permaneceu sem resultado; nenhuma autenticacao/chamada GitHub real ocorreu.

## Checks reais

Executados no worktree sobre o codigo `c2f353db0befbaa5b8bbddc69430f739206c0b7a`:

- `npm ci`: 211 pacotes instalados; auditoria reportou 0 vulnerabilidades.
- `npm run db:generate`: Prisma Client 6.12.0 gerado.
- `npm run lint`: passou, 117 arquivos.
- `npm run typecheck`: passou em contracts, runtime, API, worker e web.
- `npm test`: passou, 123 testes em 30 arquivos (14 contracts, 38 runtime, 31 API, 38 worker e 2 web).
- `npm run build`: passou; Vite transformou 115 modulos e gerou bundle de producao.
- `DATABASE_URL=postgresql://fixture:fixture@127.0.0.1:5432/fixture npm exec -w @le-fabrique/api -- prisma validate --schema prisma/schema.prisma`: schema valido sem conexao ao banco.
- `git diff --check`: passou.

A primeira rodada corrigiu a representacao do escape ANSI aceita pelo lint e estreitou tipos opcionais na UI. A segunda atualizou a fixture FAC-011A para a nova evidencia `keyring` e fortaleceu a verificacao contra `hosts.yml`; a suite integral final passou. Nao houve terceira rodada.

## Seguranca e fontes oficiais

O codigo oficial do GitHub CLI mostra que o modo web nao interativo imprime codigo/URL e que o login tenta o keyring, podendo cair para `hosts.yml`. Tambem mostra `tokenSource=keyring` para credencial segura. A implementacao usa esses sinais sem copiar o token e sem depender apenas da mensagem textual de sucesso.

- [Manual oficial `gh auth login`](https://cli.github.com/manual/gh_auth_login)
- [Manual oficial `gh auth status`](https://cli.github.com/manual/gh_auth_status)
- [Variaveis de ambiente oficiais](https://cli.github.com/manual/gh_help_environment)
- [Fluxo oficial no codigo do GitHub CLI](https://github.com/cli/cli/blob/trunk/internal/authflow/flow.go)
- [Persistencia oficial no codigo do GitHub CLI](https://github.com/cli/cli/blob/trunk/internal/config/config.go)

## Limitacoes conhecidas

- Sem `gh`, credential store e conta reais, o device flow, a versao, a rede e `tokenSource` nao foram comprovados operacionalmente.
- Se o cliente oficial gravar em `hosts.yml`, a La fabrique marca `ERROR` mas nao apaga a credencial nem executa logout automatico, evitando remover outra conta. O operador deve corrigir o keyring e tratar o arquivo sob a identidade de servico.
- PostgreSQL/Redis/BullMQ foram cobertos com contratos e doubles, sem ensaio integrado real nesta revisao.
- O painel passou por typecheck, testes de view model e build, mas nao por E2E/ensaio visual em navegador.
- O login comprova autenticacao segura do host; ainda nao comprova permissao no repositorio configurado nem cria PR.

Nao houve instalacao, login, chamada remota, migration aplicada, PR, merge, deploy, provider de IA ou consumo de cota.

## Diff, rollback e aceite

O diff funcional sanitizado e revisavel e o commit `c2f353db0befbaa5b8bbddc69430f739206c0b7a`; ele contem apenas fixtures sinteticas, sem conta, codigo real, token, path real de credencial ou host privado.

Antes de aplicar a migration, o rollback e reverter o commit funcional e este commit documental. Se a migration aditiva ja tiver sido aplicada, manter tabela/historico sem uso e criar migration compensatoria somente com autorizacao; nao apagar sessoes manualmente. Nenhuma configuracao externa precisa de rollback nesta entrega.

O responsavel aceitou explicitamente a revisao documental exata `3ac8b39de9024576df5c5709022bb4a01e5ea6a5` em 2026-10-02. O FAC-011B esta `DONE`; instalacao, keyring e login reais continuam fora deste aceite.
