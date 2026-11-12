> Leitura: [← Anterior](06-modelo-local.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](08-aprendizado.md)

# Plano futuro — Integração JEV preparada para configuração no sistema

## Objetivo e autorização específica

O responsável confirmou exceção somente para JEV e pediu documentação de contratos, configuração, segurança, acionamento e falhas. Implementação fica para tarefa posterior. A exceção não autoriza APIs de LLM, compras ou valores ilimitados. Nenhum adapter JEV existe na base inspecionada. Planejamento deve deixar claros todos os pontos que a integração precisará implementar; não alegar que estão configurados no software.

## Configuração proposta pelo sistema

| Campo | Regra proposta |
|---|---|
| enabled e modo | Desativado/observação/uso; alteração auditada |
| endpoint e modelo solicitado | Origem oficial permitida e modelo selecionado; sem URL arbitrária |
| credentialReference | Referência ao armazenamento privado; painel recebe somente estado |
| allowedDecisionTypes | Allowlist de perguntas que podem usar JEV |
| orçamento, moeda e janela | Obrigatórios antes de chamada paga; valores definidos pelo responsável |
| chamadas por tarefa e período | Limites configuráveis; incluir tentativas repetidas |
| timeout, repetição e espera | Limites explícitos; sem retry ilimitado |
| limites de payload/candidatos/perguntas | Configuráveis, validados antes do envio |
| critérios de suficiência por pergunta | Versionados e avaliados; sem default de confiança inventado |
| tarifa e validade da fonte | Fonte/data/moeda/unidade; desconhecido não vira zero |
| política de cache/retention | Versão, validade e escopo de dados sanitizados |

As janelas exatas e valores são decisões de configuração pendentes, não inferidos. Pré-condições inválidas geram estado não pronto com motivos no painel. Habilitar exige versão de configuração aprovada; qualquer alteração relevante invalida avaliação pertinente.

## Contrato externo observado e contrato interno proposto

Consulta pública em 2026-10-09: [quick start oficial](https://docs.typesafe.ai/introduction/quickstart) descreve POST `https://api.typesafe.ai/v1/systemone`, Bearer, state/model/questions, answers/model/usage. Esse é um resumo do contrato observado; revalidar API Reference na implementação. Não copiar alias latest como versão fixa. Persistir modelo solicitado e efetivo quando informado.

`JevDecisionRequest` interno contém DecisionRequest, questões versionadas e candidatos permitidos. Adapter mapeia Choice/Score/Noul para o formato oficial. `JevDecisionResult` normaliza respostas, probabilidades/confiança específicas, modelo efetivo, request ID quando disponível, uso observado, latência e erro. Schema externo e interno são separados. [Confiança oficial](https://docs.typesafe.ai/confidence) requer tratamento por tipo; Noul não recebe confidence inventado.

## Segurança e orçamento

Credencial guardada fora de DB de controle/contexto/logs/worktree/sandbox, por referência ao mecanismo privado compatível com o host. Fluxo seguro de cadastro/rotação no sistema precisa ser implementado e revisado; não escolher mecanismo ausente como existente. API recebe comandos administrativos; worker de integração acessa segredo por canal restrito e executa HTTPS. Tools do agente não recebem credencial nem cliente HTTP geral para o serviço.

Enviar apenas metadados e trechos necessários, sanitizados, e candidatos do escopo. Texto recebido é dado não confiável. Limites de gasto precisam de reserva atômica antes de envio, contagem incluindo retries, e reconciliação de uso conhecido/desconhecido. Sem estimativa conservadora verificável, não chamar sob teto monetário rígido. Custo desconhecido permanece pendente; não contabilizar como gratuito.

## Acionamento e falhas

Somente após insuficiência determinística sem bloqueio de política. Modelo local não é pré-condição inicial. Resposta inválida/opção fora do conjunto/baixa suficiência não executa ação. Auth inválida marca configuração requerida; timeout/429/5xx usam política limitada e fallback; cancelamento interrompe novas chamadas. Envio com resultado desconhecido não autoriza repetição automática: investigar suporte oficial a idempotência e reconciliação antes de habilitar retries pagos.

Cache depende de estado/pergunta/candidatos/política/modelo/revisão; evitar reutilização com alias de modelo mutável sem identidade efetiva. Fallback confirmado: LLM elegível por cliente oficial → humano se persistir dúvida. Falta de ambos pausa decisão, sem liberar gastos ou ignorar obrigação.

## Avaliação, migração e rollback

Começar em comparação sem ações; comparar com baseline e avaliação humana. HTTP direto TypeScript versus SDK oficial é alternativa pendente de compatibilidade; não introduzir Python como serviço por conveniência. Critérios: config incompleta não chama; limite concorrente não ultrapassa reserva; segredo ausente do payload/log; entrada maliciosa não altera política; resposta perdida preserva consumo desconhecido; desligamento usa fallback; modelo efetivo fica registrado.

Rollback desativa JEV, mantém auditoria/contabilidade e cancela novas chamadas, sem presumir cancelamento do lado fornecedor. Contratos de configuração exigirão migration e UI posteriores. [Guia didático](../../02-arquitetura/memoria-e-decisoes/07-jev.md).

## Como interpretar este plano

Escopo: evolução da própria La fabrique. Este documento é um plano arquitetural, sem implementação nesta entrega e sem tickets, estimativas ou sprints. As escolhas técnicas abaixo são **PROPOSTAS PARA AVALIAÇÃO**, exceto as decisões expressamente confirmadas no [registro de decisões](01-decisoes-e-dependencias.md). Exemplos de nomes, contratos e erros são propostas; não são APIs existentes. A futura implementação deverá converter os contratos em schemas Zod versionados em `packages/contracts`, sem duplicar a fonte canônica.

Stack preservada: React/TypeScript/Vite, NestJS, worker Node/TypeScript, PostgreSQL e Redis/BullMQ, VPS única. API controla e persiste; worker executa indexação, clientes e checks. Um executor inicial e um writer por workspace continuam obrigatórios. Autenticação de clientes oficiais permanece segregada. Somente JEV tem a exceção específica de API; não se habilita API de LLM por consequência.
