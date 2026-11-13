// Package gateways adapta gateways de API compatíveis com o protocolo de chat
// completions (OpenRouter e Vercel AI Gateway) ao mesmo laço de agente dos
// clientes CLI: o modelo pede ferramentas, e cada pedido passa por um
// Executor controlado (concessão, escopo e política do catálogo de ferramentas).
//
// Regras: a credencial vem do cofre (security.Secret) e nunca aparece em erro,
// log ou trilha; erros têm mensagem fixa (sem corpo remoto nem URL); a resposta
// do modelo é dado não confiável; o modelo só chama ferramentas que o agente
// tem, nunca outras.
package gateways
