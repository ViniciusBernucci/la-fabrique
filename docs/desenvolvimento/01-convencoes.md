# Convenções de desenvolvimento

## IDs
`Exx` épico, `Txx-yy` ticket, `Dxx` decisão da entrevista ([matriz](../NEW%20MVP/04-RASTREABILIDADE.md)). ADRs: `docs/adr/NNNN-titulo.md` (número nunca reutilizado; ADR aceita só é superada por outra). Entregas: `docs/entregas/AAAA-MM-DD-TICKET-titulo.md`.

## Branches, commits e PR
- Branch por ticket: `tXX-YY-descricao` a partir de `main`/`developer`. Commits pequenos, mensagem no imperativo citando o ticket (`T04-01: hash de senha com argon2id`).
- PR usa [o template](../../.github/pull_request_template.md): o quê/por quê, **como foi verificado** (comandos e resultado reais, `NOT_RUN` explícito) e checklist. Merge só com autorização do responsável.
- Proteção de `main` e checks obrigatórios: [github.md](../operacao/github.md).

## Revisão
Independente da implementação (sessão separada, skill `revisor-de-codigo`). Achados: crítico/relevante bloqueiam; baixa severidade fica registrada. Até duas rodadas de correção; depois diagnóstico e decisão do responsável. Novo diff invalida a evidência anterior do que mudou.

## Migrations
Nova migration = novo arquivo `db/migrations/NNNNN_nome.sql` com `Up` e `Down`. **Nunca editar** uma já aplicada em qualquer ambiente. Mudança incompatível em duas etapas (expandir → migrar dados → contrair). Todo ticket de schema inclui teste de integração em PostgreSQL real. Aplicar com `make migrate` ou pelo serviço `migrate` do Compose.

## Logs
JSON estruturado (`log/slog`), um campo `service`, e `correlation_id` em requisições. Atributos com nome sensível (`password`, `secret`, `token`, `cookie`, `database_url`…) são redigidos na origem; nunca logue valores de segredo, payloads com segredo ou a `DATABASE_URL`. Sondas de saúde logam em `debug`.

## Erros de API
Sempre `{"error":{"code","message","action","correlation_id"}}` ([contrato](../../contracts/openapi.yaml)). Nunca devolva texto do driver/banco.

## Testes
Unidade e contrato sem banco (`make test`); integração com PostgreSQL real (`make test-integration`, banco descartável por teste). Teste novo de API exige a rota no `contracts/openapi.yaml`.

## Documentação e evidências
Código, docs e [checklist](../NEW%20MVP/07-CHECKLIST.md) mudam juntos. Estados: PASS / FAIL / NOT_RUN / UNSUPPORTED para verificações; PLANEJADO / IMPLEMENTADO para funcionalidade. Registro de entrega descreve objetivo, mudança, como testar, resultados reais, limites e rollback. Sem segredos, dumps ou dados pessoais em docs, logs ou evidências.

## Front
Tokens em `apps/web/src/styles/tokens.css`; componentes de estado (`LoadingState`, `EmptyState`, `ErrorState`, `UnauthorizedState`) para carregamento, vazio, erro e acesso. Novas áreas entram em `navigation` (`components/AppShell.tsx`) + rota em `App.tsx`.

## Matriz de evidências do aceite

`docs/aceite/02-matriz-de-evidencias.md` é **gerada** a partir do checklist, do backlog e dos casos de aceite. Depois de alterar `07-CHECKLIST.md`, rode `make matriz` e inclua o arquivo no commit; o mesmo comando gera `docs/aceite/04-relatorio-de-aceite.md`; `make test` falha se algum estiver desatualizado ([ADR-0041](../adr/0041-matriz-de-evidencias-derivada.md), [ADR-0055](../adr/0055-ensaios-e-relatorio-de-aceite.md)). `make ensaios` roda os cenários de falha e segurança contra o PostgreSQL real e regenera `docs/aceite/03-ensaios-de-falha-e-seguranca.md`.

## Ensaio com navegador real

`make e2e` sobe uma API descartável (banco temporário removido ao encerrar, GitHub de mentira, dados de ensaio) e o front em portas livres escolhidas pelo sistema, e roda o Playwright (Chromium) em `apps/web/e2e/`. Primeira vez na máquina: `cd apps/web && npx playwright install chromium`. Telas e capturas ficam em `apps/web/e2e/.results/` (ignorado pelo Git). Mudou uma tela do painel: rode `make e2e` além de `make test` ([ADR-0057](../adr/0057-rotas-telas-e-ensaio-com-navegador.md)). `npm run typecheck` e `npm run build` fazem parte da verificação do front.
