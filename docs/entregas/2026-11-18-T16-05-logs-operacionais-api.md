# T16-05 (frente `api`) — Consolidar logs operacionais da fábrica

- **Planejamento:** [docs/planejamento/2026-11-18/00-plano-T16-05-painel-operacao-fabrica.md](../planejamento/2026-11-18/00-plano-T16-05-painel-operacao-fabrica.md)
- **Prompt executado:** [01-T16-05-api-logs-operacionais.md](../planejamento/2026-11-18/01-T16-05-api-logs-operacionais.md)
- **Base:** `developer` @ `82857c0da3ee7408e7f3a6367fb3ffbe6442fdc7`

## O que foi feito

1. **Migration** `db/migrations/00073_operational_events.sql`: tabela `operational_events`
   (nível, origem, mensagem, payload, data), com índices em `created_at` e `source`. Também
   estende `retention_runs` (coluna `operational_events_deleted`) e o `CHECK` de
   `retention_decisions.kind` para aceitar `operational_events`, reaproveitando as mesmas tabelas
   de auditoria do T16-04 em vez de criar um resumo por item novo (decisão D10 do `00-plano`,
   hipótese do planejamento — confirmada nesta implementação).
2. **Pacote `internal/opslog`** (novo): `Emit(ctx, db, level, source, message, payload)`, com
   validação dos enums de nível/origem.
3. **Wiring nos dois pontos únicos de execução periódica** (conforme a tabela de mapeamento do
   `00-plano`, sem redescobrir nada):
   - `internal/worker/jobs.go` (`exec`): emite em toda falha (nível `error`) e em todo sucesso com
     nota (nível `info`). Precisou de um ajuste pequeno não previsto no plano: `Worker.db` só era
     `Pinger` (sem acesso a consulta); `internal/worker/worker.go` agora detecta via type-assert
     se `db` também implementa `persistence.DBTX` e guarda essa referência (`Worker.ops`) — testes
     com um `Pinger` simples (como `worker_test.go` já tinha) continuam funcionando sem gravar nada.
   - `internal/sentinel/routines.go` (`RunOne`): emite nos dois branches que já atualizam
     `sentinel_routine_state` (sucesso e falha). Acrescentei um campo opcional `Scheduler.Log` só
     para avisar (sem quebrar nada) se a gravação do evento falhar; `cmd/worker/main.go` passa o
     logger do worker.
4. **Erro de boot do proxy de egress** (`cmd/worker/factory.go`): extraído para
   `reportEgressProxyUnavailable(ctx, db persistence.DBTX, log, cause)` — mesma mensagem de sempre
   no log, mais o evento operacional `error`/`fabrica`. A extração usa `persistence.DBTX` (não
   `*pgxpool.Pool`) de propósito, para o teste não precisar de banco de verdade.
5. **Endpoints**: `internal/httpapi/operational_events.go` (nome diferente do sugerido no prompt —
   `operations.go` **já existia** no repositório com outro conteúdo inteiro, delivery/memory/gitops;
   usei `operational_events.go` para não colidir). `GET /api/operations/events` (histórico com
   cursor, filtro `level`/`source`, busca `q`) e `GET /api/operations/stream` (SSE, reconexão por
   cursor), registrados em `server.go` com `s.protected(...)`. Redação de segredos via
   `s.redactor.RedactBytes`, mesmo padrão de `live.go`.
6. **Contrato**: `contracts/openapi.yaml` — paths `/operations/events` e `/operations/stream`
   (tag `operacao-ao-vivo`) e schemas `OperationalEvent`/`OperationalEvents`.
7. **Retenção** (`internal/retention/retention.go`, `Run`): apaga `operational_events` mais velhos
   que `LogDays`, com uma `Decision` por origem (`kind: "operational_events"`) e um novo campo
   `Report.OperationalEventsDeleted`, persistido em `retention_runs`.
8. **Backlog/checklist**: `T16-05` acrescentado a `docs/NEW MVP/03-BACKLOG.md` (ao lado de
   T16-01..T16-04) e a `docs/NEW MVP/07-CHECKLIST.md` (`[~]`, pendente a frente `web`). `make
   matriz` rodado com sucesso.

## Achado durante a implementação (fora do escopo do ticket, corrigido)

`docs/NEW MVP/03-BACKLOG.md` e `docs/NEW MVP/07-CHECKLIST.md` estavam com terminação de linha CRLF
nesta checkout (ambiente Windows). O parser determinístico de `internal/acceptance/matrix.go` usa
regex ancorada em `$` por linha; a `\r` sobrando fazia **toda** âncora de ticket (`### Txx-yy`)
falhar, e `make matriz`/`TestTraceabilityCloses` reportavam incorretamente que praticamente todo o
backlog estava quebrado. Normalizei os dois arquivos para LF (conteúdo idêntico, só a terminação de
linha) — isso por si só já teria corrigido falhas pré-existentes não relacionadas ao T16-05. Os
arquivos Go que esta entrega tocou (mas não criou do zero) também tinham CRLF herdado do arquivo
original; normalizei para LF só os que editei, para não deixar terminação mista.

`internal/acceptance/acceptance_test.go` tinha uma contagem fixa (`len(tickets) != 67`); como T16-05
é um ticket novo legítimo, atualizei para `68`.

## Validações executadas (resultado real)

Ambiente: este host Windows não compila o projeto nativamente (`internal/worktrees` usa
`syscall.Flock`, só existe em Unix — **pré-existente, não relacionado a esta entrega**). Toda
validação relevante rodou dentro do container Linux `le-fabrique-dev-api-1` (`go1.27`, mesma rede
do Postgres do `make up`).

| Comando | Resultado |
|---|---|
| `GOOS=linux go build ./...` (e depois nativo dentro do container) | **PASS** |
| `go vet ./...` (repo inteiro, dentro do container) | **PASS** |
| `gofmt -l` nos arquivos desta entrega (novos e editados) | **PASS** (limpo) |
| `go test -count=1 ./...` (unitários, repo inteiro) | **PASS** — todos os pacotes `ok`, inclusive `internal/acceptance` depois do ajuste da contagem |
| `go test ./tests/contract/...` | **PASS** (as duas rotas novas aparecem roteadas) |
| `TEST_DATABASE_URL=... go test ./tests/integration/...` (suíte inteira) | **PASS**, confirmado em três execuções limpas nesta sessão (13.6s, 39.7s, 15.5s) |
| Testes novos desta entrega isolados (`-run 'OperationalEvents\|WorkerJobsRecord\|SentinelRoutineRecords\|OperationalEventsAreRemoved'`) | **PASS** |
| `bash scripts/check-secrets.sh` | **PASS** — nenhum padrão de segredo |
| `make lint` (no host Windows) | **NOT_RUN** de forma confiável — `gofmt -l` aponta quase todo arquivo do repositório por causa de CRLF pré-existente neste checkout (confirmado que acontece também em arquivos nunca tocados por esta entrega, ex. `internal/worktrees/*.go`). Não é uma regressão desta entrega; é uma condição do ambiente local. O sinal equivalente (`go vet` + `gofmt -l` nos arquivos tocados) está limpo. |
| `make test-integration` (via `make up` + comando do Makefile) | **NOT_RUN** literalmente (o alvo do Makefile roda no host, que não compila); substituído por execução equivalente dentro do container, cobrindo o mesmo comando (`go test ./tests/integration/...` com `TEST_DATABASE_URL`) |

Depois das três confirmações limpas da suíte inteira, repeti a verificação pontual várias vezes em
sequência (para reforçar a confiança) e provoquei exaustão de recursos no Docker Desktop local
(container `api` ficou `unhealthy`, precisou reiniciar) — os hangs subsequentes vieram daí, não do
código (confirmado: a mesma suíte, com o mesmo código, já tinha passado limpa três vezes antes
disso; um dos hangs apontava para `TestCLILoginCodexDeviceFlowCreatesTheConnection`, teste
pré-existente e sem relação com T16-05 que depende de rede externa). Registro isso para quem for
reexecutar: rode `make test-integration` com o ambiente descansado, uma vez, sem empilhar execuções
consecutivas pesadas.

## Critérios de aceite desta frente (`00-plano`, CA-01 a CA-09 e CA-12)

| Critério | Status | Evidência |
|---|---|---|
| CA-01 (falha grava evento error) | **ATENDIDO** | `TestWorkerJobsRecordOperationalEventsOnFailureAndNote`, `TestSentinelRoutineRecordsAnOperationalEvent` |
| CA-02 (sucesso com nota grava evento info; sem nota não grava) | **ATENDIDO** | idem |
| CA-03 (proxy de egress ausente grava `fabrica`/`error`) | **ATENDIDO** | `cmd/worker/factory_test.go` (`TestReportEgressProxyUnavailableLogsAndRecordsAnOperationalEvent`); reproduz o caso real observado nesta sessão com o stack de dev |
| CA-04 (cursor) | **ATENDIDO** | `TestOperationalEventsHistoryIsFilterableSearchableAndPaged` |
| CA-05 (filtro nível/origem, 400 em valor inválido) | **ATENDIDO** | idem |
| CA-06 (busca textual) | **ATENDIDO** | idem |
| CA-07 (SSE reconexão por cursor) | **ATENDIDO** | `TestOperationalEventsStreamReconnectsByTheCursor` |
| CA-08 (redação de segredo) | **ATENDIDO** | `TestOperationalEventsRedactSecrets` |
| CA-09 (retenção 30 dias + Decision por origem) | **ATENDIDO** | `TestOperationalEventsAreRemovedAfter30DaysWithADecisionPerSource` |
| CA-12 (401 sem sessão) | **ATENDIDO** | `TestOperationalEventsRequireSession` |

## Fora do escopo (conforme o plano)

Tela "Operação" no painel web (frente `web`, prompt próprio); ações sobre o evento; gravação de
eventos pela API; logs do binário `egressproxy`; mudanças no stream de execução por projeto
(T16-01).
