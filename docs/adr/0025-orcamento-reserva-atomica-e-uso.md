# ADR-0025 — Orçamento: reserva atômica de custo, uso e cota de assinatura separados

- **Estado:** ACEITA · **Ticket:** [T09-03](../NEW%20MVP/03-BACKLOG.md#t09-03) · **Épico:** E09 · **Relacionadas:** [ADR-0013](0013-catalogo-de-agentes-e-sobrescritas.md), [ADR-0021](0021-modelos-e-gateways-de-api.md)

## Decisão
1. **Unidade:** micro-USD (1e-6) em inteiros (`bigint`); nunca ponto flutuante.
2. **Orçamentos:** um **global** (obrigatório para gastar em API) e, opcionalmente, um **por agente**, cada um por período `day` ou `month` (UTC). Sem orçamento global ativo, a reserva é recusada (`ErrNoBudget`, falha fechada); orçamento só de agente não substitui o global. Modelo sem preço (`input/output_micros_per_mtok` + `price_source`) não pode ser orçado (`ErrNoPrice`).
3. **Reserva atômica antes da chamada:** `Reserve` abre transação, bloqueia as linhas dos orçamentos aplicáveis (`FOR UPDATE`, em ordem de `id`, sem deadlock), confere **gasto do período + reservas ativas e não vencidas + pedido ≤ limite** em cada um e só então grava. Quem disputa o mesmo orçamento é serializado: a soma nunca passa do teto (provado com 40 goroutines sob `-race`).
4. **Liquidação:** `Settle` troca a reserva pelo gasto real (idempotente; liquidar de novo devolve a mesma liquidação). Gasto acima do reservado é registrado como `overrun` (não se "desgasta") e passa a contar nas próximas reservas. `Release` devolve a reserva de uma chamada que falhou; reserva abandonada **expira** (TTL, padrão 5 min, máx. 1 h) e deixa de contar sem tarefa de limpeza.
5. **Estimativa sem métrica oficial:** tokens de entrada ≈ `ceil(bytes/3)` por texto (+16 por mensagem), saída = teto da chamada (`max_tokens` do modelo, senão saída máxima, senão 4096); custo arredondado **para cima**. Liquidação sem consumo informado pelo gateway usa o valor reservado (`source=estimated`, pessimista); com consumo, `provider_reported`.
6. **Assinatura é outra coisa:** `usage_records.kind = 'subscription'` guarda unidades de cota (ex.: mensagens), **nunca custo** — o banco impõe (`CHECK`): assinatura sem custo, API sem cota. Assinatura não consome o orçamento de API. Cotas de assinatura entram só como **observação** (`quota_observations`: usado, limite, unidade, quando reinicia, **fonte e horário observados**), nunca estimadas.
7. **Reconciliação:** `Reconcile` compara o gasto registrado da conta no período com o total do provedor. Faltou registrar → lança ajuste (`source=reconciliation`) para o total bater; registramos a mais → guarda a diferença negativa e **mantém o valor maior** (conservador). Toda conferência fica em `usage_reconciliations`.
8. **No laço do gateway:** `gateways.Meter` reserva antes de **cada** chamada paga; reserva negada ⇒ nada é enviado; falha do gateway ⇒ libera; sucesso ⇒ liquida pelo consumo real. `budget.Meter` implementa (conta de API ou de assinatura).
9. **API:** `GET /api/budgets` (com gasto, reservas e saldo), `POST /api/budgets` (upsert; exige reautenticação) e `GET /api/usage` (API e assinatura em seções separadas).

## Limites
- Preços e consumo vêm do cadastro manual e do que o gateway informa; **gateways reais NOT_RUN** (a prova usa gateway de mentira). A conferência com créditos reais de OpenRouter/Vercel é do adaptador (não implementada: `Reconcile` recebe o total já obtido).
- Janela de reserva: reservas ativas contam inteiras, independentemente do período em que foram criadas (conservador).
- Limites globais de **concorrência/capacidade** são do T10-03; aqui só dinheiro.
- Tela do painel não incluída.
