> Leitura: [Índice didático](../02-INDEX.md) · [Próximo →](01-definicao-projeto-configuravel.md)

# Lessons

FAC-022: [template persistente e estado dos painéis](13-modal-edicao-configuracao.md), com identidade de nós DOM comprovada no navegador.


FAC-020A/FAC-021: [verificar serviço ativo e nome apresentado](11-baseline-regressao-review.md), distinguindo integração Git, browser na porta habitual e identificadores técnicos estáveis.


FAC-020 aplica [modais e prévias](13-modal-edicao-configuracao.md) e [baseline/verificação visual](11-baseline-regressao-review.md), com dados explicitamente simulados e foco/keyboard comprovados em browser.


FAC-018: [lifecycle e autorização CLI](08-lifecycle-processo-cli.md); FAC-019: [abas e modais](13-modal-edicao-configuracao.md).

FAC-017 valida formato real do cliente e distingue chunks de linhas completas: [lifecycle CLI](08-lifecycle-processo-cli.md).

FAC-016 coloca resultado das ações dentro do dialog ativo, com progresso e diagnóstico sanitizado: [modais](13-modal-edicao-configuracao.md).

FAC-015 distingue autenticação da conta, catálogo observado e autorização de execução: [perfis de clientes](07-perfis-permissao-cliente-cli.md).

FAC-014 reutiliza [modal com rascunho separado](13-modal-edicao-configuracao.md) e distingue cancelar campos de desfazer uma ação externa já enviada.
FAC-013 aplica [rascunho separado e modal de configuração](13-modal-edicao-configuracao.md), com salvamento versionado e limites de verificação explícitos.
OPS-009 aplica reconciliação por efeitos SQL comprovados, preservando histórico: [baseline/review](11-baseline-regressao-review.md).
OPS-008 diferencia conectividade de compatibilidade do schema e valida ambiente num subprocesso real: [baseline/review](11-baseline-regressao-review.md).
OPS-007 aplica prova de ancestry e identidade da árvore funcional antes de remover worktrees: [sandbox/worktree/snapshot](06-sandbox-worktree-snapshot.md).
FAC-012AH aplica retry pré-autoridade distinto de repetir execução: [leases/fencing](09-leases-fencing-checkpoints.md).

FAC-012AG aplica pausa de autoridade distinta de entrega/stop e deferral pré-claim: [leases/fencing](09-leases-fencing-checkpoints.md).

FAC-012AF aplica backup autenticado/staging e restore real distintos de snapshot/parada: [sandbox/snapshot](06-sandbox-worktree-snapshot.md).

FAC-012AE aplica cursor transacional em ordem de commit e invalidação SSE retomável: [outbox/idempotência](03-outbox-idempotencia.md).

FAC-012AD aplica observação atual distinta de prontidão/parada: [heartbeat/quiescência](10-worker-heartbeat-quiescencia.md).

FAC-012AC aplica exclusão persistida global independente de leases por run/concurrency: [leases/fencing](09-leases-fencing-checkpoints.md).

FAC-012AB aplica prova de composição com Git/sandbox/journal reais e portas externas sintéticas: [sandbox/worktree/snapshot](06-sandbox-worktree-snapshot.md).
FAC-012Y diferencia troca sequencial de cliente e transferência de autoridade do writer: [lifecycle CLI](08-lifecycle-processo-cli.md).

FAC-012X aplica indisponibilidade distinta de correção, sem supor stop: [lifecycle CLI](08-lifecycle-processo-cli.md).

FAC-012W aplica budgets raw/JSON/HTTP consistentes e leitura bounded: [contratos runtime](02-contratos-runtime-monorepo.md).

FAC-012V aplica identidade por instalação distinta de confinamento: [perfis de cliente](07-perfis-permissao-cliente-cli.md).

FAC-012U aplica retry de entrega distinto de execução: [leases/fencing](09-leases-fencing-checkpoints.md).

FAC-012T aplica retomada por estado externo/baseline pré-restore: [snapshots](06-sandbox-worktree-snapshot.md).

FAC-012S aplica intenção administrativa versus stop físico e confirmação por versão: [leases/fencing](09-leases-fencing-checkpoints.md).

FAC-012R aplica término unknown sticky e separa cancelamento comprovado de preservação de evidência; exemplo na lesson lifecycle, [evidências](../09-entregas/2026/operacao/2026-10-03-FAC-012R-snapshot-interrupcao.md).

FAC-012Q aplica revisão exata de patch/resultado/documentação e baseline no gate; [evidências](../09-entregas/2026/controle/2026-10-03-FAC-012Q-documentacao-aceite-entrega.md), exemplos na lesson baseline/review.

FAC-012P aplica integridade de bytes além de schemas e distingue limite/scan de garantia universal; exemplo na lesson contratos, [evidências](../09-entregas/2026/controle/2026-10-03-FAC-012P-artefatos-painel.md).

FAC-012O aplica journal antes de HTTP e distingue recuperar finalização de reexecutar IA; [evidências](../09-entregas/2026/operacao/2026-10-03-FAC-012O-journal-resultados.md), exemplo na lesson leases/fencing.

FAC-012N amplia a lesson de leases/fencing com reconciliação serializável sem retry de IA; [evidências](../09-entregas/2026/operacao/2026-10-03-FAC-012N-reconciliacao-checkpoint.md).

FAC-012M aplica DTO estrito/minimizado com desconhecidos explícitos e evidência imutável sem liberar writer; exemplos nas lessons de contratos e leases, [relatório](../09-entregas/2026/controle/2026-10-03-FAC-012M-resultados-execucao-painel.md).

OPS-006 atualiza a lesson de sandbox/worktree com identidade do user manager e separação entre integração Git e ativação operacional; [evidências](../09-entregas/2026/operacao/2026-10-03-OPS-006-integracao-local-controle-mvp.md).

FAC-012L aplica heartbeat/quiescência, lifecycle CLI e checkout confiável ao consumer gated; os conceitos existentes foram atualizados, sem criar lessons duplicadas. [Evidências](../09-entregas/2026/operacao/2026-10-03-FAC-012L-consumer-execucao-real.md).
Lessons locais têm exemplos/revisões reais preservados; não foram reexecutados nesta migração. Nenhuma lesson fictícia foi criada.

- [Contratos de runtime no monorepo TypeScript](02-contratos-runtime-monorepo.md)
- [Outbox idempotente e versão otimista](03-outbox-idempotencia.md)
- [Heartbeat e parada conservadora](10-worker-heartbeat-quiescencia.md)
- [Perfis de permissao para clientes CLI](07-perfis-permissao-cliente-cli.md)
- [Lifecycle seguro de um cliente CLI](08-lifecycle-processo-cli.md)
- [Contexto deterministico e limites conservadores](04-contexto-deterministico-limites.md)
- [Sandbox, worktree e snapshot sao limites diferentes](06-sandbox-worktree-snapshot.md)
- [Lease, fencing e checkpoint cobrem falhas diferentes](09-leases-fencing-checkpoints.md)
- [Baseline, regressao e review sao sinais diferentes](11-baseline-regressao-review.md)
- [Definicao de projeto e compilacao com fronteiras confiaveis](01-definicao-projeto-configuravel.md)
- [Checkout confiavel separa credencial e conteudo externo](05-checkout-confiavel.md)

FAC-012Z amplia [baseline/regressão/review](11-baseline-regressao-review.md) com gate documental estrutural e revisão semântica ligados ao snapshot final.

FAC-012AA amplia [perfis de permissão](07-perfis-permissao-cliente-cli.md) com allow literal, elegibilidade por prova privada e distinção entre teste do algoritmo e confinamento nativo.


FAC-023 amplia contratos-runtime-monorepo e modal-edicao-configuracao com cadastro aditivo, vínculos por projeto, limpeza referencial e erro de gravação dentro do dialog.

- [Vídeo decorativo com fronteira contínua — FAC-024](14-video-decorativo-loop.md)

FAC-025 amplia vídeo-decorativo-loop: prova técnica não é aceite visual; remoção do player inclui assets publicados, com histórico preservado.


FAC-026/027 ampliam modal-edicao-configuracao: controles da linha separados, foco entre dialogs e estado persistente do template com variável de layout.


FAC-028 amplia modal-edicao-configuracao com distinção entre moldura visual e nome/foco acessível de botão iconográfico.


FAC-029 amplia modal-edicao-configuracao com medição de alinhamento do ícone ao container nos dois estados.


FAC-030 amplia modal-edicao-configuracao: distinguir estado do template entre telas e padrão de nova abertura.


FAC-031 amplia modal-edicao-configuracao com alinhamento medido por estado do template.

- [Proveniência documental aplicada na migração](12-documentacao-proveniencia.md).

## Sequência de leitura deste capítulo

1. [definicao-projeto-configuravel](01-definicao-projeto-configuravel.md).
2. [contratos-runtime-monorepo](02-contratos-runtime-monorepo.md).
3. [outbox-idempotencia](03-outbox-idempotencia.md).
4. [contexto-deterministico-limites](04-contexto-deterministico-limites.md).
5. [checkout-confiavel](05-checkout-confiavel.md).
6. [sandbox-worktree-snapshot](06-sandbox-worktree-snapshot.md).
7. [perfis-permissao-cliente-cli](07-perfis-permissao-cliente-cli.md).
8. [lifecycle-processo-cli](08-lifecycle-processo-cli.md).
9. [leases-fencing-checkpoints](09-leases-fencing-checkpoints.md).
10. [worker-heartbeat-quiescencia](10-worker-heartbeat-quiescencia.md).
11. [baseline-regressao-review](11-baseline-regressao-review.md).
12. [documentacao-proveniencia](12-documentacao-proveniencia.md).
13. [modal-edicao-configuracao](13-modal-edicao-configuracao.md).
14. [video-decorativo-loop](14-video-decorativo-loop.md).
