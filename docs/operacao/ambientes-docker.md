# Ambientes Docker (dev e VPS)

Decisão: [ADR-0002](../adr/0002-docker-compose-dev-e-vps.md). Tudo roda por `scripts/compose.sh dev|prod` (ou `make`), a partir da raiz.

## Portas (host)
| Serviço | Dev | VPS |
|---|---|---|
| Web | `127.0.0.1:5180` | `${WEB_BIND}:${WEB_PORT}` (padrão `127.0.0.1:8280`) |
| API | `127.0.0.1:8480` | **não publicada** (via `/api` do web) |
| PostgreSQL | `127.0.0.1:5442` | **não publicado** |
| Worker | — (saúde interna `:8481`) | — |

Mude no `.env` se houver conflito. As padrão 5173/5432 estão ocupadas na máquina de dev.

## Dev
```sh
make bootstrap && make up     # http://127.0.0.1:5180
make logs S=api               # logs de um serviço
make down                     # mantém volumes; para apagar dados: scripts/compose.sh dev down -v
```
O código é montado nos containers (hot reload da API/worker via `air` e do front via Vite). O `.env` não é versionado.

## VPS
1. Instale Docker e habilite no boot (`systemctl enable docker`); os serviços têm `restart: unless-stopped`.
2. `git clone` e `cp .env.production.example .env`. Preencha `POSTGRES_PASSWORD` (`openssl rand -hex 24`) e `VAULT_KEY` (`openssl rand -base64 32`; **guarde uma cópia fora da VPS**, ver [backup](backup-restauracao.md)). Mantenha `APP_ENV=production`, `SESSION_COOKIE_SECURE=true`, `REQUIRE_HTTPS=true`. Depois do primeiro `make up-prod`, crie a conta: `make create-admin` (aponta para o banco via `DB_APP_URL`; rode na VPS com o `.env` dela) e faça o primeiro login para cadastrar o 2FA. **GitHub:** crie um OAuth App com Device Flow, ponha o Client ID em `GITHUB_OAUTH_CLIENT_ID` e, em Conexões, clique em "Conectar com GitHub" (você autoriza no seu navegador; não precisa de `gh` nem de navegador na VPS). Jira e ClickUp: token de API no formulário.
3. `make up-prod`. Confira `make ps-prod` (todos `healthy`; `migrate` sai com 0).
4. **TLS:** o `web` escuta HTTP interno; coloque na frente o proxy TLS que a VPS já usa apontando para `WEB_BIND:WEB_PORT`, repassando `Host`, `X-Forwarded-For` e `X-Forwarded-Proto: https` (a API recusa requisições sem HTTPS em produção e usa o `Host` para checar a origem). Não use `WEB_BIND=0.0.0.0` sem TLS.
5. Atualizar: `git pull && make up-prod` (migrations rodam antes da API). Rollback: voltar o commit e `make up-prod`; migrations não são desfeitas automaticamente (ver [ADR-0003](../adr/0003-persistencia-pgx-goose-e-escopo-por-projeto.md)).
6. Homologação na mesma VPS: outro `.env` (`ENV_FILE=.env.hml`) com `COMPOSE_PROJECT_NAME` e portas próprios; volumes e redes nascem separados. **NOT_RUN.**

## Verificado em 2026-11-12 (macOS, Docker 29.3.1)
PASS: stack dev sobe `healthy`; stack prod: só o web publicado, Postgres inacessível do host, api/worker `uid 10001`, rootfs somente leitura, `cap_drop ALL`, cabeçalhos de segurança, `readyz` cai para 503 com o banco parado e volta a 200, `DEV_AUTH_ENABLED` recusada (acesso dev removido no T04-01). **NOT_RUN:** reboot da VPS, TLS, homologação, limites medidos (T02-01).

**Verificado no T04 (2026-11-12, macOS, Docker 29.3.1):** stack dev reconstruído e `healthy` com `VAULT_KEY`; migrations 3 e 4 aplicadas; rotas de dados respondem 401 sem sessão e `POST /api/dev/session` 404 (PASS). `nginx -t` com `set_real_ip_from` na imagem de produção: PASS. **NOT_RUN:** ensaio completo do stack de produção após o E04, TLS real e IP real de cliente atrás de um proxy do host.

## Repositórios dos projetos (ADR-0061)
A API e o worker montam `/workspaces` (`WORKSPACES_DIR`): é onde a fábrica clona, abre e cria os repositórios. Em dev é a pasta `./workspaces` do repositório (mude com `WORKSPACES_HOST_DIR`); na VPS é um volume Docker (`workspaces`) pertencente ao usuário da imagem. Para usar uma pasta que já existe no seu computador, ponha-a (ou aponte `WORKSPACES_HOST_DIR` para a pasta que a contém) e escolha-a em "Pasta existente". Depois de mudar o Dockerfile ou o compose, rode `make up` (reconstrói as imagens; agora elas incluem o `git`).
