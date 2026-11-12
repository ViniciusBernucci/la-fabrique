// Package policy classifica ações por risco e decide se seguem sozinhas
// (baixo risco), exigem aprovação humana (sensível) ou são recusadas
// (proibidas), para scripts, CLI, API, MCP e conectores (T13-01).
//
// Regras de projeto: a decisão vem da política e dos parâmetros reais da ação,
// nunca de texto de agente, comentário externo nem anotação de ferramenta; o
// proibido vence qualquer permissão; o que não está na lista é negado (falha
// fechada); e a trilha nunca guarda valores de segredo.
package policy
