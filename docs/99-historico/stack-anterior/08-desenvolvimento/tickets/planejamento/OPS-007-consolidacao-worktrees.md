# OPS-007 — Consolidar FAC-012Z–AH e remover worktrees incorporadas

Status: AWAITING_HUMAN. Data: 2026-10-05.

## Objetivo e autorização

Pedido explícito do responsável: integrar todas as worktrees na developer e apagar as órfãs. Escopo local: FAC-012Z, AA, AB, AC, AD, AE, AF, AG e AH. Não inclui push, deploy, migrations, alteração de serviços ou aceite dos tickets funcionais.

## Baseline, caminhos, provider e limites

Developer: `745ce2d25b6120541085b4500e149cc8d313380f`. Revisão consolidada: `8b42fe4ce858d800ce97fd67cfbccf39cc239874`, branch `fix/fac-012ah-admission-capacity`. Todas as nove branches são ancestrais desta revisão e estão limpas. Preparação documental isolada: `ops/ops-007-consolidacao-worktrees`, `/home/vinicius/le-fabrique-ops-007`. Provider elegível: nenhum; Git e checks determinísticos locais. Um writer por worktree; até duas rodadas de correção. O responsável reiterou autorização para integração após identificação das sessões antigas (31401, 1438706, 3946330); não foram encerradas por esta operação. Permissões de escrita foram restauradas explicitamente.

## Critérios

1. Confirmar ausência de outro writer e estado limpo antes do fast-forward autorizado.
2. Validar typecheck, lint, testes, build e diff-check sobre a revisão consolidada, sem alterar banco existente.
3. Provar ancestralidade das nove revisões na developer resultante.
4. Remover somente worktrees incorporadas, limpas e sem processos em uso, sem force; preservar branches e arquivos privados/ignorados que não sejam artefatos regeneráveis.
5. Atualizar relatório, README operacional, índices, backlog/changelog e lesson aplicável. Aceite da revisão exata permanece separado da autorização de integração.

## Execução

Fast-forward local executado e nove worktrees funcionais removidas sem force após verificar limpeza e inclusão. Aceite funcional permanece separado. [Relatório](../../../09-entregas/2026/operacao/2026-10-05-OPS-007-consolidacao-worktrees.md).
