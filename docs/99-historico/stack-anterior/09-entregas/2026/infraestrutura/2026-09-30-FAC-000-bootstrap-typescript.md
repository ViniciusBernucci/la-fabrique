# Entrega FAC-000 — bootstrap TypeScript

Data: 2026-09-30

Status: DONE

Domínio: infraestrutura

Branch: `feat/fac-000-typescript-foundation`

Base: `e7119ae1647f8de48a65f821d7383a92da89e43c`

## O que mudou

Foi criado o monorepo npm definido no ADR-003, com React 19 e Vite no painel, NestJS na API, worker Node.js separado, contratos Zod compartilhados e interfaces de runtime. Prisma/PostgreSQL persistem projetos, tickets e outbox; Redis/BullMQ sustentam a fila. A migration inicial, health checks, testes e composições Docker de desenvolvimento e completa fazem parte da entrega.

A API não executa builds ou clientes. O worker possui somente um processador de probe sintético e concorrência igual a 1. Nenhuma API de IA, crédito adicional, autenticação de provedor ou fallback pago foi habilitado.

## Evidências

- `npm run lint`: aprovado em 38 arquivos.
- `npm run typecheck`: aprovado em todos os workspaces.
- `npm test`: 7 testes aprovados em 4 arquivos.
- `npm run build`: aprovado; bundle web de 291,12 kB, 89,68 kB gzip.
- `npm audit --audit-level=high`: zero vulnerabilidades após fixar Prisma 6.12.0.
- `docker compose config` e `docker compose -f compose.dev.yaml config`: aprovados.
- Imagens de `api`, `worker` e `web`: construídas com sucesso.
- Execução integrada: PostgreSQL, Redis e API saudáveis; worker e web em execução.
- `GET /`: HTTP 200 com o painel La fabrique.
- `GET /api/health/ready`: HTTP 200, banco e Redis com estado `ok`.
- PostgreSQL: migration registrada e tabelas `_prisma_migrations`, `projects`, `tickets` e `outbox_events` presentes.

## Limites e riscos restantes

Autenticação, CRUD, dispatcher de outbox, leases/fencing, sandbox e adapters de providers não pertencem a este bootstrap. As senhas locais são sintéticas. A VPS, HTTPS, backup/restauração e isolamento de credenciais ainda não foram validados. Os containers locais ficaram ativos para inspeção em `http://localhost:8080`.

## Rollback

Execute `docker compose down` para interromper a fundação sem remover volumes. Não use `down -v` no rollback normal, pois esse comando apaga os dados persistentes.

## Aceite

FAC-000 foi revisado no SHA `15c2198076a6a6afcbf2f15212ae1eaf53822b9f` e recebeu aceite humano explícito em 30/09/2026. A evidência final e os limites preservados estão registrados em `2026-09-30-FAC-000-aceite.md`.
