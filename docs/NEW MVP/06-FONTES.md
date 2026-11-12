# Fontes e precedência

## Fonte normativa

A [entrevista “Reconstruir MVP simplificado”](https://chatgpt.com/c/6af502cf-88a4-83e9-ae8c-17f007ed8705) e o pedido atual definem o novo MVP. As respostas do usuário têm precedência sobre recomendações nas perguntas e sobre registros anteriores que tenham sido corrigidos. A correção da D42 está no turno da D43 e foi reafirmada até o encerramento. A [matriz](04-RASTREABILIDADE.md) é o registro consolidado das 95 decisões da entrevista.

## Anexos históricos lidos

| Arquivo do kit v2.1 | Conteúdo histórico identificado |
|---|---|
| README.md | Índice, sequência anterior de início e topologia de VPS única. |
| AGENTS.md, espelhado como AGENTS (project file).md | Guia anterior de Codex, com política de assinaturas e execução por ticket. |
| ANTIGRAVITY.md | Guia anterior de Antigravity, automação e QA de interface. |
| BACKLOG.md | FAC-001 a FAC-013 e sequência de trabalho do kit antigo. |
| CHANGELOG.md | Histórico de v1, v2 e v2.1. |
| CLAUDE.md | Guia anterior de Claude Code, autenticação e revisão. |
| ESPEC-MVP.md | Entidades, API, estados, handoff e critérios da proposta v2. |
| FONTES.md | Pesquisa técnica de 29/09/2026 e limites das evidências de clientes oficiais. |
| MATRIZ-COBERTURA.md | Migração histórica v1 → v2, com classificação de risco e escopo antigos. |
| PILOTO.md | Contrato antigo de piloto com limites, dimensão sugerida e APIs desativadas. |
| PLANO-MVP.md | Plano antigo incremental, experimento de dez tickets e terceiro adapter como evolução. |
| PROMPT-INICIAL.md | Ponto de entrada e política de execução do kit antigo. |
| INDEX.md | Navegação da documentação v2.1. |
| POLITICA-IA.md | Política anterior de engenharia, documentação e aceite. |

Esses arquivos contextualizam a origem do projeto. O backlog FAC, o roteiro incremental, o piloto de dez tickets, o teto fixo de um executor, a política de assinatura sem API, a classificação de risco antiga, a dimensão sugerida de VPS, os defaults de handoff e a posição de Antigravity no plano antigo pertencem àquela versão. A especificação atual deriva seu escopo, dependências e critérios da entrevista nova e do pedido de bootstrap. Os anexos permanecem como referência no material de origem.

## Documentação de integração

- GitHub documenta criação de Draft PR e escopos do endpoint. Verificar que o plano e o repositório reais permitem a operação no preflight. [GitHub REST — pull requests](https://docs.github.com/en/rest/pulls/pulls)
- O adaptador Jira deve descobrir tipos de subtarefa, campos e transições disponíveis para a conexão e o projeto cadastrados. [Atlassian — Jira Cloud REST, issues](https://developer.atlassian.com/cloud/jira/platform/rest/v2/api-group-issues/)
- O adaptador ClickUp deve validar lista, vínculo com item pai e mapeamento de campos no ambiente real. [ClickUp — criação de tarefa](https://developer.clickup.com/reference/createtask), [ClickUp — tasks](https://developer.clickup.com/docs/tasks)

- A implementação de MCP usa a especificação de transportes e as operações de descoberta e chamada de ferramentas. [MCP — transportes](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports), [MCP — ferramentas](https://modelcontextprotocol.io/specification/2025-11-25/server/tools)

Os contratos internos dos documentos pertencem à Le Fabrique. O preflight deve comprovar versões, permissões, modos de autenticação, capacidades, cobrança e comportamento efetivo de cada integração antes da ativação. O pacote registra requisitos e cenários de teste para essa etapa.
