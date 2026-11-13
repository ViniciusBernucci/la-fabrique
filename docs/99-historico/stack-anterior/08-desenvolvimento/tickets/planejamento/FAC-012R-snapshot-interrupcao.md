# FAC-012R — Snapshot de interrupção comprovada

Status: AWAITING_HUMAN. Código `abfb033a041818131b1d05cfe71735f18e78f7e2`; 315 testes/checks passaram. Data: 2026-10-03. Baseline: `297d308`. Branch/worktree: `feat/fac-012r-interruption-snapshot`, `/home/vinicius/le-fabrique-fac-012r`.

## Objetivo e escopo

Quando AbortSignal interrompe workflow após criação/contexto, preservar observações parciais e novo snapshot somente após término conhecido de clientes/checks; publicar/journalar CANCELLED sem perder patch/untracked. Perda de lease continua fenced e writer desconhecido não libera tentativa. Falha antes de workspace/contexto e crash abrupto do host sem retorno permanecem bloqueados para recuperação explícita, não são retomados por suposição.

Caminhos: contracts, workflow/processor/journal worker e testes, reconciliação API; documentação runtime/operação/controle e lessons. Sem migrations/serviço/banco/fila real/provider/login/piloto/push/deploy. Um writer local; providers nenhum/fixtures; baseline 306 testes. Contas/modelos na interface. Até duas correções, depois checkpoint/diagnóstico.

## Critérios

1. Resultado CANCELLED/INTERRUPTED preserva checks/chamadas parciais; snapshot posterior à parada, no máximo três manifestos no DTO sem apagar artefatos locais.
2. Término desconhecido de check/exception de processo fica sticky: remover callback do Set não pode converter unknown em quiescência.
3. Cancelamento retorna após cliente/check encerrar; nenhum novo Developer/Reviewer começa. Falha de captura não declara cancelamento com evidência fictícia.
4. Consumer journal/result/artifact/checkpoint/complete suporta CANCELLED e persiste resultado de lease loss comprovado; API reconciliation deriva CANCELLED do relatório/checkpoint coerentes, sem validação/aprovação.
5. Testes de abort/cancel/unknown/snapshot/failure e checks completos; docs atualizadas com SHA exato; AWAITING_HUMAN.

## Rollback

Reverter código com writer parado, preservando snapshots/journal. Sem schema de banco alterado. Relatório em `documentacoes/operacao/2026-10-03-FAC-012R-snapshot-interrupcao.md`.
