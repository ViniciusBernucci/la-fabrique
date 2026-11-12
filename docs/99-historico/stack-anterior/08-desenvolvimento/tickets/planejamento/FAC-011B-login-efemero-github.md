# FAC-011B — Login efemero do GitHub CLI

Status: DONE

## Objetivo

Permitir que o operador inicie no Centro de Configuracoes o login web/device oficial do GitHub CLI. Sessao e intencao ficam no PostgreSQL/outbox; URL e codigo temporarios ficam somente no Redis privado e na memoria da tela; o token permanece exclusivamente no armazenamento oficial do `gh` sob a identidade do worker.

## Escopo

- Contratos Zod para sessao, job, desafio e conclusao GitHub.
- Persistencia PostgreSQL de metadados e outbox transacional/idempotente.
- Desafio efemero no Redis com TTL maximo de dez minutos e remocao ao concluir.
- Fila BullMQ `le-fabrique.github-onboarding`, com concorrencia um.
- Worker executa somente `gh auth login --hostname <host> --git-protocol https --web --skip-ssh-key --clipboard=false`, sem shell e sem stdin de token.
- Remocao de `GH_TOKEN`, `GITHUB_TOKEN`, `GH_ENTERPRISE_TOKEN` e `GITHUB_ENTERPRISE_TOKEN`; `GH_PROMPT_DISABLED=1`.
- Parser limitado ao codigo temporario e URL HTTPS `<host>/login/device`; nenhum outro output retorna.
- Confirmacao posterior por `gh auth status --hostname <host> --json hosts`.
- Classificacao do `tokenSource`: somente credential store seguro permite `CONNECTED`; fallback `hosts.yml` falha fechado com mensagem controlada.
- Painel exibe link/codigo apenas durante a sessao e acompanha o resultado.
- Cancelamento dos processos ativos em shutdown, timeout e falha ao publicar desafio.

## Fora do escopo

- Instalar `gh`, receber PAT/token/senha/cookie pelo painel ou usar `--with-token`.
- Usar `--insecure-storage`, `--show-token`, clipboard ou variavel de token.
- Copiar, ler ou persistir o token/arquivo `hosts.yml` no controle.
- Corrigir automaticamente credential store, apagar credencial ou deslogar conta existente.
- Clonar repositorio, configurar remote, criar branch/commit/PR, merge ou deploy.
- Executar provider de IA, habilitar API paga, extra usage, creditos ou fallback.

## Criterios de aceite

1. Login so pode ser solicitado quando o estado GitHub observado e `AUTH_REQUIRED`; uma sessao ativa e deduplicada.
2. Sessao e evento `github.onboarding.requested.v1` sao gravados atomicamente; outbox nao contem desafio nem segredo.
3. Worker usa binario/argv fixos, ambiente sem tokens, `shell: false`, limite de 64 KiB e janela maxima de dez minutos.
4. Desafio aceita somente HTTPS no host configurado, path `/login/device`, codigo limitado e expiracao dentro da sessao.
5. Redis indisponivel ou publicacao rejeitada mata o processo e falha fechado; PostgreSQL/log nao recebem URL, codigo ou output bruto.
6. Exit code zero nao basta: `CONNECTED` exige conta ativa `success` e `tokenSource` de credential store; `hosts.yml`, auth ausente ou JSON invalido nao promovem conexao.
7. Host alterado durante o login invalida o resultado sem sobrescrever a configuracao nova.
8. Shutdown cancela a arvore de processos; conclusao remove o desafio ou depende do TTL bounded se Redis estiver indisponivel.
9. Painel inicia e acompanha login sem coletar credencial; novo pedido fica desabilitado enquanto ha sessao ativa.
10. Lint, typecheck, testes, build, validacao Prisma e `git diff --check` passam.

## Baseline, caminhos e limites

- Base aceita e registro do FAC-011A: `eed7f85f564387ce6ed006e7cac4e46cd874ac30`.
- Branch/worktree: `feat/fac-011b-github-login`, `/home/vinicius/le-fabrique-fac-011-github-login`.
- Caminhos: `packages/contracts`, `apps/api`, `apps/worker`, `apps/web`, migration Prisma, configuracao/operacao e lesson pertinente.
- GitHub CLI observado: ausente; nenhuma instalacao ou autenticacao real sera feita neste ticket.
- Sessao/timeout: dez minutos; output: 64 KiB; um job por vez; nenhuma IA; budget adicional zero.
- Um writer, ate duas rodadas de correcao, sem merge/deploy/migration aplicada.

## Provider e fontes oficiais

GitHub CLI em modo `GH_CLI`. O fluxo web/device e o armazenamento pertencem ao cliente oficial. A documentacao informa que o credential store e preferido e que pode haver fallback para arquivo em texto simples; a La fabrique nao promovera `CONNECTED` nesse fallback.

- <https://cli.github.com/manual/gh_auth_login>
- <https://cli.github.com/manual/gh_auth_status>
- <https://cli.github.com/manual/gh_help_environment>
- <https://github.com/cli/cli/blob/trunk/internal/authflow/flow.go>

## Sequenciamento

FAC-011B implementa apenas login. Verificacao de permissao no repositorio e criacao de PR sob gate humano pertencem ao FAC-011C ou ticket equivalente; merge permanece manual.

## Revisao entregue

- Ticket READY: `da02b574ba60a03e136da18848b259f9c5207d15`.
- Codigo verificado: `c2f353db0befbaa5b8bbddc69430f739206c0b7a`.
- Relatorio: `documentacoes/configuracao/2026-10-01-FAC-011B-login-efemero-github.md`.
- Estado real: fixtures aprovadas; `gh` ausente, migration nao aplicada e login remoto nao executado.

## Aceite

Entrega aceita explicitamente pelo responsavel em 2026-10-02 na revisao documental exata `3ac8b39de9024576df5c5709022bb4a01e5ea6a5`.
