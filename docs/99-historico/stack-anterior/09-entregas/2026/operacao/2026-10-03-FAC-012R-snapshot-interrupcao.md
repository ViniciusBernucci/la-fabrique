# FAC-012R — Snapshot de interrupção comprovada

Data: 2026-10-03. Status: AWAITING_HUMAN. Baseline `297d308`; ticket READY `3a937c8`; código `abfb033a041818131b1d05cfe71735f18e78f7e2`.

## Objetivo e funcionamento

AbortSignal durante workflow após workspace/contexto preserva observações parciais e snapshot novo somente depois da parada conhecida. Runtime observa a chamada retornada antes de classificar interrupção; checks são registrados individualmente antes de abortar a sequência. Nenhum novo Developer/Reviewer começa. Resultado CANCELLED/INTERRUPTED não equivale a sucesso/aprovação.

Unknown fica sticky em DeveloperWorkflow: resultado de check sem stoppedConfirmed ou promessa de execução rejeitada não vira quiescência só porque callbacks saíram do Set. cancelActive permanece false e nova execução na mesma instância é recusada. Captura de snapshot é externa ao código e posterior à parada. DTO retém até três manifestos recentes, sem apagar nenhum artefato local.

Consumer inclui CANCELLED no journal/result/artifact/checkpoint/complete. Perda de lease com resultado cancelado/quiescência confirmada preserva evidência antes de concluir, mantendo fence original. Sem resultado recuperável após workflow iniciado, InterruptionEvidenceError mantém attempt fenced (não registra checkpoint fictício). Falha de captura também não libera tentativa. API reconcile aceita relatório CANCELLED coerente com checkpoint/snapshot/digest, nunca transforma isso em VALIDATING/DONE.

## Checks e diff sanitizado

npm ci/db:generate locais sem banco, typecheck, todos os testes, build, lint e git diff --check passaram. 315 testes: launcher 2, contracts 32, runtime 44, API 102, worker 125, web 10; lint 177 arquivos.

Primeira rodada de testes encontrou expectativa antiga de reject no abort (agora deve retornar evidência CANCELLED) e fixture com status de check CANCELLED, que não pertence ao contrato do SandboxRunner. Atualizada expectativa e fixture para TIMED_OUT, classificação real de abort do sandbox; nenhum enum fictício adicionado a checks. Rodada final passou completa.

Novos testes cobrem baseline interrompida/Developer cancelado com observação e snapshot, processo/check unknown sticky, falha de captura sem evidência, journal cancelado, lease loss preservado e reconcile cancelado sem aprovação. Tests de runtime/Linux existentes continuam verdes; novos cenários usam portas sintéticas, não cliente/serviço autenticado ou reboot real.

Diff `git diff 3a937c8 abfb033a041818131b1d05cfe71735f18e78f7e2`: 9 arquivos, 511 inserções/290 remoções, principalmente indentação do bloco try/catch no workflow. Revisão sem whitespace: `git diff -w 3a937c8 abfb033`. Alterações semânticas: CANCELLED/INTERRUPTED, progress callbacks/sticky termination/snapshot, finalização conservadora e journal/reconcile/tests. Sem segredo real, conta/modelo/piloto hardcoded.

## Limitações, rollback, documentação e aceite

Não recupera kill abrupto/queda do host ou falha antes de workspace/contexto. Não agenda retry de jobs FAILED, não restaura/retoma automaticamente, não preserva novas mudanças de Developer com falha normal fora de AbortSignal, e não substitui prova do serviço/identidade/Claude. Artefato acima do limite P ou indisponibilidade de API preserva journal/snapshot mas impede conclusão.

Nenhum provider/login, serviço, banco/fila, migration, piloto, push/deploy iniciado; gate execução false. Rollback por revert com writer parado preservando journal/snapshots; nenhum banco alterado. READMEs runtime/operação/controle, índices/backlog/changelog/controle e lesson lifecycle atualizados. Integração local autorizada após checks/árvore limpa, branch preservada na limpeza. AWAITING_HUMAN até aceite do SHA exato; MVP ainda incompleto.
