# ADR-0019 — Escrita na origem de tarefas sem duplicação

- **Estado:** ACEITA · **Ticket:** [T06-03](../NEW%20MVP/03-BACKLOG.md#t06-03) · **Épico:** E06 · **Decisões:** D03, D05–06, D50, D85, D88–92

## Decisão
1. **Toda escrita na origem passa pelo ledger** ([ADR-0018](0018-ledger-de-efeitos-externos.md)): subtarefa, dependência, link e mudança de estado têm **chave de negócio estável** (`<projeto>:<feature>:<tentativa>:<tarefa>…`, com o tipo e o alvo no final) e são gravadas como intenção antes do envio.
2. **Prova de existência por operação**, mesmo sem chave de idempotência nas APIs:
   - **subtarefa:** o texto da descrição leva o marcador `[lf:key=<16 hex do SHA-256 da chave>]`; a prova lista as subtarefas do pai (`Tracker.Children`) e procura o marcador (nenhuma → ausente, repetir é seguro; achou → confirmado; falha ao listar → inconclusivo e bloqueio);
   - **dependência:** relê o item e procura o alvo entre as dependências;
   - **link:** `Tracker.HasLink` (Jira: links remotos; ClickUp: comentários);
   - **estado:** relê o item e compara com o estado mapeado.
3. **Revalidar o alvo antes de repetir** (estado): se o item já está no estado desejado, não escreve; se não há transição válida agora, recusa **sem efeito** e sem nova tentativa (`ErrNotApplied`). Estados da fábrica (`in_progress`, `awaiting_merge`, `done`) viram estados da origem por um mapa de projeto; estado sem mapa é recusado.
4. **Draft PR e conclusão:** `PublishDraftPR` anexa o link e leva o item a **aguardando merge**, nunca a concluído. `Complete` só move o item para concluído com **merge confirmado** por um `MergeVerifier` (merge declarado **e** commit de integração); PR aberta, merge sem commit ou falha ao consultar → `ErrMergeNotConfirmed`; PR fechada sem merge → `ErrPRClosed` (o chamador abre uma decisão, T13-02). Nesses casos nada é escrito na origem.
5. **ID remoto associado** à chave de negócio na prova do ledger (`remote_id`), lido por `RemoteIDFor`.

## Limites
- O verificador de merge real é o adaptador GitHub (T15-03/T15-01), ainda inexistente; o contrato e os testes usam um verificador de teste. Os adaptadores Jira/ClickUp foram exercitados só por fixtures e por um destino em memória (Jira e ClickUp reais NOT_RUN).
- A prova de subtarefa depende de listar os filhos do pai (Jira: até 5 páginas de 100); acima disso a prova é inconclusiva por desenho.
- A chave de estado inclui o estado: um novo ciclo legítimo precisa de outro `keyBase` (outra tentativa).
