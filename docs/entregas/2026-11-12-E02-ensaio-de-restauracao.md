# Entrega 2026-11-12 — E02: verificação e ensaio de restauração (T02-04)

**Ticket:** [T02-04](../NEW%20MVP/03-BACKLOG.md#t02-04). Decisões: [ADR-0032](../adr/0032-verificacao-de-restauracao.md). Revisão independente: NOT_RUN.

## O que mudou
- `internal/restorecheck` (sete conferências) e `cmd/verify-restore`.
- `scripts/restore-drill.sh` e `make restore-drill`.
- `security.AdminTOTPAAD` exposto para a conferência do TOTP.
- Sem migration.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Integração PostgreSQL com `-race`: banco íntegro passa; chave errada reprova cofre e TOTP sem vazar valor; adulteração de objeto de checkpoint, arquivo de skill, pacote de contexto e migration é apontada só na conferência certa | PASS |
| `make restore-drill` em dev (Docker real): backup 40 kB com SHA conferido, restore isolado, banco de teste descartado | PASS na mecânica; a verificação **reprovou de verdade** num dev sem migrations (versão 5 de 22) e, depois de `make migrate`, só por **não haver administrador** (nenhum foi criado, para não inventar credencial) |
| Restore a partir do snapshot da VPS; execução em produção; rotação de chave | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Restaurar em destino isolado:** `restore-drill.sh` (banco `restore_drill_<UTC>`, descartado ao fim).
- **Sessão/configuração, desencriptar conexões, banco e hashes de artefatos:** `TestRestoredDatabaseIsVerifiedForKeysConnectionsAndArtifactHashes`.

## Rollback
`git revert` (sem migration).
