# FAC-012G — Workflow com rota por função

## Resultado

O `DeveloperWorkflow` consulta `ConfiguredAgentRouter` antes de cada tentativa Developer e Reviewer, inclusive em rodadas de correção. Cada runtime recebe o modelo e permissão da rota configurada para sua função, e o `RuntimeGuard` contabiliza o provider roteado. Ausência de rota ou Reviewer não-read-only resulta em falha segura antes da chamada que violaria a política. Código nos commits `e1e9828378061f85758de68f59bfa7bd65b81dc4` e `4bed9780ecb8a4791a137d1382a998507452e17b`.

## Escopo e evidências

- Código: `apps/worker/src/developer-workflow.ts`; testes sintéticos em `apps/worker/src/developer-workflow.spec.ts`; motivo aditivo `RUNTIME_ROUTE_UNAVAILABLE` em `packages/contracts/src/index.ts`.
- Diff sanitizado: seleção de adapter/modelo baseada em configuração existente, proteção Reviewer, motivo estruturado de bloqueio e testes; nenhuma conta, token, configuração real ou dado externo.
- Verificações: `npm run lint` (145 arquivos), `npm run typecheck`, `npm test` (187 testes), `npm run build` e `git diff --check` passaram.
- Não houve execução de provider, banco, fila, checkout, migration ou piloto. `apps/worker/src/main.ts` não foi alterado e o consumer segue no probe.

## Limites, rollback e aceite

O workflow ainda não é chamado pelo consumer. Faltam perfil operacional confiável/checkout e isolamento end-to-end do writer; a política `maxProviderSwitches` precisa ser compatível com providers distintos em um perfil validado. A validação sob a identidade systemd do serviço e o piloto são manuais posteriores. Rollback: reverter o commit de implementação FAC-012G, sem alteração de banco. Aguarda aceite humano da revisão exata; não mesclado nem implantado.
