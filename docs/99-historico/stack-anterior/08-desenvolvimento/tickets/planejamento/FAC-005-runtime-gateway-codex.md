# FAC-005 — Runtime Gateway e primeiro adapter Codex

Status: DONE

Revisao aceita: `9102fcc614739fd7bd7ca4992b16617db92ca66f`, em 2026-09-30. O commit documental posterior apenas registra o aceite da revisao funcional apresentada.

## Objetivo

Implementar o contrato executavel do Runtime Gateway e o primeiro adapter para o Codex CLI validado no FAC-002, com lifecycle controlado, JSONL validado em runtime, cancelamento, timeout e erros normalizados.

## Escopo

- Contratos compartilhados em `packages/contracts`.
- Portas, parser JSONL, lifecycle e adapter em `packages/runtime`.
- Fixtures e testes do runtime sem chamadas reais ao fornecedor.
- Documentacao de runtime, operacao, planejamento e lesson aplicada.

Ficam fora: claim de tickets, dispatcher da outbox, integracao do adapter ao loop BullMQ, leases/fencing, cgroups, sandbox de projeto externo, segundo provider, deploy e piloto real.

## Criterios de aceite

1. Requests, resultados, eventos, uso e erros possuem esquemas Zod e rejeitam payloads invalidos.
2. O adapter chama binario e argumentos sem shell, envia prompt por stdin e remove `OPENAI_API_KEY`/`CODEX_API_KEY`.
3. O perfil de permissao nega o host, limita leitura/escrita ao workspace solicitado e desliga rede dos comandos.
4. JSONL concluido produz resultado `COMPLETED`, sessao e uso observado; modelo ausente permanece `null`.
5. Auth, rate limit, permissao, contexto, falha transitoria e resultado desconhecido sao normalizados sem expor stderr bruto.
6. `cancel(executionId)` encerra uma execucao ativa e confirma `CANCELLED`; timeout confirma `TIMED_OUT`.
7. Logs/eventos sao limitados e itens de raciocinio nao sao persistidos nem encaminhados.
8. Testes usam fixture local e nao consomem assinatura/API.

## Baseline e limites

- Base: `developer` em `3963f4aeb8741718eb8f94e78a6ccf85764dfc2c`.
- Branch/worktree: `feat/fac-005-runtime-gateway` em `/home/vinicius/le-fabrique-fac-000-accepted`.
- Provider elegivel: Codex CLI `0.159.2`, autenticacao ChatGPT aceita no FAC-002.
- Uma execucao por adapter; timeout maximo contratual de 30 minutos; logs limitados pelo request.
- Nenhuma chamada real ao Codex durante os testes desta entrega.

## Rollback

Reverter os commits FAC-005 restaura a porta minima anterior de `packages/runtime`. Nao ha migration, estado persistente, alteracao de login ou deploy.
