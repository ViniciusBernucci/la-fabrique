// Package projectres trata o perfil técnico e os recursos de um projeto (T05-03).
//
// Descoberta: inspeciona, de forma segura, manifestos que o chamador fornece
// (package.json, go.mod, Makefile, Compose, pyproject…) e SUGERE comandos,
// serviços e nomes de segredos, sempre com a origem do valor. Nada é executado,
// nada é aplicado sozinho e arquivos de segredo (.env) nunca são lidos: só os
// NOMES de .env.example. Os valores sugeridos viram perfil apenas por escolha
// explícita.
//
// Perfil: comandos (com origem: manifesto ou manual), serviços e referências de
// segredo no formato connection:<id> (nunca o valor; literal de segredo é recusado).
//
// Recursos: associação EXPLÍCITA de workflows, agentes, skills, ferramentas e
// modelos globais. A sobrescrita local (modelo, conta, tempo limite de um
// agente) vive na associação e nunca altera o recurso global.
//
// Validação: acrescenta ao relatório de ativação do projeto os itens de
// workflow ainda em rascunho (ou que deixou de ser publicável), conta/modelo
// incompatível, conexão revogada e referência de segredo inválida.
package projectres
