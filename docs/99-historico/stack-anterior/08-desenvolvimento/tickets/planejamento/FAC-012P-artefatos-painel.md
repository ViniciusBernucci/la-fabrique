# FAC-012P — Diff e artefatos no painel

Status: AWAITING_HUMAN. Código `739cff001ad7c2ae53ca6719e2cbf92a11522683`; 289 testes/checks passaram. Data: 2026-10-03. Baseline: `2d5a9a3`. Branch/worktree: `feat/fac-012p-artifacts`, `/home/vinicius/le-fabrique-fac-012p`.

## Objetivo e escopo

Transportar bundle íntegro do último snapshot normal para leitura/download pelo painel. Patch binário Git e bytes untracked em base64, nomes/hashes/modos relativos, sem paths host. Primeiro limite explícito: JSON até 64 KiB para tickets pequenos; oversized não pode declarar entrega completa. Não baixar arquivos do host pela API, nem executar/restaurar bundle no browser.

Caminhos: contracts; worker artifact-reader/processor/control-client; API artifact service/controller/schema/migration; painel e testes; documentação controle/operação/runtime/infra e lessons. Migration apenas versionada, não aplicada. Sem banco/fila/serviço/provider/login/piloto/push/deploy. Provider nenhum, fixtures; um writer. Contas/modelos continuam pela interface. Baseline 266 testes; até duas rodadas de correção.

## Critérios

1. DTO estrito/limitado, caminhos relativos seguros, base64 canônico, hashes/tamanhos/manifesto íntegros, nenhuma leitura por caminho fornecido pelo painel.
2. Leitor confiável resolve somente UUID sob snapshot root privado, rejeita symlink/escape, verifica patch/untracked/manifesto contra último snapshot do relatório. Padrões conhecidos de segredo/arquivos sensíveis bloqueiam exportação sem redigir silenciosamente o patch; não é garantia de detecção universal.
3. Worker envia resultado, bundle e checkpoint/conclusão nessa ordem. Falha de exportação/persistência impede conclusão e mantém journal/evidência local. Recovery de journal reaplica mesma sequência sem IA.
4. Persistência fenced/idempotente/imutável, relatório correspondente obrigatório; GET administrativo confere run/attempt e devolve bundle ou null. Sem liberar writer.
5. UI carrega sob demanda com AbortSignal, troca de seleção descarta resposta antiga, renderiza diff escapado e oferece download JSON somente após ação. Não interpreta HTML nem aplica patch.
6. Checks/docs completos por SHA; sem DONE antes de aceite humano.

## Rollback/limites

Revert com writer parado preservando evidências. Campos aditivos, migration não aplicada; preservar dados se implantada futuramente. Artefatos maiores, retenção/backup e interrupções ficam pendentes. Relatório em `documentacoes/controle/2026-10-03-FAC-012P-artefatos-painel.md`.
