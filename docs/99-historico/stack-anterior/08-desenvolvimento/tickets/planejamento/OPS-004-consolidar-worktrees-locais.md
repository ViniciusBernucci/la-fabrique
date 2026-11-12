# OPS-004 — Consolidar worktrees locais autorizados

Status: AWAITING_HUMAN

## Objetivo

Consolidar em `developer` os branches FAC-012D e FAC-012E–J que o responsável pediu para integrar, executar verificações sobre a árvore combinada e retirar somente worktrees já integrados. Preservar os branches locais e os status de aceite humano; não publicar remotos.

## Critérios de aceite

1. Antes do merge, origem `developer`, worktrees/branches, SHAs, estado limpo e ancestry FAC-012A/B/OPS-003 estão verificados e registrados.
2. FAC-012D (`439e95429b81a4bd12e01c0d501d9241a18393ee`) e FAC-012J/ancestrais (`17fed5bd2d55eaa4dc4eb23d607cd98893f6c674`) são integrados à branch local `developer` sem perda de conteúdo; conflitos, se houver, são resolvidos e revisados por critério/arquivo.
3. Suíte total, typecheck, lint, build e diff-check passam na revisão exata resultante de `developer`.
4. D e J permanecem `AWAITING_HUMAN`; o merge local não equivale a aceite. A/B e OPS-003 permanecem registrados como aceitos/integrados.
5. Worktrees D e OPS-004 (que hospedava a branch de consolidação/J) removidas após confirmar estado limpo e ancestralidade integral; refs locais preservadas.
6. Não fazer push, deploy, migration, acessar banco/fila/provider/rede Git ou iniciar consumer/piloto.

## Execução e evidências

- FAC-012D integrado por merge `1bc5a2c51b362555f8e253d5168aaa7b7e5f4e29`.
- FAC-012E–J integrados a partir de `35507a78b6f2176a4b0c004ea6b51f86196a12a0`; conflitos foram reconciliados mantendo os limites de escrita do D e as rotas por função do G.
- Suíte, typecheck, lint, build e `git diff --check` passaram após regenerar artefatos locais de contratos, Prisma e runtime. Nenhuma migration ou serviço foi iniciado.
- Relatório operacional: `documentacoes/operacao/2026-10-03-OPS-004-consolidacao-worktrees.md`.
- A revisão humana deste exato resultado continua pendente; merges não alteram aceite de FAC-012C–J.
- A lista final de worktrees contém somente `/home/vinicius/le-fabrique`; remoções executadas sem `--force`.

## Baseline, caminhos e limites

`developer` em `a6fdf9ce57cc2293ddf310198204ea1093db9522`; D em `439e95429b81a4bd12e01c0d501d9241a18393ee`; J em `17fed5bd2d55eaa4dc4eb23d607cd98893f6c674`. Worktrees: `/home/vinicius/le-fabrique`, `/home/vinicius/le-fabrique-fac-012d`, `/home/vinicius/le-fabrique-fac-012f`. Execução local apenas; sem provider elegível ou gastos.

## Provider elegível

Nenhum; integração Git local e checks determinísticos apenas.
