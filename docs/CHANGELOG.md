## 2026-11-12 — E15: memória técnica pendente e oficial (T15-05)
Registros de memória com evidência, deduplicação e escopo por projeto; pendentes só na própria branch; só o merge confirmado promove, e só o que vale para a revisão integrada (duplicata soma evidência, fato mudado substitui o oficial); lições extraídas de achados de revisão corrigidos. Ligação ao contexto/worker e telas NOT_RUN. [Entrega](entregas/2026-11-12-E15-memoria-tecnica.md) · [ADR-0053](adr/0053-memoria-tecnica-pendente-e-oficial.md).

## 2026-11-12 — E15: revisão externa e PR fechada (T15-04)
Comentários de autores autorizados descobertos sob demanda viram um só item de correção (o mesmo comentário não duplica trabalho); PR fechada conserva artefatos e oferece cancelar, reabrir (conferida no GitHub) e nova tentativa limitada, com histórico e decisão respeitada. GitHub real e execução da correção pelo worker NOT_RUN. [Entrega](entregas/2026-11-12-E15-revisao-externa-e-pr-fechada.md) · [ADR-0052](adr/0052-revisao-externa-e-pr-fechada.md).

## 2026-11-12 — E15: confirmar merge e propagar conclusão (T15-03)
Consulta da PR sob demanda: aberta ou fechada sem merge não libera nada; merge com SHA conclui o item na origem pelo ledger e devolve à fila o que esperava; falha ao projetar o status fica visível e reconcilia sem duplicar. GitHub real e acionamento pelo worker NOT_RUN. [Entrega](entregas/2026-11-12-E15-merge.md) · [ADR-0051](adr/0051-observacao-do-merge-e-propagacao.md).

## 2026-11-12 — E16: retenção com integridade (T16-04)
Logs detalhados removidos após 30 dias com resumo permanente e seq preservado; checkpoints de execuções concluídas removidos após carência, sem tocar em bloqueados nem em ligados a incidente, com conferência de integridade, simulação e auditoria por item. Agendamento e tela NOT_RUN. [Entrega](entregas/2026-11-12-E16-retencao.md) · [ADR-0050](adr/0050-retencao-com-integridade.md).

## 2026-11-12 — E15: documentação viva e Draft PR (T15-02)
Gate de documentação por módulo e histórico vivo da feature (commits por tarefa, checks, risco, revisão, E2E); a entrega só abre a Draft PR com os gates em dia, de forma idempotente pelo ledger (repetir e resposta perdida não duplicam). GitHub real NOT_RUN. [Entrega](entregas/2026-11-12-E15-documentacao-e-draft-pr.md) · [ADR-0049](adr/0049-documentacao-viva-e-draft-pr.md).

## 2026-11-12 — E16: status, controles e logs em tempo real (T16-01)
Resumo do projeto com controles persistidos e a causa de cada bloqueio, histórico de eventos pesquisável e fluxo SSE com reconexão pelo cursor (sem perder nem repetir), com painel de execução no detalhe do projeto. Executores gravando eventos e navegador real NOT_RUN. [Entrega](entregas/2026-11-12-E16-execucao-ao-vivo.md) · [ADR-0048](adr/0048-painel-de-execucao-ao-vivo.md).

## 2026-11-12 — E14: revisão independente e correções (T14-03)
Revisor por projeto com fallback e primeira rodada independente (garantida no banco), achados com severidade (crítico e relevante bloqueiam, baixo fica registrado), correção só com checks aprovados e rodada de revalidação, limite padrão de duas rodadas e esgotamento que suspende a tarefa com checkpoint seguro e abre decisão. Agente revisor real e tela NOT_RUN. [Entrega](entregas/2026-11-12-E14-revisao-independente.md) · [ADR-0047](adr/0047-revisao-independente-e-correcoes.md).

## 2026-11-12 — E14: testes e E2E por impacto (T14-04)
Orientação de testes por arquivo alterado (bloqueia em risco alto), E2E exigido por nível e impacto na UI, jornada em ambiente isolado com fixtures e capturas com hash presas à revisão, e justificativa de inaplicabilidade com evidência alternativa existente. Navegador real NOT_RUN. [Entrega](entregas/2026-11-12-E14-testes-e-e2e.md) · [ADR-0046](adr/0046-testes-e-e2e-por-impacto.md).

## 2026-11-12 — E12: recuperação após falha ou reboot (T12-03)
Tentativas interrompidas (lease vencida) são diagnosticadas item a item — quietude, checkpoint íntegro, efeitos e estado verificável — e voltam à fila sozinhas quando seguras; incerteza abre decisão e bloqueia só o afetado (a vaga só volta com quietude e checkpoint provados); limites separados por classe de falha. Inspetor real, varredura no worker e tela NOT_RUN. [Entrega](entregas/2026-11-12-E12-recuperacao.md) · [ADR-0045](adr/0045-recuperacao-de-tentativas-interrompidas.md).

## 2026-11-12 — E10: controles operacionais (T10-04)
Controles globais e por projeto, persistentes e auditados: pausar após a feature atual (feature nova não inicia, a atual termina) e suspender agora (para com checkpoint seguro, devolve a vaga e invalida o dono antigo); retomar sem reativar o que outro escopo mantém suspenso. Suspensor real e tela NOT_RUN. [Entrega](entregas/2026-11-12-E10-controles.md) · [ADR-0044](adr/0044-controles-operacionais-pausa-e-suspensao.md).

## 2026-11-12 — E14: classificação e reavaliação de risco (T14-02)
Nível de risco = máximo entre a estimativa da IA e pisos determinísticos (arquivos críticos e tamanho do diff, por projeto), reavaliado com o diff real e justificado; os três níveis escolhem as verificações previstas. Diff real, checks de segurança e tela NOT_RUN. [Entrega](entregas/2026-11-12-E14-risco.md) · [ADR-0043](adr/0043-classificacao-de-risco-por-pisos-deterministicos.md).

## 2026-11-12 — E10: limite global, leases e justiça (T10-03)
Reclamação de vaga em transação serializada com limite global persistente, lease com heartbeat e geração, envelhecimento determinístico e prioridade de projeto; escritor obsoleto não conclui, não bloqueia e não grava checkpoint. Worker real e controles NOT_RUN. [Entrega](entregas/2026-11-12-E10-scheduler.md) · [ADR-0042](adr/0042-scheduler-limite-global-leases-e-justica.md).

## 2026-11-12 — E17: matriz de evidências e ficha de aceite (T17-01)
Matriz de evidências gerada dos casos P01–P18, critérios dos tickets e checklist (com teste de rastreabilidade e de deriva) e ficha de preparação pré-preenchida com as pendências do responsável. Projeto, feature e acessos reais NOT_RUN. [Entrega](entregas/2026-11-12-E17-matriz-e-ficha-de-aceite.md) · [ADR-0041](adr/0041-matriz-de-evidencias-derivada.md).

## 2026-11-12 — E14: checks no baseline e classificação de falhas (T14-01)
Execução dos checks no baseline e na revisão candidata (sandbox real), evidência com assinatura da falha, regressão bloqueia, falha anterior comprovada é registrada e origem desconhecida fica em investigação sem virar baseline. Preparo do baseline e tela NOT_RUN. [Entrega](entregas/2026-11-12-E14-checks-e-baseline.md) · [ADR-0040](adr/0040-checks-baseline-e-classificacao-de-falhas.md).

## 2026-11-12 — E10: fila local por projeto (T10-02)
Fila de tarefas técnicas: só inicia o elegível (pré-requisitos, projeto ativo, requisito sem pendência, vaga única por projeto garantida por índice), correção de PR na próxima vaga, esperas por merge/provedor/decisão/recursos e vaga devolvida só com checkpoint seguro. Scheduler, portas de provedor e tela NOT_RUN. [Entrega](entregas/2026-11-12-E10-fila-local.md) · [ADR-0039](adr/0039-fila-local-por-projeto.md).

## 2026-11-12 — E06: mudança de requisitos e troca de plataforma (T06-05)
Revisões do requisito com histórico e impacto (metadado adiado até marco seguro, conteúdo sem trabalho iniciado pede novo plano, conteúdo com trabalho iniciado abre decisão sem alterar nada) e troca de origem por mapeamento validado, com histórico de identidades e uma única origem. Plataformas reais, rotas e tela NOT_RUN. [Entrega](entregas/2026-11-12-E06-mudanca-de-requisitos-e-troca-de-origem.md) · [ADR-0038](adr/0038-mudanca-de-requisitos-e-troca-de-origem.md).

## 2026-11-12 — E16: consumo, orçamento e provedores (T16-03)
Tela de consumo com tetos e reservas, gasto em API por origem do valor, assinatura em unidades próprias, disponibilidade por conta e cota observada (ou "não observada"), e o bloqueio de novas chamadas visível. Navegador real e provedores reais NOT_RUN. [Entrega](entregas/2026-11-12-E16-consumo-e-provedores.md) · [ADR-0037](adr/0037-painel-de-consumo-e-provedores.md).

## 2026-11-12 — E12: retomada seletiva e reconciliação de insumos (T12-04)
Retomada num nó pelo grafo congelado: valida checkpoint, versão e efeitos, incorpora mudanças compatíveis, abre decisão no conflito sem alterar nada e refaz só o conjunto afetado (dependentes só se a saída mudar). Worker, Git real e tela NOT_RUN. [Entrega](entregas/2026-11-12-E12-retomada-seletiva.md) · [ADR-0036](adr/0036-retomada-seletiva-e-reconciliacao.md).

## 2026-11-12 — E10: plano versionado da feature (T10-01)
Contrato do plano com critérios verificáveis, risco e DAG; validador que recusa ciclo, referência ausente e critério inverificável; versões que preservam a anterior em caso de conflito; publicação das subtarefas na origem sem duplicar no retry. Agente planejador, scheduler e tela NOT_RUN. [Entrega](entregas/2026-11-12-E10-planejamento.md) · [ADR-0035](adr/0035-plano-versionado-da-feature.md).

## 2026-11-12 — E05: perfil técnico e recursos do projeto (T05-03)
Descoberta segura de manifestos (sugere com origem, nunca lê `.env`), perfil com segredo só por referência, associação explícita de recursos globais com sobrescrita local, validação que bloqueia workflow em rascunho e conta incompatível, e aptidão para executar. Repositório real e tela NOT_RUN. [Entrega](entregas/2026-11-12-E05-perfil-e-recursos.md) · [ADR-0034](adr/0034-perfil-tecnico-e-recursos-do-projeto.md).

## 2026-11-12 — E06: sincronização manual e preflight (T06-04)
Leitura da origem só sob demanda (manual ou preflight), com diferenças registradas uma vez, importação deduplicada, elegibilidade/prioridade reavaliadas e conflito com edição humana. Plataformas reais, GitHub e tela NOT_RUN. [Entrega](entregas/2026-11-12-E06-sincronizacao-manual-e-preflight.md) · [ADR-0033](adr/0033-sincronizacao-manual-e-preflight.md).

## 2026-11-12 — E02: verificação e ensaio de restauração (T02-04)
Verificador de banco restaurado (migrations, TOTP e conexões decifráveis com a chave, hashes de checkpoints, skills e contexto) e `make restore-drill` (backup, restore isolado, verificação, descarte). Snapshot da VPS e produção NOT_RUN. [Entrega](entregas/2026-11-12-E02-ensaio-de-restauracao.md) · [ADR-0032](adr/0032-verificacao-de-restauracao.md).

## 2026-11-12 — E09: fallback e disponibilidade (T09-04)
Saúde de contas por causa (autenticação, cota, indisponibilidade, financeiro) combinando sinal oficial e histórico com backoff; fallback só pela cadeia explícita, compatível e dentro do orçamento, após checkpoint e quiescência; espera sem ocupar a vaga; sem rodízio. Provedores reais, worker e tela NOT_RUN. [Entrega](entregas/2026-11-12-E09-fallback-e-disponibilidade.md) · [ADR-0031](adr/0031-fallback-e-disponibilidade-de-contas.md).

## 2026-11-12 — E12: contexto seletivo e renovação de sessão (T12-02)
Contexto seletivo com manifesto de fontes, limites e omissões, sem segredos; renovação por métrica oficial ou estimativa com margem; retomada com pacote e checkpoint reconferidos; troca de modelo só autorizada. Worker, memória e tela NOT_RUN. [Entrega](entregas/2026-11-12-E12-contexto-e-sessao.md) · [ADR-0030](adr/0030-contexto-seletivo-e-renovacao-de-sessao.md).

## 2026-11-12 — E13: compensação de efeitos parciais por risco (T13-05)
Compensação idempotente de baixo risco pelo ledger, sensível só com autorização, irreversível preservado; falha conserva o trabalho e bloqueia operações agravantes. Adaptadores reais NOT_RUN. [Entrega](entregas/2026-11-12-E13-compensacao.md) · [ADR-0029](adr/0029-compensacao-de-efeitos-por-risco.md).

## 2026-11-12 — E07: seleção de fluxo e subfluxo por regras (T07-04)
Regras ordenadas com atributos normalizados, padrão explícito, simulação e rastro persistido com o motivo; corrige o snapshot de publicação sem agentes. [Entrega](entregas/2026-11-12-E07-selecao-por-regras.md) · [ADR-0028](adr/0028-selecao-de-fluxo-por-regras.md).

## 2026-11-12 — E07: publicar e congelar configuração efetiva (T07-03)
Versões publicadas imutáveis com snapshot sanitizado e subfluxos fixados; rascunho não executa; execução presa à versão; revalidação em runtime respeita revogação. [Entrega](entregas/2026-11-12-E07-publicacao-e-congelamento.md) · [ADR-0027](adr/0027-publicacao-e-congelamento-de-workflows.md).

## 2026-11-12 — E13: escopo, expiração e revogação de autorizações (T13-03)
Autorização presa à ação canônica com prazo e usos, consumo atômico (a corrida pelo último uso autoriza um), recusa de divergências e revogação com parada segura e checkpoint. [Entrega](entregas/2026-11-12-E13-autorizacoes.md) · [ADR-0026](adr/0026-autorizacoes-com-escopo-uso-e-revogacao.md).

## 2026-11-12 — E09: orçamento e uso (T09-03)
Reserva atômica de custo (teto global e por agente), liquidação, reconciliação e cota de assinatura registrada separadamente do gasto em API; orçamento imposto antes de cada chamada do gateway. Gateways reais NOT_RUN. [Entrega](entregas/2026-11-12-E09-orcamento-e-uso.md) · [ADR-0025](adr/0025-orcamento-reserva-atomica-e-uso.md).

## 2026-11-12 — E15: GitHub, branches, commits e push (T15-01)
Conexão validada por clone, commit por tarefa validada (fast-import, trailers), push restrito à branch da feature com prova por ref pelo ledger. GitHub real NOT_RUN. [Entrega](entregas/2026-11-12-E15-github-branch-commit-push.md) · [ADR-0024](adr/0024-github-branch-commit-e-push.md).

## 2026-11-12 — E12: checkpoints íntegros (T12-01)
Captura do workspace com hashes, gravação atômica, restauração em container novo com comparação de árvore e parada segura. Git real NOT_RUN. [Entrega](entregas/2026-11-12-E12-checkpoints.md) · [ADR-0023](adr/0023-checkpoints-integros.md).

## 2026-11-12 — E11: execução de ferramentas confinada e com identidade (T11-02)
Ferramentas de arquivo resolvem links simbólicos no sandbox e só seguem dentro do workspace; chamadas brokeradas exigem identidade e registram o ator. Provado nos modos API e CLI com CLI/gateway de mentira; reais NOT_RUN. [Entrega](entregas/2026-11-12-E11-execucao-de-ferramentas.md) · [ADR-0022](adr/0022-execucao-de-ferramentas-confinada-e-com-identidade.md).

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
