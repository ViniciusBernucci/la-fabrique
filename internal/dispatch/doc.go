// Package dispatch entrega a cada execução de agente (CLI ou API) o contrato imutável no cabeçalho
// da tarefa (AJ04-03). O despacho fixa a versão recebida: reentregar a mesma execução devolve a
// mesma versão; uma revisão dos insumos publica OUTRA versão para o despacho seguinte, sem reescrever
// o histórico. As evidências ligam versão do contrato, execução e resultado.
package dispatch
