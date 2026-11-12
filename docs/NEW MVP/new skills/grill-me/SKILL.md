---
name: grill-me
description: Entrevista requisitos de uma demanda ou lote até esclarecer decisões de produto e critérios de aceite. Use para questionar um plano antes da especificação técnica; não implementa nem define arquitetura.
---

# Grill me

Questione o plano até que as decisões importantes estejam claras. Leia a demanda, o código e os
documentos disponíveis **antes** de entrevistar.

## Como perguntar

- Em português. **Até 4 decisões independentes por rodada.** Use a ferramenta de perguntas do
  cliente quando existir; senão, uma lista numerada. Decisão que depende de outra espera a próxima
  rodada.
- Cada pergunta traz opções concretas, a consequência de cada uma e a sua recomendação, quando
  houver. Escreva a pergunta de modo que "sim" aceite a recomendação.
- Temas: resultado esperado, quem usa, situação atual, restrições, regra de negócio, erros,
  permissões, dados legados, limites e critérios de sucesso.
- **Não pergunte o que dá para verificar** no repositório ou na documentação: verifique. Não repita
  pergunta já respondida.
- Confronte premissas com evidência. Se a ideia parecer ruim ou irrealista, diga.
- **Confronte o que o usuário diz com o código.** Se divergirem, mostre: "o código cancela a PN
  inteira, mas você disse que o cancelamento pode ser parcial; qual vale?". Termo vago ou com dois
  sentidos ("conta", "aprovação") vira pergunta, com um cenário de borda concreto.

## O que registrar

- **Decisões**, cada uma com a origem: usuário, evidência (arquivo/trecho) ou hipótese.
- **Hipóteses** ainda não confirmadas.
- **Questões abertas.** Sem resposta essencial, a questão continua aberta: não declare o requisito
  resolvido.
- **Exemplos Dado/Quando/Então** dos comportamentos combinados.
- **Fora do escopo.**

Não implemente nem crie arquitetura durante a entrevista.

A entrevista termina quando não restam decisões a perguntar **e** o usuário confirma o resumo do
entendimento. Sem essa confirmação, não siga para a próxima etapa.

## Saída

- **Chamada pelo `planejador-de-lote-v1`:** devolva o registro ao planejador, que segue para o
  tech-lead. Não chame outras etapas.
- **Chamada direta:** entregue no chat o registro acima e as perguntas restantes. Não encadeie
  outras skills nem comece a implementação.
