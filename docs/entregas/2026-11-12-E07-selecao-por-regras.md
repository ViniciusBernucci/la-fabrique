# Entrega 2026-11-12 — E07: seleção de fluxo e subfluxo por regras (T07-04)

**Ticket:** [T07-04](../NEW%20MVP/03-BACKLOG.md#t07-04). Decisões: [ADR-0028](../adr/0028-selecao-de-fluxo-por-regras.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 19 (`selection_rules`, `selection_defaults`, `selection_traces`).
- `internal/selection`: normalização, regras ordenadas, `Select`, `Simulate`, `Traces`.
- API: seleção (GET/PUT), simulação e rastros; contrato em `contracts/openapi.yaml`.
- Correção no T07-03: o snapshot de um fluxo sem agentes nem skills era gravado como `null` e violava o `CHECK`; agora é sempre um array.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL (duas regras, nenhum match, padrão, falha explícita, reordenação, determinismo ×20, normalização, simulação sem rastro, validação atômica, destino não publicado, API) | PASS |

## Aceite → evidência
- **Duas regras e nenhum match retornam a escolha determinística, com motivo persistido:** `TestRulesPickTheFirstMatchTheDefaultOrFailExplicitlyAndPersistTheReason`.
- Simulação, validação e API: `TestSimulationExplainsEveryRuleWithoutRecordingAndDisabledRulesAreSkipped`, `TestSelectionConfigValidationAndPublishedTargets`, `TestSelectionAPI`.

## Rollback
`git revert`. A migration 19 tem `Down`.
