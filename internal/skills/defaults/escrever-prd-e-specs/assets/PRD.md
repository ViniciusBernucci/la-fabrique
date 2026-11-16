# PRD — <demanda>

- **Referência da demanda:** <ticket, solicitação ou referência documental>
- **Versão e estado:** <versão> · <RASCUNHO ou PRONTO_PARA_PLANEJAMENTO>
- **Especificação técnica:** [Especificação](ESPECIFICACAO.md)
- **Fontes principais:** <entrevista, documentos aprovados e revisões consultadas>

## Problema e resultado esperado

<Descreva quem enfrenta o problema, o fluxo atual com evidência, por que mudar, o resultado desejado e como o resultado será observado. Para métricas, indique valor inicial, alvo e medição quando definidos. Registre alvos essenciais sem definição como pendência.>

## Atores, escopo e limites

| Ator ou sistema | Necessidade | Permissões e limites |
|---|---|---|
| <ator> | <necessidade> | <ação, recurso e restrição> |

- **Dentro do escopo:** <fluxos, superfícies e entregas incluídas>
- **Fora do escopo:** <limites que evitam interpretações divergentes>
- **Restrições e dependências de produto:** <políticas, compatibilidade e dependências externas>

## Vocabulário e regras de negócio

<Defina os termos que afetam interpretação e registre as regras aplicáveis com IDs estáveis, como RN-01. Para cada regra, indique gatilho, pré-condição, resultado, exceções, dados legados e fonte conforme aplicável.>

## Fluxos e estados

<Descreva o fluxo principal e alternativas relevantes, com ator, gatilho, pré-condição, estado anterior e posterior, efeitos e comportamento na falha. Quando houver uma máquina de estados, liste transições permitidas e o resultado de uma tentativa inválida. Inclua repetição, concorrência, abandono ou recuperação quando mudarem o comportamento.>

## Requisitos

| ID | Tipo | Comportamento ou restrição verificável | Prioridade ou obrigatoriedade acordada | Fonte |
|---|---|---|---|---|
| RF-01 | Funcional | <comportamento esperado> | <classificação> | <decisão ou evidência> |
| RNF-01 | Não funcional | <limite, condição e como medir> | <classificação> | <decisão ou evidência> |

<Considere segurança, privacidade, acessibilidade, desempenho, disponibilidade, observabilidade e compatibilidade conforme o impacto. Use valores sustentados por fonte ou decisão.>

## Critérios de aceite

| ID | Requisitos | Dado | Quando | Então |
|---|---|---|---|---|
| CA-01 | RF-01 | <pré-condição observável> | <ação ou gatilho> | <resultado verificável> |

<Inclua cenários negativos e casos de borda que distingam resultados aceitáveis. A especificação aponta os testes e evidências de cada critério.>

## Decisões, hipóteses e questões abertas

| ID | Estado | Decisão ou questão | Origem | Impacto ou responsável pela resolução |
|---|---|---|---|---|
| DEC-01 | <confirmada, hipótese ou aberta> | <registro> | <fonte> | <impacto> |

<Registre a confirmação do entendimento e as pendências essenciais ainda abertas.>
