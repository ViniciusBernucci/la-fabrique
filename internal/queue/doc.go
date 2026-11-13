// Package queue é a fila local de tarefas técnicas por projeto.
//
// Um item só INICIA se estiver elegível no momento do início (reconferido sob o
// bloqueio do projeto): projeto ativo, pré-requisitos do plano concluídos (a
// evidência é a tarefa em "done"), requisito da feature sem replanejamento nem
// conflito pendentes, nenhum outro item ocupando a vaga do projeto e a porta de
// espera liberada. Há UMA vaga por projeto (garantida por índice único), então dois
// workers que disputam o mesmo projeto nunca iniciam duas tarefas.
//
// Ordem: correções de PR primeiro; depois prioridade (0 é a mais urgente) e a ordem
// de chegada. Entre projetos há paralelismo: a fila oferece no máximo um candidato
// por projeto.
//
// Esperas são representadas (merge, provedor, decisão, recursos) e a vaga só é
// devolvida quando há um checkpoint seguro da execução; sem ele, o item espera
// segurando a vaga. O retorno de uma espera exige a porta correspondente liberada.
package queue
