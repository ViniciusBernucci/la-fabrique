> Leitura: [← Anterior](03-indexacao-e-grafo.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](05-decision-engine.md)

# Plano — Context Packs e continuidade

## Estado observado e próxima versão

[ContextBuilder](../../../packages/runtime/src/context-builder.ts) e [DeveloperWorkflow](../../../apps/worker/src/developer-workflow.ts) já montam contexto e o reconstruem em retomada/handoff. Recebem fontes explícitas; não há recuperação híbrida nem gateway de expansão comprovado. Perfil confiável limita 100 arquivos, 256 KiB/arquivo e 2 MiB totais; prompt tem limite próprio no workflow. Esses números são código observado, não dimensionamento aprovado para a nova seleção.

Próxima versão inclui regras obrigatórias, recuperação determinística/grafo, pacote por papel, expansão controlada, manifesto versionado, invalidação e consulta pelo painel. Embeddings ficam para o plano futuro.

## Contratos propostos

`ContextPackRequest`: tarefa/versão, papel, revisão base e código, memória aceita aplicável, sobreposição, fontes obrigatórias e limites configurados. `ContextPackManifest`: ID/hash, fontes/trechos/hashes, motivo de inclusão, natureza normativa/observada, conflitos, lacunas, omissões, bytes e uso de tokens nullable quando não observado. `ExpandContext`: pack/hash esperado, pergunta ou referências solicitadas, revisão e limite restante. Retorna nova versão imutável e vínculo à anterior.

Contratos atuais continuam canônicos em [Zod](../../../packages/contracts/src/index.ts). Evolução usa schema v2 ou contrato separado com negociação explícita de consumidor; não inserir campos obrigatórios no v1 e assumir compatibilidade.

## Seleção, cache e segurança

Separar obrigatórios conhecidos antes do ranking. JEV/LLM só avaliam candidatos opcionais válidos. Validação final reconfere hashes, revisão, restrições e cobertura declarada. Biblioteca de seleção é executada pelo worker; acesso externo dos agentes exige gateway limitado, sem DB/Redis geral. Arquivo de pacote permite integração inicial; expansão exige contrato seguro comprovado para cada cliente. Sem ponte compatível, pausar expansão ou preparar nova sessão/pacote sob controle do workflow.

Cache proposto chaveado por tarefa/versão/papel/revisão/sobreposição/fontes/política/estratégia. Nenhuma reutilização por semelhança de texto sozinha. Política pode dispensar cache inicialmente se sua invalidação aumentar risco.

## Dependências, migração e alternativas

Depende de memória e índice. Comparar arquivos completos com trechos por símbolo: completos simplificam proveniência, trechos economizam conteúdo, mas podem perder contexto. Recomenda-se selecionar trecho com fonte inteira recuperável, sem resumos como única evidência normativa. Persistência do conteúdo sanitizado versus reconstrução por revisão será avaliada por retenção, disponibilidade da fonte e espaço; ambas devem reproduzir os bytes entregues.

Handoff reutiliza mecanismos de parada/checkpoint/snapshot existentes e novos manifestos, sem exportar sessão privada. Mudança de provider não muda contrato de negócio. Rollback conserva manifesto v1 e fluxo atual para tarefas compatíveis; não continuar quando obrigação nova não puder ser atendida pelo fallback.

## Critérios de aceite e riscos

- Fonte obrigatória ausente, secreta, conflitante ou acima do limite impede pacote declarado completo.
- Expansão permanece na revisão correta e audita cada fonte adicional.
- Alteração de fonte, aceite, tarefa ou papel invalida cache pertinente.
- Handoff preserva patch/untracked e regras sem autorizar writer concorrente.
- Revisão recebe contexto específico e não perde critérios obrigatórios.
- Avaliação mede fontes críticas recuperadas, irrelevantes enviados e retrabalho; não exige economia percentual inventada.

Riscos: resumo impreciso, lacuna desconhecida, cache obsoleto e leitura extra do cliente não observável. Métricas distinguem contexto entregue e uso total desconhecido.

[Guia didático](../../02-arquitetura/memoria-e-decisoes/04-context-packs.md).

## Como interpretar este plano

Escopo: evolução da própria La fabrique. Este documento é um plano arquitetural, sem implementação nesta entrega e sem tickets, estimativas ou sprints. As escolhas técnicas abaixo são **PROPOSTAS PARA AVALIAÇÃO**, exceto as decisões expressamente confirmadas no [registro de decisões](01-decisoes-e-dependencias.md). Exemplos de nomes, contratos e erros são propostas; não são APIs existentes. A futura implementação deverá converter os contratos em schemas Zod versionados em `packages/contracts`, sem duplicar a fonte canônica.

Stack preservada: React/TypeScript/Vite, NestJS, worker Node/TypeScript, PostgreSQL e Redis/BullMQ, VPS única. API controla e persiste; worker executa indexação, clientes e checks. Um executor inicial e um writer por workspace continuam obrigatórios. Autenticação de clientes oficiais permanece segregada. Somente JEV tem a exceção específica de API; não se habilita API de LLM por consequência.
