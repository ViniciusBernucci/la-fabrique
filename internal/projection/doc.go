// Package projection projeta o status canônico da Issue na origem (Jira/ClickUp). Parte das
// intenções que a Sentinela grava na outbox junto de cada transição e usa o registro de efeitos
// existente: a escrita é uma única transição de status (nenhum outro campo é tocado, então o que o
// humano editou fica preservado), é revalidada por leitura antes de repetir (timeout após gravação
// remota é reconciliado, não reenviado) e, enquanto não confirmada, aparece como pendência.
package projection
