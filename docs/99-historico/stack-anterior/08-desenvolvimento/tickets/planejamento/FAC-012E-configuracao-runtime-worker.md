# FAC-012E — Leitura interna da configuração de runtime pelo worker

Status: AWAITING_HUMAN

## Objetivo

Disponibilizar ao worker uma cópia versionada e validada da configuração salva no Centro de Configurações, para que futuras etapas usem contas/modelos selecionados na interface, sem valores de provider hardcoded. A leitura será somente interna, autenticada pelo token do worker e sem segredos no contrato.

## Critérios de aceite

1. Contrato compartilhado valida `{version, observedAt, configuration}`; versão zero representa defaults não persistidos, com providers e funções inativos.
2. API oferece endpoint GET interno protegido por `WorkerAuthGuard`; leitura não cria nem modifica configuração/DB.
3. Resposta contém somente campos de `FactoryConfiguration`, que não inclui credenciais; campos extras/secretos falham na validação.
4. `ControlClient` busca e valida a resposta com o bearer interno, sem enviar credencial no body.
5. Testes cobrem defaults sem mutação, versão/configuração persistida, autenticação worker (nega ausente/incorreta) e transporte/validação no client.
6. Lint, typecheck, testes, build e `git diff --check` passam.
7. Não ligar consumer, consultar fila/banco real, executar client/provider, autenticar, alterar settings, efetuar migration ou selecionar piloto.

## Baseline, caminhos e limites

Base `a6fdf9ce57cc2293ddf310198204ea1093db9522`; branch `feat/fac-012e-worker-settings-read`; worktree `/home/vinicius/le-fabrique-fac-012e`. Domínios: `packages/contracts`, `apps/api/src/settings`, `apps/worker/src/control-client` e documentação de configuração/controle/runtime/operação. Ajustes limitam-se a tipos HTTP/worker, sem alterar persistência. Nenhuma conta precisa estar `AVAILABLE`; nenhum provider é elegível para este ticket.

## Segurança

A API administrativa existente continua autenticada por `ADMIN_API_TOKEN`; o novo GET usa somente `WORKER_API_TOKEN`, já usado nos endpoints internos. A configuração compartilhada não transporta sessões, tokens ou caminhos de auth. Não retornar settings brutos do Prisma. Não usar cache persistente no worker: o snapshot é lido sob demanda e carrega versão/horário de observação.

## Fora de escopo

Roteamento de job, checkout/clonagem, execução de workflows, início de provider, habilitar BullMQ, revisar frescor de evidência `AVAILABLE`, GitHub write/PR, mudanças UI ou migration. A decisão do piloto continua manual.

## Implementação e evidências

Implementado no código `4aca1e10085f047712766f51ca7e4e749508a7a9`; relatório operacional `documentacoes/operacao/2026-10-02-FAC-012E-configuracao-runtime-worker.md`. Endpoint e client foram cobertos por mocks, testes de schema/auth; lint, typecheck, 178 testes, build e `git diff --check` passaram. Não houve acesso ao banco/fila real, mudança de configuração, consumer, provider ou migration. Aguarda revisão/aceite humano.
