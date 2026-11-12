## 2026-11-12 — E09: modelos e gateways de API (T09-02)
Catálogo de modelos com compatibilidade explicada, sobreposição agente/nó verificável e adaptadores OpenRouter e Vercel AI Gateway com ferramentas controladas. Gateways reais NOT_RUN. [Entrega](entregas/2026-11-12-E09-modelos-e-gateways.md) · [ADR-0021](adr/0021-modelos-e-gateways-de-api.md).

## 2026-11-12 — E08: MCP (T08-04)
Cadastro de servidores MCP (Streamable HTTP), descoberta, allowlist por agente com hash do contrato pinado e revalidação a cada chamada. Servidor real NOT_RUN, stdio UNSUPPORTED. [Entrega](entregas/2026-11-12-E08-mcp.md) · [ADR-0020](adr/0020-mcp-cadastro-descoberta-e-allowlist.md).

# Changelog

## 2026-11-12 — E06: escrita na origem sem duplicação (T06-03)
Subtarefas, dependências, links e estados mapeados escritos pelo ledger com prova de existência; Draft PR deixa a tarefa aguardando merge e a conclusão exige merge confirmado. [Adendo da entrega](entregas/2026-11-12-E06-adaptadores-jira-clickup.md) · [ADR-0019](adr/0019-escrita-na-origem-sem-duplicacao.md).

## 2026-11-12 — E13: ledger de efeitos externos (T13-04)
Intenção antes do envio, prova verificável para reconciliar respostas perdidas (confirmado não duplica, ausente permite repetir, inconclusivo bloqueia e abre decisão com evidência) e reconciliação de envios abandonados. [Entrega](entregas/2026-11-12-E13-ledger-de-efeitos.md) · [ADR-0018](adr/0018-ledger-de-efeitos-externos.md).

## 2026-11-12 — E13: pedidos e decisões humanas (T13-02)
Central de Decisões com pedidos por execução, resolução transacional restrita à execução afetada, decisão repetida e obsoleta com resultado definido, e reautenticação para operações sensíveis. [Entrega](entregas/2026-11-12-E13-decisoes-humanas.md) · [ADR-0017](adr/0017-pedidos-e-decisoes-humanas.md).

## 2026-11-12 — E11: supervisão de processos (T11-03)
Prazos absoluto e de inatividade, atividade por saída/CPU/I-O da árvore, encerramento confirmado e quarentena do workspace quando o estado é incerto. [Adendo da entrega](entregas/2026-11-12-E11-sandbox.md) · [ADR-0016](adr/0016-supervisao-de-processos.md).

## 2026-11-12 — E08: catálogo de ferramentas (T08-03)
Ferramentas nativas e personalizadas com esquema tipado, permissões e limites; validação, concessão e escopo antes de qualquer efeito; política de risco; saída tipada. MCP segue no T08-04. [Entrega](entregas/2026-11-12-E08-catalogo-de-ferramentas.md) · [ADR-0015](adr/0015-catalogo-de-ferramentas.md).

## 2026-11-12 — E08: catálogo de skills e importação segura (T08-02 parcial)
Quatro modalidades de skill, importação de pacote/arquivo como dado (sem executar), scripts contra a política, hash e trilha auditável. Importação por Git indisponível até o T15-01. [Entrega](entregas/2026-11-12-E08-catalogo-de-skills.md) · [ADR-0014](adr/0014-catalogo-de-skills-e-importacao-segura.md).

## 2026-11-12 — E08: catálogo de agentes e sobrescritas por nó (T08-01)
Catálogo global de agentes com conta por referência de conexão, sobrescritas por nó que não alteram a base (ferramentas só reduzem) e configuração efetiva sanitizada. [Entrega](entregas/2026-11-12-E08-catalogo-de-agentes.md) · [ADR-0013](adr/0013-catalogo-de-agentes-e-sobrescritas.md).

## 2026-11-12 — E13: política de risco no executor (T13-01)
Classificação de ações por risco (permitir, aprovar, proibir) para scripts, API, MCP, conectores, Git e CLIs, com proibições embutidas, parâmetros canônicos e trilha redigida. Central de aprovações e autorizações seguem em T13-02 e T13-03. [Entrega](entregas/2026-11-12-E13-politica-de-risco.md) · [ADR-0012](adr/0012-politica-de-risco-no-executor.md).

## 2026-11-12 — E09: contas e clientes CLI de IA (T09-01 parcial)
Adaptadores de Claude Code e Codex CLI com modo autorizado, contas isoladas por volume, eventos normalizados e cancelamento da árvore de processos no sandbox. CLIs reais NOT_RUN. [Entrega](entregas/2026-11-12-E09-clientes-cli.md) · [ADR-0011](adr/0011-clientes-cli-de-ia-no-sandbox.md).

## 2026-11-12 — E07: motor de workflows (T07-02)
Motor determinístico com estado serializável, condições, loops limitados, espera durável, decisão humana, ramificação/convergência, rotas de erro e subfluxos expandidos em linha (sem recursão). Executores reais seguem em outros épicos. [Entrega](entregas/2026-11-12-E07-motor-de-workflows.md) · [ADR-0010](adr/0010-motor-de-workflows.md).

## 2026-11-12 — E07: editor visual de workflows (T07-01)
Documento de workflow, catálogo de nós, validador com nó/campo, API de rascunhos e editor de canvas. Publicação, motor e seleção por regras seguem em T07-02 a T07-04. [Entrega](entregas/2026-11-12-E07-editor-de-workflows.md) · [ADR-0009](adr/0009-documento-de-workflow-e-validacao.md).

## 2026-11-12 — E11: sandbox por feature (T11-01 parcial)
Container, workspace e rede por feature com endurecimento, quotas e testes adversariais em Docker real. Sem saída de rede controlada nem clone remoto ainda. [Entrega](entregas/2026-11-12-E11-sandbox.md) · [ADR-0008](adr/0008-sandbox-por-feature-em-containers.md).

## 2026-11-12 — E02: scripts de inventário da VPS e CI (T02-01, T02-03 parciais)
Scripts somente leitura para inventariar e medir a VPS e workflow de CI com imagens por commit e evidência. Nada executado contra a VPS nem no GitHub (NOT_RUN). [Entrega](entregas/2026-11-12-E02-vps-e-ci.md).

## 2026-11-12 — E05: origem do projeto com verificação e preview (T05-02 parcial)
Critérios de elegibilidade, mapa de prioridade e escopo; verificação remota da origem (Jira/ClickUp) com preview dos itens elegíveis; ativação exige verificação atual; uma lista não alimenta dois projetos ativos. GitHub segue não verificado (T15-01). Detalhes: [adendo da entrega](entregas/2026-11-12-E05-cadastro-de-projeto.md).

## 2026-11-12 — E06: adaptadores Jira e ClickUp (T06-01, T06-02 parciais)
Interface `Tracker` simétrica com adaptadores Jira e ClickUp (leitura paginada, subtarefas, transições válidas, dependências, links) e suíte de contrato com fixtures. Instâncias reais NOT_RUN. T06-03 a T06-05 bloqueados por dependências. Detalhes: [entrega](entregas/2026-11-12-E06-adaptadores-jira-clickup.md) · [ADR-0007](adr/0007-adaptadores-jira-e-clickup-simetricos.md).

## 2026-11-12 — E05: cadastro de projeto em rascunho (T05-01)
Assistente de cadastro por etapas (Dados, Repositório, Origem, Revisão) com persistência por etapa, relatório de validação com ação por pendência, ativação bloqueada sem preenchimento válido, suspensão/retomada e trilha de alterações. T05-02 e T05-03 seguem pendentes (dependem de E06 a E09). Corrige também o `make check-secrets`, que falhava desde o E04. Detalhes: [entrega](entregas/2026-11-12-E05-cadastro-de-projeto.md) · [ADR-0006](adr/0006-cadastro-de-projeto-em-rascunho.md).

## 2026-11-12 — E04: acesso e credenciais (T04-01, T04-02, T04-03 parcial)
Login com senha + TOTP, códigos de recuperação de uso único, sessões revogáveis, reautenticação, CSRF, HTTPS obrigatório em produção e cofre de conexões cifrado com chave externa ao banco. **Removido** o acesso dev sem senha. Detalhes: [entrega](entregas/2026-11-12-E04-acesso-e-credenciais.md) · [ADR-0005](adr/0005-autenticacao-sessoes-e-cofre-de-conexoes.md).

## 2026-11-12 — Nova stack: fundação (E01, E02 parcial, E03)
Stack Go + React + PostgreSQL + Docker substitui NestJS/Redis. Skills novas instaladas. Front limpo no design "Novo". Detalhes: [entrega](entregas/2026-11-12-E01-E03-fundacao.md).
