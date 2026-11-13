// Package acceptance monta a matriz de evidências do aceite (T17-01) a partir das
// fontes que já são a verdade do projeto: os casos P01–P18 de 05-ACEITE-E-PILOTO.md,
// os critérios de cada ticket em 03-BACKLOG.md e o estado e as entregas em
// 07-CHECKLIST.md. A matriz é DERIVADA: um teste confere que o arquivo versionado
// é igual ao gerado e que a rastreabilidade fecha (todo ticket do checklist está no
// backlog e na matriz; todo caso aponta para épicos existentes; toda referência da
// matriz de decisões aponta para um ticket que existe). Nada aqui inventa evidência:
// a coluna do projeto real fica "pendente" até alguém registrá-la.
package acceptance
