# FAC-012K - Lease vivo e cancelamento conservador
Status: AWAITING_HUMAN

## Objetivo
Fornecer ao worker uma primitiva reutilizável para renovar o lease de uma tentativa durante uma operação assíncrona e solicitar parada do writer imediatamente quando a renovação falhar.

## Atual e esperado
`ControlClient` já expõe `renew`, e a API rejeita renovações de tentativa expirada ou não RUNNING. Ainda não há primitiva no worker que mantenha uma lease viva durante trabalho longo. O novo utilitário não será registrado no consumer nem executará checkout/workflow; somente prepara comportamento testável antes da integração.

## Escopo permitido e proibido
Permitido: utilitário puro/injetável do worker e testes com relógio/funções sintéticas. Proibido: ativar consumer, acessar Redis/DB, chamar provider/GitHub, alterar sandbox, fila, VPS ou serviço ativo.

## Critérios de aceite verificáveis
1. Renova em intervalo menor que o lease configurado e inclui fencing token em cada pedido.
2. Na primeira renovação rejeitada, sinaliza perda de autoridade uma única vez e invoca parada; não tenta renovar nem autoriza continuação.
3. Só informa quiescência se o callback confirmar parada; callback ausente/rejeitado/sem confirmação mantém resultado como não quiescente e falha fechada.
4. Encerramento normal para o timer e aguarda operação de renovação/parada pendente sem corrida de estado.
5. Testes determinísticos cobrem renovação, erro, duplicidade, parada confirmada/ausente e shutdown.
6. A primitiva permanece sem ligação ao consumer e não constitui prova de execução real.
7. Typecheck, lint, testes, build e `git diff --check` passam.

## Baseline e comandos de checks
Baseline `ce93b20440c3f7ecc216ae6513396596cdf9a76e` (`developer`). Branch isolada `feat/fac-012k-lease-guard`, worktree `/home/vinicius/le-fabrique-fac-012k`. Arquivos limitados a `apps/worker/src`, `documentacoes/planejamento`, `documentacoes/operacao`, `documentacoes/INDEX.md`, `CHANGELOG.md`, `BACKLOG.md` e lesson existente de heartbeat/quiescência. Checks: `npm test --workspace @le-fabrique/worker`, `npm run typecheck`, `npm run lint`, `npm run build`, `git diff --check`.

## Risco e orçamento
Risco principal: renovação da lease não mata por si só o processo externo. O helper depende de callback de quiescência injetado e jamais pode liberar um writer substituto sem confirmação. Até duas rodadas de correção. Sem provider de IA; fixtures sintéticas e sem gastos.

## Dependências
FAC-008 (lease/fencing/checkpoint) e OPS-005 como preparação host futura; nenhuma integração/aceite desses tickets é presumida por este utilitário isolado.

## Entregáveis e documentação afetada
Código/testes do worker; relatório operacional datado; README operacional, índice, changelog, backlog e lesson `worker-heartbeat-quiescencia.md` atualizados.

## Evidências e aceite
READY em 2026-10-03; código implementado na revisão `8f254e8`. Evidências e limitações em `documentacoes/operacao/2026-10-03-FAC-012K-lease-vivo-cancelamento.md`. Aceite humano da revisão documental exata obrigatório antes de integrar ou ligar ao consumer.
