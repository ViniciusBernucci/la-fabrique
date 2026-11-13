// Package contextpack monta o contexto seletivo de uma sessão de agente e
// renova a sessão sem depender da memória do fornecedor.
//
// O contexto é composto por contrato (requisitos, critérios, decisões, memória,
// arquivos, evidências e saídas estruturadas), em ordem de prioridade e dentro
// de um limite de tokens. Cada fonte entra no manifesto com hash do conteúdo
// efetivamente incluído, tamanho e motivo de omissão ou truncamento; segredos
// são removidos ANTES de calcular hash e tamanho, e o pacote é recusado se ainda
// restar um segredo reconhecível.
//
// Itens obrigatórios (requisitos e critérios marcados como tal) nunca são
// descartados em silêncio: se não couberem, a montagem falha.
//
// A renovação decide pela métrica oficial quando existe; sem ela usa uma
// estimativa auditável (bytes/3 com margem), sempre registrando o método. Uma
// nova sessão só nasce ligada a um checkpoint íntegro e a um pacote cujo hash
// confere, e a troca de modelo exige autorização explícita.
package contextpack
