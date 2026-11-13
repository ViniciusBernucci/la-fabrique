> Leitura: [← Anterior](04-context-packs.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](06-modelo-local.md)

# Plano — Decision Engine

## Objetivo e estado atual

Centralizar decisões auditáveis sem entregar autoridade a modelos. RuntimeGuard, validação Zod, rotas configuradas e gates já aplicam decisões locais; um motor genérico de decisão versionado não foi localizado. Próxima versão: baseline determinístico, catálogo de decisões, registro de estratégia e escalonamento preparado para JEV → LLM elegível → responsável. Modelo local participa futuramente em decisões permitidas, sem sequência obrigatória de chamadas.

## Contratos propostos

`DecisionRequest`: ID/tipo, tarefa/run/attempt, revisão/fontes, candidatos já filtrados, política/versionamento, resultado determinístico e motivo de insuficiência. `DecisionPolicy`: estratégias autorizadas, obrigatórios, critérios de suficiência por tipo, limites configurados e exigência de intervenção humana. `DecisionResult`: estratégia/versão/modelo observado, status resolvido/incerto/bloqueado, opções escolhidas, evidências, motivo, latência/uso nullable e validação final. Nomes não implicam novos estados no enum do ticket.

Cada resultado distingue recomendação, decisão normativa humana e ação executada. ID de decisão permite idempotência; entrada alterada gera nova decisão, não reaproveita resultado anterior. Alteração de política invalida cache e resultados que dependam dela.

## Fronteiras e autoridade

Regras determinísticas filtram permissões, custos, disponibilidade, paths, revisão, obrigatórios e limites. Não escalonar bloqueio de segurança para “ver se o modelo permite”. Modelos podem priorizar opcionais e sugerir classificação; não alterar risco mínimo, autorizar gastos, confirmar parada ou aprovar entrega.

Tech Lead propõe escolha de agente/modelo por tarefa; controle manual registra override e autoria. Proposta: uma tarefa READY usa snapshot da escolha; alteração posterior invalida configuração pertinente e exige revalidação antes da próxima execução. Para trabalho já em andamento, não trocar rota silenciosamente; definir aplicação na próxima tentativa segura. Esse comportamento precisa de aceite específico antes de tickets de implementação.

## Alternativas e dependências

Manter decisões espalhadas exige menos mudança inicial, mas dificulta auditoria e comparação. Motor central com catálogo explícito exige contrato comum sem se tornar autorizador universal. Recomenda-se biblioteca de políticas e persistência na API, execução no worker, integrada gradualmente aos pontos catalogados. Não criar microserviço por padrão.

Depende de fontes/contexto e configuração do sistema. Contratos preparados para JEV nesta versão de planejamento; chamada real é implementação posterior. Critérios de confiança precisam de avaliação por pergunta; ausência de critério aprovado retorna incerto e usa fallback, sem inventar 0,8 como garantia.

## Recuperação, compatibilidade e aceite

Integrar primeiro em modo de observação, comparando decisões com o comportamento atual antes de alterar ações. Não mudar enums, workflow ou gates só para combinar com desenho da transcrição. Em falha, usar baseline quando suficiente; senão LLM elegível, depois humano. Mesma dúvida repetida respeita limites existentes e preserva checkpoint.

Critérios: bloqueios nunca consultam modelo para contorno; obrigatório não entra em competição por score; resposta com opção inexistente é recusada; override inválido não executa provider; audit trail reproduz entradas/versões; resultado atrasado não vale para nova revisão; falta de JEV segue fallback confirmado.

Riscos: abstração excessiva, confiança não calibrada, ações indevidas por resposta estruturada e mudança de rota em execução. Rollback desliga novas decisões executivas e mantém registros de observação; restaura comportamento anterior apenas quando preserva gates.

[Guia](../../02-arquitetura/memoria-e-decisoes/05-decision-engine.md).

## Como interpretar este plano

Escopo: evolução da própria La fabrique. Este documento é um plano arquitetural, sem implementação nesta entrega e sem tickets, estimativas ou sprints. As escolhas técnicas abaixo são **PROPOSTAS PARA AVALIAÇÃO**, exceto as decisões expressamente confirmadas no [registro de decisões](01-decisoes-e-dependencias.md). Exemplos de nomes, contratos e erros são propostas; não são APIs existentes. A futura implementação deverá converter os contratos em schemas Zod versionados em `packages/contracts`, sem duplicar a fonte canônica.

Stack preservada: React/TypeScript/Vite, NestJS, worker Node/TypeScript, PostgreSQL e Redis/BullMQ, VPS única. API controla e persiste; worker executa indexação, clientes e checks. Um executor inicial e um writer por workspace continuam obrigatórios. Autenticação de clientes oficiais permanece segregada. Somente JEV tem a exceção específica de API; não se habilita API de LLM por consequência.
