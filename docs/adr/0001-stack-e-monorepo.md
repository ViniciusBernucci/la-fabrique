# ADR-0001 — Stack inicial, PostgreSQL e monorepo

- **Estado:** ACEITA (decisão do responsável, 2026-11-12, UTC-03:00)
- **Ticket:** [T01-01](../NEW%20MVP/03-BACKLOG.md#t01-01) · **Épico:** [E01](../NEW%20MVP/02-EPICOS.md#e01) · **Decisão de origem:** D40 ([matriz](../NEW%20MVP/04-RASTREABILIDADE.md))
- **Substitui:** ADR-003 da stack anterior (TypeScript/NestJS/Redis), arquivada em [99-historico/stack-anterior](../99-historico/stack-anterior/06-decisoes/ADR-003-stack-typescript.md). A decisão anterior foi superada por escolha explícita do responsável; o histórico foi preservado, não apagado.

## Contexto

A fábrica é um painel pessoal que gerencia execuções de desenvolvimento assistido por IA numa VPS existente, com um administrador. Precisa de estado durável e transacional, processos de longa duração supervisionados, isolamento de execução e baixo consumo de memória (a VPS é compartilhada e a capacidade ainda não foi medida — [T02-01](../NEW%20MVP/03-BACKLOG.md#t02-01)).

## Decisão

| Tema | Escolha |
|---|---|
| Web | React 19 + TypeScript + Vite; React Flow no editor de workflows (E07) |
| API e worker | Go (`cmd/api`, `cmd/worker`), compartilhando módulos em `internal/` |
| Migrações | `cmd/migrate` (goose, SQL embutido), com identidade própria |
| Banco | PostgreSQL 17 como único armazenamento transacional: estado, fila durável, eventos, outbox e auditoria. **Sem Redis/fila externa** |
| Ambientes | Docker Compose em dev e na VPS (mesmos arquivos base) — ver [ADR-0002](0002-docker-compose-dev-e-vps.md) |
| Repositório | **Monorepo** privado para web, API, worker, contratos, migrations e infraestrutura |

### Monorepo × repositórios separados

| Critério | Monorepo (escolhido) | Repositórios separados |
|---|---|---|
| Mudança que cruza web, API, worker e schema | Uma PR, um review, um commit de referência | Várias PRs coordenadas; risco de versões incompatíveis |
| Contratos (OpenAPI/schemas) | Uma fonte, testada contra a implementação | Publicação/versionamento entre repos |
| Imagens | API e worker publicados da mesma revisão | Combinação de versões a gerenciar |
| Custo | CI roda tudo (mitigar com filtros por caminho) | CI menor por repo, mas mais repos para administrar |
| Equipe | Um administrador: sem ganho de autonomia em separar | — |

Os repositórios **dos projetos atendidos** continuam independentes; a fábrica só os vincula (E05).

### Fronteiras

- **Web:** só consome a API autenticada; nenhum segredo ou lógica privilegiada no navegador. Validação visual é complementada pela do servidor.
- **API:** configuração, autenticação, catálogo, comandos e leitura de estado. **Não** executa builds, CLIs de IA nem checks.
- **Worker:** reclama trabalho, mantém um escritor por projeto, prepara ambientes isolados e recupera tarefas. É o único executor.
- **Sandbox (E11):** container efêmero por feature; sem acesso a banco, rede de controle ou socket do runtime.
- **Banco/volumes:** inacessíveis ao código executado no sandbox.

## Consequências

- Binários Go estáticos e pequenos reduzem memória e simplificam imagens finais (alpine, usuário sem privilégios, rootfs somente leitura).
- Concentrar fila e transações no PostgreSQL elimina um serviço, ao custo de a fila (E10) ser implementada com `SELECT … FOR UPDATE SKIP LOCKED`/leases e testada sob concorrência.
- O time precisa manter duas toolchains (Go e Node); versões fixas em `go.mod`, `apps/web/package.json`/lock e nos Dockerfiles.

## Critérios de viabilidade e vínculo ao benchmark (D40)

A escolha é **sujeita à medição** do [T02-01](../NEW%20MVP/03-BACKLOG.md#t02-01) (CPU, RAM, I/O, builds e navegador na VPS). Estado: **NOT_RUN**. A stack é reavaliada se, com carga representativa, painel e banco não preservarem folga operacional. Qualquer revisão vira nova ADR.

## Decisões delegadas e responsáveis

| Decisão | Responsável | Estado |
|---|---|---|
| Toolchain, versões e libs | Responsável | Fixadas neste ticket |
| Limites de memória iniciais dos containers | Responsável, após T02-01 | Valores provisórios **não medidos** em `.env.example` |
| Parâmetros de fila (janela de envelhecimento, timeouts) | Responsável, no E10 | PLANEJADO |
| Plano do GitHub e proteção de `main` | Responsável | Ver [github.md](../operacao/github.md) — NOT_RUN |
