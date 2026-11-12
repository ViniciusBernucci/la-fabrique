# Entrega 2026-11-12 — E08: catálogo de agentes e sobrescritas (T08-01)

**Ticket:** [T08-01](../NEW%20MVP/03-BACKLOG.md#t08-01). Decisões: [ADR-0013](../adr/0013-catalogo-de-agentes-e-sobrescritas.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 9 (`agents`).
- **Backend:** `internal/agents` (catálogo, validação de conta, `Resolve`, `EffectiveForNode`); 5 operações novas; nó `agent` com campos de sobrescrita e validação deles.
- **Front:** menu **Agentes** (cadastro e edição) e "Ver configuração efetiva" no painel do nó de agente.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| `go test -race ./internal/agents ./internal/workflows` | PASS |
| `make test-integration` (CRUD, versão, contas incompatíveis/revogadas, dois workflows com efetivos diferentes e base íntegra, sobrescritas inválidas, CSRF; rollback de 9 migrations) | PASS |
| vitest (59 testes; 6 novos), `tsc`, Biome | PASS (cópia no disco interno) |
| Execução real de um agente com a configuração efetiva | **NOT_RUN** (depende de T09-01 com CLIs reais e do motor ligado) |

## Aceite → evidência
- "Dois workflows escolhem configurações diferentes e catálogo original permanece íntegro": `TestTwoWorkflowsResolveDifferentConfigsAndCatalogStaysIntact` (modelo/tempo num fluxo; conta/ferramentas/instruções no outro; o agente lido depois é idêntico, mesma versão) e `TestResolveKeepsBaseAndAppliesNodeOverrides` (nada compartilhado por referência).
- "Execução mostra configuração efetiva": endpoint de efetivo e visão no editor (com a origem de cada valor). **Limite:** mostra o rascunho salvo, não uma execução congelada (T07-03).

## Rollback
`git revert`. A migration 9 tem `Down` (descarta o catálogo).
