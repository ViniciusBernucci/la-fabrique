// Package runner conduz a execução de um workflow publicado do começo ao fim (ADR-0069): cria a execução
// presa à versão publicada, guarda o estado do motor a cada passo, registra os eventos que o painel mostra,
// reconhece decisões humanas e esperas duráveis e deixa o worker retomar de onde parou. Quem faz o trabalho
// de cada nó (agente em CLI, scripts, checks, git) é um workflows.Executor injetado.
package runner
