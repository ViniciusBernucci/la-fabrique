# Skills

Fonte única das skills deste repositório (cada pasta com `SKILL.md`, opcionalmente `agents/` e `references/`):
`grill-me`, `tech-lead`, `worktree-planner`, `planejador-de-lote-v1`, `revisor-de-codigo`. Origem: `docs/NEW MVP/new skills/` (versão original, preservada); as de `skills/` são as adaptadas e valem.

| Agente | Pasta de descoberta no projeto |
|---|---|
| Claude Code | `.claude/skills/` |
| Codex e Antigravity (App/CLI/IDE) | `.agents/skills/` |
| Antigravity (legado) | `.agent/skills/` |

Edite só `skills/`; depois `make sync-skills` (ou `node scripts/sincronizar-skills.mjs --check` para só verificar; retorna 1 se houver divergência). O script nunca escreve fora do repositório nem apaga arquivos extras.

**Adaptadas a este projeto:** as skills originais eram da Intranet (Laravel/Angular, ORCA). Hoje assumem monorepo Go + React, tickets `Txx-yy` do backlog, validação por `make lint/test/test-integration`, planos em `docs/planejamento/AAAA-MM-DD/`, commits na branch `developer` sem `Co-Authored-By` e worktrees só para paralelismo real.

Uso em outro projeto ou global: copie a pasta de descoberta correspondente (ex.: `~/.claude/skills/`, `~/.agents/skills/`). Instalação global não foi feita aqui.
