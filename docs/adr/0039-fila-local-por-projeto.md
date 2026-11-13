# ADR-0039 — Fila local por projeto

- **Estado:** ACEITA · **Ticket:** [T10-02](../NEW%20MVP/03-BACKLOG.md#t10-02) · **Épico:** E10 · **Relacionadas:** [ADR-0035](0035-plano-versionado-da-feature.md), [ADR-0038](0038-mudanca-de-requisitos-e-troca-de-origem.md), [ADR-0023](0023-checkpoints-integros.md)

## Decisão
1. **Item de fila** = tarefa técnica do plano vigente (`task`) ou correção de PR (`pr_fix`, com a PR como referência). Enfileirar é **idempotente** (índice único entre itens ativos por feature, tarefa, tipo e referência). Só entra tarefa que existe no plano e não foi removida.
2. **Só inicia o elegível**, reconferido no início sob o bloqueio do projeto: projeto **ativo**; **pré-requisitos do plano concluídos** (a evidência é a tarefa em `done`); tarefa ainda pendente; requisito da feature **sem replanejamento ou decisão pendentes** (`requirement_revisions`); **nenhum outro item ocupando a vaga**; e ser o **primeiro da ordem** do projeto. Cada recusa lista os motivos (`NotEligibleError`, `Explain`).
3. **Uma vaga por projeto**, garantida por **índice único** (`slot_held`) além do bloqueio consultivo: dois workers que disputam o mesmo projeto nunca iniciam duas tarefas. **Paralelismo entre projetos:** `Candidates` devolve no máximo um item por projeto, na ordem global.
4. **Ordem:** correções de PR primeiro; depois prioridade (0 a 4, 0 é a mais urgente) e ordem de chegada. A correção de PR ganha a **próxima vaga elegível** do projeto (não preempta quem está rodando).
5. **Esperas representadas:** `merge`, `provider`, `decision`, `resources`, com referência (ex.: URL da PR). A tarefa do plano vai a `blocked` e volta a `pending` ao retornar.
6. **Vaga só volta após checkpoint seguro:** `Block` com checkpoint **íntegro, da execução do item e sandbox quieto** devolve a vaga na hora; sem ele, o item espera **segurando** a vaga até `ReleaseSlot` com um checkpoint seguro. Checkpoint inexistente, corrompido ou de outra execução é recusado (`ErrNotSafe`) e o estado não muda.
7. **Retorno por portas (`Gate`):** `Resume` só devolve o item à fila se a porta do motivo estiver liberada — `DecisionGate` (sem decisão pendente na execução) e `MergeGate` (PR integrada com SHA; fechada sem merge não libera); `provider`/`resources` exigem uma porta injetada, e sem porta o retorno é **recusado** (falha fechada). Ao voltar, o item precisa passar na elegibilidade de novo.
8. `Complete` conclui, devolve a vaga e marca a tarefa como `done` (liberando os dependentes); `Cancel` devolve a vaga.

## Limites
- Não há scheduler, lease/heartbeat/geração, limite global, envelhecimento de prioridade nem controles de ativar/pausar/suspender (T10-03). A prioridade do item é a informada ao enfileirar (não deriva sozinha do projeto/origem).
- Portas de `provider` (disponibilidade, T09-04) e `resources` não estão ligadas; só decisão e merge têm implementação. O verificador de merge real do GitHub é T15-03.
- Sem rotas nem tela; ninguém chama a fila a partir do worker ainda.
