# OPS-008 — Comandos Prisma e diagnóstico das telas

Data: 2026-10-05. IMPLEMENTADO parcialmente / AWAITING_HUMAN. Baseline `51196bd`; revisão do código `450a56c00e5314069df7af0a3cbc7018f360d52b`; developer diretamente conforme autorização do responsável. Provider elegível: nenhum; zero chamadas IA/gastos, sem push/deploy/alteração de serviços. Execução real permanece desabilitada.

## Problema e funcionamento

Reprodução no banco de desenvolvimento: GET definition e github/pull-requests falham com schema antigo; GET operation falha ao consultar factory_operations. Query read-only confirmou ausência de project_definitions/github_pull_requests/factory_operations. Há 11 migrations concluídas; uma se chama 20261002010538_, ausente da árvore atual. Código tem 21 migrations e 11 nomes não aplicados devido à substituição do nome antigo. Dois registros RUNNING/stoppedConfirmed=false impedem instalar o índice de writer único. Origem/prova de parada aguardam informação do operador; nenhuma evidência foi alterada.

Comando antes indicado `node --env-file=.env --run db:deploy` reproduziu P1012: variável DATABASE_URL ausente no processo npm/Prisma, apesar de presente no Node pai. O .env foi carregado sem mostrar tokens. Nenhuma migration foi aplicada. `scripts/db.mjs` carrega o .env raiz e chama Prisma por process.execPath/argv/env explícitos, resolvendo o pacote a partir de apps/api (instalações locais ou hoisted), sem shell. Comandos fixos status/deploy/migrate; reset e outros não são aceitos. Deploy/migrate continuam manuais; npm run dev não aplica schema automaticamente.

`npm run db:status` lê histórico e encerra não-zero quando há pendências/divergência. Erro não é ocultado nem interpretado como sucesso. `db:generate` permanece inalterado. O painel referencia favicon.svg próprio em public; não depende de favicon.ico inexistente.

## Verificações e diff sanitizado

- Teste novo de subprocesso real: 1 passou, confirma ambiente do .env num filho com cwd diferente e recusa reset; não usa banco real ou credenciais.
- `npm run db:status`: diagnóstico esperado de histórico divergente/11 nomes pendentes, sem P1012. Log local `/tmp/ops-008-status.log` contém endereço do banco dev, sem token/senha.
- Query read-only PostgreSQL comprovou tabelas ausentes, nome antigo da migration e 2 attempts RUNNING sem stop. `/health/ready` OK não comprova schema: a consulta atual testa SELECT 1.
- `GET /favicon.svg` via Vite: HTTP 200.
- Primeira rodada de correção: resolução inicial pelo .bin raiz falhou porque Prisma está instalado no workspace API; resolvido via createRequire do package API. Lint inicial exigiu title no SVG; corrigido. Nenhuma mudança de implementação existente da API/worker.

Checks finais: `npm test` passou com **529 testes** (scripts 7, contracts 45, runtime 65, API 177, worker 202, web 33); `npm run typecheck`, `npm run build` e `git diff --check` passaram. `npm run lint` passou com somente um warning anterior optional chaining. Logs efêmeros `/tmp/ops-008-{tests,typecheck,build,lint}.log`. Checks executados sobre a implementação final; alterações posteriores somente Markdown. Diff sanitizado: `git diff 51196bd -- package.json scripts/db.mjs scripts/db.test.mjs apps/web/index.html apps/web/public/favicon.svg documentacoes README.md BACKLOG.md CHANGELOG.md lessons`; nenhum .env/token/banco exportado no patch.

## Limites, rollback, documentação e aceite

Não declara as telas corrigidas: a atualização operacional do banco continua pendente. Antes de db:deploy, reconciliar por evidência o nome antigo e garantir que o índice global possa ser aplicado. Não executar reset, apagar attempts ou marcar stoppedConfirmed por suposição. Worker ativo no modo execution=false não é prova de parada de tentativa histórica. Nenhuma migration foi aplicada nesta entrega; nenhuma tabela, índice, evento ou evidência removida. Histórico e dados persistidos preservados.

Rollback: reverter launcher/package e favicon em revisão autorizada; isso não resolve banco antigo e não altera dados. Documentação infraestrutura/operação/planejamento, README raiz, índice, backlog/changelog e lesson baseline/review atualizados. Não houve mudança arquitetural/contratual. Aceite humano da revisão exata e correção operacional do banco pendentes; não DONE.
