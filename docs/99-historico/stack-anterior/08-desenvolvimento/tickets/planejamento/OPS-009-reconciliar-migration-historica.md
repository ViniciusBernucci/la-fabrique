# OPS-009 — Reconciliar migration histórica no banco de desenvolvimento

Status: AWAITING_HUMAN. Data: 2026-10-05. Baseline: `3aefb96`, developer diretamente conforme autorização do responsável. Provider elegível: nenhum. Escopo: registro Prisma da migration 20261002010538_reconcile_prisma_schema_drift e documentação infraestrutura/planejamento; sem reset/dados de aplicação/stop presumido. Até duas rodadas de correção.

## Problema, autorização e critérios

Responsável executou db:deploy para atualizar banco dev e reportou P3018/42P01. Histórico antigo contém 20261002010538_; migration atual falhou ao renomear índice já renomeado. Confirmar os quatro UUID sem defaults e definição exata do índice (host,owner,repository,base_branch,created_at), nome novo presente e antigo ausente. Só após essa evidência usar migrate resolve --applied no nome atual, preservando histórico antigo e registros de aplicação. Verificar ausência de migration em falha ativa. Não executar próximo deploy enquanto duas attempts RUNNING sem stop impedirem proteção de writer único. Docs/checks/rollback completos; não DONE sem aceite humano exato.

## Resultado

Todos os efeitos já presentes confirmados; migrate resolve --applied executado. Zero falhas ativas; nove migrations restantes pendentes. [Relatório](../../../09-entregas/2026/infraestrutura/2026-10-05-OPS-009-reconciliar-migration-historica.md).
