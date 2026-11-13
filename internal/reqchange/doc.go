// Package reqchange trata a mudança do requisito de uma feature e a troca da
// plataforma de origem.
//
// Cada revisão do requisito é gravada (o histórico nunca é apagado) e o impacto
// é classificado:
//   - metadado (prioridade, rótulos, estado): compatível, atualizado em marco
//     seguro (adiado enquanto uma tarefa está em execução) sem tocar o plano;
//   - conteúdo (título, descrição, dependências) sem trabalho iniciado: o plano
//     precisa ser refeito (replanejar);
//   - conteúdo com trabalho já iniciado: conflito. Nada é alterado (plano e
//     requisito vigente ficam como estão) e uma decisão é aberta, com a
//     diferença como evidência.
//
// Troca de origem: o responsável informa o mapeamento das identidades antigas
// para as novas. A validação exige origem diferente, nova origem acessível, toda
// feature mapeada, identidades novas existentes, sem repetição nem colisão. Só
// então a troca pode ser ativada, com o projeto suspenso: a configuração muda,
// as features passam à nova identidade e a antiga fica no histórico. Há sempre
// uma única origem por projeto.
package reqchange
