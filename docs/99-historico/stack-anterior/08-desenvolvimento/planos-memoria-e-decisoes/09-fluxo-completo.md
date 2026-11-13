> Leitura: [← Anterior](08-aprendizado.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](10-matriz-fluxo.md)

# Plano — Fluxo completo e matriz de decisões

## Objetivo e estado observado

Preservar todo o fluxo discutido com pontos de controle, responsabilidade e tratamento de exceções. O workflow atual já coordena Developer/checks/Reviewer, documentação e handoff; Prisma possui Ticket/Run/Attempt/Checkpoint. A conversa é proposta anterior e não prova que os 145 passos estejam implementados.

Na próxima versão: mapear pontos atuais, integrar memória/grafo/contexto/decisão e consulta/validação no painel. Preparar JEV/modelo local/aprendizado para implementação futura; seu mapeamento não exige habilitá-los. Features além do escopo confirmado da memória/decisão permanecem sujeitas ao planejamento e aceite próprio.

## Contratos e fluxo propostos

`WorkflowDecisionPoint`: ID da matriz, etapa, entradas, obrigatórios, estratégias permitidas, resultado esperado, evidências e tratamento de falhas. `WorkflowObservation`: tarefa/run/attempt/revisão, evento, ator, versão e evidência. Não tratar texto livre da LLM como transição persistida. Eventos duplicados/atrasados precisam de idempotência e revisão esperada.

Estados físicos continuam conforme [contrato atual](../../05-contratos/schemas/01-workflow.md). A máquina completa futura deve especificar transições por entidade, pré-condições, efeitos, timeout/cancelamento e recuperação; não adicionar estado genérico BLOCKED só porque aparece na conversa.

## Matriz e conciliação

[145 operações, 56 exceções e 20 candidatas JEV](10-matriz-fluxo.md) possuem rótulo original, mecanismo discutido, plano de referência, horizonte e nota de autoridade. Estado de implementação por item: NÃO VERIFICADO individualmente; evidências de componentes existentes estão no registro de decisões. O Tech Lead deverá conferir cada ponto contra código e critérios antes de gerar tickets, evitando reimplementar FAC aceito.

FAC-014/015/016/025 usados na transcrição são propostas/exemplos antigos que colidem com IDs existentes. Não renumerar passado nem criar tickets com esses nomes. Priorização deriva de dependências: fontes → índice → pacotes → decisões e observação → atualização e conflitos → avaliação/otimização. Workflow é eixo de integração desde o início, não um serviço novo ao final.

## Alternativas e gates

Automatizar todas as operações de uma vez amplia risco e não distingue capacidades existentes de novas. Recomenda-se integrar por fronteira e preservar gates existentes. Papéis são responsabilidades; não exigem um processo ou fornecedor por papel. Reviewer independente continua separado do writer. Configuração de Tech Lead e override humano não anulam elegibilidade e limites.

Aceite de código, publicação da memória, merge e deploy são eventos separados. Seu aceite publica o conhecimento; execução usa vínculo à revisão e incorporação explícita. Modelo pode sugerir, ferramenta comprova execução, responsável aceita. Tentativas repetidas observam limites sem loop infinito; stop desconhecido bloqueia recovery.

## Critérios de aceite, migração e rollback

Todos os IDs 001–145, X01–X56 e J01–J20 aparecem uma única vez na matriz; cada item mantém origem e destino; nenhuma proposta antiga é chamada de implementada sem evidência. Caminho feliz e cenários de conflito, cota, stop desconhecido, resultado desconhecido e aceite obsoleto têm critérios verificáveis. Nova versão não quebra contratos v1 sem migração explícita.

Preservar workflow atual; integrar capacidades com flags/configuração versionadas e parar execução quando fallback violar novas obrigações. Rollback desliga integração nova sem apagar checkpoint/auditoria e sem liberar writer por idade. Revisão independente e QA futuros são gates pendentes; esta entrega verifica somente documentação.

[Guia didático](../../02-arquitetura/memoria-e-decisoes/09-fluxo-completo.md).

## Como interpretar este plano

Escopo: evolução da própria La fabrique. Este documento é um plano arquitetural, sem implementação nesta entrega e sem tickets, estimativas ou sprints. As escolhas técnicas abaixo são **PROPOSTAS PARA AVALIAÇÃO**, exceto as decisões expressamente confirmadas no [registro de decisões](01-decisoes-e-dependencias.md). Exemplos de nomes, contratos e erros são propostas; não são APIs existentes. A futura implementação deverá converter os contratos em schemas Zod versionados em `packages/contracts`, sem duplicar a fonte canônica.

Stack preservada: React/TypeScript/Vite, NestJS, worker Node/TypeScript, PostgreSQL e Redis/BullMQ, VPS única. API controla e persiste; worker executa indexação, clientes e checks. Um executor inicial e um writer por workspace continuam obrigatórios. Autenticação de clientes oficiais permanece segregada. Somente JEV tem a exceção específica de API; não se habilita API de LLM por consequência.
