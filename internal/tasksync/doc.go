// Package tasksync escreve na origem de tarefas (Jira ou ClickUp) sem duplicar
// (T06-03): subtarefas, dependências, links e mudanças de estado passam pelo
// ledger de efeitos (T13-04), com chave de negócio estável e prova de existência
// no destino. O estado do item só chega a "concluído" com o merge confirmado.
package tasksync
