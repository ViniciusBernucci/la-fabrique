# Checklist do backlog de adequação (LF-AJ)
Acompanhamento dos 35 tickets de [Le-Fabrique-Adequacao.md](Le-Fabrique-Adequacao.md). **Fonte de verdade do estado da adequação**: atualize ao concluir ou alterar um ticket (marcar `[x]` só com evidência real). Fica separado do [07-CHECKLIST.md](07-CHECKLIST.md) porque a matriz de aceite do MVP é derivada dele.
Legenda: `[x]` feito e verificado · `[~]` parcial (ver nota) · `[ ]` pendente.
_Última atualização: 2026-11-13 — AJ03-01._
## Resumo
**3 feitos · 0 parciais · 32 pendentes** de 35 tickets.

## LF-AJ-01 — Sincronização e gerenciamento de Issues
- [ ] **AJ01-01** — Leitura Jira/ClickUp: descobrir Issues elegíveis e atualizar conhecidas por ID · dep.: AJ02-01 · aceite: CT01.
- [ ] **AJ01-02** — Sync matinal e manual com resultado na página Issues · dep.: AJ01-01, AJ03-01 · aceite: CT01, CT15.
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
- [ ] **AJ03-02** — Transição validada e efeitos de status centralizados na Sentinela · dep.: AJ03-01, AJ02-02 · aceite: CT02, CT03, CT15.
- [ ] **AJ03-03** — Admissão por categoria, prioridade, antiguidade e vagas Front/Back/planejamento · dep.: AJ03-02, AJ04-01 · aceite: CT05, CT12.
- [ ] **AJ03-04** — Retomada nativa, fallback por checkpoint e coleta de órfãos · dep.: AJ03-03, AJ07-01, AJ06-01 · aceite: CT07, CT15.
- [ ] **AJ03-05** — Migração aditiva, ativação por workflow publicado e reversão operacional · dep.: AJ03-03, AJ05-03 · aceite: CT16.

## LF-AJ-04 — Contratos determinísticos de execução
- [ ] **AJ04-01** — Schema, proveniência, invariantes e snapshot validado pela Sentinela · dep.: AJ02-01, E12, E13 · aceite: CT04.
- [ ] **AJ04-02** — Gate PRD e contrato inicial antes de liberar o planejamento · dep.: AJ04-01, AJ01-01, AJ03-02 · aceite: CT03.
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
