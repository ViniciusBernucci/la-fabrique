> Leitura: [← Anterior](02-memoria-persistente.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](04-context-packs.md)

# Plano — Indexação, grafo e análise de impacto

## Objetivo e escopo temporal

Na próxima versão: busca determinística em código/documentação da La fabrique, indexação TypeScript, relações de imports/símbolos, vínculos explícitos de regras/contratos/testes e consulta de impacto pelo painel. Não foi encontrada implementação de grafo ou indexador estrutural na base inspecionada. Extração de outras linguagens fica fora do escopo inicial confirmado.

## Contratos e ownership propostos

`IndexRequest`: escopo autorizado, revisão Git exata, sobreposição opcional, versão do extrator e chave de idempotência. `IndexGeneration`: estado pendente/em processamento/completo/incompleto/falhou, arquivos analisados, fontes ignoradas e lacunas. `KnowledgeRelation`: origem/destino, tipo, proveniência, revisão, hash, natureza observada/proposta/aceita. `ImpactQuery`: intenção e componentes iniciais, revisão, limites de expansão. `ImpactResult`: nós afetados, caminhos justificativos, obrigatórios conhecidos e lacunas.

Worker executa extração limitada; API agenda e consulta; banco guarda gerações e relações. Fontes e relações são sempre filtradas pela revisão solicitada, inclusive nas consultas de cache. Publicação de geração usa comparação da revisão alvo para impedir que trabalho atrasado sobreponha índice mais novo.

## Alternativas e decisão técnica ainda proposta

Comparar TypeScript Compiler API diretamente com um wrapper como ts-morph durante implementação: API direta evita dependência adicional, wrapper pode reduzir código de manutenção. Não instalar agora. PostgreSQL para relações evita um banco novo; buscas com ripgrep e índices textuais são candidatos complementares. Extração de eventos e filas exige vínculos de contrato explícitos; import não prova toda dependência de negócio.

## Sequência, migração e recuperação

Depende do contrato de fontes da memória. Construir extração completa como baseline; só então implementar atualização por diff e comparar seus resultados. Ciclos no grafo usam conjunto de visitados. Delete/rename remove relações derivadas na nova geração, preservando geração antiga para consultas históricas. Regras aceitas não são apagadas pela remoção de um arquivo: registrar divergência.

Reprocessamento usa revisão/hash/extrator para identificar a mesma operação; jobs duplicados não duplicam relações. Banco recebe geração candidata e troca o ponteiro apenas quando consistente. Em falha, manter geração anterior identificada como antiga e bloquear uso que dependa de atualização completa. Rollback desativa extrator novo, conserva gerações e permite reconstrução com versão anterior.

## Critérios de aceite

- Imports e referências de fixtures TypeScript conhecidas aparecem com proveniência.
- Atualização incremental produz as mesmas relações que extração completa para inclusão, remoção e rename.
- Ciclos e limites não travam consultas; limites deixam lacunas explícitas.
- Eventos sem imports diretos são cobertos por vínculos declarados e testados.
- Índice de branch A não responde consulta de B; geração atrasada não substitui a atual.
- Evidências distinguem relação observada de dependência normativa aceita.

## Riscos e capacidade

Dependências dinâmicas incompletas, custo de análise, invalidar demais ou de menos. Medir extração inicial/incremental e consulta com código da La fabrique e conjunto sintético rotulado. VPS 2 vCPUs/8 GB: processar jobs respeitando admissão e um executor inicial; orçamento de CPU/RAM depende de benchmark. Escala vertical é possibilidade, sem contratação nesta entrega.

[Guia didático](../../02-arquitetura/memoria-e-decisoes/03-indexacao-e-grafo.md).

## Como interpretar este plano

Escopo: evolução da própria La fabrique. Este documento é um plano arquitetural, sem implementação nesta entrega e sem tickets, estimativas ou sprints. As escolhas técnicas abaixo são **PROPOSTAS PARA AVALIAÇÃO**, exceto as decisões expressamente confirmadas no [registro de decisões](01-decisoes-e-dependencias.md). Exemplos de nomes, contratos e erros são propostas; não são APIs existentes. A futura implementação deverá converter os contratos em schemas Zod versionados em `packages/contracts`, sem duplicar a fonte canônica.

Stack preservada: React/TypeScript/Vite, NestJS, worker Node/TypeScript, PostgreSQL e Redis/BullMQ, VPS única. API controla e persiste; worker executa indexação, clientes e checks. Um executor inicial e um writer por workspace continuam obrigatórios. Autenticação de clientes oficiais permanece segregada. Somente JEV tem a exceção específica de API; não se habilita API de LLM por consequência.
