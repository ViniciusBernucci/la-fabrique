# OPS-008 — Diagnóstico das telas e comandos Prisma com ambiente raiz

Status: AWAITING_HUMAN. Data: 2026-10-05. Baseline: `51196bd`. Branch: developer, diretamente conforme autorização explícita do responsável. Provider elegível: nenhum; checks locais, sem IA/financeiro. Até duas rodadas de correção.

## Objetivo, escopo e critérios

Corrigir os comandos de diagnóstico/aplicação Prisma para carregarem explicitamente o .env raiz, diagnosticar erros HTTP das telas e fornecer favicon. Caminhos: scripts/db.mjs e teste, package.json, apps/web/index.html e public/favicon.svg, documentação infraestrutura/operação/planejamento e lesson existente. Não alterar contratos ou enfraquecer proteção de writer.

1. db:status e db:deploy recebem DATABASE_URL do .env no processo Prisma, sem expor credenciais, sem shell ou migrations automáticas no start.
2. Status não escreve dados; aplicação permanece comando explícito. Sem reset/limpeza/fabricação de stoppedConfirmed.
3. Diagnóstico identifica tabelas/migrations ausentes e evidências históricas que impedem aplicar índice global.
4. Favicon referenciado existe e é servido no desenvolvimento/build.
5. Teste de subprocesso real, lint/build/typecheck adequados, documentação/evidências/diff sanitizado completos. Aceite humano da revisão exata separado; sem DONE automático.

## Baseline observado

Banco dev possui 11 migrations concluídas, incluindo nome antigo `20261002010538_`. Ausentes project_definitions, github_pull_requests e factory_operations. Duas attempts RUNNING/stoppedConfirmed=false impedem índice único global; origem/parada aguardam informação do operador. Health ready testa conectividade, não migrations. Comando anteriormente sugerido node --env-file=.env --run db:deploy reproduziu P1012/DATABASE_URL ausente no processo npm/Prisma, apesar de variável presente no processo Node pai. Nenhuma migration aplicada pelo diagnóstico.

## Entrega parcial

Launcher Prisma/teste e favicon implementados. Correção operacional do banco segue pendente de reconciliação histórica/prova das attempts; sem migrations/stop inferidos. [Evidências](../../../09-entregas/2026/infraestrutura/2026-10-05-OPS-008-bootstrap-migrations-telas.md).
