# FAC-012X — Espera por provider com trabalho preservado

Data: 2026-10-03. Status: IMPLEMENTADO / AWAITING_HUMAN. Baseline `9da35a4`, READY `9a793e9`; código `345e8efc09a8c7eefdbf3354bf578c1a084e2274`. Branch `feat/fac-012x-waiting-provider`, worktree `/home/vinicius/le-fabrique-fac-012x`.

## Objetivo, critérios e funcionamento

Conectar indisponibilidade/auth/cota ao consumer sem desperdiçar correções ou perder trabalho, com estado WAITING_PROVIDER distinguível de limite/defeito. `ProviderUnavailableError` identifica somente instalação indisponível no router; config inválida, permissão imprópria, ambiente inseguro, ausência de assignment/adapter não viram espera falsa. Developer/Reviewer tratam AUTH_REQUIRED, RATE_LIMITED e PROVIDER_BUSY somente após retorno terminal conhecido. Não repetem chamada/login, não mudam cobrança e não contam essa falha como correção de código.

Workflow exige cancelActive/quiescência/checks parados, captura snapshot posterior à parada (inclusive limpo quando nenhuma chamada começou), preserva observações e devolve WAITING_PROVIDER/PROVIDER_UNAVAILABLE. Unknown/rejeição do runtime ou falha de snapshot continuam propagando erro, sem fabricar stop/espera. Reviewer sem provider preserva trabalho Developer/checks, sem invalidar por simples falta de verdict.

Processor → journal privado → result/artifact → checkpoint WAITING_PROVIDER → complete fenced. Journal verifica coerência status/outcome/checkpoint; API usa mesmo mapping para reconcile e STOPPED para tentativa em espera. Migration adiciona WAITING_PROVIDER somente a RunStatus, TicketStatus já tinha esse valor. Nenhum banco recebeu migration. Handoff automático ainda não é este incremento.

Painel mostra mensagem de intervenção e oferece retomada explícita existente: autenticar/escolher instalação elegível na configuração antes de autorizar nova tentativa. API exige versão, tentativa/fence atuais, stop/result/checkpoint/bundle/hash exatos, objetivo original congelado e outbox. Claim continua único writer e só cria fence novo para intenção autorizada. WAITING_PROVIDER não dispara retry/novo login sozinho. Limites da nova tentativa são confirmados pelo operador como no T.

## Diff e verificação

Diff sanitizado: `git show 345e8efc09a8c7eefdbf3354bf578c1a084e2274 -- apps packages/contracts/src/index.ts`. 17 arquivos, 308 inserções/73 remoções; fixtures sintéticas, sem credenciais reais.

- `npm ci --ignore-scripts`: dependências privadas, audit zero vulnerabilidades.
- `npm run db:generate`: geração local sem banco.
- `npm run typecheck`: todos passaram na revisão final.
- `npm test`: 399 passaram (launcher 2, contracts 34, runtime 46, API 139, worker 162, web 16).
- `npm run build`: cinco workspaces passaram; teste processor extra posterior foi validado na suíte/typecheck finais, implementação inalterada.
- `npm run lint`: 196 arquivos sem fixes.
- `git diff --check`: passou.

Doze novos testes cobrem três causas em Developer/Reviewer, rota indisponível antes de chamada com snapshot limpo, runtime rejeitado unknown sem espera falsa, journal incoerente, persistência fenced no processor, reconcile e resume por evidência. Painel estático existente também verifica a mensagem de intervenção; nenhum browser/login/IA reais. Suíte inicial 398, final 399 após teste do processor. Não houve falha de checks/correção de implementação; dois patches auxiliares sem efeito útil/contexto foram corrigidos antes dos checks, sem evidência falsa.

## Limitações, rollback e aceite

Provider/cota efetivos, preflight de serviço e migração são operação manual; consumer false. WAITING_PROVIDER não mede/resetta cota, atualiza por suposição o estado global da instalação nem escolhe fallback não configurado. Handoff automático, confinamento de escrita Claude e documentação técnica do projeto seguem pendentes. Não há piloto, produção, filas/banco reais, login, API/extras/fallback pago, push/deploy neste incremento.

Rollback com worker comprovadamente parado: reverter código mediante ticket autorizado, preservar enums/linhas/journal/snapshots; clientes anteriores não entendem WAITING_PROVIDER em runs, portanto manter revisão compatível para recuperação. Não remover enum/dados como rollback automático. Aceite exato humano pendente, não DONE.

## Docs, lessons e IA

Atualizados READMEs runtime/controle/operação/infraestrutura/planejamento, README raiz, controle MVP, ticket, índice/changelog/backlog e lesson lifecycle. Nenhuma chamada de IA da fábrica; modelo/tokens/custo do assistente não observáveis. APIs/extras/créditos/autorecharge permanecem proibidos.
