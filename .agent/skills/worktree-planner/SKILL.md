---
name: worktree-planner
description: Planeja a execução de tickets já especificados da La fabrique em frentes (api, web, infra), com base fixa, branches/worktrees quando houver paralelismo real, prompts prontos por frente e plano de integração em developer. Use depois do tech-lead; não cria worktrees, não implementa, não faz merge.
---

# Planejador de frentes e worktrees

Transforme tickets já especificados num plano operacional. **Não crie worktrees, não implemente,
não rode Git de escrita.** O usuário cria as worktrees (`git worktree add`) quando quiser
paralelismo; outra sessão executa cada prompt. Só `git fetch` é permitido, para registrar a base.

Reutilize o `00-plano` do tech-lead. Não refaça a entrevista. Lacuna essencial impede marcar o
prompt como pronto: registre o bloqueio.

## Regras do projeto que valem em todo prompt

- Commits na branch **`developer`**, **sem `Co-Authored-By`** nem atribuição ao agente, **ao final de
  cada tarefa**. Frente em branch/worktree separada é exceção (paralelismo real) e termina
  integrada em `developer`.
- Sem push, PR, merge em `main` ou deploy sem autorização explícita. Sem segredos em Git, log ou
  prompt. Portas padrão 5173/5432 estão ocupadas: use as do `.env` (web 5180, API 8480, Postgres 5442).
- Um writer por workspace. Código, docs e `docs/NEW MVP/07-CHECKLIST.md` mudam no mesmo commit.

## Topologia

Um único repositório (monorepo). As frentes são partes do mesmo repositório:

| Frente | Caminhos | Validação |
| --- | --- | --- |
| `api` (Go) | `cmd/`, `internal/`, `db/migrations/`, `contracts/` | `make lint`, `make test`, `make test-integration` |
| `web` (React/TS) | `apps/web/` | `make lint`, `make typecheck`, `make test-web`, build |
| `infra` (Docker) | `infra/`, `Makefile`, `scripts/` | `docker compose config`, subir o stack, `make ps` |

Confirme a raiz com `git rev-parse --show-toplevel`. Uma worktree tem a estrutura completa do
repositório.

## Base

- `git fetch origin` (se houver remoto) e `git rev-parse developer` (e `origin/developer`).
  Registre no manifesto e nos prompts a branch e o **SHA completo**.
- Tarefa independente nasce desse SHA. Tarefa que depende do código de outra nasce do commit final
  da tarefa de origem, que só existe depois que ela termina. Registre a dependência e diga que o
  prompt só pode começar então.
- Não inclua `git pull` nos prompts: mudaria a base planejada.

## Frentes

- **Padrão: uma frente sequencial em `developer`**, sem worktree. Só proponha worktrees paralelas
  com independência real: identifique arquivos, símbolos, migrations e contratos disputados antes.
  Nada de paralelismo fictício. Separação diferente pedida pelo usuário prevalece.
- No máximo uma worktree por frente por feature. Tickets da mesma feature e frente ficam no mesmo
  prompt.
- Branch de frente paralela: `feat|fix|refactor|chore/{chave}-{assunto}` (a mesma chave nas frentes
  da mesma feature). A frente aparece no nome do arquivo do prompt, não na branch.
- Migrations: só uma frente cria migration por vez (numeração sequencial); registre a ordem.

## Prompt por frente (`01-{chave}-{frente}-{assunto}.md`)

Texto pronto, sem placeholders pendentes. Cabeçalho: frente, branch, base (ref + SHA), tickets,
perfil de risco, caminho absoluto do manifesto e do `00-plano`, dependências.

Seção **"Prompt para o executor (Claude, Codex ou Antigravity)"**, nesta ordem:

1. **Conferência do workspace.** Antes de qualquer escrita, confira:
   - `git rev-parse --show-toplevel`;
   - `git branch --show-current`: `developer` (caso padrão) ou a branch do plano. Em worktree com
     outro nome, renomeie com `git branch -m {branch-do-plano}` só se o nome ainda não existir
     (nunca `-M`);
   - `git status --porcelain` vazio (ou só os arquivos do planejamento);
   - `git merge-base --is-ancestor {SHA-base} HEAD` com sucesso.

   Qualquer divergência: **pare e reporte**. Nunca escreva em outro checkout.
2. **Leituras.** `CLAUDE.md`/`AGENTS.md`, `docs/desenvolvimento/01-convencoes.md`, o `00-plano`
   (caminho absoluto) e `contracts/openapi.yaml` quando a frente toca a API.
3. **Preparação.** `.env` presente (`make bootstrap` se faltar); para a frente `web`,
   `cd apps/web && npm ci` sem alterar `package.json` nem o lockfile; para `api`/testes de banco,
   o stack de dev no ar (`make up`).
4. **Implementação.** Tickets e critérios `CA-xx` desta frente, por ID, com os arquivos prováveis e
   o que está fora do escopo. Acrescente só decisões operacionais desta frente; não copie a
   especificação.
5. **Registro de entrega e commit.** Registro em `docs/entregas/AAAA-MM-DD-TICKET-titulo.md`,
   `docs/NEW MVP/07-CHECKLIST.md` atualizado e docs/ADR/contrato afetados. Commit na branch
   combinada com tudo junto, **sem `Co-Authored-By`**. Sem push, sem PR, sem merge.
6. **Relatório final** no chat: SHA do commit, caminho do registro, validações executadas com
   resultado (PASS/FAIL/NOT_RUN), e o que ficou sem validar e por quê. Para cada critério `CA-xx`
   da frente: ATENDIDO, NÃO ATENDIDO ou NÃO VALIDADO, com a evidência.

Depois, uma seção **"Prompt de revisão (sessão nova)"**, em poucas linhas: use a skill
`revisor-de-codigo` na branch da frente, com a base `{SHA}` e a spec `{caminho do 00-plano}`.

## Integração (`99-integracao-{slug}.md`)

Só existe quando há worktrees paralelas. Ações futuras, executadas só quando o usuário pedir:

1. No checkout principal, com `git status` limpo e na branch `developer`.
2. Merge `--no-ff` das branches das frentes, na ordem de dependência do manifesto. Conflito:
   **pare** e reporte. Não resolva por conta própria se a resolução mudar comportamento.
   Migrations conflitantes: renumere só se nenhuma foi aplicada em outro ambiente; senão, pare.
3. Validação após a integração: `make lint`, `make test`, `make test-integration` e, se tocou o
   web, o build. Resultados reais.
4. Commit do merge sem `Co-Authored-By`. Sem push.
5. Limpeza: remover uma worktree (`git worktree remove`) só depois de confirmar que a branch dela
   está integrada (`git branch --merged developer`) e que a worktree está limpa. Não apague
   branches.

## Conferência antes de devolver

Todo prompt tem caminhos reais, branch, base com SHA, critérios verificáveis, contratos entre as
frentes, validação viável, registro, commit e fora do escopo. Nenhum prompt pede push/PR, apaga
branch, atualiza a base em silêncio, adiciona co-autoria ou depende de ferramenta exclusiva de um
cliente.

## Saída

- **Chamado pelo planejador:** devolva os caminhos dos prompts e da integração, a ordem e os
  bloqueios.
- **Chamado direto:** entregue o mesmo no chat, a partir da especificação indicada, e termine no
  plano. Não inicie a execução.
