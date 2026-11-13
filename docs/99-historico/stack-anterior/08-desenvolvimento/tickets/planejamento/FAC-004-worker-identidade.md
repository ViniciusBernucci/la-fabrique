# FAC-004 — worker interno e identidade de serviço

Status: DONE

## Objetivo

Dar identidade persistida ao único worker inicial e estabelecer um protocolo interno autenticado para registro e heartbeat. O worker deve encerrar de forma conservadora após três falhas consecutivas de heartbeat, preparando o gate que impedirá writers órfãos quando leases forem adicionados.

## Escopo

- Contratos Zod de registro, identidade e heartbeat.
- Modelo Prisma e migration aditiva de `worker_identities`.
- Rotas internas `POST /internal/workers/register` e `POST /internal/workers/{id}/heartbeat` protegidas por credencial própria.
- Rota administrativa `GET /workers` protegida pelo token do operador.
- Cliente do controle no processo worker, registro no bootstrap e heartbeat periódico.
- Docker Compose com identidade/credencial injetadas, dependência da API saudável e nenhuma porta do worker.
- Nginx bloqueando `/api/internal/*` no proxy público.

## Fora do escopo

Claim, lease, fencing, execução de clientes oficiais, jobs reais, systemd no host, rotação automática de credenciais e múltiplos workers. O BullMQ continua limitado ao probe sintético existente.

## Critérios

- Registro cria ou atualiza a mesma identidade sem duplicação.
- Heartbeat só é aceito para worker registrado e atualiza `lastHeartbeatAt`.
- Token ausente/incorreto recebe `401` e não aparece em payload ou log.
- Rota interna recebe `404` pelo Nginx público e funciona apenas na rede backend.
- Três falhas consecutivas de heartbeat encerram o processo antes de qualquer trabalho futuro continuar sem controle.
- Composição publica somente `8080`; worker, API, PostgreSQL e Redis permanecem internos.
- Lint, typecheck, testes, build, migration e ensaio integrado passam.

## Baseline, risco e limites

Base: FAC-003 aceito no SHA `1807330bcf7b1374fa626d9fcbfc47dd8002f433`. Nenhum provider é necessário. Risco R2; dados sintéticos; um worker, concorrência global 1 e até duas rodadas de correção.

## Documentação

Atualizar operação, contratos, planejamento, índice, changelog, backlog e relatório datado. Registrar revisão exata, evidências, limites e rollback. DONE somente após aceite humano.

## Evidência

Implementado e verificado no SHA de código `6d73041bdcbd50215b6a018c9937475c29f542b1`. O relatório está em `documentacoes/operacao/2026-09-30-FAC-004-worker-identidade.md`.

O responsável concedeu aceite explícito em 30/09/2026 para os commits apresentados. FAC-004 está DONE.
