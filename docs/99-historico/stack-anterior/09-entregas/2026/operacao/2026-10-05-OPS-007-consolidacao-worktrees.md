# OPS-007 — Integração de FAC-012Z–AH na developer

Data: 2026-10-05. IMPLEMENTADO / AWAITING_HUMAN.

## Autorização, baseline e resultado

O responsável autorizou explicitamente reunir todo o restante na developer e apagar as worktrees órfãs. Reiterou a autorização após a identificação de sessões antigas do Codex na raiz e restaurou acesso completo após o sandbox bloquear `.git/ORIG_HEAD.lock`. As sessões antigas não foram encerradas por esta operação; não se afirma parada física delas. Nenhum serviço foi parado ou atualizado.

Baseline developer: `745ce2d25b6120541085b4500e149cc8d313380f`. Código consolidado: `8b42fe4ce858d800ce97fd67cfbccf39cc239874`. Ticket READY/documentação inicial: `d8825ac4cf9ff84b5ebca0276a1464be3de2e3aa`, preparado em branch/worktree isolada OPS-007. Não houve alteração de implementação ou resolução de conflitos: todas as branches formam uma sequência única. Developer recebeu fast-forward, incluindo 27 commits de Z–AH e o ticket de integração.

| Ticket | Branch preservada | Revisão incorporada |
|---|---|---|
| FAC-012Z | feat/fac-012z-documentation-gate | ca41328 |
| FAC-012AA | feat/fac-012aa-claude-confinement | 2edbb91 |
| FAC-012AB | test/fac-012ab-integrated-mvp | a54876f |
| FAC-012AC | fix/fac-012ac-global-writer | 8a47bbf |
| FAC-012AD | feat/fac-012ad-operation-panel | f12b48a |
| FAC-012AE | feat/fac-012ae-project-events | c1aa79a |
| FAC-012AF | feat/fac-012af-evidence-backup | 41ecfa2 |
| FAC-012AG | feat/fac-012ag-factory-pause | 2eced85 |
| FAC-012AH | fix/fac-012ah-admission-capacity | 8b42fe4 |

`git merge-base --is-ancestor` confirmou cada revisão antes e depois da integração. Todas as worktrees estavam limpas, inclusive untracked. Os arquivos ignorados listados eram somente node_modules, dist e tsbuildinfo regeneráveis; nenhum arquivo privado foi identificado nessa listagem. As nove worktrees funcionais foram removidas com `git worktree remove`, sem force, somente após comprovar inclusão em developer. A limpeza inclui também a worktree temporária OPS-007 após incorporar este relatório; a verificação final deve listar somente a raiz.

## Checks e evidências

Na worktree AH, geração local Prisma, typecheck, lint, testes e build completaram sequencialmente. Suíte: 528 testes passaram (scripts 6, contracts 45, runtime 65, API 177, worker 202, web 33). Lint: 234 arquivos, um warning anterior optional chaining em run-delivery.service.ts. Sem correção funcional. Logs locais `/tmp/ops-007-{db-generate,typecheck,lint,test,build}.log`.

A árvore de implementação da developer foi comparada com `8b42fe4` por `git diff --exit-code 8b42fe4 developer -- apps packages scripts package.json package-lock.json`: idêntica. `git diff --check 745ce2d developer` passou. Na raiz já integrada em `d8825ac`, `npm run db:generate`, `npm run typecheck`, `npm run lint`, `npm test` e `npm run build` passaram sequencialmente, sem edição de implementação entre os checks. **528 testes passaram**, com o mesmo warning anterior no lint. Documentação final altera somente Markdown; identidade da implementação com AH é verificada novamente após o commit documental.

Logs locais efêmeros da execução na raiz (SHA-256 dos bytes):

| Check | Log | SHA-256 |
|---|---|---|
| db-generate | /tmp/ops-007-root-db-generate.log | b9db8ee289e1abb2dbce04a7b53979308da787903d5f82363968db4f35f93e68 |
| typecheck | /tmp/ops-007-root-typecheck.log | 0d9600179a891af3f294e493af5d694a4e6d4e955a83dfcb7b59cc0d241eae64 |
| lint | /tmp/ops-007-root-lint.log | 8b21cf27bcc19537087a5660fde4f6ce5dd9747bd513863aad3d2247bd8cde75 |
| test | /tmp/ops-007-root-test.log | 7b4a949320f3afcd09ec39cf187c49a9e2e475f9fabaf3c1e24e5d06431eec78 |
| build | /tmp/ops-007-root-build.log | 863bd4c0cee4109b545a9ab2a0ebcf8e6e019d095b24c704886570f1b85478b8 |

Diff revisável/sanitizado: `git diff 745ce2d25b6120541085b4500e149cc8d313380f 8b42fe4ce858d800ce97fd67cfbccf39cc239874 -- apps packages scripts documentacoes lessons README.md BACKLOG.md CHANGELOG.md` e `git show d8825ac -- documentacoes/planejamento/OPS-007-consolidacao-worktrees.md`. O delta funcional/documental de Z–AH soma 113 arquivos, 6488 inserções/232 remoções. Não inclui arquivos ignorados ou credenciais.

## Limites, rollback, documentação e aceite

Integração local não promove FAC-012Z–AH a DONE: todos permanecem AWAITING_HUMAN. Provider elegível: nenhum; zero chamadas IA, autenticação, gasto ou consumo de assinatura nesta integração. Não houve push, deploy, migration em banco existente ou alteração de serviços. Testes PostgreSQL/Redis adicionais têm evidências anteriores no relatório AH e não foram repetidos aqui. A geração Prisma só atualiza o cliente local.

Rollback requer ticket autorizado e writers parados: reverter de forma revisada os commits integrados, preservando evidências/dados. O SHA anterior `745ce2d` e as branches funcionais continuam disponíveis; não executar reset destrutivo automaticamente. Worktree removida pode ser recriada a partir da branch preservada com `git worktree add <novo-caminho> <branch>`.

Atualizados README raiz/operação/planejamento, relatório, ticket, índices, backlog/changelog e lesson existente sandbox/worktree/snapshot com exemplo de ancestry/fast-forward/remoção segura. ADRs/contratos mantêm conteúdo integrado de Z–AH; nenhuma nova decisão arquitetural. Aceite humano da revisão documental final continua pendente, sem inferência de aceite a partir da autorização de merge.
