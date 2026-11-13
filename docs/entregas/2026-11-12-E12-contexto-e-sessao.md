# Entrega 2026-11-12 — E12: contexto seletivo e renovação de sessão (T12-02)

**Ticket:** [T12-02](../NEW%20MVP/03-BACKLOG.md#t12-02). Decisões: [ADR-0030](../adr/0030-contexto-seletivo-e-renovacao-de-sessao.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 21 (`context_packs`, `agent_sessions`, ligados a `checkpoints`).
- `internal/contextpack`: `Build` (contexto determinístico com manifesto, limites, omissões e redação), `Pack.Verify`, `Decide` (oficial vs estimativa), `Store.Renew` e `Store.Resume`.
- Sem rotas nem tela.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Unitários (determinismo, segredos, limite/omissão, truncamento UTF-8, obrigatória, adulteração, decisão oficial/estimada) | PASS |
| Integração PostgreSQL com `-race` (renovar e retomar, adulteração de pacote e de checkpoint, checkpoint de outro projeto/corrompido, troca de modelo, estimativa registrada, corrida de sequência) | PASS |
| Ligação ao worker; memória real; rotas/tela | **NOT_RUN / não feitos** |

## Aceite → evidência
- **Nova sessão retoma com artefatos e contexto verificáveis:** `TestNewSessionResumesWithVerifiableArtifactsAndContext` (pacote idêntico, checkpoint e artefato conferidos, checkpoint posterior sinalizado) e `TestResumeRefusesTamperedPackOrCheckpoint`.
- **Estimativa sem métrica oficial:** `TestDecideOfficialAndEstimate` e `TestRenewRecordsEstimateWhenThereIsNoOfficialMetric` (método e tokens gravados).
- **Ausência de segredos:** `TestBuildRemovesSecretsBeforeHashing` e o teste de retomada (valor registrado no cofre não aparece no pacote gravado).

## Rollback
`git revert`. A migration 21 tem `Down`.
