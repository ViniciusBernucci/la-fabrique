# 00-plano — T16-05: Consolidar logs operacionais da fábrica

- **Lote:** `docs/planejamento/2026-11-18/lote-2026-11-18-painel-operacao-fabrica.md`
- **Épico:** [E16 — Painel operacional, histórico, uso e retenção](../../NEW%20MVP/02-EPICOS.md#e16)
- **Backlog:** novo ticket `T16-05`, a acrescentar em [03-BACKLOG.md](../../NEW%20MVP/03-BACKLOG.md) ao lado de T16-01..T16-04
- **Base:** branch `developer`, SHA `82857c0da3ee7408e7f3a6367fb3ffbe6442fdc7` (local, à frente de `origin/developer` em `7172abc3b8043d6d225990f9be3a4c44d028afc4`)
- **Estado:** `PRONTA_PARA_IMPLEMENTAR`
- **Perfil de risco:** **crítico** — migration nova + dado operacional interno exposto por API (exige redação de segredos no payload; um único gatilho já basta, mas há dois: migration e exposição de dado novo por rota autenticada).

## Contexto

O painel por projeto (T16-01/D27, `internal/httpapi/live.go`) já mostra o stream de eventos de
**execução de um projeto**. Não existe hoje nenhuma visão do funcionamento **geral** do worker: os
jobs de fundo (sentinela, admissão, retenção, sincronização, cli-login) só escrevem logs
estruturados (`log/slog`, JSON) no `stdout` do container — nada é persistido, nada aparece no
painel. O responsável quer um painel global ("Operação") com isso, complementando (sem substituir)
o que já existe por projeto.

Levantamento de código (sessão de planejamento, 2026-11-18):

- Todo job periódico do worker passa por um único ponto: `(w *Worker) exec` em
  `internal/worker/jobs.go:68`. Ele já sabe o nome do job, a nota (`note`) e o erro (`runErr`) de
  cada execução, e já loga via `w.log` (slog). Jobs registrados por `w.AddJob` hoje:
  `liberar-decisoes`, `recuperar-tentativas`, `retencao`/`retencao-simulada`, `sentinela-eventos`
  (na verdade a reavaliação da **admissão** por evento — nome genérico, ver mapeamento abaixo),
  `sentinela-rotinas` (dispara as rotinas agendadas abaixo), `login-contas-ia`, `fabrica`.
- Toda rotina agendada (sync de Issues, PRs, projeção de status, varredura de manutenção, admissão
  por varredura) passa por outro ponto único: `(s *Scheduler) RunOne` em
  `internal/sentinel/routines.go:142`. Ele também já sabe o nome da rotina e o erro de cada
  execução.
- Há exceções: alguns erros de **configuração do boot** do worker (ex.: `cmd/worker/factory.go:50`,
  "proxy de saída indisponível") acontecem **uma vez**, fora do laço periódico, e não passam por
  nenhum dos dois pontos acima. Alguns desses (`cmd/worker/main.go`, `cmd/worker/connect.go`)
  acontecem **antes** do pool do Postgres existir — não há banco para gravar, ficam só no stdout
  (fora do escopo). Os que acontecem **depois** do pool existir (ex.: `factory.go:50`) entram no
  escopo e precisam de uma chamada explícita.
- Retenção (T16-04, `internal/retention/retention.go`) já tem o padrão exato a reaproveitar:
  `retention_runs` e `retention_decisions` são o resumo de auditoria **permanente** que sobra depois
  de apagar o detalhe. Não é preciso inventar uma tabela de resumo nova para os eventos
  operacionais: as mesmas tabelas já servem.
- Rotas autenticadas usam `s.protected(...)` (sessão de admin único; não há múltiplos papéis no
  projeto, ADR-0005). `s.redactor` (campo do `Server`) já redige segredos em `[]byte` antes de
  escrever no stream (`live.go:210`).
- Modelo de teste de integração para stream/eventos: `tests/integration/live_test.go`.

## Resultado esperado

Um administrador abre "Operação" no menu global do painel, vê os eventos operacionais da fábrica
(sentinela, admissão, retenção, sincronização, cli-login e os erros de boot do worker cobertos
abaixo) em tempo real, filtra por nível e origem, busca por texto, reconecta sem perder nem
duplicar evento, e sabe que o detalhe desaparece depois de 30 dias mantendo o resumo de auditoria.

## Escopo

1. Migration: tabela `operational_events` (nível, origem, mensagem, payload, `created_at`).
2. Pacote `internal/opslog`: `Emit(ctx, db, level, source, message, payload)` — grava um evento;
   falha ao gravar não derruba o job que chamou (loga um aviso e segue).
3. Wiring em `internal/worker/jobs.go:exec` e `internal/sentinel/routines.go:RunOne` (cobre todos os
   jobs/rotinas listados no contexto, por mapeamento de nome → origem, seção "Regras de negócio").
4. Wiring explícito em `cmd/worker/factory.go:50` (proxy de egress ausente) — único erro de boot
   pós-conexão identificado nesta especificação. Se a implementação encontrar outro caso
   equivalente (erro de configuração após o pool existir, fora do laço periódico), registre-o do
   mesmo jeito e anote no registro de entrega; não é motivo para bloquear o ticket.
5. Extensão de `internal/retention/retention.go`: `Retention.Run` também apaga `operational_events`
   mais velhos que `LogDays`, registrando uma `Decision` agregada por origem em
   `retention_decisions` (mesmo padrão já usado para `execution_events`/checkpoints).
6. `GET /api/operations/events` (histórico paginado por cursor, filtro por nível/origem, busca `q`).
7. `GET /api/operations/stream` (SSE, reconexão por `Last-Event-ID`/`?after=`, mesmo padrão de
   `executionStream`).
8. Tela "Operação" em `apps/web`: novo item de menu global, lista com filtro por nível/origem,
   busca, atualização ao vivo via SSE com reconexão.
9. Documentação: `contracts/openapi.yaml`, este `00-plano`, ADR se a implementação achar necessário
   registrar uma decisão nova (ver "Riscos e questões abertas"), e `docs/NEW MVP/07-CHECKLIST.md`.

## Fora do escopo

- Logs do binário `egressproxy` em si (requisições permitidas/negadas) — só o erro que o *worker*
  reporta ao tentar subir o proxy entra.
- Ações sobre o evento (marcar como resolvido, arquivar, abrir Central de Decisões a partir dele).
- Gravação de eventos operacionais pela API (`cmd/api`) — só o worker grava nesta entrega.
- Erros de boot anteriores à conexão com o Postgres (`cmd/worker/main.go` antes do `pool`,
  `cmd/worker/connect.go` durante a retentativa de conexão) — continuam só no stdout.
- Qualquer mudança de comportamento no stream de execução por projeto já existente (T16-01,
  `internal/httpapi/live.go`).
- Alterar o intervalo, a política ou o comportamento de qualquer job/rotina existente — só
  acrescentar a gravação do evento ao lado do que já roda.

## Decisões (com origem)

| # | Decisão | Origem |
|---|---|---|
| D1 | Fontes: jobs de fundo do worker (sentinela, admissão, retenção, sincronização, cli-login) e erros de infraestrutura do boot do worker pós-conexão. Execução por projeto fica fora. | Usuário (entrevista 2026-11-18) |
| D2 | Persistência em tabela nova (`operational_events`), gravada só pelo worker; a API só lê. | Usuário |
| D3 | Tempo real via SSE com cursor/reconexão, mesmo padrão do stream de execução. | Usuário |
| D4 | Filtro por nível e por origem, mais busca textual livre. | Usuário |
| D5 | Mesma política de retenção do T16-04/D41 (30 dias de detalhe, resumo permanente). | Usuário |
| D6 | Só visualizar/buscar nesta entrega; sem ações sobre o evento. | Usuário |
| D7 | Novo item global no menu ("Operação"), ao lado de Decisões/Consumo/Segurança; só o admin único acessa (sessão já exige isso em toda rota `protected`). | Usuário + evidência (`AppShell.tsx`, ADR-0005) |
| D8 | Os dois pontos únicos de execução periódica (`jobs.go:exec`, `routines.go:RunOne`) bastam para cobrir sentinela/admissão/retenção/sincronização/cli-login sem instrumentar cada pacote individualmente. | Evidência (leitura do código, ver Contexto) |
| D9 | `sentinela-eventos` (nome do job) mapeia para a origem `admissao` — é a reavaliação por evento do laço de admissão (`cmd/worker/admission.go:27`), apesar do nome genérico. | Evidência (código) |
| D10 | Resumo de auditoria da retenção reaproveita `retention_runs`/`retention_decisions`; não há tabela de resumo nova para eventos operacionais. | Hipótese (consistente com D5 e com o padrão já usado para checkpoints/execução) — **confirmar com o usuário só se a revisão discordar**; não bloqueia a implementação. |

## Regras de negócio

- **Nível** (`level`): `info` | `warn` | `error`.
- **Origem** (`source`): `sentinela` | `admissao` | `retencao` | `sincronizacao` | `clilogin` |
  `fabrica`.
- **Mapeamento nome do job/rotina → origem** (único lugar que decide isso; a implementação não
  precisa redescobrir):

  | Job (`internal/worker/jobs.go`, nome registrado) | Origem |
  |---|---|
  | `retencao`, `retencao-simulada` | `retencao` |
  | `login-contas-ia` | `clilogin` |
  | `fabrica` | `fabrica` |
  | `sentinela-eventos` | `admissao` |
  | `sentinela-rotinas`, `liberar-decisoes`, `recuperar-tentativas` | `sentinela` |
  | qualquer outro nome futuro não listado aqui | `sentinela` (default) |

  | Rotina (`internal/sentinel/routines.go` / `internal/admission`, nome registrado) | Origem |
  |---|---|
  | `RoutineIssueSync`, `RoutineCorrections`, `RoutinePRs`, `RoutineProjection` | `sincronizacao` |
  | `RoutineMaintenance` | `sentinela` |
  | `admission.RoutineName` | `admissao` |
  | qualquer outro nome futuro não listado aqui | `sentinela` (default) |

- **Quando emitir** (para não encher a tabela com ruído de polling vazio):
  - Em `jobs.go:exec`: emitir sempre que `runErr != nil` (nível `error`, mensagem
    `runErr.Error()`); emitir com nível `info` só quando `note != ""` (mesmo gate que já existe
    para `w.log.Info`). Sem nota e sem erro: não emite.
  - Em `routines.go:RunOne`: emitir só quando a rotina **realmente rodou** (`tag.RowsAffected() >
    0`, o mesmo branch que já atualiza `sentinel_routine_state`); nível `error` com
    `runErr.Error()` quando falhar, nível `info` quando tiver sucesso. "Due mas não reservou"
    (outra execução em andamento) não emite.
  - Em `cmd/worker/factory.go:50`: emitir nível `error` com a mesma mensagem já logada, uma vez
    por tentativa de boot (não é periódico).
- **Falha ao gravar o evento operacional nunca derruba o job/rotina que o originou.** `opslog.Emit`
  devolve erro; o chamador loga um aviso (`slog`) e continua — o resultado do job/rotina em si (seu
  próprio `note`/`runErr`) não muda.
- **Redação:** o `payload` passa por `Redactor.RedactBytes` antes de sair pela API (histórico e
  stream), mesmo padrão de `execution_events`.
- **Retenção:** eventos operacionais com `created_at` mais velho que `Policy.LogDays` (30 por
  padrão) são apagados na mesma execução de `Retention.Run` que já cuida de `execution_events` e
  checkpoints; dry-run simula sem apagar (`would_delete`), igual ao que já existe. Uma `Decision`
  por origem (`kind: "operational_events"`, `ref: source`) registra quantos eventos saíram.
- **Acesso:** `GET /api/operations/events` e `GET /api/operations/stream` exigem sessão (`s.protected`).
  Não há papel diferente de admin neste projeto (ADR-0005): qualquer sessão válida vê tudo.

## Critérios de aceite (Dado/Quando/Então)

- **CA-01** — Dado o worker rodando com `SENTINEL_ADMISSION`/retenção/sentinela ativos, quando um
  job periódico roda e falha, então um evento `level=error` com a origem mapeada (tabela acima)
  aparece em `operational_events` com a mensagem do erro.
- **CA-02** — Dado o mesmo cenário, quando o job roda com sucesso e produz uma nota (`note != ""`),
  então um evento `level=info` com a mesma origem é gravado; quando não produz nota (nada a fazer),
  nenhum evento é gravado.
- **CA-03** — Dado `FACTORY_EGRESS_FILTER=true` e a imagem `le-fabrique/egress:dev` ausente, quando
  o worker sobe, então um evento `level=error`, `source=fabrica` com a mensagem "proxy de saída
  indisponível..." aparece em `operational_events` (reproduz o caso real observado nesta sessão).
- **CA-04** — Dado eventos existentes, quando o cliente chama `GET /api/operations/events?after=N`,
  então só eventos com `seq > N` voltam, ordenados por `seq`, com `next_cursor` igual ao maior `seq`
  devolvido (ou `N` se não houver nenhum).
- **CA-05** — Dado o parâmetro `level` ou `source`, quando fora do enum (`CA-01`), então a API
  devolve `400 invalid_request`; valor dentro do enum filtra corretamente.
- **CA-06** — Dado o parâmetro `q`, quando preenchido, então só eventos cuja `message` ou `payload`
  contenham o texto (case-insensitive) voltam — mesmo comportamento de busca de
  `executionEvents`.
- **CA-07** — Dado um cliente conectado em `GET /api/operations/stream`, quando a conexão cai e
  reconecta com `Last-Event-ID` (ou `?after=`), então recebe exatamente os eventos que faltaram,
  sem lacuna nem duplicata.
- **CA-08** — Dado um evento com segredo no `payload` (ex.: token), quando servido por qualquer um
  dos dois endpoints, então o segredo sai redigido (mesma verificação que já existe para
  `execution_events`, adaptada ao novo endpoint).
- **CA-09** — Dado `operational_events` com `created_at` mais velho que `Policy.LogDays`, quando
  `Retention.Run` roda (`dryRun=false`), então essas linhas são apagadas e uma `Decision`
  (`kind=operational_events`) é gravada em `retention_decisions` com a contagem por origem; em
  `dryRun=true`, nada é apagado e a `Decision` sai como `would_delete`.
- **CA-10** — Dado o painel web aberto em "Operação", quando um evento novo chega pelo SSE, então
  aparece na lista sem recarregar a página, respeitando o filtro de nível/origem e a busca ativos.
- **CA-11** — Dado o menu principal do painel, quando o admin está autenticado, então "Operação"
  aparece como item global (fora de qualquer projeto), ao lado de Decisões/Consumo/Segurança.
- **CA-12** — Dado um cliente sem sessão válida, quando chama qualquer um dos dois endpoints novos,
  então recebe `401`.

## Contratos de API

Seguem o envelope de erro já padronizado do projeto:
`{"error":{"code","message","action","correlation_id"}}` (ver `internal/httpapi/errors.go`).

### `GET /api/operations/events`

- **Query:** `after` (integer ≥ 0, cursor; default `0`), `level` (enum `info|warn|error`,
  opcional), `source` (enum `sentinela|admissao|retencao|sincronizacao|clilogin|fabrica`,
  opcional), `q` (string, até 100 caracteres, opcional), `limit` (integer 1–500, default 200).
- **200:** `{"items":[{"seq":int64,"level":string,"source":string,"message":string,"payload":object,"created_at":datetime}], "next_cursor": int64}`
- **400** `invalid_request`: cursor inválido, `level`/`source` fora do enum, `q` > 100 caracteres.
- **401**: sem sessão válida.

### `GET /api/operations/stream`

- **Query:** `after` (integer ≥ 0, opcional). **Header:** `Last-Event-ID` (integer ≥ 0, opcional;
  tem prioridade sobre `?after=` — mesma regra de `cursorOf` em `live.go`).
- **200** `text/event-stream`: cada evento como `id: <seq>\ndata: {"seq","level","source","message","payload","created_at"}\n\n`;
  heartbeat `: ping`; encerra após `StreamMaxAge` com `event: reconnect` para o cliente reconectar
  pelo cursor. Sem filtro por nível/origem no stream em si (o filtro do painel web é client-side
  sobre o que chega — mesma simplicidade do stream de execução, que também não filtra no servidor).
- **400** `invalid_request`: cursor inválido. **401**: sem sessão válida.

### `contracts/openapi.yaml`

Acrescentar os dois paths acima (tag `operacao-ao-vivo`, mesmo padrão de
`/projects/{id}/executions/{executionId}/events` e `.../stream`) e os schemas
`OperationalEvent`/`OperationalEvents` (mesmo formato de `ExecutionEvents`, com os campos
`level`/`source` adicionais).

## Tickets

### T16-05 — Consolidar logs operacionais da fábrica (frentes `api` + `web`)

- **Objetivo:** painel global de logs operacionais da fábrica (worker), complementando o stream por
  projeto (T16-01).
- **Dependências:** nenhuma de outro ticket deste lote. Depende apenas do código já em `developer`
  no SHA base.
- **Contratos afetados:** `contracts/openapi.yaml` (dois paths novos, dois schemas novos).
- **Migrations:** uma nova, `db/migrations/000NN_operational_events.sql` (número exato = próximo
  livre no momento da implementação; hoje a última aplicada é `00072`).
- **Riscos:** instrumentar `jobs.go:exec`/`routines.go:RunOne` errado pode duplicar eventos ou
  (pior) introduzir uma falha que impeça o job original de rodar — a chamada a `opslog.Emit` deve
  vir **depois** de `j.status`/`sentinel_routine_state` já terem sido atualizados, nunca antes nem
  no meio, e nunca `return` em caso de erro do próprio `Emit`. Vazamento de segredo no `payload` se
  a redação for esquecida num dos dois endpoints — CA-08 cobre isso explicitamente.
- **Arquivos prováveis:**
  - `db/migrations/000NN_operational_events.sql` (nova)
  - `internal/opslog/opslog.go` (novo pacote) + `internal/opslog/opslog_test.go`
  - `internal/worker/jobs.go` (`exec`)
  - `internal/sentinel/routines.go` (`RunOne`)
  - `cmd/worker/factory.go` (linha do `log.Error` do proxy de egress)
  - `internal/retention/retention.go` (`Run`) + `internal/retention/retention_test.go`
  - `internal/httpapi/operations.go` (novo: handlers `operationsEvents`/`operationsStream`,
    reaproveitando `cursorOf`/padrão de `live.go`) + `internal/httpapi/server.go` (registrar rotas)
  - `tests/integration/operations_test.go` (novo, modelo: `tests/integration/live_test.go`)
  - `contracts/openapi.yaml`
  - `apps/web/src/pages/OperationsPage.tsx` (nova) + rota em `apps/web/src/App.tsx` + item de menu
    em `apps/web/src/components/AppShell.tsx` + chamadas em `apps/web/src/lib/api.ts`
  - `apps/web/src/pages/operations.test.tsx` (novo, modelo: `apps/web/src/pages/runpanel.test.tsx`)
  - `docs/NEW MVP/03-BACKLOG.md` (acrescentar T16-05), `docs/NEW MVP/07-CHECKLIST.md`
    (acrescentar linha), `contracts/openapi.yaml`, `docs/CHANGELOG.md` se o convencionado pelo
    projeto pedir.
- **Testes:**
  - `internal/opslog`: `Emit` grava a linha certa (nível/origem/mensagem/payload); teste unitário
    simples com banco de teste.
  - `tests/integration/operations_test.go`: CA-01, CA-02, CA-04 a CA-08, CA-12 — modelo direto em
    `tests/integration/live_test.go` (troque execução por projeto pelos endpoints novos).
  - `internal/retention/retention_test.go`: estender o teste existente de `Run` com uma linha de
    `operational_events` velha e uma nova — CA-09.
  - CA-03: teste de integração que registra `registerFactory` com `FactoryEgressFilter=true` e uma
    imagem de egress inexistente (ou teste unitário direto na função extraída, se a implementação
    isolar a chamada do `log.Error` numa função testável) e confere a linha gravada.
  - `apps/web/src/pages/operations.test.tsx`: CA-10, CA-11 — modelo em `runpanel.test.tsx` (mock do
    SSE) e em `App.test.tsx`/menu para o item novo.
- **Documentação:** `docs/NEW MVP/03-BACKLOG.md`, `docs/NEW MVP/07-CHECKLIST.md`,
  `contracts/openapi.yaml`, registro de entrega em `docs/entregas/`.
- **Estado:** `PRONTA_PARA_IMPLEMENTAR`.

## Dependências e ordem de migrations/deploy

- Migration única deste ticket, depois da `00072` (última aplicada no SHA base).
- Ordem dentro do ticket: migration → `internal/opslog` → wiring no worker (jobs/routines/factory)
  → extensão da retenção → endpoints da API → `contracts/openapi.yaml` → tela web. Cada etapa é
  committável isoladamente, mas o ticket inteiro fecha num commit só (convenção do projeto).
- Sem paralelismo real entre `api` e `web`: a tela web depende do contrato que a frente `api`
  define. Frente única sequencial em `developer`, sem worktree.

## Riscos e questões abertas

- **D10 é hipótese, não decisão confirmada pelo usuário.** Reaproveitar `retention_decisions` em
  vez de criar uma tabela de resumo nova é a leitura mais simples e consistente com o padrão
  existente; se a revisão ou o usuário discordar depois de ver o código, é um ajuste pequeno e
  isolado (trocar `Decision` por uma tabela de resumo), não um bloqueio para começar a implementar.
- **Nenhuma questão essencial em aberto.** A entrevista (2026-11-18) cobriu fontes, persistência,
  tempo real, filtros, retenção, ações e navegação; todas as decisões têm origem registrada acima.
