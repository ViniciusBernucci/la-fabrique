# FAC-012L - Consumer de execução autônoma
Status: AWAITING_HUMAN

## Objetivo
Conectar evento `le-fabrique.execution` ao checkout confiável, compilador de workflow, runtime configurado e protocolo de claim/lease/fencing/checkpoint, sem embutir projeto, conta ou modelo.

## Atual e esperado
As bibliotecas de checkout, compilação, workflow e lease existem, mas `main.ts` ainda consome o evento com `processOrchestrationFixture`, que pode marcar execução sintética como `VALIDATING`. O consumer real deverá substituir esse fixture quando explicitamente habilitado; o default permanece desligado. Cada job precisa materializar checkout no SHA congelado, usar configuração de IA atual da interface e persistir checkpoint antes da conclusão.

## Escopo permitido e proibido
Permitido: worker/configuração, processo de orquestração injetável, cancelamento/quiescência, testes sintéticos, configuração local genérica e documentação. Proibido: conectar banco/Redis ativos, habilitar em serviço, instalar/iniciar systemd, autenticar clientes, chamar provider/GitHub/repo remoto, cadastrar piloto, executar migration, push ou deploy.

## Critérios de aceite verificáveis
1. `WORKER_EXECUTION_ENABLED` é `false` por padrão; `true` exige raízes absolutas, hosts Git explicitamente permitidos e configuração de cliente segura. O consumer fixture nunca é usado em modo habilitado.
2. Consumer real valida job estrito, obtém claim idempotente e fencing token, renova lease durante checkout/workflow, e envia checkpoint/conclusão com o mesmo fencing token.
3. Checkout vincula project/repository/SHA do evento READY e fica fora do workspace gravável. O compilador reconcilia caminhos, contextos e checks com a definição versionada; nenhuma rota de provider/modelo vem do job.
4. Developer/Reviewer são resolvidos em cada chamada pelo snapshot atual do Centro de Configurações; falta de rota/adapter, Reviewer com escrita, auth/cota e configurações financeiras proibidas falham fechados, sem fallback por API.
5. Perda/expiração da lease envia cancelamento a todo subprocesso ativo; workflow e sandbox aguardam término e a parada só é confirmada com evidência de processo. Sem quiescência não concluir nem liberar writer.
6. Resultado aprovado vai a `VALIDATING`; limite/falha/cancelamento mapeiam estados sem marcar aprovação falsa. Checkpoint carrega SHA-base e snapshot/hash quando disponíveis e é idempotente.
7. Tests sintéticos cobrem caminho aprovado, configuração/rota ausente, conflito/idempotência, renovação falha/timeout, cancelamento e não-quiescência. Nenhum teste abre provider/rede/banco/fila.
8. Typecheck, lint, testes, build e `git diff --check` passam; documentação explica ativação manual posterior e rollback seguro.

## Baseline e comandos de checks
Baseline `a305b38607bb70c5747b4fa8d6f0e5a579621258` na branch `feat/fac-012k-lease-guard`. Branch isolada `feat/fac-012l-real-execution`, worktree `/home/vinicius/le-fabrique-fac-012l`. Checks: `npm run typecheck`, `npm run lint`, `npm test`, `npm run build`, `git diff --check`. Nenhuma execução externa durante o ticket.

## Risco e orçamento
Alto: primeiro caminho que pode chamar clientes CLI e executar código de projeto. Default deve permanecer desabilitado; credenciais/controle nunca entram no sandbox. Até duas rodadas de correção e depois checkpoint. Provider elegível para esta implementação: nenhum; fakes determinísticos somente.

## Dependências
FAC-012A/B e C–K (os últimos aguardam aceite humano) e OPS-005 para futura topologia host. A implementação depende do código de trabalho dessas branches, não presume aceite nem autoriza ativação operacional.

## Entregáveis e documentação afetada
Composição/consumer/config do worker; testes; documentação operacional e planejamento; README do domínio; índice, changelog, backlog e lessons de lease/checkout/processo CLI.

## Evidências e aceite
READY em 2026-10-03. Escopo de integração deliberadamente sem provider/repositório externo. Aceite humano da revisão exata obrigatório antes de habilitar o consumer.

Implementação: `54bf483473cc2f2c75b7d6b6d52fa151d47455dc`. Evidências/limites/rollback: [relatório FAC-012L](../../../09-entregas/2026/operacao/2026-10-03-FAC-012L-consumer-execucao-real.md). Gate desabilitado; nenhum serviço ativado. Recuperação, artefatos no painel e handoff operacional permanecem pendentes.
