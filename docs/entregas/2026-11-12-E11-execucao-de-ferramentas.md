# Entrega 2026-11-12 — E11: execução de ferramentas com limites (T11-02)

**Ticket:** [T11-02](../NEW%20MVP/03-BACKLOG.md#t11-02). Decisões: [ADR-0022](../adr/0022-execucao-de-ferramentas-confinada-e-com-identidade.md). Revisão independente: NOT_RUN.

## O que mudou
- `internal/tools`: confinamento por resolução de links simbólicos nas ferramentas de arquivo (`ErrEscape`), `Identity` no contexto, `ErrIdentity` e `actor` na trilha da política.
- `internal/gateways`: `RegistryExecutor` aceita `Identity`.
- Sem migration, sem mudança de API.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| `go test -race ./internal/tools` (16 fugas por link/raiz/cadeia/relativo/pendente/socket em 6 ferramentas; links internos; busca sem seguir links; traversal; socket e `docker` recusados pela política e ausentes no container; identidade) | PASS |
| `go test -race ./internal/providers` (modo CLI: trabalho no workspace, rootfs RO, traversal, sem socket, sem docker, sem root) | PASS |
| `go test -race ./internal/gateways` (modo API: ferramentas autorizadas + 6 fugas falham, modelo só recebe o erro) | PASS |
| Claude Code / Codex CLI reais, gateways reais | **NOT_RUN** |
| `git_*` em repositório real | **NOT_RUN** (imagem sem git) |
| Isolamento do host em Linux | **NOT_RUN** (Docker Desktop no macOS) |

## Aceite → evidência
- **Ferramentas autorizadas funcionam em ambos os modos de IA:** `TestAPIModeAuthorizedToolsWorkAndEscapesFail` (API) e `TestCLIModeWorksInTheWorkspaceAndCannotEscapeTheContainer` (CLI), com executor/CLI de mentira.
- **Path traversal, symlinks e socket falham:** `TestSymlinksCannotLeaveTheWorkspaceInAnyFileTool`, `TestSearchDoesNotFollowLinksInsideTheTree`, `TestTraversalAndRuntimeSocketAreUnreachable`.
- **Chamadas brokeradas com identidade e política por operação:** `TestBrokeredCallsRequireIdentityAndRecordTheActor`, `TestLocalToolsRecordTheActorWhenIdentified`.

## Rollback
`git revert` (sem migration).
