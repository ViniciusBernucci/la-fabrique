// Package budget impõe orçamento e registra uso. Valores em micro-USD (1e-6).
//
// Gasto em API (custo) e cota de assinatura (unidades observadas) são registrados
// separadamente e NUNCA se somam: a assinatura não tem custo por chamada e a cota
// só entra com fonte e horário observados. O teto de API é imposto por reserva
// atômica antes da chamada: concorrentes disputam o mesmo orçamento serializados
// pelo bloqueio da linha dele, então reservas ativas + gasto nunca passam do teto.
// Sem orçamento global ativo, o gasto em API é recusado (falha fechada); modelo
// sem preço também não pode ser orçado.
package budget
