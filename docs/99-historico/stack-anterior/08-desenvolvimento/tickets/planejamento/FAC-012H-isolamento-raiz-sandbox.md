# FAC-012H — Isolamento da raiz do SandboxRunner

Status: AWAITING_HUMAN

## Objetivo

Fechar a exposição do filesystem do host em comandos executados pelo `SandboxRunner`. Verificação local FAC-012H reproduziu leitura de `/etc/hostname` apesar da árvore do projeto estar read-only: o launcher entrava em namespace de mount, mas mantinha a raiz do host visível. Antes de ligar checks de projeto, o processo precisa enxergar uma raiz mínima, sem poder atravessar para diretórios do host.

## Critérios de aceite

1. Launcher cria uma raiz de execução isolada e troca a raiz efetiva antes de executar o comando; não deixa mountpoint de escape para a raiz antiga.
2. A raiz expõe somente workspace, runtime/toolchains explicitamente permitidos somente leitura, `/proc` da execução, devices mínimos e temporários isolados; caminhos de host como `/etc/hostname`, `/home`, `/run` e `/var/lib` não são legíveis.
3. Workspace é read-only por padrão; caminhos relativos autorizados permanecem graváveis sem escape por symlink, caminho absoluto ou `..`.
4. Sem rede, limites de CPU/memória/processos/arquivos, timeout, quiescência e sanitização ambiental existentes continuam valendo.
5. Testes Linux com comandos sintéticos tentam ler marcadores do host, escrever fora/dentro dos escopos e escapar da raiz; acesso não autorizado precisa falhar e processos devem parar confirmadamente.
6. Lint, typecheck, testes, build e `git diff --check` passam.
7. Nenhum CLI/provider, worker/consumer, checkout remoto, banco, fila, migration, credencial, piloto, gasto ou deploy é utilizado.

## Baseline e limites

Base `9c1e11e351cefad8a0eb38bb0b64430bdc4d5fa0`, branch `feat/fac-012h-sandbox-root-isolation`, worktree `/home/vinicius/le-fabrique-fac-012f`. Alvos em `packages/runtime/src/sandbox-runner.ts`, `sandbox-launcher.cjs` e testes; docs de infraestrutura/operação e lesson de isolamento. Runtime/toolchains vindos do worker confiável apenas; nenhum path originado do ticket/job. Sem novo provider/dependência.

## Fora de escopo

Confinamento equivalente do Developer CLI (FAC-012D separado), identidade operacional systemd do worker, checkout/materialização de repositório e ativação do consumer. Tais gates continuam obrigatórios depois desta correção.

## Implementação e evidências

Implementado no commit `2392a0491074a0bf1034c469595b888f8a0b3f04`. A prova prévia, via comando sintético `/usr/bin/cat /etc/hostname`, completava e lia o host (11 bytes); nenhuma configuração/segredo foi lido. O launcher agora pivota para um tmpfs root, mantém apenas `/usr` e workspace montados, oculta host root/home/run/var/sys, remove a raiz antiga e elimina CapEff/CapPrm antes do comando. Teste de integração também tenta remontar `/mnt` como RW e falha; verifica symlink host, temporário isolado, rede bloqueada e saída confirmada. `npm run lint` (145 arquivos), typecheck, `npm test` (187 testes), build e `git diff --check` passaram. Nada foi executado sob identidade systemd de produção ou com provider real; worker/consumer não foi ligado. Limite: toolchains externas a `/usr` não são montadas; planejar whitelist confiável antes de projeto que necessite delas. Aguarda revisão/aceite humano exato.
