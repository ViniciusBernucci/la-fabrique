// Package recovery recupera tentativas interrompidas (crash do worker, reboot da VPS).
//
// Uma tentativa interrompida é um item em execução cuja lease venceu. Antes de devolvê-lo
// à fila, a recuperação PROVA, item a item: o sandbox está quieto (nenhum escritor vivo), o
// último checkpoint está íntegro (hashes), não há efeito externo incerto da execução e o
// estado verificável (Git, artefatos, políticas) não aponta problema. Com tudo provado, o item
// volta à fila AUTOMATICAMENTE, com o checkpoint a usar. Qualquer incerteza abre uma decisão
// humana e bloqueia só o item afetado: a vaga do projeto só é devolvida com quietude provada e
// checkpoint íntegro (um único escritor por workspace); os demais itens elegíveis seguem.
//
// Os limites de tentativas são SEPARADOS por classe de falha (infraestrutura, IA, revisão).
// Esgotado o limite da classe, a recuperação deixa de ser automática e pede decisão. Uma
// inspeção que não consegue responder (ex.: o runtime de containers indisponível) adia o item
// sem consumir tentativa e sem alterá-lo.
package recovery
