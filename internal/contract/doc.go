// Package contract é o contrato JSON de execução (LF-AJ-04): híbrido (dados determinísticos do
// backend + propostas técnicas do Tech Lead), validado pela Sentinela e imutável por execução.
//
// O Tech Lead só consegue propor os campos de Proposal: o decodificador recusa qualquer chave
// desconhecida, de modo que tentar elevar permissões, trocar IDs ou apontar para fora do worktree
// nunca chega a ser publicado. Cada publicação gera um snapshot com versão e hash; uma alteração
// válida cria outra versão encadeada pelo hash da anterior, preservando as antigas.
package contract
