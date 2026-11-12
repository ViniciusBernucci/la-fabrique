# Estado e gates do lote

## Persistência

Pasta de planos configurada (Intranet: {FONTE_CLAUDE}/worktrees/AAAA-MM-DD/):

- lote-AAAA-MM-DD-{slug}.md: manifesto único por lote. ID, data fuso do projeto, escopo, tarefas, grafo, bases ref/SHA por repo, ordem, branches/candidatos aprovados e SHAs, status e pacote.
- estado-{chave}-{assunto}.md: estado por tarefa; se uma mesma tarefa existir em outro lote, use nome distinto e referência explícita. Não sobrescreva o estado anterior.
- revisao-humana-AAAA-MM-DD-{slug}.md: pacote final.

O planejador cria; durante execução só o coordenador escreve esses arquivos. Subagentes entregam eventos. Retomada confere o estado com Git/arquivos; não repete commit, merge, índice ou remoção já comprovados. Uma execução ativa mantém identificação da sessão proprietária do lote; não execute dois coordenadores escrevendo o mesmo estado. Uma sessão órfã só é retomada após verificar que não está executando.

## Perfis e fonte operacional

Aplique ../../_shared/perfis-de-risco.md. Gates condicionais são NA_SEM_ACHADOS ou COBERTO_NO_REVIEW com evidência; nunca PASS fictício. Registre perfil e justificativa. Planos antigos mantêm critico até migração explícita.

Com Contexto, SQLite é a fonte operacional e render gera projeções somente na pasta local do projeto. Eventos têm ID único, expected_revision e sessão. Manifesto/estado central pode apontar para essa projeção; não mantenha Markdown divergente como segunda fonte. Sem ferramenta, preserve o protocolo manual.

## Estado mínimo da tarefa

- Chave e ID/link externo opcional; lote e caminhos 00/01/02/99/especificação.
- Fase, próxima ação e responsável; última atualização com fuso.
- Repositório/camada, worktree, branch, base ref e SHA completo, HEAD atual, registro de entrega e commits de implementação/correção/documentos.
- Gates por camada: implementação, review (relatório + SHA), auditoria (relatório + SHA), correções/revalidação por ID, QA (matriz + SHA), integração, revisão/QA final e limpeza; estado, evidência e limitações de cada um.
- Decisões do auditor por relatório/ID: severidade, decisão/motivo, regra de negócio, evidência, commit corrigido e resultado independente.
- Rodadas de correção técnica usadas: 0/3; QA: 0/3. Contadores por tarefa, persistidos e compartilhados entre camadas e integração final; registre a rodada em andamento para não contar de novo na retomada.
- Dependências, bloqueios, branches candidatas/rejeitadas/aprovadas por repo, merge SHAs, arquivos em conflito e worktrees mantidas/removidas.
- Histórico de transições com data, fase anterior/nova e evidência. Mudança de base/HEAD lista quais evidências foram invalidadas.

## Transições

BLOQUEADA_NO_PLANEJAMENTO ou PRONTA_PARA_ORQUESTRAR → CRIACAO_WORKTREES → IMPLEMENTACAO → CODE_REVIEW → AUDITORIA_REVIEW → CORRECOES_REVIEW → REVALIDACAO_REVIEW → QA → CORRECOES_QA → REVALIDACAO_QA → PRONTA_PARA_INTEGRAR → INTEGRACAO → VALIDACAO_INTEGRADA → PRONTO_PARA_REVISAO_HUMANA.

Estado alternativo ISOLADA contém motivo, evidência, trabalho preservado e condição de retomada; não satisfaz entrega. Não há AGUARDANDO_* nem checkpoint humano na execução. Fases sem trabalho são “não aplicável” com motivo; gates essenciais bloqueados nunca são pulados.

| Gate | Evidência para avanço |
| --- | --- |
| Planejamento | critérios e contratos completos; manifesto e planos reais; nenhuma lacuna essencial |
| Worktrees | registro Git da branch/path/base corretos, sem alteração alheia apropriada |
| Implementação | commit próprio + registro + validações + índice idempotente |
| Review | cobertura essencial completa, relatório com base/HEAD e limitações |
| Auditoria | todo achado decidido; em critico sempre independente; nos demais NA_SEM_ACHADOS somente com review completo sem achados |
| Correções | IDs APLICAR commitados e verificados pelo revisor independente |
| QA | critérios essenciais PASS; simples pode COBERTO_NO_REVIEW com evidência independente completa |
| Pronta para integrar | todos os gates das camadas, relatórios commitados, dependências aprovadas |
| Integração | commits das duas camadas incluídos nas candidatas compatíveis e merges registrados |
| Validação integrada | verificações pós-merge, review/auditoria de interação e QA afetado aprovados |
| Revisão humana | commits finais, branches aprovadas registradas, limpeza e pacote final verificáveis |

Veredito original do review não é apagado pelo auditor. Gate consolidado pode liberar achado rejeitado/adiado fundamentadamente, mas nunca falha essencial ou risco bloqueante confirmado. Commit novo exige revalidação do código e dos critérios que pode afetar; documentação sem efeito executável não exige repetir testes sem motivo.

## Limite e isolamento

Até três rodadas de correção técnica e três de QA por tarefa; após a terceira sem solução, ISOLADA. Uma falha de permissão, identidade de branch, critério, arquitetura/negócio não definido ou prova de banco pode isolar imediatamente. Não repita três vezes uma ação arriscada apenas para consumir contador. Dependentes também ficam isoladas. Independentes continuam. Decisões necessárias vão ao pacote final.

Estados legados AGUARDANDO_WORKTREES_ORCA/AGUARDANDO_CORRECOES_REVIEW não são novas pausas: confira evidências, registre migração para CRIACAO_WORKTREES/AUDITORIA_REVIEW e continue pelo contrato atual. Não aceite aprovação histórica sem conferir escopo e hashes.

## Pacote de revisão

1. Resultado honesto do lote: PRONTO_PARA_REVISAO_HUMANA, PARCIAL_COM_BLOQUEIOS ou BLOQUEADO. Tabela de tarefas, camadas, dependências, estado, branches finais/revisão e commits.
2. Branches aprovadas por repositório/grupo, base SHA, HEAD, merges por tarefa. Comandos de leitura com SHAs concretos: git log --first-parent BASE..BRANCH, git diff BASE...BRANCH e git diff MERGE^1 MERGE. Não substitua a base fixada por origin/main móvel.
3. Roteiro de telas, sempre no checkout principal: branch `revisao/` já ativa nele (ou, se não foi possível trocar, motivo e comandos para trocar), branch anterior para voltar, reinstalação de dependências quando os lockfiles mudaram, ambiente, perfis/permissões, cenários e evidências. Roteiro de diff por tarefa/arquivo, gerado por review-guide quando disponível, senão pelo coordenador com localização verificada.
4. Aceite/QA observado, testes executados e não executados; decisões ADIAR/REJEITAR do auditor e motivos; falhas isoladas e trabalho preservado. Diferencie dívida residual de bloqueio.
5. Migrations pendentes, dependências entre PRs e ordem backend/frontend. Sugira um PR por grupo coeso/camada; dependentes podem exigir PR empilhado ou entrega conjunta. Branch de revisão aprovada é a candidata do conjunto testado; branches individuais continuam disponíveis para diff/PR quando os gates correspondentes cobrirem esse recorte.
6. Títulos/descrições de PR preparados, sem criar PR ou fazer push. Resumo ClickUp pronto, sem publicar automaticamente.
7. Inventário de limpeza: path, branch, removida/mantida e motivo. Inclua candidatas reprovadas preservadas e comandos apenas de leitura para inspecioná-las.

PR e publicação externa não são gates para concluir a execução local: são ações posteriores do usuário. Nunca marque tarefa ISOLADA como concluída por ter produzido pacote.
