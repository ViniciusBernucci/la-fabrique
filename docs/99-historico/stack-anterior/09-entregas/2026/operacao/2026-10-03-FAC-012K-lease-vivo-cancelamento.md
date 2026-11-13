# FAC-012K — lease vivo e cancelamento conservador

Status: AWAITING_HUMAN  
Branch: `feat/fac-012k-lease-guard`  
Worktree: `/home/vinicius/le-fabrique-fac-012k`  
Baseline: `ce93b20440c3f7ecc216ae6513396596cdf9a76e` (`developer`)  
Revisão do código: `8f254e8` (`feat(fac-012k): add conservative lease guard`)

## O que mudou

`apps/worker/src/lease-guard.ts` fornece `LeaseGuard`, uma primitiva injetável que vincula renovações ao fencing token, agenda a renovação antes da expiração e limita cada chamada ao tempo restante da lease. Erro, resposta expirada/inválida ou chamada pendente até o vencimento aborta o `AbortSignal` da operação e aciona exatamente uma tentativa de parada. `LeaseAuthorityLostError.writerQuiescent` fica `true` somente se o callback `stopWriter` confirmar; exceções e `false` permanecem não quiescentes.

A guarda espera a operação e eventual renovação/parada em andamento antes de resolver ou rejeitar, não trata timeout como morte do processo e não aceita resultado após perda de autoridade. Duração segue limites do contrato da API (15–300 segundos), o fencing token deve ser positivo e a renovação deve devolver expiração futura.

O helper é uma biblioteca isolada: não foi conectado a `main.ts`, BullMQ, `ControlClient`, `DeveloperWorkflow`, checkout, adapter, banco, Redis nem sandbox. O atual worker em execução não foi lido, parado ou alterado. Este incremento não habilita execução autônoma.

## Critérios e evidência

- Renovação periódica repassa o fencing token; timer termina com o trabalho normal.
- Falha e vencimento de renovação abortam, chamam parada uma vez e rejeitam sem aceitar sucesso; confirmação e ausência de confirmação são diferenciadas.
- Validação de intervalo impede renovação no próprio prazo de expiração.
- `apps/worker/src/lease-guard.spec.ts`: cinco testes novos, relógio falso e dependências sintéticas, cobrindo sucesso, rejeição, parada confirmada/não confirmada, renovação pendente até expirar e configuração inválida.

## Verificações

Na worktree isolada: `npm ci` instalou o lockfile (211 pacotes adicionados, 0 vulnerabilidades). O primeiro `npm test --workspace @le-fabrique/worker` não conseguiu carregar `@le-fabrique/runtime/dist` ainda não gerado; após `npm run build -w @le-fabrique/contracts && npm run build -w @le-fabrique/runtime`, passaram 86 testes do worker. A primeira suíte geral também apontou Prisma Client local desatualizado; `npm run db:generate` regenerou somente o artefato local, sem conexão ou migration.

Na revisão de código `8f254e8`, passaram `npm run typecheck`, `npm run lint` (151 arquivos), `npm run build`, `npm test` (211 testes: launcher 2, contracts 21, runtime 42, API 55, worker 86, web 5) e `git diff --check`. Após a verificação completa, uma validação adicional restringiu o intervalo de lease aos limites do contrato; depois dela passaram novamente typecheck do worker, seus 86 testes e `git diff --check`. Nenhuma alteração de código ocorreu após o commit citado; documentação deste relatório será commit separado.

Nenhum provider IA elegível foi necessário/usado. Nenhuma conta, modelo ou segredo foi configurado ou embutido. Não houve chamada a Codex/Claude/GitHub, fila ou serviço ativo, nem piloto externo, migration, push ou deploy.

## Limitações e próximo passo

O cancelamento real ainda precisa de integração que mate e confirme toda a árvore de processos do CLI/sandbox; o `DeveloperWorkflow` hoje não recebe esse `AbortSignal`, e o retorno de `stopWriter` é uma fronteira de confiança. Portanto a primitiva sozinha não autoriza claim concorrente ou retry. O ticket seguinte deve integrar lease, fencing, tentativa/checkpoint e cancelamento comprovado sem ligar o consumer antes de testes sintéticos completos e revisão operacional.

As evidências provam somente a lógica determinística da primitiva. Não provam API disponível durante provider longo, semântica do cancelamento dos CLIs, fencing end-to-end, systemd/cgroups sob a identidade do host nem execução de piloto.

## Rollback e aceite

Nada foi aplicado fora da worktree. Rollback do incremento de código é reverter `8f254e8`; os registros de ticket/relatório podem permanecer como histórico. Não ligar o consumer ao reverter e não restaurar o consumer fixture antigo. Aceite humano pendente para a revisão documental exata; sem merge para `developer` ou push.
