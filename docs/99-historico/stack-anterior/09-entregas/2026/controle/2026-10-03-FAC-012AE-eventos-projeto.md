# FAC-012AE — Eventos autenticados e retomáveis de projeto

Data: 2026-10-03. IMPLEMENTADO / AWAITING_HUMAN. Código `9173532`; baseline `f12b48a`, READY `7b7d033`. Branch `feat/fac-012ae-project-events`, worktree exclusiva `/home/vinicius/le-fabrique-fac-012ae`, writer único. Piloto/provider não são dependências; não DONE automático.

## Funcionamento e contrato

PostgreSQL guarda project_event_cursors e project_events. Triggers de tickets, runs, tentativas e checkpoints publicam invalidação minimizada com projeto, entidade, tipo, sequência e horário na mesma transação da mudança. Renovação de lease isolada não gera evento. Não transportam texto do agente, resultados, arquivos, comandos, tokens ou credenciais. Não são autorização nem auditoria de cada ferramenta; cliente consulta o estado HTTP atual.

Sequência bigint por projeto usa incremento em linha bloqueada até commit, não nextval: uma transação posterior do mesmo projeto não pode publicar sequência maior antes de uma anterior. Rollback desfaz evento/cursor. Contratos mantêm decimal string para precisão além de Number.MAX_SAFE_INTEGER; rejeitam negativos, notação exponencial, leading zero, overflow PostgreSQL, campos desconhecidos, projeto diferente e cursor regressivo/incoerente.

GET /api/projects/:projectId/events?after=0 consulta até 100 eventos ordenados e devolve nextCursor; GET .../events/stream é SSE com AdminAuthGuard, Last-Event-ID prevalece sobre query after. Poll de persistência sequencial 1 s, drena páginas cheias sem esperar e emite heartbeat sem ID quando vazio. Desconexão cancela timer e descarta resposta pendente; erros internos do stream são genéricos. Eventos antigos não são inventados para registros pré-migration; a leitura HTTP inicial preserva estado atual.

React abre fetch streaming com Bearer no cabeçalho, nunca URL. Decoder bounded 64 Ki caracteres de buffer/16 Ki por frame, UTF-8 estrito, CRLF fragmentado, identidade/ID/schema validados. Duplicatas/regressos são descartados sem recuar cursor. Reconnect 2 s retoma o último evento entregue; 15 s sem bytes aborta conexão. Cleanup aborta requisição/leitor/timers e impede atualização tardia. Mudanças invalidam consultas de tickets/runs/detalhe sem perder seleção. Poll HTTP sequencial 10 s permanece fallback; não depende da stream para progredir. Proxy nginx dessa rota desativa buffering/cache e mantém heartbeat/timeout.

## Verificações reais e diff

- npm ci --ignore-scripts e db:generate passaram; geração sem banco existente.
- npm run typecheck, npm test, npm run lint, npm run build e git diff --check passaram. Checagem strict adicional da nova spec API passou (specs API ficam fora do tsconfig de produção).
- Suíte 485 passou: scripts 5, contracts 45, runtime 50, API 163, worker 190, web 32. Novos testes verificam input hostil, precisão, autenticação HTTP/SSE, frame real em localhost, sequential/cancel/reconnect/cursor/identidade/limites; portas banco no teste HTTP são mocks.
- npm run test:postgres: 7 passaram/0 skipped no container exclusivo network none/tmpfs/512 MiB/1 CPU, todas migrations reais aplicadas. Rollback real não avança cursor/evento; transações concorrentes em tickets distintos do mesmo projeto mantêm incremento contíguo; lease isolada não gera churn e status/stop geram. Provas anteriores do índice global continuam passando.
- nginx -t passou em container exclusivo da imagem local existente com config montada read-only, sem porta ou serviço alterado.
- Lint 223 arquivos, um warning optional-chain preexistente; nenhum novo. Duas rodadas de correção: tipos/supressões explícitas de invalidação por cursor e validação BigInt que lançava em 1e9. A segunda virou rejeição 400 e contratos negativos versionados. Não mascarar a falha inicial como baseline.

Logs efêmeros /tmp/fac-012ae-{tests,postgres,typecheck,lint,build,nginx,events-spec-typecheck}.log. Diff sanitizado: git show 9173532 -- apps packages/contracts infrastructure/nginx/default.conf scripts/global-writer.postgres.test.mjs (15 arquivos, 781 inserções/9 remoções). Sem credenciais; identidades sintéticas nos testes. Container de teste removido.

## Limites, operação, rollback e aceite

Migration aplicada só no PostgreSQL efêmero; nenhum serviço antigo atualizado. SSE oferece invalidations retomáveis, não replay completo de estado intermediário nem exatamente uma entrega. Persistência sem limpeza automática: incluir tabelas/cursors no backup; cliente novo sem cursor relê histórico e consulta HTTP atual. Sem benchmark de conexões/carga, browser real ou serviços remotos. Banco/HTTP sintético não contam como conta/provider real; nenhum cliente/login/gasto/push/merge/deploy.

Rollback preserva eventos/cursors/triggers e dados; frontend anterior usa HTTP normalmente. Reverter backend streaming/proxy mediante ticket autorizado não altera leases ou artefatos. Remoção de tabelas/triggers não é rollback automático. Aceite humano da revisão exata pendente. Próximo desenvolvimento interno: ferramentas de backup/restauração verificável com dados sintéticos, independentemente de piloto.

## Docs e lessons

README controle/operação/infraestrutura/planejamento, ADR-003, handoff, INDEX, BACKLOG, CHANGELOG, CONTROLE-MVP e lesson outbox/idempotência atualizados. Conceito aplicado: sequência alocada fora da transação não prova ordem de commit; eventos servem para invalidar estado consultável, autenticação permanece no cabeçalho e HTTP é fallback.
