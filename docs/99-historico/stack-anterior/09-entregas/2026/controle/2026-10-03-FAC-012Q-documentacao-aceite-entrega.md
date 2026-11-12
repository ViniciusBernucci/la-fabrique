# FAC-012Q — Documentação e aceite da entrega

Data: 2026-10-03. Status: AWAITING_HUMAN. Baseline `3ff8bdd`, READY `68e81b5`; código `bfd853265360c27f5112063bc7f8ab9ba49b03a9`.

## Checkpoint e diagnóstico

Duas rodadas expuseram fronteiras de validação antes da entrega: lint identificou version como dependência sem leitura na UI; foi corrigido capturando loadedVersion para invalidar aceite de documento antigo. Typecheck identificou que RunStatus PostgreSQL não possui DONE (TicketStatus já possui). Não basta atribuir string no serviço: adicionar estado do enum por migration versionada e regenerar Prisma Client localmente, sem banco.

Uma execução de testes anterior ao build do runtime passou API/contracts, mas cinco suites worker não resolveram `packages/runtime/dist`; diagnóstico é artefato derivado ausente no worktree, não regressão funcional. Build de dependências já recuperado pelo typecheck; todos os checks finais deverão ser repetidos após ajuste do enum. Nenhum resultado dessa rodada equivale a aprovação final.

## Entrega final e funcionamento

GET administrativo `/api/runs/:id/delivery` gera relatório Markdown determinístico a partir de READY imutável e evidências persistidas, ou motivo controlado de inelegibilidade. Inclui objetivo/critérios, base/HEAD/hash do patch (HEAD não é SHA de alterações não commitadas), checks/baseline, review, modelos/uso observados/desconhecidos, artefatos/rollback/limites. Sem prompt, sessão ou token. Textos HTML/brackets são escapados; painel mostra texto, não interpreta Markdown/HTML.

Gate exige latest attempt/fence COMPLETED/parado, checkpoint COMPLETED/parado, resultado/review APPROVE, chamadas Developer/Reviewer concluídas, checks aprovados com baseline e pós sem nova regressão, bundle íntegro/digests/metadados compatíveis e vínculo à especificação READY. Preexistente precisa corresponder à baseline; flag sozinha não basta. Checks/review não asseguram semanticamente todo critério.

POST administrativo `/api/runs/:id/approve-delivery` recebe expectedVersion/attemptId/deliveryDigest estritos e só após ação humana explícita grava approval com documento/hashes/data (sem credencial) e promove run/ticket VALIDATING para DONE numa transação serializável. Digest vincula relatório+artefato+documento; replay exato retorna estado existente sem duplicar versões. Outra tentativa/digest/versão é recusada. Leitura de aceite revalida vínculo e bytes atuais, não confia só em colunas de digest.

Painel carrega documentação sob demanda, oferece Markdown/download/bundle e exige checkbox confirmando critérios/diff/checks/docs antes do botão de aceite. loadedVersion invalida aceite se a versão mudou; requests abortam na troca. Aceite não implica merge/deploy/publicação.

Migration aditiva de runs.approval JSONB e enum RunStatus.DONE somente versionada; Prisma Client regenerado localmente, nenhum banco alterado.

## Checks finais e diff sanitizado

Typecheck, todos os testes, build, lint, diff check e Prisma validate com URL sintética sem conexão passaram. 306 testes: launcher 2, contracts 32, runtime 44, API 101, worker 117, web 10. Lint 177 arquivos; aviso de optional chain normalizado após revisão, seguido de typecheck/API 101 testes/lint novamente. Build API final repetido na revisão de código.

Testes cobrem snapshot imutável, digest/documento, unknown/HTML, preexistente falso/verdadeiro, campos/guard, stale versão/tentativa/conteúdo, evidência ausente/divergente, aprovação idempotente e alteração de bytes após aceite. UI testada por renderização estática; não E2E/browser, DB/HTTP/fila ou serviço real. Falhas intermediárias e diagnóstico estão preservados acima, não eram regressão da baseline.

Diff `git diff 68e81b5 bfd853265360c27f5112063bc7f8ab9ba49b03a9`: 11 arquivos, 867 inserções/1 remoção. Contratos; campo/enum/migration; service/controller/tests/module API; painel/cliente/tests. Sem piloto/conta/modelo hardcoded ou segredo real. Docs/lessons em commit posterior separado.

## Limitações, rollback, docs e aceite

Relatório documenta a entrega, mas não atualiza documentação técnica dentro do repositório externo (ADRs/README/lessons desse projeto). Ainda falta esse gate se o projeto o exigir; painel/relatório avisam. Critérios exigem revisão semântica humana. Pausa/cancelamento/retomada, interrupção/recovery e providers/handoff/identidades continuam pendentes; gate execução false.

Não foram iniciados provider/login, piloto, banco, fila, serviço, push ou deploy. Rollback por revert com writer parado preservando approval/documento/artefatos; não remover coluna/dados. Enum PostgreSQL aditivo não deve ser removido automaticamente. READMEs controle/operação/infra/planejamento, índices/backlog/changelog/controle e lesson baseline/review atualizados. Integração local autorizada após checks/árvore limpa; branch preservada na limpeza. Este ticket da fábrica permanece AWAITING_HUMAN até aceite do SHA exato; não declara MVP completo.
