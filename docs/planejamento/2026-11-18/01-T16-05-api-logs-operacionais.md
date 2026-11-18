# Prompt — T16-05, frente `api`

- **Frente:** `api` (Go — `cmd/`, `internal/`, `db/migrations/`, `contracts/`)
- **Branch:** `developer` (sem worktree — frente única sequencial)
- **Base:** `developer` @ `82857c0da3ee7408e7f3a6367fb3ffbe6442fdc7`
- **Tickets:** T16-05 (parte `api`)
- **Perfil de risco:** crítico (migration + dado novo exposto por API autenticada)
- **Manifesto:** `C:\Users\vinicius.silva\Documents\Documentos\Projetos Dev\la_fabriquev2\docs\planejamento\2026-11-18\lote-2026-11-18-painel-operacao-fabrica.md`
- **Especificação canônica:** `C:\Users\vinicius.silva\Documents\Documentos\Projetos Dev\la_fabriquev2\docs\planejamento\2026-11-18\00-plano-T16-05-painel-operacao-fabrica.md`
- **Dependências:** nenhuma (nasce direto da base acima). A frente `web` deste mesmo ticket depende
  do commit desta frente.

## Prompt para o executor (Claude, Codex ou Antigravity)

1. **Conferência do workspace.** Antes de qualquer escrita:
   - `git rev-parse --show-toplevel`
   - `git branch --show-current` → deve ser `developer`
   - `git status --porcelain` → vazio (ou só arquivos deste planejamento)
   - `git merge-base --is-ancestor 82857c0da3ee7408e7f3a6367fb3ffbe6442fdc7 HEAD` → sucesso

   Qualquer divergência: **pare e reporte**. Não escreva em outro checkout.

2. **Leituras obrigatórias.** `CLAUDE.md`, `docs/desenvolvimento/01-convencoes.md`, o `00-plano`
   acima (caminho absoluto, leia inteiro) e `contracts/openapi.yaml` (seções `operacao-ao-vivo` e
   os schemas `ExecutionEvents`/`ExecutionEvent` como modelo).

3. **Preparação.** `.env` presente (`make bootstrap` se faltar). Stack de dev no ar (`make up`) para
   `make test-integration`.

4. **Implementação.** Tickets e critérios desta frente: **CA-01 a CA-09 e CA-12** do `00-plano`.
   Nesta ordem (cada etapa é committável, mas o commit final é um só para a frente inteira):

   a. **Migration** `db/migrations/000NN_operational_events.sql` (NN = próximo número livre; rode
      `make migrate-status` para confirmar — hoje a última aplicada é `00072`). Tabela
      `operational_events` com `id bigserial PK`, `level text NOT NULL CHECK (level IN
      ('info','warn','error'))`, `source text NOT NULL CHECK (source IN
      ('sentinela','admissao','retencao','sincronizacao','clilogin','fabrica'))`, `message text
      NOT NULL`, `payload jsonb NOT NULL DEFAULT '{}'::jsonb`, `created_at timestamptz NOT NULL
      DEFAULT now()`. Índices em `created_at` e `source`. Siga o padrão de
      `db/migrations/00043_sentinel_routines_events.sql` (comentário `+goose Up`/`+goose Down`,
      `DROP TABLE` no Down). Rode `make migrate`.

   b. **Pacote `internal/opslog`** (novo): `func Emit(ctx context.Context, db
      persistence.DBTX, level, source, message string, payload map[string]any) error` — insere uma
      linha. Valide `level`/`source` contra os enums da seção "Regras de negócio" do `00-plano`
      (erro se fora do enum — é bug de chamada, não entrada do usuário). `payload` vira JSON
      (`nil`/vazio → `{}`). Teste unitário simples (banco de teste) cobrindo insert e enum inválido.

   c. **Wiring no worker**, exatamente como a tabela de mapeamento da seção "Regras de negócio" do
      `00-plano` descreve — não redescubra o mapeamento, use o que já está especificado:
      - `internal/worker/jobs.go`, função `exec`: depois de atualizar `j.status` (runs, lastRun,
        lastNote, lastError), chame `opslog.Emit` com nível/origem conforme a tabela. Gate de
        emissão: erro → sempre emite (nível `error`); sem erro e `note != ""` → emite (nível
        `info`); sem erro e `note == ""` → não emite. Se `Emit` falhar, logue um aviso (`w.log`) e
        **não** propague — o retorno de `exec` continua o mesmo de hoje.
      - `internal/sentinel/routines.go`, função `RunOne`: nos dois branches que já atualizam
        `sentinel_routine_state` (sucesso e falha, ambos dentro do `if tag.RowsAffected() > 0`),
        chame `opslog.Emit` com a origem mapeada pelo nome da rotina. Mesmo cuidado: falha ao
        gravar o evento não muda o `Result` devolvido por `RunOne`.
      - `cmd/worker/factory.go`, linha do `log.Error("proxy de saída indisponível...")`: logo
        depois dessa chamada (o `pool` já existe neste ponto do boot), chame `opslog.Emit(ctx, pool,
        "error", "fabrica", <mesma mensagem>, map[string]any{"error": err.Error()})`. Use um
        contexto de timeout curto (ex.: reaproveite `epCtx`/um novo `context.WithTimeout` antes de
        `cancelEP()`), já que isso roda uma vez no boot, não dentro de um job com contexto próprio.

   d. **Extensão da retenção** (`internal/retention/retention.go`, método `Run`): apague
      `operational_events` com `created_at < logCut` (mesmo `logCut` já calculado para
      `execution_events`). Sem permanent-summary novo: registre uma `Decision{Kind:
      "operational_events", Ref: source, Action: deleteWord, Reason: "<N> evento(s) com mais de
      <LogDays> dias"}` por origem presente (agrupe por `source` antes de apagar, para contar
      certo). Em `dryRun`, só conte e registre `would_delete`, sem `DELETE`. Estenda o teste
      existente de `Run` (veja o arquivo de teste do pacote) com linhas antigas e recentes em
      `operational_events` e confira a contagem e a `Decision`.

   e. **Endpoints**, novo arquivo `internal/httpapi/operations.go` (siga o estilo de
      `internal/httpapi/live.go`, reaproveite `cursorOf` de lá):
      - `GET /api/operations/events`: valida `after` (cursor), `level`/`source` (enum — `400
        invalid_request` se inválido), `q` (≤100 chars), `limit` (1–500, default 200). Query em
        `operational_events` com os mesmos filtros opcionais (`AND ($x = '' OR source = $x)` etc.).
        Aplique `s.redactor.RedactBytes` no `payload` antes de responder, igual
        `fetchEvents`/`live.go` faz. Resposta: `{"items":[...], "next_cursor": <int64>}`.
      - `GET /api/operations/stream`: mesmo padrão de `executionStream` (heartbeat, `StreamMaxAge`,
        `event: reconnect`, `Last-Event-ID` tem prioridade sobre `?after=`), sem filtro de
        nível/origem no servidor (só no histórico). Registre as duas rotas em
        `internal/httpapi/server.go` com `s.protected(...)`, perto das rotas de
        `/projects/{id}/executions/...`.

   f. **Contrato:** acrescente os dois paths em `contracts/openapi.yaml` (tag `operacao-ao-vivo`,
      mesmo formato dos paths de execução) e os schemas `OperationalEvent`/`OperationalEvents`
      (mesmo molde de `ExecutionEvent`/`ExecutionEvents`, com `level`/`source` a mais). Rode `make
      matriz` depois de tocar o checklist (próximo passo).

   g. **Fora do escopo desta frente:** a tela web (frente `web`, prompt separado); qualquer ação
      sobre o evento; gravação de eventos pela própria API; logs do binário `egressproxy`; qualquer
      erro de boot anterior à conexão com o Postgres.

5. **Registro de entrega e commit.**
   - Crie `docs/entregas/2026-11-18-T16-05-logs-operacionais-api.md` com o que foi feito, os
     critérios atendidos (CA-01 a CA-09, CA-12) e a evidência de cada um (saída de teste, não
     narrativa).
   - Acrescente `T16-05` em `docs/NEW MVP/03-BACKLOG.md` (ao lado de T16-01..T16-04, mesmo formato)
     e a linha correspondente em `docs/NEW MVP/07-CHECKLIST.md` (marque `[x]` só a parte `api`, ou
     `[~]` se preferir esperar a frente `web` para marcar o ticket inteiro — registre a nota).
   - Rode `make matriz` depois de editar o checklist.
   - Commit único na branch `developer`, **sem `Co-Authored-By`** nem qualquer atribuição ao
     agente, com código + migration + docs + checklist juntos.
   - Sem push, sem PR, sem merge em `main`.

6. **Relatório final no chat:** SHA do commit, caminho do registro de entrega, validações
   executadas (`make lint`, `make test`, `make test-integration`) com resultado real
   (PASS/FAIL/NOT_RUN), e para cada critério desta frente (CA-01 a CA-09, CA-12): ATENDIDO, NÃO
   ATENDIDO ou NÃO VALIDADO, com a evidência.

## Prompt de revisão (sessão nova)

Use a skill `revisor-de-codigo` na branch `developer`, com a base `82857c0da3ee7408e7f3a6367fb3ffbe6442fdc7`
e a especificação `C:\Users\vinicius.silva\Documents\Documentos\Projetos Dev\la_fabriquev2\docs\planejamento\2026-11-18\00-plano-T16-05-painel-operacao-fabrica.md`
(critérios CA-01 a CA-09 e CA-12).
