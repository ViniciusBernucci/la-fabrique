# ADR-0037 — Painel de consumo, orçamento e provedores

- **Estado:** ACEITA · **Ticket:** [T16-03](../NEW%20MVP/03-BACKLOG.md#t16-03) · **Épico:** E16 · **Relacionadas:** [ADR-0025](0025-orcamento-reserva-atomica-e-uso.md), [ADR-0031](0031-fallback-e-disponibilidade-de-contas.md)

## Decisão
1. **Uma tela, três leituras** (`/usage`, item "Consumo" do menu): tetos de gasto em API, uso do mês e provedores. Reaproveita `GET /api/budgets` e `GET /api/usage` (T09-03) e acrescenta `GET /api/providers`.
2. **Gasto em API e assinatura nunca se somam:** o gasto aparece em dinheiro (micro-USD → US$), com a **origem do valor** (reportado pelo provedor, estimado, ajuste de reconciliação), reservas e o excedente sobre o estimado; a assinatura aparece em unidades próprias por conta.
3. **Valor não observado aparece como tal:** `GET /api/providers` devolve `quota: null` quando nenhuma cota foi observada e a tela diz "Cota não observada: o provedor ainda não informou"; consumo de assinatura sem cota observada leva "Sem limite observado"; conta de API diz que não tem cota de assinatura. Nenhum valor é inventado nem mostrado como zero.
4. **O bloqueio de novas chamadas é visível:** teto com saldo zero (gasto + reservas) mostra "Bloqueia novas chamadas"; sem teto global ativo a tela avisa que as chamadas em API são recusadas (falha fechada de T09-03). Tetos por agente mostram o nome do agente.
5. **Disponibilidade com causa e sinal:** por conta de IA (CLI e API), o estado (disponível, autenticação, cota, indisponível, financeiro), a fonte (oficial ou histórico), o motivo e quando volta a ser tentada (ou "só volta após revalidação"), vindos de `availability.Tracker.Get` (T09-04). Conexões que não são de IA não aparecem; nenhum segredo trafega.
6. **Somente leitura:** definir ou alterar teto continua sendo `POST /api/budgets` (sensível, reautenticação); a tela não grava nada.

## Limites
- Sem edição de tetos pela tela e sem seletor de período (mês corrente); sem gráficos.
- Sem ligação a provedores reais (OpenRouter/Vercel/CLIs): os números refletem o que o ledger e os adaptadores registrarem.
- Os agentes aparecem por nome apenas quando o catálogo carrega; caso contrário, pelo prefixo do ID.
