# OPS-005 — Worker host para o sandbox da VPS

Status: AWAITING_HUMAN

## Resultado

Revisão de código: `8f3dbda34892c466c0907f38767e1d1d1b41c859` na branch `ops/ops-005-host-worker-service`, worktree `/home/vinicius/le-fabrique-ops-005`.

O caminho de produção do worker passa a ser um processo Node.js dedicado no host da mesma VPS. O Compose contém a control plane e publica somente Nginx `127.0.0.1:8080`, API `127.0.0.1:3000` e Redis `127.0.0.1:6379`; PostgreSQL segue sem porta publicada. O serviço worker foi removido do Compose padrão, pois o container restrito não tem os utilitários e a interação `systemd-run --user` exigidos pelo supervisor de sandbox.

O código `main.ts` não registra mais `le-fabrique.execution`. A fixture de protocolo segue disponível apenas para testes; nenhum job READY deve avançar artificialmente para `VALIDATING`. Foi adicionado um unit systemd com usuário dedicado, `StateDirectory`, `HOME` sob `/var/lib`, `EnvironmentFile` externo, filesystem restrito e nenhuma capability. Exemplo sem credenciais: `infrastructure/systemd/worker.env.example`.

Não foram incluídos piloto, provider, modelo, conta, host Git ou token. Esses dados continuam em configuração/interface ou arquivo operacional fora do repositório.

## Checks e evidências

- `npm test`: passou — 206 testes (2 launcher, 21 contracts, 42 runtime, 55 API, 81 worker, 5 web).
- `npm run typecheck`, `npm run lint` (149 arquivos), `npm run build` e `git diff --check`: passaram.
- `docker compose --env-file .env.production.example config --quiet`: passou. `config --services` listou `redis`, `postgres`, `api`, `web`; não há worker Compose.
- `systemd-analyze verify infrastructure/systemd/le-fabrique-worker.service`: exit 0. O ambiente emitiu avisos para `netplan-ovs-cleanup` (permissão ao ler unidade do host) e `snapd` (`RestartMode` desconhecido nesta versão); nenhum aponta para o unit OPS-005.
- `npm run db:generate` apenas atualizou Prisma Client local; nenhuma migration, conexão de banco/fila, checkout remoto ou chamada a provider ocorreu.
- Diff revisável: `compose.yaml` limita binds e remove o worker container; `main.ts` remove import/instanciação/fechamento/log do consumidor fixture; os arquivos `infrastructure/systemd/*` declaram serviço/env file sem segredos.

## Limite da validação e operação ativa

Antes da mudança, foi observado `le-fabrique-worker-1` vivo desde 2026-09-30 como container sem bind mounts para o checkout. A inspeção usou metadados do container (processo, estado, mounts e rede), sem ler environment/log, banco ou fila. Não foi provado se havia job em execução ou pendente.

Esse container não foi parado, atualizado nem recriado; o código e Compose deste branch não afetam a instância ativa. Portanto, não afirmar que a execução fixture cessou no deployment atual. Qualquer adoção requer parada/quiescência, inspeção autorizada de filas e janela manual de mudança. Nenhum usuário de sistema foi criado, nenhum linger habilitado e nenhum unit instalado/iniciado. Git/GH/CLIs oficiais e keyring sob identidade de serviço seguem sem verificação.

## Bootstrap manual futuro

Somente após ticket de execução integrado e revisão operacional: provisionar o usuário fixo `le-fabrique-worker` e `/opt/le-fabrique`; instalar Node/Git/GH e CLIs oficiais por fonte verificada; definir `/etc/le-fabrique/worker.env` fora do repo com modo restritivo e valores criados pelo responsável; alinhar token do worker com a API; autenticar contas pelos fluxos oficiais nessa identidade; criar os roots privados sob `/var/lib/le-fabrique-worker`; habilitar user manager/linger e provar que `systemd-run --user`, user/mount namespaces e isolamento de credenciais funcionam. Só então considerar instalar/ativar o unit em uma janela autorizada. OPS-005 não executa estes passos.

## Rollback e aceite

Não houve efeito no deployment ativo, então não há rollback operacional. Para reverter o código, reverta a revisão `8f3dbda` somente após um plano que mantenha o consumidor de execução desativado; reverter cegamente reintroduz o fixture `VALIDATING` e o container incompatível. O rollback não remove dados/volumes.

OPS-005 aguarda revisão humana desta revisão exata. Não é aceite FAC-012C–J, não liga consumer/workflow, não demonstra identidade/keyring reais e não declara o MVP operacional.
