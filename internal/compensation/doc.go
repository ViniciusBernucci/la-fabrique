// Package compensation desfaz efeitos externos parciais conforme o risco:
//   - baixo risco e reversível: compensação idempotente, pelo ledger de efeitos,
//     com prova de que o efeito original deixou de existir;
//   - sensível: só com uma autorização consumida (T13-03) presa exatamente a
//     essa compensação;
//   - irreversível ou desconhecido: nada é desfeito, o efeito é preservado e a
//     recusa fica registrada.
//
// Toda decisão (aplicada, recusada, falha) é auditável. A falha conserva o
// trabalho (nada é apagado) e bloqueia novas operações agravantes da execução
// até a resolução do bloqueio.
package compensation
