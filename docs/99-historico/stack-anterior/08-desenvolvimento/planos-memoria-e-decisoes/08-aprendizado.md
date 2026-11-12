> Leitura: [← Anterior](07-jev.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](09-fluxo-completo.md)

# Plano futuro — Dataset e classificador próprio

## Objetivo e divisão temporal

Próxima versão prepara registros de decisões, fontes iniciais/adicionais, resultados e candidatos a exemplos. Rotulagem, treino, avaliação e implantação do classificador são evolução futura separada. Não há LearningPipeline/dataset/classificador implementado identificado. Não alterar memória oficial por aprendizado automático.

## Contratos e responsabilidades propostos

`TrainingExampleCandidate`: tarefa/revisão/pack/decisão, candidatos, sinais/versionamento, seleção inicial, fonte adicional, resultado e evidências. `ValidatedLabel`: relevância declarada, responsável, apoio de IA identificado, revisão e motivo; rótulo disputado não entra no conjunto confiável. `DatasetManifest`: fontes permitidas, exclusões, hashes, versão de extração/sinais, divisão treino/validação/teste e proveniência de cada exemplo. `ModelEvaluation`: dataset separado, baseline, métricas por cenário, custo e revisão/configuração do modelo.

Worker registra candidatos sanitizados; API/painel apresentam validação; responsável aceita rótulos com auxílio de IA. Histórico registra correção de rótulo sem apagar a anterior. Dados operacionais permanecem separados de dados confiáveis para treino.

## Alternativas e sequência futura

Primeiro usar regras/pesos explícitos como baseline. Depois comparar classificador leve tradicional com alternativas aprendidas, evitando treinar LLM do zero. Linguagem e ferramentas de treino são escolha futura: laboratório separado pode exportar modelo para worker TypeScript; não criar backend Python como decisão implícita.

Coletar também falhas, fontes omitidas e casos negativos. Separar por tarefa/tempo/grupos relacionados para evitar cópias entre treino e avaliação. Avaliação deve representar conflitos, impacto indireto, retomada e alteração de revisão. Sinais de versão diferente não podem misturar-se silenciosamente.

## Critérios e promoção

Rótulos com evidências e autoria; conjunto congelado por hash; testes de vazamento; reexecução reproduzível com versão de ferramentas; comparação por cenário; nenhum obrigatório pode ser removido; modo sombra antes de ação; promoção específica pelo responsável, com possibilidade de rollback.

Sem metas numéricas ainda: primeiro medir baseline e reunir amostra representativa. A decisão de suficiência da amostra depende de diversidade e erros, não de um número arbitrário de tickets. Recursos de treino não cabem automaticamente na VPS; medir ou definir ambiente de experimento antes.

## Riscos, retenção e rollback

Viés de seleção, rótulos incorretos, dados sensíveis, drift e influência circular das próprias sugestões. Drift é mudança dos casos reais em relação aos usados para treinar. Separar métricas por versão e acompanhar fontes necessárias omitidas. Não transferir dados de projeto externo para treino; escopo atual é La fabrique.

Retenção, permissões de export e localização de artefatos precisam de definição anterior ao primeiro treino; sem prazo presumido. Rollback seleciona ranker anterior e mantém dataset/modelo histórico, removendo uso executivo sem apagar auditoria. Atualização periódica automática não entra sem decisão futura.

[Guia didático](../../02-arquitetura/memoria-e-decisoes/08-aprendizado.md).

## Como interpretar este plano

Escopo: evolução da própria La fabrique. Este documento é um plano arquitetural, sem implementação nesta entrega e sem tickets, estimativas ou sprints. As escolhas técnicas abaixo são **PROPOSTAS PARA AVALIAÇÃO**, exceto as decisões expressamente confirmadas no [registro de decisões](01-decisoes-e-dependencias.md). Exemplos de nomes, contratos e erros são propostas; não são APIs existentes. A futura implementação deverá converter os contratos em schemas Zod versionados em `packages/contracts`, sem duplicar a fonte canônica.

Stack preservada: React/TypeScript/Vite, NestJS, worker Node/TypeScript, PostgreSQL e Redis/BullMQ, VPS única. API controla e persiste; worker executa indexação, clientes e checks. Um executor inicial e um writer por workspace continuam obrigatórios. Autenticação de clientes oficiais permanece segregada. Somente JEV tem a exceção específica de API; não se habilita API de LLM por consequência.
