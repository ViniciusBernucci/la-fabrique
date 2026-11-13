> Leitura: [← Anterior](01-decisoes-e-dependencias.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](03-indexacao-e-grafo.md)

# Plano — Memória persistente

## Objetivo, estado observado e alvo

Evitar perda de regras, decisões e evidências entre tarefas. Hoje Git, documentação, tickets, checkpoints e persistência de runs existem; não foi encontrado um catálogo de conhecimento com ciclo de aprovação integrado. Base inspecionada: `896ef0d2c804b80711d5ad00ad96695eaaf9f2d7`. A [implementação do contexto](../../../packages/runtime/src/context-builder.ts) registra hashes, sem persistir automaticamente o conteúdo.

Na próxima versão: catálogo consultável pelo painel, seis categorias confirmadas, vínculos a fontes Git, revisão humana de propostas, atualização incremental, divergências e sobreposição por branch. Futuro: refinamentos de recuperação conforme medições; conversas continuam excluídas até decisão diferente.

## Responsabilidades e dados propostos

Git mantém conteúdo versionado; PostgreSQL mantém catálogo, relações de proveniência, aceites e estado operacional. API recebe propostas e aceites; worker lê fontes e constrói índices. O painel apresenta origem, versão, evidências, conflitos e relação com main. Conhecimento não concede permissões de runtime.

Contrato lógico `KnowledgeRecord`: identificador estável, categoria, tipo de afirmação, fonte/revisão/hash/trecho, domínio responsável, status da proposta, aceite identificado, revisão de incorporação nullable, substituições e conflitos. Observação do código e regra normativa são naturezas distintas. Não substituir um texto aprovado in-place: nova versão aponta a anterior.

`AcceptKnowledge` recebe ID da proposta, hash esperado, revisão e decisão do responsável. Rejeita proposta que mudou após ser apresentada. Idempotência evita dois eventos de aceite por repetição da mesma operação. A publicação ocorre no aceite humano; a incorporação ao código é registrada separadamente. O responsável é o usuário, com apoio de IA, sem aprovação automática pela IA.

## Alternativas e recomendação para avaliação

Arquivos apenas simplificam armazenamento, mas deixam consultas e relações sem índice. Conteúdo oficial só no banco criaria fonte concorrente ao Git. Recomenda-se catálogo relacional reconstruível e fontes Git, com aceites auditáveis no banco; formatos exatos de metadados em arquivos precisam de revisão técnica antes da implementação.

## Dependências, sequência e recuperação

Primeiro definir identidades de fonte e revisão; depois indexar, consultar, propor, aceitar e invalidar caches. A integração futura com aceite de entrega deverá incluir atomicidade entre registro de aceite e evento de publicação, usando outbox existente quando aplicável. Se a indexação falhar, o aceite não desaparece: publicação fica registrada e processamento pendente é observável. Não apresentar índice incompleto como atualizado.

Migração aditiva; não importar todo histórico como vigente. Preparar inventário de fontes atuais, preservando status de ADRs e aceites FAC. Rollback desativa consultas novas e conserva fontes e auditoria, sem apagar regras aceitas. Conflito afeta apenas o escopo identificado; impacto ainda desconhecido pausa a tarefa após investigação inconclusiva.

## Critérios de aceite e riscos

- Consulta identifica origem e bytes de toda informação normativa apresentada.
- Aceite de proposta obsoleta é recusado; repetição de aceite não duplica publicação.
- Regra aceita antes de merge aparece com estado de incorporação explícito.
- Duas branches não compartilham alterações candidatas; exclusão/rename não deixa fonte vigente fantasma.
- Falha de índice permite reconstrução a partir da mesma revisão sem perder auditoria.
- Conversas e arquivos de credenciais não entram no catálogo.

Riscos: fonte desatualizada, correlação errada entre regra e revisão, aceite sem evidência e fonte Git removida. Retenção, localização de snapshots e política de exclusão serão configuradas/avaliadas antes da ativação, sem prazo numérico inventado.

[Explicação didática](../../02-arquitetura/memoria-e-decisoes/02-memoria-persistente.md).

## Como interpretar este plano

Escopo: evolução da própria La fabrique. Este documento é um plano arquitetural, sem implementação nesta entrega e sem tickets, estimativas ou sprints. As escolhas técnicas abaixo são **PROPOSTAS PARA AVALIAÇÃO**, exceto as decisões expressamente confirmadas no [registro de decisões](01-decisoes-e-dependencias.md). Exemplos de nomes, contratos e erros são propostas; não são APIs existentes. A futura implementação deverá converter os contratos em schemas Zod versionados em `packages/contracts`, sem duplicar a fonte canônica.

Stack preservada: React/TypeScript/Vite, NestJS, worker Node/TypeScript, PostgreSQL e Redis/BullMQ, VPS única. API controla e persiste; worker executa indexação, clientes e checks. Um executor inicial e um writer por workspace continuam obrigatórios. Autenticação de clientes oficiais permanece segregada. Somente JEV tem a exceção específica de API; não se habilita API de LLM por consequência.
