# Entrega 2026-11-12 — E06 adaptadores Jira e ClickUp (T06-01, T06-02)

**Tickets:** [T06-01](../NEW%20MVP/03-BACKLOG.md#t06-01) · [T06-02](../NEW%20MVP/03-BACKLOG.md#t06-02), ambos **parciais**. [T06-03](../NEW%20MVP/03-BACKLOG.md#t06-03), [T06-04](../NEW%20MVP/03-BACKLOG.md#t06-04) e [T06-05](../NEW%20MVP/03-BACKLOG.md#t06-05) **não iniciados: bloqueados** (T13-04 ledger de efeitos, T05-02, T10-01, T12-04 pendentes). Decisões: [ADR-0007](../adr/0007-adaptadores-jira-e-clickup-simetricos.md). **Revisão independente: NOT_RUN.**

## O que mudou
- `internal/integrations`: interface `Tracker` + adaptadores `jira.go` e `clickup.go` (identidade, tipos, estados, campos, prioridades, listagem paginada, leitura, transições válidas, criação de subtarefa, dependência e link de PR), erros tipados, cliente HTTP comum.
- **Contrato com fixtures:** uma suíte (`tracker_contract_test.go`) roda **igual** contra servidores falsos de Jira e ClickUp com payloads realistas em `testdata/`. Sem dependências novas nem mudança de banco ou de contrato HTTP.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Contrato Jira e ClickUp (`go test -race ./internal/integrations`) | PASS |
| `make lint` | PASS |
| `make test` — parte Go (config, httpapi, logging, security, worker, integrations, contrato) | PASS |
| `make test` — parte do front (vitest) nesta rodada | **NOT_RUN**: os workers do vitest não iniciaram (a leitura de `node_modules` no SSD externo estagnou; até `require('jsdom')` passou de 120 s). `apps/web` não mudou neste commit; a última execução válida (45 testes) foi PASS na entrega do E05 |
| `make test-integration`, `make check-secrets` | NOT_RUN nesta rodada (sem mudança de banco ou de contrato; última execução válida: PASS no E05) |
| Instância real de teste do Jira / lista real do ClickUp | **NOT_RUN** (sem credenciais autorizadas) |
| Revisão independente | **NOT_RUN** |

### Critérios de aceite → evidência
- **T06-01 "distingue itens, subtarefas, transições válidas, paginação, permissão e rate limit":** subtestes "lista pagina e distingue item de subtarefa", "só transições válidas são enviadas", "falha forbidden" e "falha ratelimit" (`RetryAfter` lido de `Retry-After`) em `TestJiraContract`.
- **T06-02 "pai/filho, status, prioridade e tratamento de erros":** os mesmos subtestes em `TestClickUpContract` (+ `X-RateLimit-Reset`).
- **Segurança:** o servidor falso ecoa a credencial nos corpos de erro; nenhuma mensagem de erro a contém, nem URL, nem corpo. Entradas malformadas não geram chamada. Redirecionamento não é seguido; Jira recusa hosts fora de `*.atlassian.net`.

## Por que ficam parciais
O aceite cita **instância de teste**; só fixtures foram exercitadas. Shape real dos campos personalizados, permissões finas, limites reais e o comportamento de `search/jql` em sites antigos não foram medidos.

## Limites
- ClickUp: link de PR vira comentário (repetir duplica); 401 cobre "sem acesso"; página fixa de 100.
- Sem retry/backoff no adaptador (decisão do chamador).
- Sem persistência de snapshots nem sincronização (T06-04).

## Rollback
`git revert` do commit. Sem migration.
