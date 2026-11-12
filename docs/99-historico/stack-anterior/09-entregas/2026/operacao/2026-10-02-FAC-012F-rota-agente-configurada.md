# FAC-012F — Rota de agente por configuração

## Resultado

Implementado em `950f3b62651b1e918bae09d89996af0d95e9fbee`. `ConfiguredAgentRouter` obtém um snapshot novo das configurações para cada resolução, valida a atribuição de função/provider/instalação/modelo e devolve a rota junto ao adapter injetado correspondente, versão e instante observado. Não mantém cache, não escolhe fallback e não executa o adapter. `REVIEWER` com permissão de escrita falha fechado.

## Escopo e evidências

- Arquivos de código: `apps/worker/src/agent-route.ts`, `apps/worker/src/configured-agent-router.ts` e testes em `apps/worker/src/configured-agent-router.spec.ts`.
- Documentação de domínio: configuração, runtime e operação; lesson de contratos runtime atualizada.
- Diff sanitizado: somente resolução de rota, metadados do snapshot e testes; nenhuma configuração local, credencial, conta/modelo real, piloto, banco ou fila foi incluída ou alterada.
- Verificações: `npm run lint` (145 arquivos), `npm run typecheck`, `npm test` (185 testes), `npm run build` e `git diff --check` passaram no worktree da feature.
- Sem migration, chamada a provider, checkout ou consumer real. Testes usam snapshots/adapters sintéticos.

## Limites, rollback e aceite

O BullMQ ainda usa o probe sintético; o router não chama `DeveloperWorkflow`. Ainda faltam integração com checkout/perfil confiável, validação de disponibilidade operacional recente sob a identidade do serviço, isolamento do CLI Developer e prova de ponta a ponta. O piloto segue manual e não foi cadastrado. Rollback do incremento: reverter o commit `950f3b6` (sem alteração de schema/dados). Estado: aguarda revisão e aceite humano desta revisão exata; não mesclado nem implantado.
