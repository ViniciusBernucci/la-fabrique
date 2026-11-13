# OPS-005 — Worker confiável no host da VPS

Status: AWAITING_HUMAN

## Objetivo

Preparar o processo worker Node/TypeScript para rodar como serviço dedicado do host na mesma VPS, deixando PostgreSQL, Redis, API e painel em Docker Compose. O `SandboxRunner` exige `systemd-run --user`/namespaces, que não existem no container worker atual; a imagem também não inclui as CLIs oficiais/Git nem recebe os diretórios de checkout. Não executar nem provisionar o serviço neste ticket.

## Critérios de aceite

1. O serviço Compose por padrão não inicia um consumidor de execução que atualmente apenas simula o workflow; o código não pode marcar tickets READY como `VALIDATING` via fixture.
2. O Compose mantém API e Redis acessíveis ao worker host somente por bind `127.0.0.1`; portas não são publicadas em interfaces externas.
3. Entregar unit systemd genérico para o worker host com usuário dedicado, `HOME`/estado sob `StateDirectory`, env file externo ao repositório, sem tokens/modelos/contas embutidos e sem `privileged`/capabilities amplas.
4. O serviço host não inicia execução real de ticket: checkout/DeveloperWorkflow continuam desligados até ticket de integração próprio e gates operacionais/aceites cumpridos.
5. Documentar bootstrap manual futuro, requisitos de `loginctl enable-linger` para `systemd-run --user`, caminhos de instalação, acessos locais e rollback sem executar comandos administrativos.
6. Atualizar ADR-002/README de infraestrutura somente como detalhamento compatível da VPS única; manter a stack React/Nest/worker TS/PostgreSQL/Redis+BullMQ e o Docker Compose da control plane.
7. `docker compose config --quiet`, verificação sintática do unit systemd quando disponível, lint, typecheck, testes, build e `git diff --check` passam sem iniciar/recriar containers, tocar banco/fila, instalar cliente ou autenticar provider.

## Baseline e escopo

Base `ce93b20440c3f7ecc216ae6513396596cdf9a76e` em `developer`. Alvos principais: `compose.yaml`, `apps/worker/src/main.ts`, processador de orquestração, `infrastructure/systemd/`, `.env.production.example`, README e documentação `infraestrutura`/`operação`/`arquitetura`. Relato de implementação será `documentacoes/infraestrutura/2026-10-03-OPS-005-worker-host-sandbox.md`.

## Provider, efeitos e limites

Nenhum provider de IA elegível/usado; apenas testes determinísticos. Não acessar banco/Redis reais, não ler logs/env do serviço ativo, não executar setup systemd, não parar/atualizar serviços, não instalar CLIs, não criar piloto, não fazer migration/push/deploy. A composição existente contém worker container em execução sem bind mounts para o checkout; ela não será reiniciada neste ticket.

Limite: incremento de infraestrutura pequeno, até duas rodadas de correção. Evidência sintética não prova identidade, keyring, sandbox ou uso financeiro da futura VPS.

## Implementação e evidências

Implementado no commit `8f3dbda34892c466c0907f38767e1d1d1b41c859`. `compose.yaml` agora compõe apenas web/API/PostgreSQL/Redis; API (3000) e Redis (6379) são publicados em `127.0.0.1`, PostgreSQL permanece sem porta publicada. O worker Compose foi removido. `main.ts` não registra consumidor `le-fabrique.execution`; `processOrchestrationFixture` permanece sob teste, sem marcar jobs READY como `VALIDATING` por simulação.

Foi adicionado `infrastructure/systemd/le-fabrique-worker.service`, com usuário fixo dedicado, home/StateDirectory persistente, env file externo, sistema read-only exceto estado do worker, sem capabilities e sem `privileged`. O exemplo `worker.env.example` tem IDs/token vazios e URLs locais; contas, modelos e hosts Git continuam configuráveis, não embutidos. O unit não foi instalado ou iniciado.

Checks: `docker compose --env-file .env.production.example config --quiet` passou e `config --services` listou somente `redis`, `postgres`, `api`, `web`. `systemd-analyze verify infrastructure/systemd/le-fabrique-worker.service` terminou com exit 0; mostrou somente avisos de unidades do host não relacionadas (`netplan-ovs-cleanup`, `snapd`). `npm test` passou (206 testes: launcher 2, contracts 21, runtime 42, API 55, worker 81, web 5); `npm run typecheck`, `npm run lint` (149 arquivos), `npm run build` e `git diff --check` passaram. `npm ci` instalou lockfile; `npm run db:generate` gerou artefato local. Sem migration ou conexão a banco/fila.

Limite operacional: foi observado antes da edição que `le-fabrique-worker-1` continuava rodando desde 2026-09-30 em container, sem bind mounts para o checkout. Não lemos env/logs nem alteramos/parámos containers; portanto a configuração live ainda contém a imagem antiga e não usa este código. Não foi provado se havia job pendente. Troca exige parada/quiescência e janela manual autorizada. Nenhuma instalação, user account, linger, cliente oficial, keyring, provider, piloto, push ou deploy ocorreu.

Rollback: não aplicar automaticamente a configuração antiga, pois isso religa o fixture que respondia `VALIDATING` sem workflow real. Antes de qualquer rollback/recriação, manter consumidor de execução desligado e reconciliar workers/filas com operador. Como nada foi aplicado aos serviços ativos, o código no branch pode ser revertido sem efeitos persistentes; não há dados a restaurar.
