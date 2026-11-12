# Entrega 2026-11-12 — Skills adaptadas e regra de commits

- **Skills:** removidas as referências à Intranet (Laravel/Angular, ORCA, `docker-intranet`, `.claude/rules`, SQL Server, ClickUp como fonte). Passam a assumir monorepo Go + React, tickets `Txx-yy`, `make lint/test/test-integration`, planos em `docs/planejamento/AAAA-MM-DD/`, checklist de risco para Go/PostgreSQL/React/Docker, e worktrees só para paralelismo real. Original preservado em `docs/NEW MVP/new skills/`.
- **Regra de commits** (decisão do responsável): sempre em `developer`, sem `Co-Authored-By`, ao final de cada tarefa. Registrada nos 3 guias de agentes e nas skills.
- **Autoridade:** a decisão atual do responsável prevalece sobre políticas/ADRs arquivados; registrado nos guias.
- **Verificação:** `node scripts/sincronizar-skills.mjs --check` PASS; busca por termos da Intranet nas skills sem ocorrências (restam só ClickUp/Jira como integrações do produto). Revisão independente: NOT_RUN.
