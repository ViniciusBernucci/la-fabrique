# Guia do projeto — ANTIGRAVITY.md

Ponto de entrada do Antigravity (App, CLI e IDE). Skills em `.agents/skills/` (padrão) e `.agent/skills/` (compatibilidade legada). Confirme na versão instalada quais arquivos de instrução são lidos; se este não for, abra-o explicitamente no contexto.

## Antes de qualquer trabalho

1. Leia [docs/README.md](docs/README.md), o ticket em [docs/NEW MVP/03-BACKLOG.md](docs/NEW%20MVP/03-BACKLOG.md), o épico em [02-EPICOS.md](docs/NEW%20MVP/02-EPICOS.md) e a [arquitetura](docs/NEW%20MVP/01-ARQUITETURA.md). Links não carregam conteúdo sozinhos: abra os arquivos.
2. Acompanhe o estado em [07-CHECKLIST.md](docs/NEW%20MVP/07-CHECKLIST.md) e **atualize-o** ao concluir ou alterar um ticket (marque `[x]` só com evidência; `[~]` para parcial, com nota).
3. Convenções (PR, revisão, migrations, logs, docs, evidências): [docs/desenvolvimento/01-convencoes.md](docs/desenvolvimento/01-convencoes.md). Decisões: [docs/adr](docs/adr).

## Stack (aprovada; não reabrir sem nova ADR)

Go (API `cmd/api`, worker `cmd/worker`, `cmd/migrate`) · React 19 + TypeScript + Vite (`apps/web`) · PostgreSQL 17 · Docker Compose em dev e VPS. Sem Redis. A stack antiga (NestJS/Node) está só em `docs/99-historico/stack-anterior/` e **não** vale como regra.

## Regras

- **Autoridade:** a decisão atual do responsável prevalece sobre qualquer política, ADR ou documento arquivado (inclusive `docs/99-historico/`). Em conflito, siga o responsável e registre a mudança em ADR.
- **Commits:** sempre na branch **`developer`**, **sem `Co-Authored-By`** e sem qualquer atribuição ao agente (nem em PRs). Faça o commit **ao final de cada tarefa**, com código, docs e checklist juntos. Isto vale também para commits/PRs que ferramentas ofereçam atribuição automática: remova-a. Push, PR e merge em `main` continuam exigindo autorização.
- **Portas:** as padrão (5173, 5432) estão ocupadas na máquina do responsável. Use as do `.env`: web 5180, API 8480, Postgres 5442. Nunca fixe portas no código.
- **Segredos:** nunca em Git, logs, prompts ou testes. Só `.env` (ignorado). Rode `make check-secrets`.
- **Um writer por workspace.** Sem prova de término de outro agente, não escreva no mesmo diretório.
- **Sem merge, push, deploy, mudança de plano/gastos ou acesso a credenciais** sem autorização explícita do responsável.
- **Verifique de verdade:** rode `make lint`, `make test` e, quando tocar o banco, `make test-integration`. Relate resultados reais (PASS/FAIL/NOT_RUN); exit code ou fala do modelo não provam sucesso.
- **Código + docs + checklist no mesmo diff.** Contrato de API em `contracts/openapi.yaml` (há teste de contrato). Mudança de schema = nova migration em `db/migrations` (nunca editar uma já aplicada).
- **Front:** siga o design estabelecido (`apps/web/src/styles/tokens.css`, componentes de `components/`); só tokens `--hud-*`, nunca cores fixas. O menu e o painel crescem com os épicos; não invente dados de demonstração.
- **Acesso dev provisório** (`DEV_AUTH_ENABLED`, [ADR-0004](docs/adr/0004-sessao-no-servidor-e-acesso-dev-provisorio.md)) deve sumir com o T04-01.

## Skills

Fonte única em [`skills/`](skills/README.md): `grill-me`, `tech-lead`, `worktree-planner`, `planejador-de-lote-v1`, `revisor-de-codigo`. Copiadas por `make sync-skills` para `.claude/skills`, `.agents/skills` (Codex e Antigravity) e `.agent/skills` (Antigravity legado). As skills foram adaptadas a este projeto (monorepo Go + React, tickets `Txx-yy`, planos em `docs/planejamento/AAAA-MM-DD/`). Nenhuma skill autoriza subagentes, gastos, merge, push ou deploy.

## Comandos

`make help` lista tudo. Principais: `make bootstrap`, `make up`, `make lint`, `make test`, `make test-integration`, `make seed`, `make db-backup`.
