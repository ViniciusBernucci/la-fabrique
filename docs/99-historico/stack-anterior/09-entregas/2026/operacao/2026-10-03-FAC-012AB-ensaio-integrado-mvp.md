# FAC-012AB — Ensaio integrado interno do MVP

Data: 2026-10-03. Estado: IMPLEMENTADO / AWAITING_HUMAN; não DONE. Revisão exata de teste/tooling `171434758b12a290ba9314c76942382d7dfb706a`. Baseline `2edbb915d53d06ec52f70d31a53a1e5ee9fa7738`; ticket READY `e26408c`. Branch `test/fac-012ab-integrated-mvp`, worktree `/home/vinicius/le-fabrique-fac-012ab`, writer exclusivo. Stack aprovada preservada. Nenhuma alteração de comportamento de produção.

## Objetivo e funcionamento

O teste `apps/worker/src/mvp.integration.spec.ts` compõe compilador e workflow reais com ContextBuilder, RuntimeGuard, WorkspaceManager, SandboxRunner Linux, SnapshotManager, gate documental, ArtifactReader, ResultJournal e processOrchestrationExecution. Cada cenário cria repositório Git temporário com commit-base exato. Um check Node executado no sandbox mede valor 0 antes e 1 depois da edição. README tracked e relatório Markdown untracked permitem verificar restauração de ambas as classes de arquivo. O Reviewer sintético exige READ_ONLY e lista de escrita vazia.

Controle/lease/checkout remoto e adapters são portas sintéticas tipadas e explicitamente identificadas. Git/worktrees/filesystem, checks confinados, contexto, snapshots/hashes/bundle e journal são reais. O teste exige resultados observados e bytes, não mensagens de sucesso do adapter. Não usa clientes oficiais, contas reais ou rede externa. `typecheck:integration` verifica o ensaio apesar de specs estarem excluídas do tsconfig de produção; integra o comando raiz `npm run typecheck`.

## Evidências por cenário

| Cenário | Resultado observado |
|---|---|
| Entrega normal | Baseline 0 e pós 1 em checks parados; VALIDATING; gate documental PASS ligado ao manifesto final; bundle contém relatório com hash exato; restore em nova worktree reproduz bytes tracked/untracked. |
| Upload interrompido | Journal real preservado; nenhum checkpoint após falha; redelivery entrega e conclui sem novo claim, adapter ou check. |
| Provider sintético sem cota após editar | Parada confirmada, snapshot real, nova worktree e contexto atualizado com valor 1; alternativa sintética explicitamente configurada, uma troca, zero correções. |
| Patch preservado corrompido | Replay recusa integridade, mantém journal e não chama writer/check/checkpoint/complete. |
| Documentação incompleta | PAUSED_LIMIT / DOCUMENTATION_INCOMPLETE com gate FAIL e snapshot preservado; nenhum Reviewer ou entrega aprovada. |

## Checks, baseline e diff sanitizado

Node `v22.23.3`, npm `10.9.9`. Dependências instaladas com `npm ci --ignore-scripts`; `npm run db:generate` apenas gera cliente local, sem migration ou acesso a banco. Build inicial confirmou baseline utilizável. Não foi demonstrada regressão na implementação.

- Ensaio isolado: 5 testes passaram, 4,67 s na execução final isolada; suíte completa também os executou.
- `npm test`: 438 passaram (scripts 5, contracts 34, runtime 50, API 141, worker 190, web 18). Worker: 23 arquivos, 190 testes, 9,16 s.
- `npm run typecheck`: passou; inclui scripts e, na revisão final, integração. Verificação explícita strict da spec também passou.
- `npm run lint`: passou, 210 arquivos, um warning anterior de optional chaining em `run-delivery.service.ts`; nenhum warning novo.
- `npm run build`: cinco workspaces passaram.
- `git diff --check`: passou.

A primeira checagem explícita de tipos da fixture encontrou mocks incompletos de renovação/adapter e role ampla; corrigidos em uma rodada e revalidados. Não houve correção de código de produção. A formatação corrigiu estilo e concatenação do output sintético antes dos checks finais.

Logs locais efêmeros: `/tmp/fac-012ab-integration-final.log`, `/tmp/fac-012ab-tests.log`, `/tmp/fac-012ab-typecheck-final.log`, `/tmp/fac-012ab-lint-final.log`, `/tmp/fac-012ab-build.log`; não são backup permanente. A evidência reproduzível está no teste versionado. Diff sanitizado: `git show 171434758b12a290ba9314c76942382d7dfb706a -- apps/worker/src/mvp.integration.spec.ts package.json tsconfig.integration.json` (3 arquivos, 481 inserções/2 remoções). Fixtures usam conteúdo/identidades sintéticos, sem credenciais. Não registrar auth ou conteúdo de conta.

## Limitações, rollback e aceite

Este ensaio não comprova HTTP/PostgreSQL/Redis/BullMQ reais, identidade de serviço, confinamento nativo de clientes oficiais, elegibilidade financeira, piloto externo, reboot/backup externo ou operação contínua. Prova Claude real continua ausente e gate de execução não foi ativado. Zero chamadas a modelo/cliente oficial; provider/modelo efetivos não medidos. Nenhum login, extra usage, crédito, fallback pago, migration, serviço persistente, merge/push/deploy.

MVP operacional permanece pendente desses gates e aceite humano exato. Root developer segue em `745ce2d`, intacto; branch inclui Z e AA por ancestry. Rollback deste incremento: reverter commit de testes/tooling em ticket autorizado, preservando relatórios/evidências; não requer rollback de banco ou runtime. Este ticket aguarda revisão exata, não promove outros tickets a DONE.

## Documentação e lessons

Atualizados README raiz/operação/planejamento, INDEX, BACKLOG, CHANGELOG, CONTROLE-MVP, ticket e lesson de sandbox/worktree/snapshot. Conceito aplicado: distinguir a prova de composição de módulos reais da prova das portas externas; referência concreta aos testes de replay, corrupção e restore, sem duplicar uma lesson.
