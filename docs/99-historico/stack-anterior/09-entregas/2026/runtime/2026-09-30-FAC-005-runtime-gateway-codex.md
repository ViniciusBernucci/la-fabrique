# FAC-005 — Runtime Gateway e adapter Codex

Data: 2026-09-30
Estado: DONE
Revisao aceita: `9102fcc614739fd7bd7ca4992b16617db92ca66f`, em 2026-09-30

## Resultado

O `packages/runtime` agora implementa o primeiro adapter oficial da fabrica. `CodexAdapter` valida requests, inicia o Codex sem shell, envia o prompt por stdin, aplica perfil de permissao restrito, interpreta JSONL, limita logs, normaliza falhas e controla cancelamento/timeout da arvore de processos.

Os contratos ficam em `packages/contracts` e usam Zod em runtime. Request, eventos normalizados, resultado, uso, erro, status do provider e cancelamento rejeitam formatos invalidos antes de atravessar os limites do gateway.

## Funcionamento

- `execute(request, eventSink)` exige workspace absoluto existente e apenas uma execucao ativa por adapter.
- O processo recebe argumentos em array, `--ignore-user-config`, `--ephemeral`, `--json` e prompt por stdin.
- `OPENAI_API_KEY` e `CODEX_API_KEY` sao removidas do ambiente filho.
- O perfil Codex nega `:root`, libera `:minimal`, concede somente leitura ou escrita na raiz ativa e mantem rede de comandos desligada.
- Eventos de raciocinio e conteudo bruto de comandos nao entram no stream normalizado. O consumidor recebe apenas categoria e lifecycle.
- `turn.completed` fornece uso quando o cliente o expõe. Modelo ausente permanece `null`; cota/reset permanecem `UNKNOWN`.
- `cancel(executionId)` envia `SIGTERM` ao grupo, usa `SIGKILL` apos o grace period e so confirma `CANCELLED` depois do fechamento.
- Timeout, limite de log, JSONL invalido, auth, cota, contexto, permissao e falha transitoria recebem codigos normalizados.

## Evidencias

- Fixture local comprovou prompt por stdin, remocao das duas chaves, perfil read/write e ausencia de conteudo privado nos eventos.
- Cancelamento e timeout encerraram processos reais Node criados pelo teste.
- Falhas sinteticas foram classificadas como `AUTH_REQUIRED`, `RATE_LIMITED`, `CONTEXT_TOO_LARGE`, `TOOL_DENIED`, `TRANSIENT`, `RESULT_UNKNOWN` e `LOG_LIMIT`.
- A sondagem real no usuario atual da VPS, sem inferencia, retornou provider `codex`, estado `AVAILABLE`, auth `chatgpt`, `codex-cli 0.159.2`; uso permaneceu `UNKNOWN` com fonte `client-not-exposed`. A identidade futura do servico ainda deve repetir o preflight.

## Checks

- `npm run lint`: 56 arquivos, passou.
- `npm run typecheck`: contratos, runtime, API, worker e web, passou.
- `npm test`: 29 testes em 9 arquivos, passou; 12 testes pertencem ao adapter.
- `npm run build`: contratos, runtime, API, worker e web, passou.
- `git diff --check`: passou.
- Diff sanitizado: nenhuma chave ou token encontrado.

As duas rodadas de ajuste foram de formatacao Biome. Os testes funcionais e o typecheck passaram desde a primeira execucao.

## Limites

O adapter ainda e uma biblioteca. O worker nao recebe runtime jobs, nao persiste eventos e nao chama o adapter pelo BullMQ; claim, lease, fencing e orquestracao pertencem a tickets posteriores. Cgroups, isolamento de servicos/sockets e snapshots pertencem ao FAC-007. O FAC-005 nao realizou nova inferencia real, pois o cliente e a assinatura ja foram comprovados no FAC-002.

Cancelamento confirma a arvore iniciada pelo adapter, mas nao reconcilia efeitos externos que um comando ja tenha produzido. Resultado desconhecido nao deve ser repetido automaticamente.

## Rollback

Reverter `d7488ccff1b5cb5f33a9bfcfb3b36f526cbdcde8` remove contratos e implementacao do adapter e restaura a porta minima anterior. Nao ha migration, dado persistente, login alterado ou deploy.
