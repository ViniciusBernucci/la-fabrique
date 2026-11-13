# ADR-0051 — Observação do merge e propagação da conclusão

- **Estado:** ACEITA · **Ticket:** [T15-03](../NEW%20MVP/03-BACKLOG.md#t15-03) · **Épico:** E15 · **Relacionadas:** [ADR-0049](0049-documentacao-viva-e-draft-pr.md), [ADR-0019](0019-escrita-na-origem-sem-duplicacao.md), [ADR-0039](0039-fila-local-por-projeto.md)

## Decisão
1. **Consulta só sob demanda (D89):** `Completer.Observe` lê a PR no GitHub (`GitHubAPI.GetPR`, via `gitops.MergeVerifier`) apenas quando chamado; não há polling. Consulta que falha (rede, limite, permissão) deixa tudo como está e fica registrada como `unknown` em `delivery_observations`.
2. **O que cada estado faz:**
   - **aberta** (rascunho ou não): nada é liberado;
   - **fechada sem merge:** a entrega vira `closed`; nada é liberado nem concluído na origem (a decisão humana é o T15-04);
   - **"integrada" sem o SHA do merge:** sem prova, nada é liberado (`unknown`);
   - **integrada com SHA:** a entrega vira `merged` com o SHA e o horário, o item é concluído na origem pelo ledger (`tasksync.Publisher.Complete`) e os itens da fila que esperavam **esse merge** (motivo `merge`, referência = URL da PR) voltam à fila pela porta `MergeGate`, que reconfere a prova.
3. **Falha de status visível e reconciliável:** a liberação não depende da projeção na origem. Se concluir o item na origem falhar (ou ficar inconclusivo), a entrega guarda `projection = failed` com o erro (`Status` mostra) e `Reconcile` (ou nova observação) tenta de novo; o ledger garante que não se conclui duas vezes. Já `done`, repetir não chama a origem.
4. **Auditoria:** cada observação (`open`, `closed`, `merged`, `unknown`) é gravada com o detalhe; o `merged` é registrado uma única vez.

## Limites
- Sem GitHub real (a API dos testes é um servidor HTTP de mentira) e sem rotas/tela. Nada chama `Observe` a partir do worker ou do painel ainda; o acionamento é o do ponto D89 (sync manual ou preflight), que fica para a ligação.
- Dependentes são os itens da fila em espera por esse merge; dependência entre features diferentes e a promoção de memória no merge são T15-05.
- O fechamento sem merge só marca a entrega; as três ações (cancelar, reabrir, nova tentativa) são T15-04.
