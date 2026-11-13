// Package techplan valida a proposta do Tech Lead e materializa o plano em subtarefas técnicas
// que vivem SOMENTE na fábrica (AJ05-02). Cada subtarefa é FRONTEND ou BACKEND; tarefa mista é
// recusada, a menos que seja marcada como indivisível (com a responsabilidade predominante
// declarada). Arquivos de contrato de API, schema, dependências e configuração viram recursos
// compartilhados reservados com exclusividade na admissão. Nada é criado no Jira/ClickUp.
package techplan
