# FAC-012E — Configuração versionada disponível ao worker

## Resultado

Implementação no commit `4aca1e10085f047712766f51ca7e4e749508a7a9`, sobre a base da branch `a24f1fa` (ticket READY), derivada de `developer` em `a6fdf9ce57cc2293ddf310198204ea1093db9522`. Status `AWAITING_HUMAN`; não houve merge, push, deploy, migration nem mudança de settings persistidos.

A API expõe `GET /api/internal/worker-settings`, protegido pelo `WorkerAuthGuard` e pelo token interno já usado pelo worker. Retorna `{version, observedAt, configuration}` validado por `workerConfigurationSnapshotSchema`. Se não existe linha de settings, responde versão `0` e defaults inativos sem executar `upsert`. Configuração persistida retorna sua versão efetiva. O schema é estrito e não contém credenciais.

`ControlClient.getWorkerConfiguration()` consulta o endpoint via GET, envia o bearer em header e valida a resposta novamente antes de retornar ao chamador. O método ainda não está ligado ao consumer, roteamento ou execução: nenhuma conta/modelo foi lido ou iniciado.

## Evidências

- `npm ci`: instalou 211 pacotes; auditoria informou zero vulnerabilidades.
- `npm run db:generate -w @le-fabrique/api`: gerou Prisma Client local para compilar/testar; nenhuma migration aplicada.
- `npm run lint`: passou, Biome em 143 arquivos.
- `npm run typecheck`: passou em contracts, runtime, API, worker e web.
- `npm test`: passou: scripts 2, contracts 21, runtime 40, API 54, worker 57, web 4 (178 no total).
- `npm run build`: passou nos cinco workspaces; Vite production build processou 118 módulos.
- `git diff --check`: passou.
- Testes de unidade validaram versão/default sem persistência, settings persistidas, resposta estrita, autenticação bearer ausente/incorreta e GET do client sem token no body. Prisma e HTTP foram mocks; nenhum serviço real foi consultado.

## Diff sanitizado

- `packages/contracts`: novo contrato snapshot estrito com versão não negativa, timestamp e configuração sem segredos.
- `apps/api/src/settings`: leitura não mutante, rota worker-only e integração no módulo.
- `apps/api/src/worker-identity`: teste do guard de bearer interno.
- `apps/worker/src/control-client`: método de leitura/validação do snapshot e teste do transporte.
- READMEs, backlog, changelog, índice, planejamento e lesson de contratos atualizados. Não há arquivo de configuração com contas/modelos fixos.

## Limitações e próximos passos

1. Consumer BullMQ ainda executa o probe sintético; este método não foi ligado ao loop.
2. Rota configurada ainda não é elegibilidade operacional: login/preflight na identidade real, cobrança/extras e frescor do estado `AVAILABLE` permanecem gates separados.
3. Checkout/materialização confiável do repositório, perfil local de checks/limites, execução via DeveloperWorkflow e persistência/checkpoints precisam de tickets próprios. A escolha do piloto segue manual.

## Rollback e aceite

Rollback lógico: reverter o commit `4aca1e1`; não requer rollback de banco porque nenhum schema/persistência mudou. Não remover settings existentes nem fazer rollback destrutivo.

Estado: `AWAITING_HUMAN`; o resultado não equivale à execução autônoma de projeto, aceite de piloto, ativação de provider, consumer, deploy ou push.
