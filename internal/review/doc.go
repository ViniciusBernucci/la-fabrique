// Package review conduz a revisão independente de uma tarefa e as correções.
//
// Independência: a primeira rodada é feita por um revisor DIFERENTE de quem implementou (o
// banco também garante). O revisor e a cadeia de fallback são configurados por projeto; o
// primeiro disponível e independente é usado. Nas rodadas seguintes o revisor pode corrigir.
//
// Achados têm severidade: CRÍTICO e RELEVANTE bloqueiam; BAIXO fica registrado e não bloqueia.
// A correção só é aceita com os checks da nova revisão aprovados (T14-01) e abre a rodada
// seguinte (revalidação) sobre a nova revisão. O limite padrão é de DUAS rodadas (configurável):
// esgotado com achados bloqueantes, o item da fila é suspenso com checkpoint seguro e uma decisão
// humana é aberta; nada segue sozinho.
//
// A aprovação vale só para a revisão revisada: uma alteração depois dela exige nova revisão.
package review
