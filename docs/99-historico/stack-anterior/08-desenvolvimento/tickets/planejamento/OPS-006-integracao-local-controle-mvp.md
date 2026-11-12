# OPS-006 — Integração local e controle do MVP

Status: AWAITING_HUMAN. Data: 2026-10-03.
Baseline: `ce93b20440c3f7ecc216ae6513396596cdf9a76e` (`developer`).
Branch/worktree: `ops/ops-006-final-local-integration`, `/home/vinicius/le-fabrique-ops-006`.

## Objetivo e autorização

Consolidar os deltas locais OPS-005/FAC-012K/FAC-012L na developer e remover worktrees limpos após provar integração, conforme pedido prévio do responsável para reunir todo o trabalho local. Criar controle atual do MVP com evidências e lacunas explícitas. Merge local não implica aceite de cada ticket nem implantação.

## Escopo

Merges locais das revisões `7df2765` (OPS-005) e `a1d904b` (FAC-012L, que inclui FAC-012K), resolução dos conflitos preservando os objetivos dos dois incrementos, env example do worker host alinhado e correção de documentação atual divergente. Sem banco/fila/serviços ativos, instalação systemd, autenticação, piloto, migrations, push ou deploy. Provider elegível: nenhum; testes com fakes/fixtures.

## Critérios verificáveis

1. Ambas as revisões e FAC-012K são ancestrais de developer; nenhuma mudança única é descartada.
2. Compose conserva controle/bancos e binds locais; o host worker segue com execução desligada por padrão e nenhuma fixture consome tickets reais.
3. O example host incorpora os novos parâmetros; user manager/DBus usam o UID real da identidade provisionada, sem presumir que um specifier do system manager identifica o User do serviço.
4. Checks combinados: geração local Prisma, typecheck, lint, testes, build, Compose config, verificação sintática systemd e diff check, sem iniciar serviços.
5. Controle do MVP diferencia implementado, integrado, aceito e operacionalmente verificado; FAC-012A/B permanecem aceitos, piloto/deploy manuais, lacunas de execução explicitadas.
6. Somente worktrees limpos e comprovadamente integrados são removidos; branch refs e commits permanecem recuperáveis. Developer final limpo e sem push.

## Riscos e orçamento

Conflitos em main/env/docs são esperados. Um writer, até duas rodadas de correção; falha anterior de bootstrap deve ser diferenciada de regressão. Não instalar/iniciar/reiniciar serviços nem ler credenciais. Preservar histórico documental e checkpoints/artefatos externos.

## Entregáveis

Relatório `documentacoes/operacao/2026-10-03-OPS-006-integracao-local-controle-mvp.md`, `CONTROLE-MVP.md`, README/índice/backlog/changelog e lessons pertinentes. DONE somente após aceite humano da revisão exata; integração local é autorizada separadamente.

Implementação/integracão: `26e412d`; 227 testes e checks combinados passaram. OPS-005/K/L comprovadamente integrados em developer; worktrees respectivos removidos, refs preservadas. Ver [relatório](../../../09-entregas/2026/operacao/2026-10-03-OPS-006-integracao-local-controle-mvp.md).
