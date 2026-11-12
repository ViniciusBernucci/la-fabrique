> Leitura: [← Anterior](07-jev.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](09-fluxo-completo.md)

# Aprendizado: transformar resultados em exemplos confiáveis

## O que a fábrica poderá aprender

A memória guarda conhecimento oficial. Um classificador aprende padrões para localizar e priorizar conhecimento. Ele não substitui as regras. Por exemplo, pode aprender que tickets de recuperação costumam precisar de documentação de concorrência.

Um **exemplo de treinamento** reúne entrada, características e rótulo. **Características** são sinais como proximidade no grafo e correspondência textual. **Rótulo** indica a resposta validada, como “este documento era necessário”. Um **dataset** é um conjunto versionado desses exemplos.

Ler um arquivo não prova que ele era necessário. Concluir uma tarefa não prova que todas as decisões estavam corretas. É preciso registrar a evidência e validar o rótulo, com você e apoio de IA.

## Exemplo didático

O pacote inicial omitiu uma regra de concorrência. Durante revisão, um achado demonstra que a regra era necessária. Registrar tarefa, revisão, candidatos, seleção inicial, regra recuperada posteriormente e evidência do achado produz um exemplo útil. A opinião do Developer sozinha não vira rótulo confiável.

## Treinar, avaliar e promover

Treino ajusta o modelo. Validação auxilia escolhas durante o experimento. Avaliação final verifica generalização com dados separados. Se exemplos quase iguais aparecem nos dois conjuntos, o modelo pode parecer melhor por memorizar casos: isso é vazamento entre treino e avaliação.

Modo sombra deixa o novo modelo ordenar candidatos, sem alterar pacotes reais. Só após comparação e avaliação do responsável pode haver proposta de promoção. **Rollback** volta à estratégia anterior quando a nova versão piora resultados.

## Métricas explicadas

**Recall** mede quantas fontes necessárias conhecidas foram recuperadas. **Precisão** mede quantas fontes recuperadas eram pertinentes. Cobertura de regras críticas declaradas é um gate independente: não deve cair para obter menos texto. Tempo, recursos e retrabalho completam a avaliação.

Não há número mínimo de exemplos confirmado nem economia prometida. A coleta operacional será preparada na próxima versão; treinamento e promoção ficam para evolução futura. Regressão logística e árvores são alternativas da conversa, não modelo escolhido.

Leia o [plano](../../08-desenvolvimento/planos-memoria-e-decisoes/08-aprendizado.md).
