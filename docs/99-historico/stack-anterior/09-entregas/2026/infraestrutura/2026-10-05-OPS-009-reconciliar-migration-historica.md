# OPS-009 — Reconciliação da migration histórica no banco dev

Data: 2026-10-05. IMPLEMENTADO / AWAITING_HUMAN. Baseline `3aefb96`, developer diretamente autorizada. Operação somente no banco dev localhost:5432; sem provider/IA/gasto, push/deploy de serviços, reset ou dados de aplicação alterados.

## Problema e evidência anterior

Responsável chamou db:deploy: criação github_pull_requests completou; migration 20261002010538_reconcile_prisma_schema_drift falhou P3018/42P01 porque o índice antigo não existia. Registro histórico 20261002010538_ já estava concluído. Query read-only de information_schema.columns comprovou os quatro UUID id em attempts/checkpoints/provider_verifications/runs sem defaults. pg_indexes confirmou ausência do índice antigo e índice novo válido com colunas host,owner,repository,base_branch,created_at. Todos os efeitos da SQL atual já estavam presentes; não repetir nem substituir índice.

## Ação e checks

Executado `node --env-file=.env apps/api/node_modules/prisma/build/index.js migrate resolve --applied 20261002010538_reconcile_prisma_schema_drift --schema apps/api/prisma/schema.prisma`. Ambiente carregado no processo Prisma direto, sem imprimir token/senha. Resolve registrou nome atual como aplicado e registro da tentativa falha como rolled_back; histórico antigo mantido. Query posterior confirmou **zero migrations em falha ativa**. Logs locais efêmeros `/tmp/ops-009-resolve.log` e `/tmp/ops-009-status.log`.

`npm run db:status` agora reconhece a migration atual como último ponto comum; nove migrations seguem pendentes e nome histórico antigo continua registrado. A divergência residual é histórica, não uma nova falha ativa. Confirmado github_pull_requests presente, project_definitions/factory_operations ainda ausentes. Duas attempts sem stop permanecem inalteradas. Não executado deploy subsequente: índice global não pode ser instalado sobre duas tentativas sem confirmação. Nenhum stoppedConfirmed fabricado.

## Limites, rollback e documentação

Apenas o P3018 relatado foi reconciliado; não declara atualização completa das telas/banco. Próximo passo exige evidência externa da origem/parada das duas tentativas e reconciliação autorizada, antes de instalar proteção global. Nome antigo não foi apagado para esconder histórico. Sem mudança em código SQL/schema/contratos/ADRs; não necessário repetir testes de implementação da OPS-008 por alteração somente documental e metadata Prisma. Verificação adequada foi por estado real do schema e histórico, além de diff-check documental.

Rollback não apaga linhas de histórico ou dados: se evidência for contestada, bloquear novas migrations e diagnosticar em ticket autorizado. Preservar logs/histórico; nenhuma reversão automática da metadata. Atualizados README raiz/infraestrutura/planejamento, índice, backlog/changelog e lesson baseline/review. Diff sanitizado é o commit documental OPS-009 sobre esses caminhos, sem .env ou export de dados. Aceite humano da revisão exata pendente, não DONE.
