# La fabrique

Painel pessoal que gerencia a execução de desenvolvimento assistido por IA em projetos de software (Jira/ClickUp → plano → execução isolada → Draft PR no GitHub). Especificação completa em [docs/NEW MVP](docs/NEW%20MVP/00-LEIA-ME.md); progresso em [07-CHECKLIST.md](docs/NEW%20MVP/07-CHECKLIST.md).

**Stack:** Go (API + worker) · React/TypeScript/Vite · PostgreSQL · Docker Compose.

## Começar (dev)

Requisitos: Docker, Go 1.27, Node 24, make.

```sh
make bootstrap   # cria .env e instala dependências
make up          # sobe postgres, api, worker e web (hot reload)
make seed        # projetos fictícios (opcional)
```

Crie a conta única com `make create-admin` (usuário + senha de 12+ caracteres) e abra <http://127.0.0.1:5180>: no primeiro login você cadastra o app autenticador (TOTP) e guarda os códigos de recuperação. Não existe mais acesso sem senha. Portas (configuráveis no `.env`): web 5180 · API 8480 · Postgres 5442 — as padrão 5173/5432 estão ocupadas nesta máquina.

```sh
make lint test            # qualidade (sem banco)
make test-integration     # com PostgreSQL real (precisa do make up)
make down                 # derruba o stack
```

## VPS

`cp .env.production.example .env`, preencha `POSTGRES_PASSWORD` e rode `make up-prod`. Só o web é publicado. Detalhes em [docs/operacao/ambientes-docker.md](docs/operacao/ambientes-docker.md).

## Estrutura

```text
apps/web/        painel React        cmd/{api,worker,migrate}/   binários Go
internal/        domínio e infra     db/migrations/              SQL versionado
contracts/       OpenAPI             infra/{compose,images,proxy}/  Docker
tests/           contrato, integração, fixtures      skills/      skills de agentes
docs/            manual, ADRs, operação, spec do MVP
```

Para agentes de IA: [CLAUDE.md](CLAUDE.md) · [AGENTS.md](AGENTS.md) · [ANTIGRAVITY.md](ANTIGRAVITY.md). Índice: [docs/README.md](docs/README.md).
