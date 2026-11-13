// Package risk classifica o risco de uma tarefa em baixo, médio ou alto e escolhe as
// verificações previstas para cada nível.
//
// O nível final combina a estimativa da IA com PISOS DETERMINÍSTICOS: arquivos críticos
// (migrations, autenticação, segredos, CI, containers…) e o tamanho da mudança. Os
// pisos só elevam: uma sugestão baixa da IA não rebaixa um arquivo crítico, e uma alta
// não é reduzida pelos pisos. O planejamento usa os caminhos previstos da tarefa; depois
// da implementação (e de cada correção) o diff REAL é reavaliado. Toda classificação
// guarda as razões do nível final (quem exigiu o quê), para auditoria.
//
// Os níveis escolhem verificações: baixo roda o conjunto básico; médio acrescenta
// revisão; alto acrescenta checks de segurança aplicáveis, revisão independente e E2E.
// Check de segurança indisponível no projeto é listado como não aplicável (com a
// justificativa pedida a quem aprova), nunca omitido em silêncio.
package risk
