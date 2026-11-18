# Lote 2026-11-18 — Painel de Operação da fábrica

- **Branch de integração:** `developer` (sem worktrees paralelas — uma única frente sequencial por
  vez; sem necessidade de `99-integracao`).
- **Base:** `developer` @ `82857c0da3ee7408e7f3a6367fb3ffbe6442fdc7` (local, à frente de
  `origin/developer` @ `7172abc3b8043d6d225990f9be3a4c44d028afc4`).

## Demandas incluídas

- **T16-05 — Consolidar logs operacionais da fábrica.** Épico
  [E16](../../NEW%20MVP/02-EPICOS.md#e16). Extensão formal do backlog (novo ticket, ao lado de
  T16-01..T16-04). Perfil de risco: **crítico**.
  - Especificação: [`00-plano-T16-05-painel-operacao-fabrica.md`](00-plano-T16-05-painel-operacao-fabrica.md)
  - Prompt `api`: [`01-T16-05-api-logs-operacionais.md`](01-T16-05-api-logs-operacionais.md) — CA-01 a CA-09, CA-12
  - Prompt `web`: [`01-T16-05-web-logs-operacionais.md`](01-T16-05-web-logs-operacionais.md) — CA-10, CA-11
  - Estado: `PRONTA_PARA_IMPLEMENTAR`

## Demandas excluídas

Nenhuma — este lote cobre uma única demanda, levantada pelo usuário durante a sessão (fora do
backlog original; virou ticket formal por decisão do planejamento, ver `00-plano`).

## Ordem de execução e integração

1. Frente `api` (T16-05) a partir da base acima. Commit único em `developer`.
2. Frente `web` (T16-05) a partir do commit da frente `api`. Commit único em `developer`.

Sem grafo de dependência além dessa ordem linear (um ticket, duas frentes sequenciais, zero
paralelismo real: a tela depende do contrato que a API define).

## Riscos e pendências

- **D10 do `00-plano` é hipótese** (reaproveitar `retention_decisions` como resumo de auditoria dos
  eventos operacionais, sem tabela de resumo nova). Não bloqueia o início; se a revisão discordar,
  é um ajuste isolado no método `Run` de `internal/retention`.
- Nenhuma questão essencial em aberto. Nenhuma `BLOQUEADA_NO_PLANEJAMENTO`.

## Próximos passos

1. Executar o prompt da frente `api` numa sessão (pode ser esta mesma, se o usuário pedir agora;
   senão, uma sessão nova lendo `01-T16-05-api-logs-operacionais.md`).
2. Depois do commit da frente `api`, abrir uma sessão nova com a skill `revisor-de-codigo` (prompt
   de revisão no fim do arquivo da frente `api`).
3. Executar o prompt da frente `web` (depende do commit revisado/integrado da frente `api`).
4. Depois do commit da frente `web`, abrir uma sessão nova com a skill `revisor-de-codigo` (prompt
   de revisão no fim do arquivo da frente `web`).
5. Sem worktrees nesta entrega: não há etapa de merge/integração separada — cada commit já cai
   direto em `developer`.
