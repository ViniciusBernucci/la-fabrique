// Package memory mantém a memória técnica do projeto (T15-05): registros pendentes por branch,
// com evidência e deduplicação, promovidos a oficiais só pelo merge confirmado e só quando valem
// para a revisão integrada. Tudo é escopado por projeto: um projeto nunca lê a memória de outro.
package memory
