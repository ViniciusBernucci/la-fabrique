> Leitura: [← Anterior](00-README.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](02-memoria-persistente.md)

# Decisões confirmadas, propostas e evidências

## Requisitos confirmados nesta sessão

| ID | Decisão do responsável | Consequência |
|---|---|---|
| C01 | Oito planos separados e documentação didática | Guia e plano para cada feature |
| C02 | Avaliar arquitetura juntos | Propostas técnicas não viram decisões aceitas automaticamente |
| C03 | Escopo inicial na própria La fabrique | Não planejar integração inicial de projetos externos |
| C04 | Próxima versão e evolução futura | Separar capacidades imediatas de modelos futuros |
| C05 | Manter seis categorias de memória | Institucional, negócio, relações, decisões, operacional e trabalho |
| C06 | Conversas fora da memória por enquanto | Transcrição é fonte de planejamento, não ingestão operacional |
| C07 | Git para fontes; PostgreSQL para índices/registros | Não duplicar conteúdo oficial editável no banco |
| C08 | Seu aceite publica conhecimento; main é referência | Aceita e incorporada são condições diferentes por revisão |
| C09 | Sobreposição por branch/worktree | Candidatos de uma branch não contaminam outra |
| C10 | Conflitos bloqueiam tarefas afetadas | Impacto desconhecido é investigado; se persistir dúvida, pausa |
| C11 | Todas as seis capacidades de memória/contexto na próxima versão | Busca, grafo, packs, expansão, atualização/conflitos e painel |
| C12 | Modelo local e treinamento para execução futura | Preparar contratos sem instalar ou treinar agora |
| C13 | Exceção de API específica para JEV | Documentar integração futura e limites pelo sistema |
| C14 | JEV indisponível/inconclusivo → LLM elegível → humano | Não habilitar fallback pago de LLM |
| C15 | Tech Lead escolhe agentes/modelos por tarefa; override no controle | Configuração com autoria, versão e elegibilidade |
| C16 | Índices automáticos; relações de código observadas; regras aguardam aceite | Separar extração de aprovação normativa |
| C17 | VPS 2 vCPUs e 8 GB RAM; escala vertical possível | Benchmark antes de fixar limites ou contratar recursos |
| C18 | Limites/qualidade medidos; matriz completa preservada | Sem metas percentuais ou prazos inventados |
| C19 | Responsável valida com apoio de IA | IA não se torna aprovadora normativa |
| C20 | Arquiteto entrega Markdown; Tech Lead cria backlog depois | Esta entrega não implementa nem cria sprints |

## Prioridade proposta sob autorização para ordenar

1. Preservar fontes, versões, aceites e limites de autoridade: sustenta todos os passos seguintes.
2. Recuperar obrigatórios conhecidos e estabelecer baseline de busca.
3. Indexar símbolos, contratos e relações para analisar impacto.
4. Construir/expandir Context Packs e preparar continuidade entre sessões/provedores.
5. Integrar decisões auditadas, atualização incremental, conflitos e painel. Essas capacidades são construídas ao longo das etapas, não adiadas para depois da próxima versão.
6. Avaliar qualidade e recursos; só então evoluir seleção com JEV, modelo local e classificador, em implementações separadas.

Workflow e rastreabilidade acompanham todas as etapas. Esta sequência é recomendação técnica, não cronograma nem autorização de execução.

## Evidência de estado atual

Base inspecionada `896ef0d2c804b80711d5ad00ad96695eaaf9f2d7`, branch `feat-arquitetura-contexto`, workspace inicialmente limpo. [ContextBuilder](../../../packages/runtime/src/context-builder.ts), [schemas](../../../packages/contracts/src/index.ts), [DeveloperWorkflow](../../../apps/worker/src/developer-workflow.ts), [Prisma](../../../apps/api/prisma/schema.prisma) e [perfil](../../../apps/worker/src/execution-profile.ts) lidos. Busca local não encontrou grafo, EmbeddingProvider, adapter JEV ou pipeline de treino; ausência não é teste operacional.

main local observada em `15c2198076a6a6afcbf2f15212ae1eaf53822b9f`, bootstrap. main permanece referência escolhida; antes de implementar conferir seu conteúdo e alinhamento com o código inspecionado, sem merge automático. Não foi consultado remoto.

Perfil atual permite 6 GiB de memória e CPU quota 400%; esses são tetos, não consumo medido. Na VPS declarada de 2 vCPUs/8 GB, admissão/limites precisam de benchmark e ajuste autorizado, reservando recursos para banco e controle. Não aplicar esses tetos como orçamento dos novos componentes.

## Fonte e carregamento explícito

Transcrição: `/home/vinicius/le-fabrique/transcricao-integral-memoria-contexto-persistente.md`, 2247 linhas, SHA-256 `9badc77b393c45137bc403c5254fe5bacc4385b8e2e29b8c3030793f1ac2579b`. Fonte externa não copiada para memória ou histórico do repositório. Matriz preserva os itens tabulares, com origem; o documento não promete reproduzir imagens/conteúdo visual ausente.

AGENTS lido explicitamente, hash `7ca4664cba5de95bb0d19d0b8131c078f3d917486b143afda60edc841a0665ca`. Política IA lida integralmente no início, hash anterior à adição da exceção `cf459b7d679a9796427d40b734dc0f2cabf5405de94c7ab05912574e1c3e6971`. Política documental lida integralmente, hash `10d419ece5476593c37a092525d18f8c358f95f582eb5faf6f42540f1b1c022e`. Perfil local e skill adaptada lidos. Autoload NÃO VERIFICADO; não usar leitura manual como prova de descoberta automática.

## Pendências de avaliação, sem requisitos inventados

- Avaliar formatos de catálogo/relações, schemas e API propostos antes de gerar tickets.
- Aprovar semântica de override durante execução, persistência/reconstrução de packs, retenção e armazenamento histórico.
- Preencher pelo sistema orçamento/moeda/janelas/chamadas JEV e critérios avaliados por pergunta, antes da primeira chamada paga.
- Verificar serviço/conta/modelo/tarifa/contrato de idempotência e tratamento de envio desconhecido na implementação JEV.
- Medir CPU/RAM/tempo/qualidade; escolher runtime/modelo/artefatos locais só na evolução futura.
- Concluir revisão independente e aceite documental da revisão exata; checks estruturais não os substituem.

Pendências são explicitadas, não completadas por inferência. Os planos são revisáveis mesmo quando esses parâmetros ainda dependem de configuração ou experimento.
