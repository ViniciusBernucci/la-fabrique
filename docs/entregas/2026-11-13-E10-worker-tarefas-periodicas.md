# Entrega 2026-11-13 — Worker: tarefas periódicas (T10-02, T12-03, T16-04)

**Decisões:** [ADR-0056](../adr/0056-tarefas-periodicas-do-worker.md). Revisão independente: NOT_RUN.

## O que mudou
- `internal/worker`: `Job`, `AddJob`, `Jobs`, `RunJobOnce`, tarefas `liberar-decisoes`, `recuperar-tentativas` e `retencao`/`retencao-simulada`; `GET /jobs` no endpoint de saúde.
- `internal/recovery/docker.go` (`DockerInspector`), `execution.Docker.ManagedContainers` e `Attach`.
- `cmd/worker` registra as tarefas; `internal/config` ganha `WORKER_JOBS_INTERVAL`, `RETENTION_INTERVAL`, `RETENTION_APPLY` (documentadas no `.env.example`).
- Sem migration.

## Verificação (2026-11-13)
| Verificação | Resultado |
|---|---|
| Unitário `-race` (tarefas rodam periodicamente, falha isolada, `/jobs`) e configuração | PASS |
| Integração PostgreSQL com `-race`: decisão resolvida libera só o seu item; recuperação adia sem inspeção e retoma com inspeção; retenção só simula sem `apply` | PASS |
| Docker real: inspetor distingue sandbox quieto de ocupado | PASS |
| Worker executando tarefas com agentes; inspeção real dentro do contêiner do worker | **NOT_RUN / não feito** |

## Rollback
`git revert`.
