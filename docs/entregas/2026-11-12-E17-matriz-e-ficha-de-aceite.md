# Entrega 2026-11-12 — E17: matriz de evidências e ficha de aceite (T17-01)

**Ticket:** [T17-01](../NEW%20MVP/03-BACKLOG.md#t17-01). Decisões: [ADR-0041](../adr/0041-matriz-de-evidencias-derivada.md). Revisão independente: NOT_RUN.

## O que mudou
- `internal/acceptance` e `cmd/matriz` (`make matriz`): geram `docs/aceite/02-matriz-de-evidencias.md` (18 casos × 17 épicos × 67 tickets).
- `docs/aceite/01-ficha-de-preparacao.md`: ficha pré-preenchida com o que o repositório prova e as pendências F01–F12 do responsável.
- Teste de rastreabilidade, de deriva da matriz e de cobertura da ficha. Convenção: rodar `make matriz` ao alterar o checklist.
- Sem migration, sem API.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Rastreabilidade (checklist × backlog × casos P × épicos × matriz de decisões), deriva da matriz, cobertura da ficha | PASS |
| Seleção do projeto e da feature reais; contas, VPS e GitHub do piloto | **NOT_RUN: dependem do responsável (F01–F12)** |

## Aceite → evidência
- **Contrato revisável com critérios rastreáveis:** `TestTraceabilityCloses`, `TestTheCommittedMatrixIsTheGeneratedOne`, `TestTheFichaCoversEveryContractField`.
- **Contrato preenchido (ambiente, projeto, usuários, conexões):** parcial — usuários e conexões necessárias estão descritos; ambiente e projeto reais ficam em F02–F06.

## Rollback
`git revert` (sem migration).
