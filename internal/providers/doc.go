// Package providers adapta os clientes CLI de IA (Claude Code e Codex CLI) para
// rodar dentro do sandbox: monta o comando com o modo autorizado, isola cada
// conta em um diretório de configuração próprio, normaliza os eventos de saída
// e permite cancelar. Os adaptadores de gateways de API (OpenRouter, Vercel AI
// Gateway) chegam no T09-02.
//
// Regras: a autenticação é a oficial de cada CLI (o login é feito pelo
// responsável no diretório da conta); este pacote nunca lê, grava nem repassa
// tokens, e o prompt vai por stdin, não por argumento de processo.
package providers
