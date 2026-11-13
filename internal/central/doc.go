// Package central consolida a Central de decisões e a auditoria operacional (T16-02): lista as
// autorizações com validade, uso e revogação e monta a sequência do incidente e da retomada de uma
// execução a partir do que o servidor registrou. Só leitura: aprovar, recusar e revogar acontecem
// nos pacotes de domínio (decisions, authorizations) e deixam evento próprio.
package central
