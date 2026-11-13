> Leitura: [Catálogo de decisões](00-README.md) · [Índice didático](../02-INDEX.md) · [Próxima decisão →](ADR-008-excecao-jev.md)

# ADR-007 — Memória e decisões versionadas

Data: 2026-10-09, Europe/Berlin. Status: **PROPOSTA TÉCNICA PARA AVALIAÇÃO**; requisitos explicitamente confirmados estão no registro abaixo. Implementação: PLANEJADA. Não substitui ADR-002/003 aceitos ou aceita propostas multi-tenant posteriores.

## Contexto

Context Builder atual lê fontes selecionadas e registra hashes; memória estruturada, grafo e decisões auditáveis precisam de planejamento próprio. A transcrição usa versões/tickets históricos; o código base `896ef0d2c804b80711d5ad00ad96695eaaf9f2d7` e backlog atual delimitam estado observado.

## Requisitos confirmados

Memória independente das sessões; seis categorias; fontes Git e índices/registros PostgreSQL; conversas excluídas por enquanto; main como referência; aceite humano publica conhecimento antes de merge com incorporação separada; sobreposição por branch; atualização automática de índices e observações; regras novas aguardam aceite. Conflitos afetam tarefas relacionadas, impacto incerto é investigado e pausa quando dúvida persiste. Escopo inicial na própria La fabrique.

## Estrutura recomendada para avaliar

Bibliotecas/serviços lógicos dentro de API e worker existentes: catálogo de conhecimento, índice estrutural, relações em PostgreSQL, pacotes versionados e motor de políticas/estratégias. Modelo local e aprendizado futuro são substituíveis e não fontes oficiais. Tech Lead gerencia agentes/modelos por tarefa e responsável pode alterar no controle, sob elegibilidade. Fallback semântico confirmado: JEV quando configurado → LLM elegível → humano.

## Alternativas e consequências

Memória só em sessão perde portabilidade. Conteúdo oficial editável apenas no banco cria fonte concorrente. Novo banco de grafos/serviços separados amplia operação antes de medição. A proposta conserva stack/topologia e adiciona índices reconstruíveis, com custo de invalidação, proveniência e avaliação. Sem garantia de recall de regras ainda não catalogadas.

## Estado e próximos passos

[Requisitos/evidências/pendências](../08-desenvolvimento/planos-memoria-e-decisoes/01-decisoes-e-dependencias.md), [planos](../08-desenvolvimento/planos-memoria-e-decisoes/00-README.md), [guia](../02-arquitetura/memoria-e-decisoes/00-README.md) e [contratos](../05-contratos/memoria-e-decisoes/00-contratos-propostos.md). Avaliar detalhes técnicos antes de tickets; este ADR não autoriza implementação, merge ou deploy.
