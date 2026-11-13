# ADR-0050 — Retenção com integridade

- **Estado:** ACEITA · **Ticket:** [T16-04](../NEW%20MVP/03-BACKLOG.md#t16-04) · **Épico:** E16 · **Relacionadas:** [ADR-0023](0023-checkpoints-integros.md), [ADR-0048](0048-painel-de-execucao-ao-vivo.md), [ADR-0045](0045-recuperacao-de-tentativas-interrompidas.md)

## Decisão
1. **Logs detalhados: 30 dias.** `execution_events` com mais de 30 dias (configurável) são removidos, mas **antes** o **resumo permanente** da execução é gravado, na mesma transação (`execution_summaries`: total e contagem por tipo dos removidos, primeiro e último horário, `compacted_through` e o **último seq**). `AppendEvent` passou a considerar o último seq do resumo, então o **seq nunca é reaproveitado**, nem depois de remover todos os eventos: os cursores do painel e do SSE continuam válidos.
2. **Checkpoints: carência após a conclusão.** Checkpoints de execuções **concluídas** (concluída com sucesso ou cancelada) saem só depois da carência (padrão 14 dias desde a última atualização da execução). Execução em andamento, falha (não é "concluída") ou dentro da carência não é tocada. Os objetos que ficam sem referência saem junto (`Store.Delete`); os que outro checkpoint usa permanecem.
3. **Protegido continua intacto:** nada é removido de (a) checkpoint com **bloqueio explícito** (`Protect`, com motivo e referência do incidente), (b) checkpoint **usado por sessão do agente**, (c) execução **ligada a incidente**: decisão pendente, estado de espera/incerto/na fila/em execução, efeito externo sem confirmação, tarefa da fila em andamento, em espera ou suspensa. Isso vale para logs e checkpoints. Liberada a proteção (`Unprotect`), o item passa a ser elegível na execução seguinte.
4. **Integridade conferida:** ao final, cada checkpoint mantido das execuções afetadas é relido (hashes de manifesto e estado) e **todos os seus objetos** são conferidos; qualquer falha encerra com `ErrIntegrity`.
5. **Relógio e simulação:** o relógio é injetável (o teste avança ou recua o tempo) e `dryRun` mostra o que sairia sem remover nem gravar resumo (decisão `would_delete`).
6. **Auditoria:** `retention_runs` (instante, prazos, contagens) e `retention_decisions` (uma linha por execução de logs e por checkpoint: removido, "removeria" ou mantido, com o **motivo**).

## Limites
- Nada agenda a retenção sozinho (não há laço no `cmd/worker`); é uma função `Run` para o agendador/operador chamar. Sem rotas nem tela.
- O resumo permanente guarda contagens e horários, não o conteúdo dos eventos removidos; o histórico resumido do projeto (E16-02) e artefatos fora do banco não estão cobertos.
- A carência conta pela última atualização da execução (não há timestamp de conclusão separado); retenção de outras tabelas (auditoria de login, trilhas) não é tratada aqui.
