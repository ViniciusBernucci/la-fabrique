# Entrega 2026-11-12 — E09: contas e clientes CLI de IA (T09-01 parcial)

**Ticket:** [T09-01](../NEW%20MVP/03-BACKLOG.md#t09-01) **parcial**. Decisões: [ADR-0011](../adr/0011-clientes-cli-de-ia-no-sandbox.md). Revisão independente: NOT_RUN.

## O que mudou
- `internal/providers`: `BuildCommand`, `ParseLine`, `Run`, `SandboxRunner`, contas (`Account`) e capacidades.
- `internal/execution`: `Spec.Mounts` (volumes nomeados validados), `EnsureVolume`/`RemoveVolume`, `Sandbox.Start` com saída em fluxo e `Cancel` que encerra a árvore de processos (SIGTERM → SIGKILL).

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| `go test -race ./internal/providers` (comandos, recusas, parsers, resumo, cancelamento; 2 contas e cancelamento em Docker real com CLI de mentira) | PASS |
| `go test -race ./internal/execution` (+ cancelamento de árvore e de SIGTERM ignorado, saída incremental, volumes persistentes e isolados, montagens inválidas) | PASS |
| `make lint`, `make test-integration`, `make check-secrets` | PASS (ver commit) |
| Claude Code e Codex CLI **reais** (login, execução, eventos, cota) | **NOT_RUN** (exigem conta do responsável) |

## Aceite → evidência
- "Duas contas do mesmo provedor são distintas e selecionáveis": `TestTwoAccountsOfTheSameProviderAreDistinct` (volumes e tokens distintos, cada execução enxerga só o seu).
- "Rodar alteração e checks no sandbox com modo autorizado comprovado": **parcial**. Provado que o cliente recebe `--permission-mode dontAsk` + lista de ferramentas e o prompt por stdin, e que modos perigosos não são expressáveis; **não** provado com os CLIs reais nem que uma alteração e os checks rodam por eles.
- Cancelamento: `TestCancellingARunKillsTheClientAndItsChildren`.

## Rollback
`git revert`; sem migration.
