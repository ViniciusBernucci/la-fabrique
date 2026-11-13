# Entrega 2026-11-12 — E16: Central e auditoria operacional (T16-02)

**Ticket:** [T16-02](../NEW%20MVP/03-BACKLOG.md#t16-02). Decisões: [ADR-0054](../adr/0054-central-e-auditoria-operacional.md). Revisão independente: NOT_RUN.

## O que mudou
- `internal/central`: situação das autorizações e sequência do incidente e da retomada.
- API (`contracts/openapi.yaml`): `POST /decisions/{id}/refuse`, `GET /authorizations`, `POST /authorizations/{id}/revoke`, `GET /executions/{id}/sequence`.
- Painel: botão Recusar, aba Autorizações com revogação e sequência sob demanda.
- Sem migration.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race` (aprovar, recusar e revogar pelo HTTP: eventos, estado, auditoria, idempotência, reautenticação, CSRF, sequência ordenada, vencida/esgotada, nenhum efeito externo) | PASS |
| Contrato OpenAPI (rotas declaradas existem) | PASS |
| Vitest (Recusar, sequência, autorizações e revogação) | PASS |
| Navegador real; parada segura com `Halter` real | **NOT_RUN** |

## Aceite → evidência
- **Aprovar, recusar ou revogar pelo painel produz evento e estado verificáveis sem notificação externa:** `TestApprovingRefusingAndRevokingFromThePanelLeaveVerifiableEventsAndState`, `TestCentralRoutesRequireSessionAndCSRF`; tela: `decisions.test.tsx`.

## Rollback
`git revert`.
