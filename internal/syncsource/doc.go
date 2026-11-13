// Package syncsource lê a origem (Jira/ClickUp) SOMENTE quando alguém pede:
// sincronização manual ou preflight de uma nova execução. Não há leitura
// periódica de mudanças: com a fila ociosa nada consulta a plataforma.
//
// Cada leitura normaliza os itens (identidade remota, versão, conteúdo, estado,
// prioridade e elegibilidade), compara com o último estado gravado e registra
// o que mudou, de onde e quando. Uma mudança é detectada UMA vez: depois de
// registrada, o novo estado vira a referência. A importação é deduplicada pela
// identidade remota (um item, uma linha).
//
// Edição humana local (prioridade) nunca é sobrescrita em silêncio: uma mudança
// remota no mesmo campo abre um conflito, o valor local continua valendo e o
// preflight não libera o item até a decisão. Uma leitura que falha (limite,
// rede, permissão) fica registrada e não altera nenhum item (nada é dado como
// removido por uma leitura incompleta).
package syncsource
