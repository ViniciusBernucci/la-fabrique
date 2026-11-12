# Entrega 2026-11-12 — E13: política de risco no executor (T13-01)

**Ticket:** [T13-01](../NEW%20MVP/03-BACKLOG.md#t13-01). Decisões: [ADR-0012](../adr/0012-politica-de-risco-no-executor.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 8 (`policy_decisions`).
- `internal/policy`: política declarativa com proibições embutidas, política padrão, `Guard`, `GuardedExecutor`, parâmetros canônicos, redação e trilha em PostgreSQL.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| `go test -race ./internal/policy` (scripts, git, API, MCP, conectores, CLI, ordem das regras, compilação, hash canônico, redação, Guard, GuardedExecutor) | PASS |
| `make test-integration` (trilha sem segredo em nenhuma coluna; falha fechada sem trilha; rollback de 8 migrations) | PASS |
| Aplicação no runtime da API/worker | **não ligada** (não há consumidor ainda; `GuardedExecutor` pronto para o motor) |
| Destinos por DNS/redirecionamento | **NOT_RUN** (só o texto do destino é avaliado) |

## Aceite → evidência
"Suíte prova baixo risco permitido, sensível encaminhado e proibido recusado em scripts, API e MCP":
`TestScriptsLowRiskAllowedSensitiveRoutedForbiddenRefused`, `TestAPIAllowlistWritesApprovalAndInternalTargets`, `TestMCPAnnotationsNeverGrantAccess`, mais git, conectores e CLI, e `TestGuardRunsOnlyAllowedActionsAndRecordsEveryDecision` / `TestGuardedExecutorProtectsWorkflowNodes` (a ação não executa quando não é permitida).

Achado durante o teste: a trilha deixava passar um segredo **arbitrário** dentro do texto de um comando; corrigido fazendo a trilha usar o `Redactor` do cofre (valores exatos).

## Rollback
`git revert`. A migration 8 tem `Down` (descarta a trilha).
