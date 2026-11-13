// Package delivery gera a documentação viva da feature e publica a Draft PR.
//
// Gate de documentação: para cada módulo alterado no diff (pacote Go em internal/<x>/, área
// do painel em apps/web/src/<x>/ …) o diff precisa trazer a documentação do módulo atualizada
// (a regra diz qual arquivo); e a feature precisa ter o seu histórico (docs/features/<chave>.md),
// gerado pela fábrica a partir do que está registrado: requisito, plano, tarefas, commits por
// tarefa, checks por revisão, risco, revisão e E2E. Sem isso a entrega fica BLOQUEADA, com a
// lista do que falta.
//
// Draft PR idempotente: depois dos gates (documentação e os de checks, revisão e E2E da revisão
// entregue), a PR é aberta como RASCUNHO pelo ledger de efeitos (a chave é a branch da tentativa):
// repetir encontra a mesma PR, e se a resposta se perder a prova por busca decide, sem duplicar. O
// merge continua humano; a fábrica nunca o faz.
package delivery
