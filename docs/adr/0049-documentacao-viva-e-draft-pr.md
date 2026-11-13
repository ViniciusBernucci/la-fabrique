# ADR-0049 — Documentação viva e Draft PR idempotente

- **Estado:** ACEITA · **Ticket:** [T15-02](../NEW%20MVP/03-BACKLOG.md#t15-02) · **Épico:** E15 · **Relacionadas:** [ADR-0024](0024-github-branch-commit-e-push.md), [ADR-0018](0018-ledger-de-efeitos-externos.md), [ADR-0040](0040-checks-baseline-e-classificacao-de-falhas.md), [ADR-0046](0046-testes-e-e2e-por-impacto.md), [ADR-0047](0047-revisao-independente-e-correcoes.md)

## Decisão
1. **Gate de documentação por módulo:** para cada módulo com código alterado no diff (pacote Go em `internal/<x>/`, área do painel em `apps/web/src/<x>/`; regras configuráveis), o diff precisa trazer a documentação do módulo (`internal/<x>/doc.go` ou `docs/modulos/<x>.md`; no painel `docs/modulos/web-<x>.md` ou `web.md`). Só teste, documentação, migrations e `cmd/` não pedem. Cada lacuna lista o módulo e os arquivos aceitos.
2. **Histórico vivo da feature:** `docs/features/<chave>.md`, gerado a partir do que está registrado (`delivery.BuildReport`): requisito vigente, tarefas (estado, item na origem, risco, revisão, critérios, **commits por tarefa**), **checks da revisão entregue** com classificação, jornadas E2E e justificativas. É determinístico; precisa estar no diff e, se o conteúdo commitado for informado, ser igual ao gerado (senão "desatualizado"). Os commits por tarefa são registrados por `RecordCommit` (idempotente por SHA).
3. **Gates da entrega:** além da documentação, quaisquer `Gate` (checks aprovados da revisão, revisão independente por tarefa, E2E…). Qualquer pendência **bloqueia**: nenhuma PR é aberta e a tentativa fica registrada em `deliveries` (estado `blocked`, motivos).
4. **Draft PR idempotente pelo ledger:** `PRPublisher.OpenDraft` registra a intenção com a chave `<projeto>:<feature>:<tentativa>:pr:<branch>`, cria a PR **sempre como rascunho** e a branch tem que ser a da feature (nunca a base). Repetir devolve a mesma PR (`already_confirmed`). Se a resposta da criação se perde, ou se o GitHub recusa com "já existe", a **prova por busca** (PRs abertas da branch → base) confirma a mesma PR, sem duplicar; ausente comprovada repete; consulta que falha é inconclusiva e bloqueia a repetição até decisão.
5. **Origem:** com a PR aberta, a plataforma de origem recebe "Aguardando merge" (`PublishDraftPR` de `tasksync`); se falhar, a PR existe e repetir não duplica nada. **O merge continua humano.**
6. **API do GitHub:** `GitHubAPI.FindPR` e `CreateDraftPR` (token só no cabeçalho, mensagens fixas, `422 já existe` distinguido de outras recusas).

## Limites
- O GitHub real NOT_RUN (a API usada nos testes é um servidor HTTP de mentira); o `git` real (commit/push) é do T15-01 e não é exercitado junto aqui. A fábrica não registra sozinha os commits: quem commita chama `RecordCommit`.
- O "repo real contém docs e testes coerentes" é verificado pelo gate no diff informado (não há leitura do repositório real) e pela orientação de testes de T14-04; a coerência semântica da documentação não é avaliada.
- Sem rotas nem tela de entrega; o merge, a propagação da conclusão e as correções de revisão externa são T15-03/T15-04.
