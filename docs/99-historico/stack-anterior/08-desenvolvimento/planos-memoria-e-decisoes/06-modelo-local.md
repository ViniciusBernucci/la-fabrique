> Leitura: [← Anterior](05-decision-engine.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](07-jev.md)

# Plano futuro — Modelo local e recuperação semântica

## Escopo e objetivo

Preparar fronteiras na próxima versão; integrar modelo, vetores e inferência somente em implementação futura. Não baixar pesos nem instalar extensão agora. Objetivo: recuperar fontes semanticamente relacionadas sem depender de chamadas externas em cada busca. Ausência de EmbeddingProvider/modelo local/pgvector na implementação inspecionada permanece explícita.

## Contratos propostos

`EmbeddingProvider`: recebe trechos permitidos e versão/configuração; retorna vetores, modelo efetivo, dimensão, hash da entrada e métricas observadas. `ModelRegistry`: ID, origem, licença verificada, revisão dos pesos, hash do artefato, runtime, dimensão, tokenizer, limites e status de avaliação. `VectorGeneration`: revisão de fontes, modelo e versão de chunking. Chunking é divisão em trechos; registrar offsets/seções para recuperar fonte original.

`SemanticSearch`: consulta e revisão, geração compatível e limite de candidatos. Resultado contém fonte/hash, métrica e score específico, sem converter em probabilidade genérica. `ContextRanker` recebe candidatos permitidos e sinais; retorna ordenação e explicações reproduzíveis onde possível. Falha do modelo retorna erro normalizado e fallback determinístico, sem trocar a fonte da memória.

## Alternativas para avaliar

Sem vetores mantém simplicidade e mede lacunas do baseline. Modelo de embeddings local pode melhorar recuperação, mas ocupa recursos. JEV avalia candidatos ambíguos, não substitui necessariamente indexação semântica de todo acervo. Multilingual-e5-small é candidato da conversa; licença, versão, runtime Node, artefato ONNX, idiomas e qualidade precisam de verificação específica no futuro. Não fixar pesos ou versão pela transcrição.

pgvector é alternativa dentro do PostgreSQL. Começar experimento com busca exata permite baseline de qualidade antes de índice aproximado. A [fonte primária](https://github.com/pgvector/pgvector) descreve as duas modalidades; não constitui escolha aceita ou instalação local.

## Dependências e sequência futura

Fontes/rótulos, indexação determinística, Context Packs e avaliação devem existir antes. Definir corpus de avaliação independente; comparar semântica com textual/grafo; medir recursos no host de referência; executar em modo sombra; pedir avaliação do ganho antes de usar resultados na seleção real. Modo sombra calcula alternativas sem alterar ações.

Não sobrepor indexação pesada e checks sem orçamento de admissão medido. Limites CPU/RAM/lotes/concurrency pelo sistema, sem valores novos presumidos. Dados do host não foram medidos nesta entrega. Troca de modelo cria nova geração e reindexa vetores compatíveis antes de trocar ponteiro; nunca misturar dimensões ou modelos.

## Critérios de aceite, riscos e rollback

Modelo com origem/licença/hash verificados; índices segregados por revisão; português e inglês avaliados; ausência do modelo não remove fontes obrigatórias; busca híbrida melhora lacunas documentadas sem regressão crítica; CPU/RAM/tempo mensurados em carga concorrente representativa; desligamento restaura baseline e não perde fontes.

Riscos: saturação da VPS, resultados semanticamente próximos mas errados, licença/artefato incompatível e reindexação cara. Metas numéricas de qualidade e limites serão estabelecidos após baseline e aprovação, não inferidos. Retenção de gerações e custo de armazenamento precisam de definição operacional.

[Guia didático](../../02-arquitetura/memoria-e-decisoes/06-modelo-local.md).

## Como interpretar este plano

Escopo: evolução da própria La fabrique. Este documento é um plano arquitetural, sem implementação nesta entrega e sem tickets, estimativas ou sprints. As escolhas técnicas abaixo são **PROPOSTAS PARA AVALIAÇÃO**, exceto as decisões expressamente confirmadas no [registro de decisões](01-decisoes-e-dependencias.md). Exemplos de nomes, contratos e erros são propostas; não são APIs existentes. A futura implementação deverá converter os contratos em schemas Zod versionados em `packages/contracts`, sem duplicar a fonte canônica.

Stack preservada: React/TypeScript/Vite, NestJS, worker Node/TypeScript, PostgreSQL e Redis/BullMQ, VPS única. API controla e persiste; worker executa indexação, clientes e checks. Um executor inicial e um writer por workspace continuam obrigatórios. Autenticação de clientes oficiais permanece segregada. Somente JEV tem a exceção específica de API; não se habilita API de LLM por consequência.
