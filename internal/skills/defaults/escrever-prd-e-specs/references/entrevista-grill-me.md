# Entrevista grill-me incorporada

Protocolo `grill-me` incorporado à skill para rodar na mesma sessão. A entrevista esclarece o que o produto deve fazer e sob quais condições. A especificação posterior traduz as decisões para o sistema e seus contratos.

## Conduza a rodada

1. Antes de entrevistar, leia a demanda, decisões anteriores e fontes acessíveis. Inspecione código, documentação e testes para separar fatos verificáveis de escolhas que dependem do usuário. Registre a diferença entre uma regra vigente e um comportamento observado que pode ser um defeito.
2. Comece pela decisão que destrava as demais. Faça até quatro perguntas independentes por rodada, com uma quando houver dependência entre respostas. Use a ferramenta de perguntas disponível ou uma lista numerada. Quando houver alternativas reais, exponha a consequência concreta de cada uma e recomende uma com motivo.
3. Confronte termos ambíguos e divergências com um exemplo concreto. Se “cancelar” pode significar cancelar um item ou todo o pedido e o código cancela o pedido inteiro, explique o comportamento observado e pergunte qual regra vale para a demanda. Registre a resposta como regra e formule um cenário de aceite.
4. Explore conforme o impacto: quem pode agir, sobre qual recurso e em que estado, o que muda quando a ação se repete ou concorre, o que acontece com dados antigos, quem observa a falha e como ela é recuperada. Se faltar um requisito não funcional material, procure o limite aprovado antes de propor um valor. Questões de implementação reversíveis podem ficar como proposta documentada.
5. Ao receber respostas, atualize as decisões e o vocabulário. Preserve as respostas que seguem válidas e evite repetir uma pergunta já resolvida. Se uma resposta mudar decisão anterior, identifique quais regras, contratos e critérios precisam ser revistos.

## Registre o entendimento

Para cada decisão material, registre identificador, pergunta ou conflito, escolha, motivo quando relevante e origem. A origem pode ser uma resposta do usuário, documento aprovado ou evidência com arquivo e revisão. Para hipótese, indique o que a tornaria inválida e o impacto. Para questão aberta, indique quem ou o que pode resolvê-la e quais artefatos ou tarefas ela bloqueia.

Use exemplos Dado/Quando/Então para confirmar as regras que aceitam mais de uma interpretação. Cubra também limites e fora do escopo. Quando todas as decisões essenciais estiverem cobertas, devolva um resumo curto do comportamento acordado para confirmação. Se o usuário já confirmou esse mesmo entendimento, use o registro anterior. Uma recomendação, silêncio ou hipótese não encerram uma decisão essencial.

Se a entrevista for interrompida, entregue decisões, rascunhos e lacunas com a última evidência conhecida. Ao retomar, releia a revisão anterior e prossiga pelas lacunas ainda materiais.
