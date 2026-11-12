# Entrega 2026-11-12 — E15: GitHub, branches, commits e push (T15-01)

**Ticket:** [T15-01](../NEW%20MVP/03-BACKLOG.md#t15-01). Decisões: [ADR-0024](../adr/0024-github-branch-commit-e-push.md). Revisão independente: NOT_RUN.

## O que mudou
- `internal/gitops`: `Connector` (Check/Clone/RemoteRef) e `Effects` (Push) separados; cliente da API do GitHub para o preflight; commit por `git fast-import` com trailers; `Publisher` pelo ledger com prova por ref.
- Sem migration (usa a `outbox` do T13-04), sem rotas nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| `go test -race ./internal/gitops` (validação do repositório, guarda de push, clone real, preflight com API, commit e trailers, nada extraído no host, hooks não executam, credencial fora de argv/erros) com `git` real | PASS |
| Integração PostgreSQL + `git` real (publicação confirmada por ref, timeout após o efeito, timeout antes do efeito, push restrito, divergência bloqueia, ref à frente confirma, `.github/` exige aprovação) | PASS |
| GitHub real (token, permissão de push, proteção do ramo base, rate limit) | **NOT_RUN** |
| Imagem do worker com `git` | **não feita** |
| Rotas/tela, PR e merge | **não feitos** (T15-02/T15-03) |

## Aceite → evidência
- **Conexão valida clone:** `TestConnectionIsValidatedByARealClone`, `TestPreflightUsesRepoAPIWhenAvailable`.
- **Commit corresponde a tarefa e checks:** `TestCommitMatchesTaskAndChecksAndBuildsTheExactTree`, `TestCommitRefusesUnvalidatedWorkAndUnsafeContent`.
- **Timeout de push e prova por ref sem duplicidade ou alteração direta da base:** `TestPushTimeoutIsSettledByRefProofWithoutDuplicates` (1 execução de push, 1 atualização no remoto, repetição ⇒ `already_confirmed`), `TestPushTimeoutBeforeTheRemoteReceivedItIsRetriedSafely`, `TestPushIsRestrictedToTheFeatureBranchAndNeverTouchesTheBase` (zero `git push`, zero intenção, base idêntica), `TestDivergedRemoteBlocksWithEvidenceAndKeepsTheForeignCommit`, `TestRefAdvancedAfterOurPushIsStillConfirmed`.

## Rollback
`git revert` (sem migration).
