// Package issuesync mantém as Issues da fábrica alinhadas à origem oficial (LF-AJ-01).
//
// O sync geral roda pela manhã ou sob demanda: importa Issues novas em PRD-DOC ou CORRECAO_PR e
// atualiza TODAS as já conhecidas, em qualquer estado, por identidade externa estável (upsert por
// projeto + origem + ID remoto). Issues desconhecidas em outros estados são ignoradas. Uma leitura
// que falha no meio preserva o progresso das páginas já gravadas e registra o erro por escopo.
package issuesync
