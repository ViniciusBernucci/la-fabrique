// Package factory executa os nós de trabalho de um workflow (ADR-0069). O agente de IA é o CLI oficial
// (Claude Code ou Codex) rodando num terminal isolado (sandbox Docker) com a LLM escolhida, as skills do
// agente gravadas no workspace e um comando montado pela fábrica; as alterações voltam como patch para o
// repositório de trabalho da execução, onde a fábrica faz commit, push e abre a Draft PR. Scripts e checks
// rodam em sandbox sem rede. A fábrica nunca lê o token das contas de IA (ele fica no volume da conta).
package factory
