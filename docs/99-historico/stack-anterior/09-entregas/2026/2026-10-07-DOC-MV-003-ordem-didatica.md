# DOC-MV-003 — Manual em ordem didática

2026-10-07 Europe/Berlin. IMPLEMENTADO documentalmente / AWAITING_HUMAN; baseline developer 9439cde.

## Objetivo e funcionamento

143 capítulos e introduções receberam prefixos numéricos conforme dependência de entendimento. O roteiro conduz do problema/vocabulário à estrutura, responsabilidades, comportamento, interfaces, decisões e prática. Introduções oferecem listas numeradas; capítulos oferecem anterior/próximo. O índice agrupa manual e apêndices de consulta separadamente. Regra comum exige manter ordem/navegação a cada nova documentação.

Guias AGENTS/CLAUDE/ANTIGRAVITY e prompts foram atualizados. As duas pontes de política preservam referências da regra .agents somente leitura e exigem abrir a fonte numerada; autoload continua não verificado. Tickets/ADRs mantêm IDs; relatos mantêm datas; snapshots/logs/evidências anteriores não foram renomeados nem reescritos.

## Origem e revisão

[Mapa de ordem, realocação e hashes](../../00-governanca/ORDEM-LEITURA.json) registra 143 originais em docs/99-historico/ordem-9439cde, íntegros antes da transformação editorial. Mapas anteriores atualizaram apenas destinos correntes; source/archive/hash permanecem. Os manifestos de revisão DOC-MV-001/002 continuam referentes às suas revisões anteriores.

O painel teve somente paths de exemplo atualizados; configuração salva e comportamento do gate preservados. [Checks reais](evidencias/DOC-MV-003/checks.json) e [resultado documental](evidencias/DOC-MV-003/validacao.json) identificam o que foi verificado; [diff editorial](evidencias/DOC-MV-003/editorial.patch) é revisável.

## Limites e rollback

Revisão semântica própria da trilha; aceite humano/independente pendente. Checks estruturais não comprovam didática por si só. Sem nova sessão dos agentes, render Mermaid, produção, DB ou provider. .git/.agents somente leitura: sem commit/merge nesta etapa. Nenhum ticket de software promovido a DONE. Aplicada a lesson existente de documentação/proveniência, sem duplicar lesson.

Rollback: consultar mapa e restaurar nomes e bytes a partir de ordem-9439cde; reverter links/mapas/guia/validador/exemplo do painel pelo diff e baseline, preservando trabalho posterior. Não restaurar DB/auth/serviço.
