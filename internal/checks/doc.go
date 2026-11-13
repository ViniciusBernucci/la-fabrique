// Package checks executa os checks de um projeto (build, testes, lint…) no
// baseline viável e na revisão candidata, guarda a evidência e classifica cada falha.
//
// Classificação de uma falha na candidata:
//   - regressão: o mesmo check PASSOU no baseline, ou falhou lá com outra assinatura
//     (falha nova). Bloqueia;
//   - falha anterior comprovada: o mesmo check falhou no baseline com a MESMA
//     assinatura (código de saída e linhas de erro normalizadas). É registrada em
//     known_failures, com a execução de baseline que a comprova, e não bloqueia
//     automaticamente;
//   - origem desconhecida: não há baseline viável (não executado, inviável ou com
//     erro de infraestrutura). Fica em investigação e BLOQUEIA; nunca vira baseline
//     por suposição. Uma resolução explícita (com justificativa) define o destino.
//
// A evidência vale só para a revisão testada: o veredito de outra revisão não herda
// nada (uma alteração depois do check exige rodar de novo). Saídas passam pelo
// Redactor antes de ser gravadas e são truncadas.
package checks
