# FAC-012O — Journal de resultados antes da API

Status: AWAITING_HUMAN. Código `016a5ef088e5e46c178b4558ffdf4a6bec0d8bea`; 266 testes/checks passaram. Data: 2026-10-03. Baseline: `0f96dd3`. Branch/worktree: `feat/fac-012o-result-journal`, `/home/vinicius/le-fabrique-fac-012o`.

## Objetivo e escopo

Preservar relatório normal de workflow já parado antes de enviá-lo ao controle. Redelivery/reinício recupera o envio/checkpoint/complete do mesmo attempt/fence sem nova chamada de IA, inclusive após lease expirar. Não retoma writer desconhecido nem recupera trabalho interrompido antes do journal; próximos incrementos tratam esses casos.

Caminhos: worker result-journal/execution.processor/main e testes; documentação operação/runtime/contratos e lessons. Sem API/migration, serviço/fila/banco real, provider/login/piloto/push/deploy. Root local privado fica fora de workspaces e sandbox; accounts/models permanecem configuráveis pela interface. Provider elegível: nenhum; fixtures somente. Um writer local; serviços existentes intocados. Baseline 257 testes; no máximo duas rodadas de correção.

## Critérios

1. Journal bounded/strict contém apenas IDs/fence, relatório público e checkpoint/outcome derivado; nunca prompt/session/credencial/workspace. Root privado, publicação atômica/fsync e rejeição de symlink/arquivo excessivo.
2. Somente após workflow retornar e parada confirmada; gravação precede API e falha impede conclusão.
3. Entrada é imutável por evento; repetição idêntica é idempotente, alteração rejeitada. Leitura valida job/event/projeto/ticket/base e worker atual via API.
4. Recuperação acontece antes de claim: reenvia result/checkpoint/complete autenticados com fence original, sem checkout/workflow/IA, nunca infere stop de lease expirada. API existente recusa owner/fence stale.
5. Journal retido após conclusão, sem limpeza destrutiva automática; erros/corrupção falham fechados.
6. Testes de reinício/falhas/limites/replay, checks completos e documentação com SHA/diff/lessons; AWAITING_HUMAN.

## Rollback

Reverter código com writer parado preservando journal/snapshots. Nenhuma alteração no banco. Relatório em `documentacoes/operacao/2026-10-03-FAC-012O-journal-resultados.md`.
