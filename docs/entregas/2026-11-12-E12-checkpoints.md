# Entrega 2026-11-12 — E12: checkpoints íntegros (T12-01)

**Ticket:** [T12-01](../NEW%20MVP/03-BACKLOG.md#t12-01). Decisões: [ADR-0023](../adr/0023-checkpoints-integros.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 15 (`checkpoint_objects`, `checkpoints`).
- `internal/checkpoints`: manifesto e hashes, captura do workspace (tar do sandbox), registro de git, `Store` (Save atômico, Get/Latest/Object com reconferência, Prune), `Restore`/`Verify`, `CaptureAndRelease`.
- Sem API nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| `go test -race ./internal/checkpoints` (caminhos, identidade da árvore, captura, limites, tar cortado, git, adulteração do manifesto, tar de restauração) | PASS |
| Integração com PostgreSQL + Docker reais (restauração em container novo, parada com processo vivo, gravação interrompida em 3 pontos, corrupção, escopo por projeto, poda) | PASS |
| `git` real no sandbox | **NOT_RUN** (imagens sem git; HEAD/ramo/patch provados com git de mentira) |
| Integração com o motor/worker, API e tela | **não feitas** |

## Aceite → evidência
- **Restaurar em container novo e comparar hashes:** `TestCheckpointRestoresInANewContainerAndHashesMatch` (`sha256sum` de todos os arquivos igual antes/depois, links, modo executável, diretório vazio, divergência posterior detectada com o caminho).
- **Interromper gravação de snapshot mantém o checkpoint anterior consistente:** `TestInterruptedSnapshotWriteKeepsThePreviousCheckpointConsistent` (queda no meio dos objetos, antes do checkpoint e cancelamento do contexto: o anterior segue idêntico, sem órfãos; o seguinte sai com `seq` 2; o anterior ainda restaura).
- **Liberar só depois da parada confirmada:** `TestCheckpointRefusesToCaptureWhileProcessesAreAlive`.
- Corrupção e escopo: `TestCheckpointCorruptionIsDetectedOnReadAndRestore`, `TestCheckpointsAreScopedByProjectAndPrunedSafely`.

Achado durante o teste: se a leitura do tar falhasse, a captura esperava o processo `tar`, bloqueado num pipe sem leitor, até o timeout de 3 minutos; agora fechar o leitor cancela o comando (teste de 198 s → 2,7 s).

## Rollback
`git revert`. A migration 15 tem `Down` (descarta checkpoints e objetos).
