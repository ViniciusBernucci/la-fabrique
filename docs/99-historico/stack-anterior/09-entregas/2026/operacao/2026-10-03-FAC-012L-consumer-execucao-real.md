# FAC-012L — Consumer de execução real com ativação explícita

Data: 2026-10-03. Status: IMPLEMENTADO / AWAITING_HUMAN.
Domínio: operação/runtime. Branch: `feat/fac-012l-real-execution`.
Baseline: `a305b38607bb70c5747b4fa8d6f0e5a579621258`; ticket READY: `972c558`.
Revisão do código: `54bf483473cc2f2c75b7d6b6d52fa151d47455dc`.

## Objetivo e critérios de aceite

Conectar checkout, compilador, workflow e lease/fencing ao consumer de execução sem embutir piloto, contas ou modelos. Critérios detalhados no [ticket FAC-012L](../../../08-desenvolvimento/tickets/planejamento/FAC-012L-consumer-execucao-real.md). Ativação operacional continua manual.

## Implementação e funcionamento

`main.ts` só registra `le-fabrique.execution` com `WORKER_EXECUTION_ENABLED=true`; default `false`. Desabilitado, os jobs aguardam na fila, sem consumo pela fixture antiga que podia produzir VALIDATING sintético. O probe mantém sua fila própria.

`processOrchestrationExecution` valida envelope estrito e snapshot READY consistente antes do claim. Reentrega com `replayed=true` não cria segundo checkout/writer. Checkout recebe projeto/URL/SHA imutáveis e ID da tentativa; perfil reconfere o vínculo retornado antes de compilar contexto/checks aprovados na definição versionada. Não há projeto externo fixado no código.

`LeaseGuard` renova o fencing token durante checkout/workflow. Falha ou expiração abortam e aguardam operação e callback de parada. Sem confirmação não há checkpoint terminal nem complete. Shutdown/heartbeat abortam execução antes de fechar a fila. DeveloperWorkflow propaga AbortSignal aos clientes/checks, solicita cancelamento e espera término. SandboxRunner encerra unidade/cgroup e só confirma parada mediante resposta conhecida de systemd; erro de consulta não equivale a parada.

Developer/Reviewer consultam o Centro de Configurações em cada chamada. Instalação/modelo, timeout e máximo de tentativas por função vêm daquele snapshot; os tetos globais limitam o executor. Rota ausente, adapter incompatível, Reviewer com escrita e flags financeiras proibidas falham fechados. Não há fallback API.

Checkpoint precede complete com SHA-base e snapshot/revisão/digest disponíveis. Aprovação do Reviewer produz VALIDATING, nunca DONE; limite/falha/cancelamento mantêm seus estados. Check sem parada comprovada impede a conclusão.

## Configuração posterior

Novos valores: `WORKER_EXECUTION_ENABLED=false`, `WORKER_LEASE_DURATION_MS=90000`, `WORKER_EXECUTION_ROOT`, `WORKER_CODEX_BINARY` e `WORKER_CLAUDE_BINARY`. Checkout root/hosts já existiam. Habilitar exige raízes absolutas separadas, hosts DNS exatos e CLIs absolutos. Examples mantêm o gate desligado; nenhum ambiente real foi editado.

Antes de habilitar: topologia host OPS-005, identidade dedicada, autenticação oficial, versões/binários e systemd/namespaces/cgroups verificados. Confirmar credenciais inacessíveis ao código do projeto e proteções financeiras. Contas/atribuições/modelos continuam configurados no painel. O worker Docker legado e a identidade pessoal dos testes não substituem o preflight operacional.

## Alterações e diff sanitizado

Diff recuperável: `git diff 972c558..54bf483 -- apps/worker/src packages/runtime/src/sandbox-runner.ts packages/runtime/src/sandbox-runner.spec.ts .env.example .env.production.example`. Contém fixtures e configuração genérica, sem credenciais. Novos módulos: `execution.processor.ts`, seus testes e `execution-profile.ts`.

## Verificação

Passaram `npm run typecheck`, `npm run lint`, `npm run build`, `git diff --check` e `npm test` (225 testes). Após os dois casos finais de envelope/configuração, passaram novamente typecheck do worker, seus 100 testes, lint e diff check. Total final comprovado: 227 testes distintos — launcher 2, contracts 21, runtime 44, API 55, worker 100, web 5.

Processor: aprovação/checkpoint antes de complete, replay sem writer novo, conflito, falha/limite, checkout divergente, parada não confirmada, override de modelo rejeitado e shutdown. LeaseGuard existente cobre renovação, erro, vencimento e chamada pendente. Workflow cobre rota ausente/cancelamento. Dois novos testes Linux do SandboxRunner comprovam ausência de confirmação quando systemd não pode ser observado e aborto do comando/descendente, com heartbeat deixando de crescer após retorno.

Duas correções das fixtures foram necessárias: exposição do mock de cancelamento e manifesto de snapshot conforme contrato. O primeiro typecheck geral encontrou Prisma Client não gerado no worktree; `npm run db:generate` gerou apenas o artefato local, sem banco/migration. Não houve regressão da API.

## Limitações e próximos passos

Teste Linux ocorreu na identidade interativa com fixtures locais. Identidade de serviço, clientes autenticados, Git/GH/keyring e isolamento operacional seguem NÃO VERIFICADOS. Nenhum provider, repositório remoto, banco ou fila ativa foi usado.

Replay evita segundo writer, mas não reconcilia automaticamente parada entre checkpoint/complete. Falha de persistência exige recuperação controlada. Artefatos ficam no disco do worker; transporte/diff no painel, retomada e handoff automático ainda não estão ligados a este consumer. Cancelamento pode preservar arquivos em workspace sem novo manifesto terminal; não descartá-los.

O protocolo complete admite VALIDATING/PAUSED_LIMIT/FAILED/CANCELLED; auth/cota ainda não produzem WAITING_PROVIDER persistido ou troca automática. A composição usa um adapter por provider sob a identidade do worker; isolamento de múltiplas contas por instalação exige integração própria. Claude não tem a allowlist granular de escrita validada para Codex. Esses limites impedem declarar o MVP inteiro concluído.

## Rollback

Desligar admission e confirmar término das árvores antes de rollback operacional. `WORKER_EXECUTION_ENABLED=false` deixa jobs aguardando e preserva demais consumers. Reverter `54bf483` em revisão isolada para desfazer código; preservar checkpoints/workspaces/artefatos. Não restaurar a fixture em serviço.

## Documentação e lessons

Atualizados README raiz/operação/runtime/planejamento, ticket, backlog, changelog e índice. Lessons: [heartbeat/quiescência](../../../10-lessons/10-worker-heartbeat-quiescencia.md), [checkout](../../../10-lessons/05-checkout-confiavel.md), [lifecycle CLI](../../../10-lessons/08-lifecycle-processo-cli.md). Stack/topologia aprovadas preservadas.

## Uso de IA e aceite

Agente da sessão implementou o incremento; modelo efetivo/tokens/custo atribuível não disponíveis por telemetria confiável. Testes não chamaram clientes para inferência e não habilitaram gastos extra/API. Aceite humano da revisão exata pendente. Merge local autorizado não equivale a aceite/deploy.
