# Entrega FAC-003 — controle web e persistência

Data: 2026-09-30

Status: DONE

Branch: `feat/fac-003-control`

Revisão de código verificada: `1807330bcf7b1374fa626d9fcbfc47dd8002f433`

## Resultado

Foram implementados contratos Zod compartilhados, autenticação administrativa por token de ambiente, cadastro e consulta de projetos/tickets e a transição idempotente `DRAFT -> READY`. A transição usa versão otimista e cria um evento de outbox na mesma transação PostgreSQL. O painel React permite autenticar, cadastrar registros sintéticos e promover o ticket.

## Evidências reais

- `npm run lint`: aprovado em 45 arquivos.
- `npm run typecheck`: aprovado em todos os workspaces; o comando agora constrói contratos/runtime antes de consumir suas declarações em checkout limpo.
- `npm test`: 11 testes aprovados em 6 arquivos.
- `npm run build`: aprovado; bundle web 310,38 kB, 94,38 kB gzip.
- Migrations `20260929000100_initial_control_plane` e `20260930000100_fac_003_control`: aplicadas.
- Ensaio HTTP em API local: sem token `401`; criação de projeto/ticket aprovada; retry de `READY` manteve versão 2; consulta SQL confirmou uma linha de outbox.
- Docker Compose completo: API, PostgreSQL e Redis saudáveis; worker e web em execução; somente porta 8080 publicada.
- Proxy Nginx: health `200`, controle sem token `401`, projeto persistiu após reinício da API.

## Limites

O token único é uma autenticação administrativa inicial e exige HTTPS na implantação. Ele não oferece usuários, expiração ou revogação individual. O painel usa `sessionStorage`, que reduz persistência da credencial, mas não protege contra código malicioso executado na mesma origem. A outbox ainda não possui dispatcher e `READY` não inicia job. Nenhum provider, API de IA ou cobrança extra foi habilitado.

## Rollback

Reverter o commit de código restaura o painel e a API anteriores. A coluna aditiva `deduplication_key` pode permanecer no banco sem afetar o código anterior; removê-la exige migration explícita e não faz parte do rollback normal. Para interromper a composição sem apagar dados, executar `docker compose down`, sem `-v`.

## Documentação e lessons

Foram atualizados README raiz, controle, operação, planejamento, índice, changelog e backlog. A lesson `lessons/outbox-idempotencia.md` registra o conceito aplicado.

## Aceite

O responsável concedeu aceite explícito em 30/09/2026 para o código `1807330bcf7b1374fa626d9fcbfc47dd8002f433` e a documentação apresentada no commit `499caab`. O registro de aceite é documental e não altera o código verificado.
