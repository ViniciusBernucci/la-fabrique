> Leitura: [← Anterior](05-decision-engine.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](07-jev.md)

# Modelo local: encontrar semelhanças sem gerar código

## O que significa local

Um modelo local executa inferência na infraestrutura da fábrica. Isso consome CPU, RAM e disco; não é custo zero. **Inferência** é usar um modelo já treinado. **Treinamento** ajusta seus parâmetros a partir de exemplos. A primeira integração futura não precisa treinar nada.

Um **embedding** representa um trecho de texto como uma lista de números, chamada vetor. A **similaridade** compara vetores para encontrar textos próximos em significado. Exemplo didático: “limite de uso” pode se aproximar de “cota do provedor” mesmo sem a mesma palavra. Proximidade não comprova que o documento seja obrigatório nem verdadeiro.

## Como funcionaria

1. Dividir fontes em trechos, mantendo caminho, seção, revisão e hash.
2. O modelo transforma cada trecho em vetor.
3. Guardar vetores com a versão do modelo e da divisão de textos.
4. Transformar a consulta em vetor compatível.
5. Buscar candidatos semelhantes e combinar com busca textual e grafo.
6. Revalidar obrigatórios e enviar somente os candidatos pertinentes.

**Busca híbrida** combina sinais textuais, estruturais e semânticos. **Ranking** ordena candidatos. A seleção final continua submetida às regras determinísticas.

## Por que versionar

Mudar o modelo pode mudar o significado dos vetores. Misturar vetores incompatíveis pode produzir ranking sem sentido. Uma nova versão exige índice separado e avaliação antes de substituir a anterior. Se falhar, o motor determinístico continua disponível; obrigação ausente continua bloqueando.

## O que está decidido e o que está aberto

O responsável deixou modelo local para execução futura. A conversa menciona multilingual-e5-small, ONNX e quantização; são candidatos históricos, não escolha técnica aceita nem prova de que cabem nesta VPS. **Quantização** reduz a precisão numérica para tentar diminuir recursos; pode afetar qualidade.

A VPS tem 2 vCPUs e 8 GB segundo o responsável. Precisamos medir uso junto de banco, worker e checks. Escalar verticalmente significa aumentar recursos da mesma VPS; não resolve automaticamente gargalos de algoritmo.

PostgreSQL pode receber uma extensão de vetores futuramente. A [documentação do pgvector](https://github.com/pgvector/pgvector) confirma buscas exatas e aproximadas. A instalação, versão e uso não foram autorizados por este plano.

Leia o [plano futuro](../../08-desenvolvimento/planos-memoria-e-decisoes/06-modelo-local.md).
