// Package selection escolhe o fluxo (e o subfluxo) de uma feature por regras
// ordenadas. Os atributos são normalizados (minúsculas, espaços colapsados,
// listas ordenadas e sem repetição) antes de comparar; a primeira regra que casa
// vence (as demais que casariam aparecem no rastro); sem regra, vale o padrão
// explícito do projeto; sem padrão, a seleção falha de forma explícita. Toda
// seleção grava um rastro com o motivo; a simulação explica cada regra sem gravar.
package selection
