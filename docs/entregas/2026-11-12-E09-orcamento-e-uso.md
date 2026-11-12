# Entrega 2026-11-12 — E09: orçamento e uso (T09-03)

**Ticket:** [T09-03](../NEW%20MVP/03-BACKLOG.md#t09-03). Decisões: [ADR-0025](../adr/0025-orcamento-reserva-atomica-e-uso.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 16 (`budgets`, `usage_records`, `usage_reservations`, `quota_observations`, `usage_reconciliations`; preço em `models`).
- `internal/budget`: orçamentos, `Reserve`/`Settle`/`Release`, uso de assinatura e cotas observadas, `Reconcile`, `Report`, preço/estimativa, `Meter` do gateway.
- `internal/gateways`: `Meter` no laço (reserva antes de cada chamada).
- `internal/models`: preço opcional (com fonte).
- API: `GET/POST /api/budgets`, `GET /api/usage`.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race` (reserva concorrente, liquidação/liberação/expiração, falha fechada e janelas, assinatura separada, reconciliação, preço/estimativa, laço do gateway, API) | PASS |
| `TestConcurrentReservations…` repetido 8× com `-race` | PASS |
| Gateways e créditos reais (OpenRouter/Vercel) | **NOT_RUN** |
| Tela do painel | **não feita** |

## Aceite → evidência
- **Chamadas concorrentes disputam reserva sem ultrapassar teto:** `TestConcurrentReservationsNeverExceedTheAuthorizedCeiling` (40 disputando 1 000 000 em reservas de 100 000: exatamente 10 vencem; com teto por agente: agente ≤ 3 e total = 15 em qualquer intercalação).
- **Cota de assinatura separada de gasto em API:** `TestSubscriptionQuotaIsRecordedSeparatelyFromAPISpend` (assinatura não move o orçamento; relatório com seções separadas; o banco recusa misturar), `TestGatewayLoopEnforcesTheBudgetBeforeEveryCall` (conta de assinatura sem teto de API).
- Falha fechada, orçamento no laço e reconciliação: `TestBudgetFailsClosedAndWindowsRoll`, `TestGatewayLoopEnforcesTheBudgetBeforeEveryCall`, `TestReconciliationAdjustsMissingSpendAndKeepsTheHigherNumber`.

## Rollback
`git revert`. A migration 16 tem `Down` (descarta orçamento e uso; remove as colunas de preço).
