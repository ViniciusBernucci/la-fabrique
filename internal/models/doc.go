// Package models mantém o catálogo de modelos cadastrados manualmente e decide
// se um modelo pode ser usado por um agente/nó: provedor, conta, gateway e
// capacidades (ferramentas) precisam ser compatíveis. Modelo não cadastrado,
// desativado ou incompatível nunca é usado; o motivo é sempre explicado.
package models
