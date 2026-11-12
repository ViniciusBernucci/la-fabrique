// Package availability mantém a saúde das contas de IA e decide o fallback.
//
// Saúde: combina o sinal OFICIAL (cota observada pelo adaptador, erro de
// autenticação ou financeiro informado pelo provedor) com o HISTÓRICO de
// falhas (indisponibilidade repetida, com backoff exponencial). Os quatro
// motivos são distintos: autenticação, cota, indisponibilidade e bloqueio
// financeiro, e cada um tem a sua regra de retorno.
//
// Fallback: percorre SOMENTE a cadeia explícita do agente, em ordem, e só
// escolhe alternativa saudável, compatível (models.Check) e dentro do orçamento
// (reserva de teste liberada em seguida). Não há rodízio: sem falha, a escolha é
// sempre a primeira alternativa saudável da cadeia, então o principal volta a ser
// usado assim que se recupera, mas nada move uma tarefa que já está andando.
//
// Preservação antes de troca: trocar de conta exige checkpoint íntegro da
// execução e sandbox em quiescência. Sem alternativa, a tarefa espera com novo
// horário (backoff); a vaga só é liberada quando há checkpoint seguro, e as
// demais tarefas seguem normalmente.
package availability
