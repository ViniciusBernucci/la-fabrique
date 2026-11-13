// Package planning define o contrato do plano de decomposição de uma feature e
// o torna verificável, versionado e publicável na origem sem duplicar efeito.
//
// Contrato: o agente planejador devolve tarefas com chave estável, título,
// critérios VERIFICÁVEIS, risco inicial, dependências (DAG) e uma estimativa de
// ordem. O validador recusa: chave inválida ou repetida, plano vazio ou grande
// demais, tarefa sem critério, critério que não possa ser verificado (sem
// comando, teste, arquivo ou evidência observável), dependência ausente, para
// si mesma ou em ciclo (o ciclo é mostrado).
//
// Versões: um plano aceito vira a versão vigente; o mesmo conteúdo não gera
// versão nova; plano inválido ou conflitante é registrado como rejeitado e a
// versão anterior continua valendo. Conflito é alterar ou remover uma tarefa
// que já saiu de "pendente": a identidade (feature, chave) atravessa as versões
// e preserva estado e item remoto.
//
// Origem: cada tarefa vira UMA subtarefa e cada dependência UMA relação, pelo
// ledger de efeitos (T13-04), então repetir a publicação, mesmo após timeout,
// não duplica nada.
package planning
