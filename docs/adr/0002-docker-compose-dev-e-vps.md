# ADR-0002 — Docker Compose em desenvolvimento e na VPS

- **Estado:** ACEITA (2026-11-12)
- **Ticket:** [T02-02](../NEW%20MVP/03-BACKLOG.md#t02-02) (escopo reduzido: ambientes dev e prod em Docker) · **Épico:** E02

## Contexto

O responsável alterna entre a máquina local e a VPS. As portas padrão do front (5173) e do PostgreSQL (5432) já estão em uso na máquina local, e a VPS tem serviços vizinhos. É preciso o mesmo mecanismo nos dois lugares, sem expor mais do que o necessário.

## Decisão

- Um arquivo base (`infra/compose/compose.yaml`) com `postgres`, `migrate` (one-shot), `api`, `worker` e `web`, mais dois overlays: `compose.dev.yaml` (hot reload, portas em loopback) e `compose.prod.yaml` (só o `web` publicado). Sempre usados por `scripts/compose.sh dev|prod` (ou `make`).
- **Portas do host configuráveis** e fora das padrão: web `5180`, API `8480` (só dev), PostgreSQL `5442` (só dev). Em produção apenas `WEB_BIND:WEB_PORT` (padrão `127.0.0.1:8280`) é publicado.
- **Redes:** `private` (`internal: true`, sem saída) para banco e tráfego interno; `edge` para o que publica porta; `egress` para api/worker alcançarem APIs externas futuras (GitHub, Jira, ClickUp, provedores). O Postgres só entra em `edge` no overlay de dev, porque o Docker não publica portas de containers em redes internas.
- **Hardening padrão:** `cap_drop: ALL`, `no-new-privileges`, rootfs somente leitura, `tmpfs` para `/tmp`, usuário não-root (10001) em api/worker, `mem_limit`/`pids_limit`, healthchecks e `restart: unless-stopped`.
- **Segredos** só no `.env` (ignorado pelo Git; modelos `.env.example` e `.env.production.example`). O backend exige `VAULT_KEY` (chave do cofre/TOTP, fora do banco), cookie `Secure` e HTTPS (`REQUIRE_HTTPS`) em produção; o acesso dev sem senha foi removido ([ADR-0005](0005-autenticacao-sessoes-e-cofre-de-conexoes.md)).
- **Homologação:** suportada por `COMPOSE_PROJECT_NAME` e arquivo de ambiente próprios (volumes e redes nascem com o prefixo do projeto), mas **não exercitada** neste ticket.
- TLS fica no proxy do host (o `web` escuta HTTP interno em 8080); ver [ambientes-docker.md](../operacao/ambientes-docker.md).

## Consequências

- `make up` reproduz o ambiente em qualquer máquina com Docker; `make up-prod` faz o mesmo na VPS com imagens finais.
- Limites de memória são **provisórios** até o benchmark (T02-01).
- O dev usa `user: 0:0` e bind mount do código nos containers `api`/`worker`/`web` (hot reload); produção não.
- Sem prova ainda de reboot da VPS (depende do `docker` habilitado no boot) nem de TLS — ver pendências no [status](../desenvolvimento/02-status-backlog.md).
