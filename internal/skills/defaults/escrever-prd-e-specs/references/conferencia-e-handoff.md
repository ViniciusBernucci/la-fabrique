# Conferência e handoff

## Confira a cobertura

Faça a revisão cruzada na revisão corrente dos documentos.

| Verificação | Condição de passagem |
|---|---|
| Requisitos | Cada requisito obrigatório tem origem e critério observável. Critérios negativos e casos de borda cobrem as regras que mudam resultado. Limites não funcionais essenciais têm condição de medição ou pendência explícita. |
| Contratos | Cada contrato novo ou alterado tem dono, consumidores, origem, schemas, comportamento de sucesso e falha e tratamento de compatibilidade conforme aplicável. Código, PRD e spec divergem somente quando o delta está explícito. |
| Repositórios e caminhos | Cada tarefa identifica o repositório, ponto de entrada e área de mudança e teste. Um caminho proposto está marcado como tal. Ausência de acesso ou evidência aparece nas limitações. |
| Tarefas e dependências | Cada tarefa aponta para requisitos, critérios e testes, e descreve o que entrega. Dependências têm referência e etapa, o grafo é acíclico e recursos compartilhados incompatíveis têm ordem ou restrição. |
| Testes | Cada critério obrigatório tem teste ou procedimento de verificação com resultado esperado, ambiente e evidência prevista. Resultados reais só constam se observados e vinculados à revisão. |
| Rastreabilidade | Da regra até o teste e do teste de volta à regra. Investigue critérios sem tarefas, tarefas sem origem, contratos sem consumidores e testes que não demonstram o resultado. |
| Lacunas | Decisões essenciais estão confirmadas. Hipóteses reversíveis têm escopo e impacto. Divergências não resolvidas e marcadores de modelo pendentes não aparecem como decisões aprovadas. |

Faça uma leitura como implementador e outra como revisor. Para um requisito de maior impacto, simule a jornada pelos artefatos, incluindo a falha relevante. Verifique se duas pessoas poderiam chegar a comportamentos diferentes seguindo os mesmos documentos. Se puderem, explicite a decisão ou registre a lacuna antes de fechar o pacote.

Ao atualizar uma especificação, registre o que mudou e por quê. Mantenha IDs de trabalho equivalente. Se uma regra mudar ou for retirada, atualize referências e indique o impacto no plano, nos contratos e na evidência de teste anterior. Preserve as versões publicadas conforme a convenção do projeto.

## Encaminhe o pacote

- **RASCUNHO:** use enquanto faltar confirmação da intenção ou existir lacuna que permita implementações materialmente diferentes, altere um contrato obrigatório ou impeça verificações essenciais. Registre o que está pronto, o que falta, quem resolve e quais tarefas dependem disso.
- **PRONTO_PARA_PLANEJAMENTO:** use quando o entendimento do produto estiver confirmado e a conferência documental passar. Um detalhe técnico pode ficar para o planejamento se houver limites e critérios suficientes para resolvê-lo. Exemplos são o nome de um arquivo novo e a escolha entre duas estratégias internas equivalentes.

Esses rótulos descrevem o estado dos documentos. Use os equivalentes do projeto quando houver convenção. Na entrega, inclua título e referência da demanda, versão do pacote, links ou conteúdo acessível do PRD e da especificação, resumo, critérios obrigatórios e decisões pendentes. Se houver publicação autorizada, confirme que o público da entrega consegue acessar os documentos.

A equipe responsável pelo planejamento recebe as tarefas propostas e os critérios com os mesmos IDs. As etapas de revisão e aprovação seguem as convenções do projeto.

O método aproveita a sequência de especificação, esclarecimento, plano, tarefas e análise de consistência de SDD. [GitHub Spec Kit](https://github.github.com/spec-kit/quickstart)
