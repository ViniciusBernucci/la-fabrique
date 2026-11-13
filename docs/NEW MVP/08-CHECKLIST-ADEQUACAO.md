# Checklist do backlog de adequação (LF-AJ)
Acompanhamento dos 35 tickets de [Le-Fabrique-Adequacao.md](Le-Fabrique-Adequacao.md). **Fonte de verdade do estado da adequação**: atualize ao concluir ou alterar um ticket (marcar `[x]` só com evidência real). Fica separado do [07-CHECKLIST.md](07-CHECKLIST.md) porque a matriz de aceite do MVP é derivada dele.
Legenda: `[x]` feito e verificado · `[~]` parcial (ver nota) · `[ ]` pendente.
_Última atualização: 2026-11-13 — AJ01-01, AJ01-02._
## Resumo
**7 feitos · 2 parciais · 26 pendentes** de 35 tickets.

## LF-AJ-01 — Sincronização e gerenciamento de Issues
- [x] **AJ01-01** — Leitura Jira/ClickUp: descobrir Issues elegíveis e atualizar conhecidas por ID · dep.: AJ02-01 · aceite: CT01.
  - `internal/issuesync`: sync geral paginado com Issues fechadas incluídas, importa só novas em PRD-DOC/CORRECAO_PR (pelo mapa de status do projeto), ignora desconhecidas em outros estados, atualiza as conhecidas em qualquer estado por ID (GetItem quando a listagem não traz), upsert por projeto+origem+ID remoto, prioridade local preservada; novas são rastreadas pela Sentinela. Migration 00048. PASS: integração `TestIssueSyncDiscoversEntryStatesAndUpdatesKnownOnes` (jira e clickup, via Tracker simétrico; os adaptadores reais seguem cobertos pela suíte de contrato existente) e `TestIssuesAPIStatusMapManualSyncAndHistory` contra o fake Jira.
- [x] **AJ01-02** — Sync matinal e manual com resultado na página Issues · dep.: AJ01-01, AJ03-01 · aceite: CT01, CT15.
  - Rotina matinal `issues.sync` por fuso (uma vez por janela, backoff e erro/último sucesso por escopo) + sync manual `POST /projects/{id}/issues/sync` que usa a mesma função; falha paginada vira `partial` com progresso preservado; página **Issues** no painel (lista, status canônico, histórico, botão Sincronizar agora); rotas de mapa de status. PASS: integração `TestMorningIssueSync…`, `TestIssueSyncKeepsProgressWhenAPageFails`, `TestIssuesAPI…`, vitest `issues.test.tsx`, contrato OpenAPI. O worker só registra o sync com `SENTINEL_ISSUE_SYNC=true` (exige `VAULT_KEY` no worker; decisão de implantação, desligado por padrão).
- [ ] **AJ01-03** — Consulta por ID das correções aguardando liberação · dep.: AJ01-01, AJ03-01 · aceite: CT02.
- [ ] **AJ01-04** — Projeção Jira/ClickUp e pendências de mapeamento/conflito/escrita · dep.: AJ01-01, AJ02-02, AJ03-02 · aceite: CT01, CT15.

## LF-AJ-02 — Catálogo de status e transições
- [x] **AJ02-01** — Catálogo, mapeamento por projeto e migração do alias · dep.: E03, E06 · aceite: CT02, CT16.
  - Catálogo com IDs estáveis (`internal/flow`), alias `REPROVED_PR` → `REPROVADO_PR` com bruto preservado, migration 00042 aditiva e mapa por projeto validado com campo do erro. PASS: `go test ./internal/flow` e integração `TestStatusCatalogIsIdempotentAndMapsPerProject` (reaplicar mantém IDs, alias converge, versão do mapa).
- [x] **AJ02-02** — Guardas para fluxo normal, correção e estado agregado · dep.: AJ02-01 · aceite: CT03, CT06, CT13.
  - Guardas puras `flow.Check` (fluxo normal, correção, PRD incompleto, sem liberação, sem merge, cabeça obsoleta) e `flow.Aggregate` (Front revisado + Back implementando mantém IMPLEMENTACAO). PASS: `go test ./internal/flow`.
- [ ] **AJ02-03** — Visão na página Issues: estado, avanço por trilha, bloqueios e projeção pendente · dep.: AJ02-02, AJ01-04, AJ05-02 · aceite: CT06, CT15.

## LF-AJ-03 — Sentinela, escalonamento e recuperação
- [x] **AJ03-01** — Eventos duráveis e rotinas agendadas no worker, com relógios e cursores por alvo · dep.: E10, E16 · aceite: CT02, CT15.
  - `internal/sentinel`: eventos duráveis (dedupe, falha não bloqueia os demais, pendentes recuperados após reinício) e rotinas com relógio, cursor, backoff e reserva atômica independentes por rotina/escopo (janela matinal por fuso); ligados ao worker (`sentinela-eventos`, `sentinela-rotinas`, visíveis em /jobs). Migration 00043. PASS: integração `TestSentinelRoutines…` e `TestTheWorkerRunsSentinel…`. Rotinas concretas (sync, correções, PRs, manutenção) são registradas pelos tickets AJ01-02/03, AJ09-01 e AJ03-04.
- [x] **AJ03-02** — Transição validada e efeitos de status centralizados na Sentinela · dep.: AJ03-01, AJ02-02 · aceite: CT02, CT03, CT15.
  - `sentinel.Coordinator.Transition`: só o ator `sentinel` efetiva (agente é recusado e auditado), guarda `flow.Check`, versão/observação obsoleta não avança, reentrega e corrida (4 goroutines) aplicam uma vez; estado, auditoria, outbox de projeção (`status-projection`) e evento saem da mesma transação. Migration 00044. PASS: integração `TestSentinelIsTheOnlyAuthorityForTransitions` (achou e corrigiu bug de lock com join).
- [~] **AJ03-03** — Admissão por categoria, prioridade, antiguidade e vagas Front/Back/planejamento · dep.: AJ03-02, AJ04-01 · aceite: CT05, CT12.
  - `sentinel.Admit/Order` (correção → prioridade oficial → antiguidade → ID estável; vaga Front e Back compartilhada entre implementação e review; limite próprio de planejamento; capacidade global; dependências, pausa, workflow, orçamento, autorização e conflito de recurso; bloqueado não trava independente; sem preempção) e `sentinel.Reserver` (reserva atômica com geração, aquisição das duas trilhas tudo-ou-nada, escritor antigo recusado). Migration 00047. PASS: `go test ./internal/sentinel` e integração `TestReservationsAreAtomicAndFenceStaleWriters` (CT05, CT12 no nível da política). **Pendente:** o despacho vivo ainda é o `queue.Claim` legado; a troca do despachante acontece na ativação (AJ03-05).
- [ ] **AJ03-04** — Retomada nativa, fallback por checkpoint e coleta de órfãos · dep.: AJ03-03, AJ07-01, AJ06-01 · aceite: CT07, CT15.
- [ ] **AJ03-05** — Migração aditiva, ativação por workflow publicado e reversão operacional · dep.: AJ03-03, AJ05-03 · aceite: CT16.

## LF-AJ-04 — Contratos determinísticos de execução
- [x] **AJ04-01** — Schema, proveniência, invariantes e snapshot validado pela Sentinela · dep.: AJ02-01, E12, E13 · aceite: CT04.
  - `internal/contract`: proposta do Tech Lead decodificada com campos proibidos (permissões, IDs, worktree) rejeitados; caminhos fora do worktree e ciclos recusados; piso de risco e proveniência por campo; snapshots versionados com hash, encadeados e imutáveis no banco (trigger). Migration 00045. PASS: `go test ./internal/contract` e integração `TestContractSnapshotsAreVersionedHashedAndImmutable`.
- [~] **AJ04-02** — Gate PRD e contrato inicial antes de liberar o planejamento · dep.: AJ04-01, AJ01-01, AJ03-02 · aceite: CT03.
  - `internal/prdgate`: campos obrigatórios, conteúdo acessível, consistência e referências; pendências persistidas e Issue em PRD-DOC; válido publica exatamente um contrato inicial por revisão antes da transição a PRD-DOC-CONCLUIDO; evento forjado de agente recusado. Migration 00046. PASS: integração `TestPRDGateBlocksUntilCompleteThenPublishesOneInitialContract` (CT03). **Pendente:** ligar à leitura da origem (AJ01-01), pois o gate recebe o PRD já normalizado.
- [ ] **AJ04-03** — Contrato imutável no cabeçalho de cada despacho e revisão dos insumos · dep.: AJ04-01, AJ07-01 · aceite: CT04, CT07.

## LF-AJ-05 — Workflow padrão, planejamento técnico e subtarefas
- [ ] **AJ05-01** — Workflow estrutural no cadastro com caminhos de feature e correção · dep.: AJ02-02, E05, E07 · aceite: CT14.
- [ ] **AJ05-02** — Validar proposta do Tech Lead e materializar plano e subtarefas internas · dep.: AJ04-01, E10 · aceite: CT04, CT05.
- [ ] **AJ05-03** — Configuração manual dos nós, publicação, ativação e fixação de versão · dep.: AJ05-01, AJ07-01 · aceite: CT14, CT16.
- [ ] **AJ05-04** — Plano focalizado de correção a partir da rodada e da cabeça da PR · dep.: AJ05-02, AJ09-02 · aceite: CT11, CT12.

## LF-AJ-06 — Execução paralela Frontend/Backend e consolidação Git
- [ ] **AJ06-01** — Worktree e branch temporária por tentativa, com escopo e checkpoint · dep.: AJ04-01, E11, E12 · aceite: CT05, CT07.
- [ ] **AJ06-02** — Dependências e reservas de recursos compartilhados na admissão das trilhas · dep.: AJ03-03, AJ05-02, AJ06-01 · aceite: CT05, CT06.
- [ ] **AJ06-03** — Consolidar commits e registrar a cabeça candidata · dep.: AJ06-02, AJ08-01 · aceite: CT06, CT07.
- [ ] **AJ06-04** — Consolidação, checks integrados e adaptador de publicação GitHub (Draft PR) · dep.: AJ06-03, AJ08-05 · aceite: CT06, CT12.

## LF-AJ-07 — Monitoramento de agentes, terminais e interação humana
- [ ] **AJ07-01** — Capacidades CLI/API no contrato dos adaptadores de execução e recuperação · dep.: E08, E09, E11, E12 · aceite: CT07, CT10, CT14.
- [ ] **AJ07-02** — Terminal, eventos, cronômetro, progresso e subagentes observáveis · dep.: AJ07-01 · aceite: CT10.
- [ ] **AJ07-03** — Orientações ao agente com confirmação técnica e histórico na retomada · dep.: AJ07-02, AJ04-03 · aceite: CT07, CT10.

## LF-AJ-08 — Revisão automatizada, testes e ambiente visual
- [ ] **AJ08-01** — Política de testes, risco e review por trilha no pipeline · dep.: AJ03-03, AJ05-02, E14 · aceite: CT17.
- [ ] **AJ08-02** — Chrome/Chromium isolado e ferramentas MCP de navegação, evidência e limpeza · dep.: AJ06-01, AJ03-01, E08, E11, E13 · aceite: CT08, CT09.
- [ ] **AJ08-03** — Descoberta do ambiente de teste pelo repositório e proposta editável (Decisão 29) · dep.: AJ06-01, E04 · aceite: CT08.
- [ ] **AJ08-04** — Validar, versionar, persistir e reutilizar a configuração de ambiente por projeto · dep.: AJ08-02, AJ08-03 · aceite: CT08, CT09.
- [ ] **AJ08-05** — Evidência visual, resultados por trilha e gate de testes integrados antes da PR · dep.: AJ08-01, AJ08-04, AJ06-03 · aceite: CT06, CT17.

## LF-AJ-09 — Draft PR, revisão humana e ciclo de correções
- [ ] **AJ09-01** — Consultar PRs vinculadas com agenda própria e ação manual · dep.: AJ03-01, E15 · aceite: CT11, CT13, CT15.
- [ ] **AJ09-02** — Agrupar comentários da rodada e criar a Issue REPROVADO_PR · dep.: AJ09-01, AJ01-04 · aceite: CT11, CT15.
- [ ] **AJ09-03** — Despachar correção liberada, plano focalizado e atualizar a PR original · dep.: AJ01-03, AJ05-04, AJ06-04 · aceite: CT02, CT12.
- [ ] **AJ09-04** — Encerrar só Issues incorporadas após merge e tratar fechamento sem merge · dep.: AJ09-01, AJ09-03, AJ03-02 · aceite: CT13, CT15.
