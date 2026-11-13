# FAC-012M — Resultados de execução no painel

Status: AWAITING_HUMAN. Data: 2026-10-03. Código revisável: `14ae5baf9fe7b7ba98934ac581e4788e794da56c`; checks locais concluídos, sem aceite humano.
Baseline: `d24245a83e8686373f237a2a7fcb2db6cba69746`.
Branch/worktree: `feat/fac-012m-execution-results`, `/home/vinicius/le-fabrique-fac-012m`.

## Objetivo

Permitir ao operador acompanhar runs e consultar resultado estruturado (checks/revisão/snapshots e modelos/uso efetivamente observados). Hoje os resultados do workflow são locais e o painel não oferece consulta de runs. A entrega preserva o objetivo global: artefatos/diff completos, recuperação, handoff, isolamento por instalação e controles de run seguem incrementos posteriores, sem redefinir conclusão do MVP.

## Escopo

Contratos compartilhados, persistência aditiva de relatório por tentativa, protocolo interno autenticado/fenced, consulta administrativa, observações do runtime e painel de execuções. Migration apenas versionada, nunca aplicada. Sem serviço/banco/fila ativos, provider/repo remoto, piloto, instalação/login, API de IA, push ou deploy. Provider elegível: nenhum; fakes/fixtures somente.

## Critérios de aceite

1. Relatório estrito e limitado não inclui workspace/artefact paths, prompt, stdout/stderr, provider session ou credenciais. Uso/modelo efetivo desconhecidos permanecem null/sem estimativa.
2. Worker registra papel/instalação/provider/modelos/uso da chamada real e configuração observada; contas/modelos continuam vindos da interface.
3. Relatório imutável/idempotente por tentativa exige worker/fencing atuais. Repetição idêntica não duplica; conteúdo divergente ou fence obsoleto é rejeitado. Relatar não altera stoppedConfirmed nem libera writer.
4. Consumer preserva relatório antes da conclusão aprovada/pausada/falha; persistência indisponível impede declarar entrega concluída.
5. API administrativa lista runs filtrados por projeto e detalhe de tentativas/resultado/checkpoint, com schema runtime e paginação/limite; API não executa clientes ou arquivos do worker.
6. Painel mostra estados, checks baseline/pós-alteração, revisão, snapshots e observações de modelos/uso. Polling é limitado; troca de projeto/token/unmount descarta respostas antigas. Nenhum botão aprova, retoma ou publica automaticamente.
7. Testes cobrem autorização/fence/idempotência, payload indevido, sanitização/unknown e cliente UI. Typecheck/lint/testes/build/diff check passam; docs/índices/lessons atualizados e aceite humano pendente.

## Risco, limites e evidência

Um writer; até duas rodadas de correção, depois checkpoint/diagnóstico. Dados públicos do relatório são bounded e minimizados. Baseline completa OPS-006 tem 227 testes verdes; dependências locais podem exigir npm ci/db:generate (sem conexão/migration). Relatório final em `documentacoes/controle/2026-10-03-FAC-012M-resultados-execucao-painel.md`. DONE só após aceite humano da revisão exata.
