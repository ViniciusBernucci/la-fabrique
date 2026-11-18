---
name: escrever-prd-e-specs
description: Entrevista requisitos com o protocolo grill-me e produz PRD, especificação técnica e tarefas rastreáveis. Use para transformar uma ideia, funcionalidade ou mudança de produto em requisitos, contratos, critérios de aceite e testes.
---

# Escrever PRD e especificação técnica

Produza, em português, um PRD e uma especificação técnica em Markdown. Use a abordagem SDD para ligar a intenção do produto aos contratos, às tarefas e aos testes que a comprovam. Adapte a profundidade dos documentos ao impacto da mudança e às convenções do projeto.

## Investigue a demanda

Leia a demanda, respostas já dadas, documentação, ADRs, contratos, testes e código pertinentes. Em cada repositório autorizado, confira a raiz, a revisão analisada e os pontos reais de entrada. Identifique consumidores, componentes, caminhos, modelos de dados e verificações já existentes. Registre a origem das informações relevantes. Se faltar acesso, indique o que ficou sem comprovação e avance com as fontes disponíveis.

Distinga comportamento atual, mudança aprovada e proposta técnica. Marque caminhos confirmados, caminhos propostos e caminhos ainda desconhecidos. Em projeto novo, proponha a topologia, os repositórios e a estrutura de pastas com justificativa. Em projeto existente, parta da topologia encontrada e explicite o delta.

## Faça a entrevista com grill-me

Use o [protocolo grill-me incorporado](references/entrevista-grill-me.md) na própria sessão. A entrevista fecha decisões que mudam comportamento, escopo, contratos, segurança ou critérios de sucesso. Verifique fatos no material disponível antes de perguntar. Trate divergências entre intenção e código como decisões explícitas. Preserve as respostas já confirmadas.

Quando a entrevista chegar a um entendimento verificável, apresente o resumo para confirmação. A confirmação anterior pode ser reutilizada se cobrir o escopo atual. Enquanto uma decisão essencial estiver pendente, mantenha as partes independentes em rascunho e aponte o que falta para fechar o documento.

## Especifique e decomponha

1. **PRD.** Registre problema, atores, objetivos, regras de negócio, escopo, requisitos funcionais e não funcionais e critérios observáveis. Use o [modelo de PRD](assets/PRD.md). Fixe o significado dos termos e descreva fluxos principais, alternativos e de erro. Cada requisito tem fonte e ID estável.
2. **Especificação técnica.** Derive os detalhes a partir do PRD e do repositório. Use o [modelo de especificação](assets/ESPECIFICACAO.md) para arquitetura, repositórios e pastas, APIs, eventos, dados, integrações, segurança, testes, implantação e tarefas. Aprofunde cada superfície tocada. Para um tópico sem impacto, registre a razão quando ela ajudar a entender o escopo.
3. **Tarefas técnicas.** Descreva unidades pequenas com resultado verificável. Para cada uma, indique área, repositório, componente, caminhos, contratos, dependências por etapa, risco, critérios e testes. Separe as responsabilidades conforme a arquitetura do projeto, como interface, serviços, dados ou infraestrutura, quando houver fronteiras claras. Declare quais recursos compartilhados exigem ordem ou coordenação. Identifique tarefas que já podem ser especificadas e as que dependem de uma decisão.
4. **Consistência.** Cruze PRD, especificação, contratos, tarefas e testes com a [conferência de cobertura](references/conferencia-e-handoff.md). Quando encontrar lacuna material, investigue ou retome somente a pergunta afetada. Ao revisar documentos existentes, mantenha IDs estáveis, registre o delta e indique quais testes, tarefas ou consumidores precisam ser revistos.

## Entregue

Siga os nomes e caminhos adotados no projeto. Sem convenção, entregue `PRD.md` e `ESPECIFICACAO.md` na pasta da demanda, com links relativos. Informe a revisão das fontes, o estado dos documentos e as pendências. Inclua tarefas detalhadas o suficiente para que a equipe possa planejar e executar o trabalho, preservando os vínculos com requisitos e testes.

O escopo desta skill é a documentação. Se o usuário solicitar vinculação ou publicação, siga as convenções do projeto e use o destino autorizado.
