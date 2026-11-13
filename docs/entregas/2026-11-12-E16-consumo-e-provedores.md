# Entrega 2026-11-12 — E16: consumo, orçamento e provedores (T16-03)

**Ticket:** [T16-03](../NEW%20MVP/03-BACKLOG.md#t16-03). Decisões: [ADR-0037](../adr/0037-painel-de-consumo-e-provedores.md). Revisão independente: NOT_RUN.

## O que mudou
- **API:** `GET /api/providers` (contas de IA com disponibilidade e a última cota observada), em `contracts/openapi.yaml`.
- **Web:** página `UsagePage` (`/usage`), item "Consumo" no menu, tipos e chamadas em `api.ts`. Só tokens `--hud-*`.
- Sem migration.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL (só contas de IA; cota nula sem observação; cota e falha oficial aparecem; sem segredo) | PASS |
| vitest (75 testes; novos: teto/reserva/origem/não observado/bloqueio/sem teto global) | PASS |
| Biome, tsc, contrato OpenAPI | PASS |
| Navegação em navegador real; provedores reais | **NOT_RUN** |

## Aceite → evidência
- **Valor não observado aparece explicitamente:** `TestProvidersShowAvailabilityAndOnlyObservedQuota`; vitest "mostra teto, reservas, origem do valor, cota não observada e disponibilidade".
- **Teto global e por agente refletem o bloqueio de novas chamadas:** vitest "reflete o bloqueio de novas chamadas quando o teto acabou e quando não há teto global".

## Rollback
`git revert` (sem migration).
