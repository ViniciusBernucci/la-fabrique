# FAC-003 — controle web e persistência

Status: DONE

## Objetivo

Transformar a fundação do FAC-000 em um controle administrativo mínimo: autenticar o operador, cadastrar e consultar projetos e tickets, promover ticket de `DRAFT` para `READY` e registrar a intenção de despacho na outbox dentro da mesma transação PostgreSQL.

## Baseline

Base aceita: `15c2198076a6a6afcbf2f15212ae1eaf53822b9f`, acrescida apenas das atualizações documentais de aceite e replanejamento. Node.js 22, npm workspaces, NestJS, React/Vite, Prisma/PostgreSQL, Redis/BullMQ e contratos Zod permanecem fixados pelo FAC-000.

O baseline reproduzível é `npm ci`, `npm run db:generate`, `npm run db:migrate`, `npm run lint`, `npm run typecheck`, `npm test` e `npm run build`. PostgreSQL e Redis de desenvolvimento usam `compose.dev.yaml` e dados sintéticos.

## Escopo permitido

- `packages/contracts`: schemas versionados de autenticação administrativa, projeto, ticket e transição.
- `apps/api`: guard administrativo, endpoints de sessão/controle, serviços Prisma e transação de ticket/outbox.
- `apps/api/prisma`: migration aditiva necessária para os dados do controle.
- `apps/web`: login administrativo, cadastro e listagem mínima de projetos/tickets com validação dos contratos.
- Configuração example e documentação diretamente afetadas.

A autenticação inicial usa um token administrativo de alta entropia fornecido pelo operador e validado na API com comparação em tempo constante. O segredo vem de variável de ambiente, não entra no bundle, banco, logs, contratos ou Git. O painel mantém o token apenas na sessão do navegador e o envia por `Authorization: Bearer` sobre HTTPS na implantação. Evolução para identidade multiusuário exige ticket próprio.

## Fora do escopo

- Worker API, heartbeat, leases, fencing e execução de jobs.
- Dispatcher BullMQ em produção, adapters de IA, clientes oficiais e credenciais de providers.
- Piloto externo, deploy, domínio, HTTPS, contratação de VPS e backup externo.
- Aprovação final, SSE, RBAC multiusuário e recuperação de senha.

## Critérios de aceite

- Rotas administrativas rejeitam ausência ou token inválido e nunca registram o segredo.
- Entradas e saídas HTTP usam schemas compartilhados com validação em runtime.
- Operador autenticado cria e lista projeto e ticket; os dados permanecem após reinício da API.
- Transição `DRAFT -> READY` cria exatamente um evento de outbox na mesma transação e repetição idempotente não duplica o evento.
- Versão otimista impede atualização baseada em estado antigo.
- Painel permite autenticar, cadastrar e consultar os registros sintéticos e mostra erros sem revelar detalhes internos.
- Migration é aditiva e reproduzível; PostgreSQL e Redis continuam privados na composição completa.
- Lint, typecheck, testes relevantes e build passam na revisão final.

## Provider, risco e limites

Nenhum provider de IA é requisito funcional do FAC-003; a implementação segue o provider autorizado pela sessão de desenvolvimento, sem integrar API ou cliente ao produto. Risco R2, elevado a R3 apenas na superfície do token administrativo. Um writer, até duas rodadas de correção e dados exclusivamente sintéticos.

## Documentação e evidências

Criar o relatório datado do FAC-003, atualizar README dos domínios afetados, índice, changelog, backlog, contratos e operação. Registrar migration, testes de autorização/idempotência, diff sanitizado, revisão exata, limitações e rollback. O ticket só muda para DONE após revisão e aceite humano.

## Evidência de implementação

Implementado na branch `feat/fac-003-control` e verificado no SHA de código `1807330bcf7b1374fa626d9fcbfc47dd8002f433`. Evidências detalhadas estão em `documentacoes/controle/2026-09-30-FAC-003-controle-web-persistencia.md`.

O responsável concedeu aceite explícito em 30/09/2026 após receber os SHAs de código e documentação. FAC-003 está DONE.
