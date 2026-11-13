> Leitura: [← Anterior](09-fluxo-completo.md) · [Índice didático](../../02-INDEX.md)

# Matriz de rastreabilidade do fluxo discutido

Fonte: transcrição externa com hash registrado em [decisões](01-decisoes-e-dependencias.md). Rótulos e mecanismos das tabelas originais preservados abaixo. A classificação de horizonte é proposta arquitetural, não declaração de implementação. Para **cada item**, estado individual: **NÃO VERIFICADO** nesta entrega documental; capacidades atuais por componente estão no registro de evidências.

D = regras determinísticas; ML = modelo local; J = JEV; LLM = modelo de linguagem; H = humano; S = ferramentas/sistema. ML e J citados no original são participação candidata para implementação futura, mesmo em linha de integração da próxima versão. JEV tem exceção confirmada; orçamento/limites serão configurados pelo sistema. Ferramentas comprovam resultados, modelos sugerem e humano aceita. IDs do fluxo não são tickets FAC.

**Integração próxima versão** significa ponto de ligação das capacidades confirmadas com workflow; não autorização para reimplementar toda função já existente. **Evolução futura** identifica treino/modelos/datasets ou implantação posterior. **Gate preservado** conserva controle existente e limitações. Nenhum item dispensa inspeção técnica antes de tickets.

## As 145 operações

| ID original | Operação original | Mecanismo discutido | Plano de responsabilidade | Horizonte proposto |
|---|---|---|---|---|
| 001 | Receber uma solicitação de feature, bug, melhoria, refatoração ou incidente | S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 002 | Identificar usuário, projeto e permissões para criar o ticket | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 003 | Verificar se o projeto está ativo e se o repositório está cadastrado | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 004 | Interpretar a solicitação em linguagem natural | LLM — Tech Lead | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 005 | Classificar o tipo de trabalho usando categorias conhecidas | D → ML → J | [Decision Engine](05-decision-engine.md) | Integração próxima versão |
| 006 | Procurar tickets semelhantes, duplicados ou relacionados | D + ML; J para ambiguidade | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 007 | Identificar quais informações de requisito estão faltando | LLM + D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 008 | Classificar se a solicitação é ambígua ou potencialmente incompleta | D → ML/J | [Decision Engine](05-decision-engine.md) | Integração próxima versão |
| 009 | Solicitar esclarecimentos quando a dúvida muda o comportamento esperado | LLM → H | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 010 | Transformar a solicitação em descrição técnica inicial | LLM — Tech Lead | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 011 | Propor critérios de aceite verificáveis | LLM + D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 012 | Persistir ticket, versão, autor, requisitos, dúvidas e estado inicial | D + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 013 | Identificar a revisão de referência do projeto | D + Git | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração próxima versão |
| 014 | Consultar arquitetura, ADRs e políticas obrigatórias | D + S | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração próxima versão |
| 015 | Recuperar as regras de negócio diretamente relacionadas | D + ML | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração próxima versão |
| 016 | Identificar os domínios candidatos da alteração | D → ML → J | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração próxima versão |
| 017 | Encontrar classes, métodos, arquivos e módulos associados | D + análise estática | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração próxima versão |
| 018 | Consultar dependências diretas no grafo | D | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração próxima versão |
| 019 | Expandir dependências indiretas relevantes | D + ML; J para priorização | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração próxima versão |
| 020 | Localizar contratos de APIs, banco, eventos e filas | D + S | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração próxima versão |
| 021 | Identificar testes existentes e critérios associados | D + S | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração próxima versão |
| 022 | Procurar decisões históricas e implementações semelhantes | D + ML | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração próxima versão |
| 023 | Avaliar documentos candidatos semanticamente ambíguos | ML → J | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração próxima versão |
| 024 | Identificar relações não mapeadas, inconsistências e informações ausentes | D + LLM quando necessário | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração próxima versão |
| 025 | Identificar componentes de segurança, autenticação e autorização afetados | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 026 | Detectar possibilidade de migração destrutiva ou alteração irreversível | D + LLM | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 027 | Aplicar classificação mínima obrigatória de risco | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 028 | Sugerir risco adicional considerando a descrição e os efeitos indiretos | ML/J → LLM | [Decision Engine](05-decision-engine.md) | Integração próxima versão |
| 029 | Verificar se o risco é permitido pela política de automação | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 030 | Identificar incompatibilidades técnicas e arquiteturais | D + LLM | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 031 | Determinar se a feature exige nova decisão arquitetural | LLM — Arquiteto | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 032 | Comparar alternativas e trade-offs de implementação | LLM | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 033 | Identificar dependências entre tickets e bloqueios anteriores | D + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 034 | Decompor feature em etapas ou subtarefas implementáveis | LLM — Tech Lead | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 035 | Estimar esforço, complexidade e necessidades de teste | D + histórico; ML/J consultivo | [Decision Engine](05-decision-engine.md) | Integração próxima versão |
| 036 | Definir plano técnico, critérios de aceite e pedidos de aprovação | LLM + D + H quando exigido | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 037 | Validar campos e critérios mínimos do ticket | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 038 | Conferir dependências e bloqueios de outros tickets | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 039 | Verificar compatibilidade do ticket com a política de risco | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 040 | Conferir disponibilidade de worker e recursos da VPS | D + monitoramento | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 041 | Verificar autenticação e capacidades comprovadas dos provedores | D + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 042 | Excluir provedores indisponíveis, inelegíveis ou com cobrança não autorizada | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 043 | Escolher entre provedores elegíveis com base em histórico e tipo de tarefa | D → ML/J | [Decision Engine](05-decision-engine.md) | Integração próxima versão |
| 044 | Determinar limite de duração, tentativas, correções e handoffs | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 045 | Registrar a estratégia escolhida e as razões da decisão | D + S | [Decision Engine](05-decision-engine.md) | Integração próxima versão |
| 046 | Realizar transição de estado com controle de concorrência e persistência | D + PostgreSQL | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 047 | Carregar a política global vigente | D | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 048 | Carregar instruções específicas do projeto e do domínio | D | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 049 | Identificar a versão exata do código e da documentação | D + Git | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 050 | Obter regras e contratos obrigatórios | D | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 051 | Recuperar trechos de código relacionados à tarefa | D + análise estática | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 052 | Buscar documentação por palavras-chave e identificadores | D | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 053 | Buscar documentos semanticamente relacionados | ML local | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 054 | Expandir relações do grafo dentro dos limites definidos | D | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 055 | Ranquear documentos adicionais não obrigatórios | D + ML | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 056 | Resolver empate ou ambiguidade semântica entre candidatos | J, se habilitado | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 057 | Identificar contexto insuficiente, conflitante ou desatualizado | D + LLM quando necessário | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 058 | Construir Context Pack com orçamento adaptativo e referências | D + S | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 059 | Validar fontes, versões, cobertura obrigatória, omissões e segredos | D + S | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 060 | Reservar worker disponível por lease | D + PostgreSQL | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 061 | Garantir exclusividade de escrita no workspace | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 062 | Verificar se processos de uma execução anterior foram encerrados | D + sistema operacional | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 063 | Criar branch/worktree isolada | Git + D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 064 | Verificar branch e SHA de referência | Git + D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 065 | Identificar alterações preexistentes e arquivos não rastreados | Git + D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 066 | Preparar sandbox, permissões e limites de recursos | Docker/OS + D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 067 | Bloquear acesso ao host, segredos e serviços não autorizados | D + controles do ambiente | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 068 | Verificar dependências e ferramentas exigidas pelo projeto | D + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 069 | Executar verificações iniciais para registrar o baseline | S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 070 | Identificar falhas já existentes antes da feature | D + resultados de testes | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 071 | Preparar cliente oficial de IA selecionado | Runtime Gateway + D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 072 | Confirmar modo de autenticação e condições de uso | D + preflight | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 073 | Registrar snapshot/checkpoint inicial recuperável | D + Git + armazenamento | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 074 | Receber objetivo, critérios, restrições e Context Pack | D + LLM | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 075 | Confirmar arquivos e componentes que precisam ser estudados | D + ML | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 076 | Solicitar contexto adicional durante a implementação | LLM → Memory Engine | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 077 | Selecionar documentos adicionais para expansão de contexto | D → ML → J | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 078 | Interpretar regras de negócio e contratos existentes | LLM | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 079 | Formular plano de alteração limitado ao ticket | LLM — Developer | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 080 | Verificar se o plano ultrapassa o escopo autorizado | D + LLM reviewer | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 081 | Editar código incrementalmente | LLM + ferramentas | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 082 | Adicionar ou atualizar testes relacionados às mudanças | LLM | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 083 | Verificar se houve alteração em interfaces públicas ou contratos | D + ferramentas | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 084 | Detectar dependências novas e necessidade de ampliar contexto | D + ML; LLM quando necessário | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 085 | Executar verificações locais rápidas e interpretar falhas | S + D + LLM | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 086 | Registrar progresso, alterações e checkpoint intermediário | D + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 087 | Detectar quais arquivos e pacotes foram alterados | Git + D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 088 | Selecionar testes obrigatórios conforme o impacto conhecido | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 089 | Identificar testes candidatos adicionais | D + ML; J eventual | [Decision Engine](05-decision-engine.md) | Integração próxima versão |
| 090 | Executar lint e formatadores em modo apropriado | S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 091 | Executar verificação de tipos TypeScript | S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 092 | Executar testes unitários relacionados | S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 093 | Executar testes de integração necessários | S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 094 | Executar testes de contrato de APIs/eventos, se afetados | S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 095 | Executar build e verificar artefatos produzidos | S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 096 | Executar verificações de segurança configuradas | S + D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 097 | Comparar falhas com o baseline e distinguir regressões | D + S; LLM em diagnósticos ambíguos | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 098 | Consolidar resultados associados ao SHA do código | D + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 099 | Confirmar que o Developer não está mais escrevendo durante a revisão | D + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 100 | Capturar diff, revisão exata e arquivos afetados | Git + D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 101 | Construir contexto específico de revisão | D + ML | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 102 | Identificar especialidade necessária para revisão | D → ML → J | [Decision Engine](05-decision-engine.md) | Integração próxima versão |
| 103 | Selecionar provedor elegível para revisão | D → ML/J | [Decision Engine](05-decision-engine.md) | Integração próxima versão |
| 104 | Verificar aderência aos critérios de aceite | LLM Reviewer + evidências | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 105 | Analisar possíveis falhas de negócio e arquitetura | LLM Reviewer | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 106 | Examinar segurança, concorrência, tratamento de erros e regressões | LLM + ferramentas | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 107 | Identificar problemas concretos com referências a arquivos e critérios | LLM Reviewer | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 108 | Classificar achados segundo regras de severidade | D + LLM; ML/J consultivo | [Decision Engine](05-decision-engine.md) | Integração próxima versão |
| 109 | Autorizar correção limitada ou bloquear por problema crítico | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 110 | Reexecutar validações e revisão após correções | S + D + LLM | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 111 | Identificar exatamente quais componentes foram modificados | Git + D | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 112 | Identificar domínios e contratos afetados | D + ML | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 113 | Gerar relatório de entrega com base nas alterações e evidências | LLM — Documentação | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 114 | Atualizar documentação vigente dos domínios | LLM + D | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 115 | Atualizar instruções de funcionamento e exemplos relevantes | LLM | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 116 | Propor alterações em ADRs quando decisões arquiteturais mudaram | LLM + H se exigido | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 117 | Identificar conhecimentos novos e aprendizados reaproveitáveis | ML/J consultivo + LLM | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 118 | Propor inclusão, alteração ou invalidação de regras da memória | LLM + D | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 119 | Conferir conflitos com versões e decisões anteriores | D + ML; LLM quando necessário | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 120 | Atualizar índices, changelog, backlog e lessons aplicáveis | D + LLM | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 121 | Validar estrutura, links, fontes, referências e evidências | D + S | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 122 | Produzir versão candidata da memória, sem publicá-la como vigente antes do aceite | D + S | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 123 | Verificar que a revisão final tem todos os artefatos exigidos | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 124 | Conferir resultados de testes e validações da revisão exata | D + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 125 | Confirmar revisão concluída e achados resolvidos ou explicitamente aceitos | D + H quando aplicável | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 126 | Validar documentação e propostas de atualização de memória | D + LLM | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 127 | Preparar resumo executivo da implementação | LLM | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 128 | Apresentar mudanças, riscos, evidências e limitações | React + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 129 | Solicitar aceite humano associado ao SHA exato | H + D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 130 | Detectar alteração posterior ao aceite e invalidar aprovação obsoleta | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 131 | Executar merge ou deploy somente quando houver autorização específica | H + D + Git/CI | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 132 | Publicar memória vigente após incorporação aprovada, preservando histórico | D + S | [Memória persistente](02-memoria-persistente.md) | Integração próxima versão |
| 133 | Registrar estado final, decisão e trilha de auditoria | D + PostgreSQL | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Gate preservado |
| 134 | Consolidar duração e tentativas de execução | D + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 135 | Consolidar provedores e modelos efetivamente utilizados | D + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 136 | Registrar uso observado, desconhecido e custos atribuídos | D + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 137 | Calcular quantidade de handoffs e interrupções | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 138 | Medir quantidade de correções e retrabalho | D | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |
| 139 | Medir contexto inicial, expansões e recuperação de documentos | D + S | [Context Packs e continuidade](04-context-packs.md) | Integração próxima versão |
| 140 | Identificar decisões que exigiram JEV ou escalonamento | D | [Decision Engine](05-decision-engine.md) | Integração próxima versão |
| 141 | Associar decisões ao resultado validado da feature | D + H/revisão quando necessário | [Aprendizado e classificador próprio](08-aprendizado.md) | Integração próxima versão |
| 142 | Criar candidatos a exemplos de treinamento sem segredos | D + S | [Aprendizado e classificador próprio](08-aprendizado.md) | Evolução futura do aprendizado |
| 143 | Identificar exemplos que precisam de rotulagem e verificação | ML/J consultivo; H/LLM na validação | [Aprendizado e classificador próprio](08-aprendizado.md) | Evolução futura do aprendizado |
| 144 | Atualizar métricas de qualidade da seleção e das estratégias | D + S | [Aprendizado e classificador próprio](08-aprendizado.md) | Evolução futura do aprendizado |
| 145 | Encerrar execução, aplicar retenção de artefatos e liberar recursos | D + S | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração próxima versão |

## As 56 exceções

Resposta original é preservada como intenção histórica. “Fallback autorizado” agora significa LLM elegível por cliente oficial e humano se a dúvida persistir. “Bloquear” não cria enum genérico ausente. Reverter trabalho, deploy ou rollback operacional exigem autorização e revisão aplicáveis; não executar por esta matriz.

| ID original | Evento original | Detecção discutida | Resposta original | Plano relacionado | Horizonte proposto |
|---|---|---|---|---|---|
| X01 | Ticket incompleto | D + LLM | Voltar a DRAFT e solicitar informações | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X02 | Critério de aceite impossível de verificar | LLM + D | Pedir reformulação | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X03 | Ticket duplicado | D + ML/J | Sugerir vínculo ou consolidação | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X04 | Requisitos contraditórios | D + LLM | Bloquear planejamento e solicitar decisão | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X05 | Feature grande demais | D + LLM | Propor decomposição | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X06 | Dependência entre tickets não concluída | D | Manter bloqueado | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X07 | Mudança fora do escopo do projeto | D + LLM | Rejeitar ou solicitar novo escopo | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X08 | Regra de negócio não encontrada | D + ML/J | Expandir busca; se necessário, pedir esclarecimento | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X09 | Solução incompatível com ADR vigente | D + LLM | Exigir análise de mudança arquitetural | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X10 | Risco superior ao permitido | D | Exigir aprovação ou impedir execução | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X11 | Documento desatualizado | Hash, versão e Git — D | Reindexar ou invalidar | [Memória persistente](02-memoria-persistente.md) | Integração/gate próxima versão |
| X12 | Dois documentos contraditórios | D + ML/LLM | Sinalizar conflito; não escolher verdade por votação | [Memória persistente](02-memoria-persistente.md) | Integração/gate próxima versão |
| X13 | Dependência indireta desconhecida | D + ML/J | Investigar código e contratos | [Indexação, grafo e análise de impacto](03-indexacao-e-grafo.md) | Integração/gate próxima versão |
| X14 | Embedding não encontrado | D | Buscar por índice textual | [Modelo local e busca semântica](06-modelo-local.md) | Evolução futura |
| X15 | Modelo local indisponível | D | Utilizar baseline determinístico | [Modelo local e busca semântica](06-modelo-local.md) | Evolução futura |
| X16 | Confiança insuficiente do ranking | D + ML | JEV autorizado ou LLM | [Integração JEV](07-jev.md) | Evolução futura |
| X17 | JEV indisponível | D | Fallback autorizado ou pausa | [Integração JEV](07-jev.md) | Evolução futura |
| X18 | Context Pack excede o limite | D | Priorizar opcionais; dividir a tarefa se necessário | [Context Packs e continuidade](04-context-packs.md) | Integração/gate próxima versão |
| X19 | Regra obrigatória não cabe no contexto | D | Não truncar silenciosamente; decompor ou bloquear | [Context Packs e continuidade](04-context-packs.md) | Integração/gate próxima versão |
| X20 | Documento contém segredos ou dados restritos | D + verificações | Redigir/excluir ou bloquear envio | [Context Packs e continuidade](04-context-packs.md) | Integração/gate próxima versão |
| X21 | Memória de outra branch mistura estados | D + Git | Isolar pela revisão e sobreposição da branch | [Memória persistente](02-memoria-persistente.md) | Integração/gate próxima versão |
| X22 | Modelo sugere regra inexistente | D + validação de fontes | Rejeitar a sugestão não fundamentada | [Memória persistente](02-memoria-persistente.md) | Integração/gate próxima versão |
| X23 | VPS com memória insuficiente | Monitoramento — D | Pausar ou adiar novos jobs | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X24 | Disco quase cheio | Monitoramento — D | Bloquear novas execuções até recuperação segura | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X25 | Worker desconectado | Heartbeat/lease — D | Interromper e preservar estado | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X26 | Processo anterior não terminou | OS + D | Impedir novo writer | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X27 | Credencial expirada | Preflight — D | AUTH_REQUIRED | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X28 | Cota do provedor esgotada | Adapter + D | WAITING_PROVIDER ou handoff | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X29 | Modelo contratado indisponível | Preflight — D | Selecionar outro elegível | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X30 | Cliente oficial muda comportamento/versão | D + testes de compatibilidade | Bloquear adapter até validação | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X31 | Execução ultrapassa timeout | Supervisor — D | Encerrar árvore de processos e checkpoint | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X32 | Handoff falha parcialmente | D + Git + S | BLOCKED_RECOVERY até verificar integridade | [Context Packs e continuidade](04-context-packs.md) | Integração/gate próxima versão |
| X33 | Executor tenta acessar segredo | Controles do sandbox | Bloquear acesso e registrar incidente | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X34 | Duas execuções disputam workspace | Lock + fencing token | Somente execução autorizada mantém escrita | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X35 | Build falha | Compilador — S | Diagnóstico e correção limitada | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X36 | Teste falha | Test runner — S | Comparar com baseline | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X37 | Dependência nova é necessária | Git/package diff + LLM | Verificar política e autorização | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X38 | Developer modifica arquivos proibidos | Diff + D | Bloquear alteração e exigir recuperação segura | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X39 | Developer amplia escopo | D + Reviewer | Reverter/desmembrar alterações com segurança | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X40 | Falha de segurança detectada | Scanner + revisão | Bloquear aceite | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X41 | Reviewer discorda da implementação | LLM + evidências | Gerar achado concreto, sem loop ilimitado | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X42 | Duas correções não resolvem | Contador — D | PAUSED_LIMIT + diagnóstico | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X43 | Evidência não corresponde ao SHA | D + Git | Invalidar evidência e reexecutar verificação | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X44 | Teste E2E não pode ser executado | S + D | Marcar NÃO VERIFICADO e solicitar decisão | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X45 | Ambiente de teste contaminado por dados reais | D + verificações | Bloquear execução e tratar incidente | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X46 | Saída da LLM viola formato ou contrato | Validador — D | Rejeitar/repetir de forma limitada | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X47 | Relatório de entrega ausente | Document Gate — D | Impedir DONE | [Memória persistente](02-memoria-persistente.md) | Integração/gate próxima versão |
| X48 | README do domínio desatualizado | D + revisão semântica | Devolver para documentação | [Memória persistente](02-memoria-persistente.md) | Integração/gate próxima versão |
| X49 | Documento inventa teste não executado | Evidências + D/LLM | Rejeitar informação | [Memória persistente](02-memoria-persistente.md) | Integração/gate próxima versão |
| X50 | Nova decisão conflita com ADR | D + LLM | Exigir avaliação e aceite | [Memória persistente](02-memoria-persistente.md) | Integração/gate próxima versão |
| X51 | Humano rejeita implementação | H | Retornar ao planejamento ou correção | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X52 | Alteração depois do aceite | Git + D | Invalidar aceite anterior | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X53 | Merge apresenta conflitos | Git + D | Resolver em execução autorizada e revalidar | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X54 | Deploy autorizado apresenta falha | CI/monitoramento | Executar rollback aprovado ou recuperação | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X55 | Backup não restaura corretamente | Teste de recuperação | Bloquear operação dependente desse backup | [Fluxo completo de desenvolvimento](09-fluxo-completo.md) | Integração/gate próxima versão |
| X56 | Exemplo de treinamento apresenta rótulo duvidoso | Validação de dataset | Não promovê-lo ao conjunto confiável | [Aprendizado e classificador próprio](08-aprendizado.md) | Evolução futura |

## As 20 decisões candidatas ao JEV

Catálogo para avaliação; nenhuma pergunta abaixo ganha autoridade por estar listada. Filtros determinísticos mantêm apenas opções válidas. Escolha de provider continua gerenciada pelo Tech Lead e alterável pelo responsável; J13 é sugestão, não substituição automática dessa escolha. J08 sinaliza necessidade de esclarecimento, não cria requisito implícito.

| ID original | Pergunta original | Primitiva discutida | Contrato e horizonte |
|---|---|---|---|
| J01 | Qual categoria melhor representa a solicitação? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J02 | O ticket parece descrever uma mudança funcional ou correção? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J03 | Qual domínio candidato deve ser investigado primeiro? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J04 | Este documento adicional parece relevante para a feature? | Noul | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J05 | Quanto este trecho contribui para o objetivo do ticket? | Score | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J06 | Qual relação indireta merece investigação? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J07 | O ticket parece semanticamente semelhante a outro já registrado? | Noul | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J08 | A descrição sugere a existência de um requisito implícito que precisa ser esclarecido? | Noul | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J09 | Qual grupo de documentos opcionais deve ser priorizado? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J10 | Qual especialidade de revisão adicional parece apropriada? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J11 | Qual tipo de dificuldade melhor descreve um bloqueio de desenvolvimento? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J12 | O diagnóstico sugere falta de contexto ou problema de implementação? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J13 | Entre provedores elegíveis, qual parece mais adequado à natureza da tarefa? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J14 | Qual grupo de testes adicionais parece mais relevante? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J15 | Esta anotação parece regra de negócio, decisão técnica ou observação? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J16 | O documento parece semanticamente duplicado de outro? | Noul | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J17 | Este aprendizado merece proposta de incorporação à memória? | Noul | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J18 | Qual classe de incidente melhor corresponde aos sintomas apresentados? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J19 | Qual categoria representa melhor a causa provável do retrabalho? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |
| J20 | Qual grupo de documentos precisa de investigação após uma falha recorrente? | Choice | [Adapter JEV futuro](07-jev.md), catálogo sujeito à avaliação por pergunta |

## Diferenças que precisam ser preservadas

A conversa afirma em um trecho que nenhuma implementação tinha sido confirmada: isso descreve aquele momento, não o HEAD inspecionado. FAC-006 já consta aceito no backlog real. FAC-014/015/016 sugeridos na conversa não são IDs disponíveis. FAC-025 de recuperação é exemplo histórico, enquanto backlog atual usa FAC-025 para mudança da home. Não converter exemplos em tickets ou evidências.

READY → RUNNING é transição comprovada no recorte de persistência atual; WAITING_WORKER e demais etapas possuem contratos próprios. Handoff interno atual pode usar a mesma Attempt sob controles existentes; uma mudança para nova Attempt precisa de decisão/migração específica. Modelo local não será condição para uso do JEV; a integração seguirá insuficiência determinística e configuração válida.
