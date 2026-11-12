# FAC-012N — Reconciliação após checkpoint

Status: AWAITING_HUMAN. Data: 2026-10-03. Código: `94b5d641030072ef1f06e2b815eec0bec7f7880d`; 257 testes/checks passaram. Baseline: `f3b9b058c58ffbee09950f27a9147fd93e583521`.
Branch/worktree: `feat/fac-012n-reconcile`, `/home/vinicius/le-fabrique-fac-012n`.

## Objetivo e escopo

Fechar a janela de crash entre checkpoint com parada comprovada e complete. Replay reconcilia evidência persistida sem checkout, chamada de IA ou novo writer. Não recupera resultado ainda não persistido ou writer de estado desconhecido; journal e recuperação de interrupção ficam no próximo incremento.

Caminhos: packages/contracts, API orchestration, worker control-client/execution.processor, testes e documentação desses domínios. Sem migration, provider, piloto, fila/banco/serviço real, login, push/deploy. Provider elegível: nenhum; somente fixtures. Conta/modelo continuam configurados pela interface. Um writer local, processos de serviços existentes não serão alterados. Baseline FAC-012M: 240 testes/checks verdes. Até duas rodadas de correção, depois checkpoint/diagnóstico.

## Critérios

1. Endpoint WorkerAuthGuard estrito recebe apenas workerId/fence; não recebe outcome ou prova de parada inventada.
2. Sem checkpoint parado e attempt parado, retorna pendente sem mutações; tokens/owner obsoletos são rejeitados.
3. Outcome deriva do checkpoint imutável. COMPLETED/PAUSED exigem relatório válido coerente com status/snapshot; não há DONE automático.
4. Replay terminal idêntico é idempotente; não regride estado humano nem muda outcome terminal. Transação serializável protege leitura e conclusão.
5. Consumer usa reconciliação no replay sem preparar checkout/workflow; falha de rede propaga, não inventa sucesso.
6. Testes positivos/negativos, lint/typecheck/tests/build/diff; READMEs/índices/backlog/changelog/lessons e relatório com SHA exato. AWAITING_HUMAN até aceite.

## Limites/rollback

Sem retomar execução nem liberar writer desconhecido. Reverter código com writer parado; nenhum banco alterado. Relatório: `documentacoes/operacao/2026-10-03-FAC-012N-reconciliacao-checkpoint.md`.
