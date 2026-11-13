# ADR-0038 — Mudança de requisitos e troca de plataforma de origem

- **Estado:** ACEITA · **Ticket:** [T06-05](../NEW%20MVP/03-BACKLOG.md#t06-05) · **Épico:** E06 · **Relacionadas:** [ADR-0033](0033-sincronizacao-manual-e-preflight.md), [ADR-0035](0035-plano-versionado-da-feature.md), [ADR-0036](0036-retomada-seletiva-e-reconciliacao.md), [ADR-0017](0017-pedidos-e-decisoes-humanas.md)

## Decisão
1. **Revisões do requisito, com histórico:** cada leitura do requisito de uma feature vira uma `requirement_revisions` (conteúdo, hash, impacto, ação, estado, diferença). Nunca se apaga: o que deixa de valer fica `superseded` ou `rejected`. Conteúdo igual ao já conhecido não gera revisão (também não reabre uma decisão já tomada). No máximo uma revisão **aplicada** (vigente) por feature, garantido por índice.
2. **Impacto:** *metadado* (prioridade, rótulos, estado) é compatível; *conteúdo* (título, descrição, dependências) pode invalidar o plano. Mesma classificação da retomada (`resume.Reconcile`); só ordem/espaço não é mudança.
3. **Ação por impacto e por estado do trabalho:**
   - metadado, nenhuma tarefa em execução → **aplicada** na hora, a anterior vira `superseded`;
   - metadado com tarefa em execução → **adiada** até um **marco seguro** (`AtMilestone`, sem tarefa em execução); a adiada mais recente vence e as anteriores ficam substituídas;
   - conteúdo, **nenhuma tarefa iniciada** → `needs_replan`: a vigente fica até o novo plano (`CompleteReplan`); nova leitura antes disso substitui a pendente;
   - conteúdo com **trabalho iniciado** → **conflito**: nada é alterado (plano, requisito vigente e tarefas intactos), a execução vai a `waiting` e abre-se uma decisão `conflict` com a diferença como evidência; opções `keep_work` (a mudança fica `rejected` no histórico) e `replan` (a mudança vira `needs_replan`; a vigente é substituída ao concluir o replanejamento). `Settle` só conclui depois de a decisão estar resolvida.
4. **Troca de plataforma por mapeamento explícito:** o responsável informa `identidade antiga → nova`. A validação (`Switcher.Plan`, sem alterar o projeto) exige: projeto **não ativo**, nova origem completa e diferente da atual, **toda feature mapeada**, nenhuma identidade repetida ("uma identidade remota por item"), sem colisão com a identidade atual de outra feature e **todos os itens novos existentes e legíveis na nova origem**. O resultado fica registrado (`validated` ou `rejected` com os problemas).
5. **Ativação atômica (`Switcher.Activate`):** só de troca `validated`, com o projeto suspenso e sem mudança desde a validação (origem igual, mesmo conjunto de features; senão `ErrSwitchStale`). Numa transação: a identidade antiga de cada feature vai para `feature_identity_history` (com a nova e a troca), as features passam à nova identidade, o estado lido da origem antiga (`source_items`) é descartado (o histórico de sincronizações permanece), a configuração da origem muda (uma única origem por projeto) e a trilha do projeto registra `source_changed`. A nova origem **só ativa o projeto depois de nova verificação remota** (o fingerprint da origem mudou).
6. **Publicação do plano após a troca:** a chave de efeito da subtarefa inclui a identidade remota da feature (`plan:<feature>@<remoto>:<tarefa>`); depois da troca os itens são criados na nova origem em vez de dar o efeito da antiga como feito.

## Limites
- Sem rotas nem tela; nada liga a leitura de origem (T06-04) ao `Handle` automaticamente: quem sincroniza entrega o requisito normalizado. O replanejamento em si (gerar o plano novo) é do planejador (T10-x); aqui só se marca e se conclui.
- A decisão `replan` abre o caminho, mas não dispara o planejador. `Settle`/`CompleteReplan` são chamados por quem observa a decisão e o plano aceito.
- Troca com Jira/ClickUp reais NOT_RUN (origem em memória). Itens da origem antiga (subtarefas, links de PR) permanecem lá; o vínculo é só o histórico de identidades.
- Reabrir uma decisão já tomada exige que o requisito mude de novo.
