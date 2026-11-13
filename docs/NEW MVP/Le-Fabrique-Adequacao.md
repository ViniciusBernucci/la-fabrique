# Le Fabrique — especificação e backlog complementar

**Entrevista de adequação encerrada em 12/11/2026.** Esta especificação registra as 29 decisões e define os nove épicos LF-AJ-01 a LF-AJ-09. O pacote complementa o MVP completo. Os épicos E01 a E17, os tickets e os documentos originais ficam preservados. A entrega é documental.

[Épicos e tickets](#épicos-e-tickets) · [Sequência técnica](#sequência-técnica-e-migração) · [Cenários de teste](#cenários-de-teste-e-evidências) · [Matriz de rastreabilidade](#matriz-de-rastreabilidade)

## Decisão de adequação 29

A Le Fabrique descobre, a partir do repositório do projeto, como preparar o ambiente de testes visuais. Apresenta uma proposta que o administrador pode revisar e editar. A configuração validada é persistida por projeto e reutilizada nas próximas execuções. Ela pode incluir comandos e diretórios de inicialização, serviços dependentes, URLs e verificações de prontidão, dados e credenciais de teste por referência ao cofre, comunicação com o navegador isolado e limpeza. A Decisão 29 integra o [LF-AJ-08](#lf-aj-08), com aceite em AJ08-03, AJ08-04, CT08 e CT09. A entrevista foi encerrada sem questões arquiteturais bloqueantes. [Registro da entrevista](https://chatgpt.com/c/6af502cf-88a4-83e9-ae8c-17f007ed8705)

## Escopo e precedência

`AJ01` a `AJ29` designam decisões da adequação. `Dxx` designa a entrevista original. `AJnn-xx` identifica ticket deste backlog e `CTnn` identifica cenário de aceite. As dependências de tickets são de entrega técnica. Os nove épicos integram o aceite da adequação, em conjunto com a regressão do MVP completo da D42/D95.

| Base do MVP | Comportamento acrescentado pelo fluxo de adequação | Ponto de extensão |
|---|---|---|
| D05 e D50 · subtarefas e projeção de status | Subtarefas técnicas novas vivem na fábrica. Projetar o status consolidado da Issue principal e das Issues externas de correção. Manter vínculos e histórico de subtarefas remotas legadas. | E06/E10 · modo de publicação do plano e adaptador de saída |
| D06, D89 e D92 · elegibilidade e leituras externas | Importar novas Issues em `PRD-DOC` ou `CORRECAO_PR`, atualizar conhecidas em qualquer status, com sync diário ou manual. Consultar em frequências próprias PRs vinculadas e correções aguardando liberação. Consultas de preflight e reconciliação continuam com alvo definido. | E05/E06/E15 · rotinas da Sentinela e adaptadores |
| D55–57 e D52 · concorrência e ordenação | Por projeto, uma vaga Frontend e uma Backend, compartilhadas entre implementação e review da respectiva trilha. Planejamento em controle próprio. Correções elegíveis primeiro, depois prioridade oficial e antiguidade. Mantêm-se os limites da VPS. | E10/E11/E12 · política de admissão e recuperação |
| D63–69 · qualidade | Review visual Frontend obrigatório. Review Backend por IA obrigatório em alto risco. Testes reais para todos. Conservam-se classificação de risco, gates aplicáveis, revisão independente inicial e revalidação de correções. | E08/E13/E14 · política por trilha e ambiente de teste |
| D20, D50–52 e D93 · GitHub | Uma branch e uma PR da feature no ciclo normal. Rodada humana gera uma Issue de correção sujeita à liberação. Merge confirmado encerra o que foi incorporado. O tratamento autorizado de PR fechada sem merge continua pelo fluxo existente. | E06/E13/E15 · agrupamento de revisão, efeitos e reconciliação |
| D58 e D94 · cadastro e workflow | Novo projeto recebe estrutura padrão editável. Agentes, skills, modelos, contas e ferramentas são escolhidos manualmente. Workflow publicado e configuração válida são pré-condições de execução. | E05/E07/E08/E09 · cadastro, catálogos e editor existentes |

Os adaptadores devem reaproveitar o worker, os controles de pausa, o registro de efeitos externos, as políticas de segurança, os catálogos, a memória e o mecanismo de checkpoints do MVP. Aplicar as regras de adequação pelo fluxo publicado correspondente. Projetos cadastrados e execuções em curso são tratados pela migração de AJ03-05, respeitando o workflow fixado e evitando dois coordenadores ativos sobre o mesmo trabalho. [Arquitetura original](le-fabrique-mvp/01-ARQUITETURA.md) · [Épicos originais](le-fabrique-mvp/02-EPICOS.md) · [Matriz D01–D95](le-fabrique-mvp/04-RASTREABILIDADE.md)

## Fluxo, autoridades e estados

- **Origem.** Jira ou ClickUp é a referência externa de requisitos, prioridade e entradas humanas. GitHub é a referência de commits, revisões e merge. O banco da fábrica mantém o estado oficial de execução e o histórico de observações. A Sentinela é a única autoridade interna que valida e efetiva transições de desenvolvimento. O humano pode preparar a entrada da Issue e liberar uma correção pela origem oficial. Adaptadores e agentes entregam observações, resultados e evidências.
- **Fluxo principal.** `PRD-DOC → PRD-DOC-CONCLUIDO → PLANEJAMENTO → IMPLEMENTACAO → REVIEW → DRAFT_PR → CONCLUIDO`.
- **Fluxo de correção.** `REPROVADO_PR → CORRECAO_PR → PLANEJAMENTO → IMPLEMENTACAO → REVIEW → DRAFT_PR → CONCLUIDO`. A primeira transição exige liberação humana. A Issue da feature segue vinculada à mesma PR enquanto cada correção percorre o próprio fluxo. O indicador de rodada pendente pertence ao progresso interno da feature.
- **Unidade interna.** Subtarefa tem classificação `FRONTEND` ou `BACKEND`, tentativa, etapa e evidências próprias. O estado da tarefa técnica não é projetado no Jira/ClickUp. O estado da PR no GitHub, incluindo Draft ou Ready for Review, é registrado separadamente do status canônico da Issue.

| Entrada ou transição | Evidência e regra de avanço |
|---|---|
| Entrada em `PRD-DOC` | Issue elegível da origem oficial. Campos incompletos ficam visíveis em Issues e a Sentinela volta a avaliar quando a documentação mudar. |
| `PRD-DOC → PRD-DOC-CONCLUIDO` | PRD externo consistente e contrato inicial validado, publicado e vinculado à revisão dos requisitos. |
| Criação em `REPROVADO_PR` | Rodada fechada por revisão formal com alterações solicitadas ou comando autorizado, com comentários acionáveis e vínculo à PR original. |
| `REPROVADO_PR → CORRECAO_PR` | Liberação humana na origem, observada e validada pela Sentinela. Um item já aberto na origem em `CORRECAO_PR` entra pela sincronização geral e só é escalonável com vínculo inequívoco à feature e à PR e liberação reconhecida. |
| `PRD-DOC-CONCLUIDO` ou `CORRECAO_PR → PLANEJAMENTO` | Projeto ativo, versão publicada, insumos válidos e vaga de planejamento. Correção exige planejamento focalizado. |
| `PLANEJAMENTO → IMPLEMENTACAO` | Plano técnico válido e subtarefas obrigatórias materializadas internamente, com dependências e contratos de despacho. A admissão de cada subtarefa respeita a fila e suas dependências. |
| `IMPLEMENTACAO → REVIEW` | Todas as implementações obrigatórias concluídas. Reviews por trilha podem começar antes. Na ausência de review Backend obrigatório, a etapa agregada ainda verifica os testes, as evidências e os gates exigidos. |
| `REVIEW → DRAFT_PR` | Todas as trilhas obrigatórias entregues, revisões e checks exigidos aprovados, commits consolidados e verificações integradas válidas para a cabeça que será publicada. GitHub confirma criação ou atualização da PR. |
| `DRAFT_PR → CONCLUIDO` | Merge confirmado na PR vinculada. Concluir a feature e somente correções comprovadamente incorporadas. Registrar prova e sincronizar cada Issue de modo idempotente. |

**Falhas e desvios.** Bloqueio por dependência, pausa, conflito, indisponibilidade, autorização ou recuperação fica no estado de execução já usado pelo MVP, com motivo e evidência. Uma observação remota fora das entradas permitidas é registrada para reconciliação. Falha na escrita de status mantém projeção pendente com retry controlado. Após merge comprovado, a dependência de código pode ser liberada com a evidência de integração. O painel distingue merge confirmado, atualização externa pendente e conclusão sincronizada. PR fechada sem merge mantém Issues pendentes de decisão humana.

**Propostas de engenharia.** Usar a transação e a outbox existentes para salvar transição, evidência e intenção de efeito. Validar versão da entidade e geração da reserva antes de escrever. Usar `/fabrique corrigir` como comando configurável para encerrar a rodada. Adotar a entrada mais antiga na fila vigente para desempate e um ID estável para empates exatos. Horário matinal e fuso do projeto, intervalos de GitHub, correções pendentes e manutenção interna, limites de execução e retentativas são parâmetros operacionais, aferidos no piloto.

## Épicos e tickets

Cada ticket indica o trabalho complementar, sua dependência direta, o aceite verificável e o cenário que comprova o comportamento.

<a id="lf-aj-01"></a>

### LF-AJ-01 — Sincronização e gerenciamento de Issues

**Objetivo.** Manter as Issues da fábrica alinhadas à origem oficial e detectar liberações de correção com baixa latência de consulta.

**Requisitos funcionais e regras.** Cada projeto usa Jira ou ClickUp. O sync geral roda uma vez por dia pela manhã e pode ser acionado no painel. Seleciona Issues novas em `PRD-DOC` ou `CORRECAO_PR` e atualiza todas as já conhecidas em qualquer estado, inclusive fechadas e correções criadas pela fábrica. Guarda identidade externa estável, revisão observada, requisitos, prioridade e status bruto mapeado. A consulta periódica direcionada parte dos IDs locais das Issues `REPROVADO_PR` aguardando liberação, lê seu estado atual na origem e encaminha à Sentinela a observação de `CORRECAO_PR`. A página Issues exibe origem, histórico de sync, falhas e pendências.

**Requisitos não funcionais.** Paginação, rate limits, retentativas limitadas, resultados parciais observáveis e isolamento por projeto. Sync manual, agendado e consulta direcionada podem observar a mesma revisão sem duplicar Issues nem admissões. Upsert por integração, projeto e ID externo estável. Conservar as edições humanas e o último estado conhecido em falha transitória.

**Dependências.** E05/E06 para cadastro e adaptadores, LF-AJ-02 para mapas e LF-AJ-03 para agenda e eventos. Aceite do épico em CT01, CT02 e CT15.

| Ticket | História ou trabalho | Depende de | Critério de aceite e teste |
|---|---|---|---|
| AJ01-01 | Adaptar a leitura Jira e ClickUp para descobrir novas Issues elegíveis e atualizar conhecidas por ID, com deduplicação e vínculo à revisão da origem. | AJ02-01 | Nas duas plataformas, importar os dois estados de entrada, ignorar desconhecidas em outros estados e atualizar uma Issue conhecida fora dos estados de entrada. Repetir a leitura e obter um único registro por Issue. CT01. |
| AJ01-02 | Como administrador, executar o sync matinal ou manual e acompanhar seu resultado na página Issues. | AJ01-01, AJ03-01 | Relógio simulado dispara uma vez por janela matinal. Ação manual usa a mesma rotina. Sobreposição e falha paginada preservam progresso, com último sucesso e erro por escopo. CT01, CT15. |
| AJ01-03 | Consultar por ID apenas as correções conhecidas que aguardam liberação e emitir a observação normalizada. | AJ01-01, AJ03-01 | O lote não consulta Issues fora do conjunto alvo. Detecta a alteração mesmo que o filtro remoto `REPROVADO_PR` já não encontre a Issue. Reentregas e corridas com sync geral não disparam correções duplicadas. CT02. |
| AJ01-04 | Adaptar a projeção Jira/ClickUp e expor pendências de mapeamento, conflito ou escrita de status. | AJ01-01, AJ02-02, AJ03-02 | Timeout após gravação remota é reconciliado por leitura antes de novo envio. Campo humano não relacionado fica preservado. A UI mostra status, origem e projeção pendente até confirmação. CT01, CT15. |

<a id="lf-aj-02"></a>

### LF-AJ-02 — Catálogo de status e transições

**Objetivo.** Usar códigos estáveis e auditáveis e projetar o progresso real da Issue principal.

**Requisitos funcionais e regras.** Persistir IDs internos estáveis para o catálogo da tabela de transições. Mapear IDs e transições permitidas do Jira e ClickUp por projeto. Normalizar `REPROVED_PR` para `REPROVADO_PR` na compatibilidade de leitura e migração. Toda nova persistência, decisão e projeção canônica usa `REPROVADO_PR`. Manter valor bruto histórico para auditoria. Derivar o status agregado pela etapa obrigatória mais atrasada, independentemente do avanço individual das trilhas. Validar a documentação e publicar o contrato inicial para efetivar `PRD-DOC-CONCLUIDO`.

**Requisitos não funcionais.** Invariantes verificadas na transição e no reprocessamento. Mapas inválidos mostram o campo ou a transição que impede a projeção. Eventos antigos não podem desfazer estados mais novos. A auditoria identifica estado anterior e novo, autor da observação, decisão da Sentinela e evidência.

**Dependências.** E03/E06/E10. A tabela de regras é consumida por LF-AJ-03, LF-AJ-04, LF-AJ-06 e LF-AJ-09. Aceite em CT02, CT03, CT06 e CT16.

| Ticket | História ou trabalho | Depende de | Critério de aceite e teste |
|---|---|---|---|
| AJ02-01 | Adicionar catálogo, mapeamento por projeto e migração compatível do alias. | E03, E06 | Catálogo contém todos os códigos da tabela de transições. Alias legado resolve ao mesmo ID, sem linhas ou transições concorrentes. Reaplicar migração mantém IDs, vínculos e histórico bruto. CT02, CT16. |
| AJ02-02 | Formalizar guardas para fluxo normal, correção e estado agregado, com regras testáveis separadas da integração. | AJ02-01 | Fluxos válidos passam com evidência suficiente. PRD incompleto, correção sem liberação e conclusão sem merge são rejeitados. Front revisado com Back ainda implementando mantém `IMPLEMENTACAO`. CT03, CT06, CT13. |
| AJ02-03 | Como administrador, ver na página Issues o estado principal, o avanço interno por trilha e os motivos de bloqueio e de projeção pendente. | AJ02-02, AJ01-04, AJ05-02 | Visão mostra a etapa obrigatória que segura o avanço, commits e evidências vinculados e a diferença entre estado interno e projeção externa. CT06, CT15. |

<a id="lf-aj-03"></a>

### LF-AJ-03 — Sentinela, escalonamento e recuperação

**Objetivo.** Coordenar o fluxo em um único worker lógico e manter execução, fila e recursos consistentes após falhas.

**Requisitos funcionais e regras.** Eventos internos de conclusão, falha, alteração de estado e liberação de vaga provocam nova avaliação imediata. Rotinas internas independentes fazem sync geral, consulta de correções pendentes, consulta de PRs vinculadas e varredura de agentes inativos, timeouts, execuções inconsistentes e recursos órfãos. Todas alimentam o mesmo coordenador de transição e fila. Despacho filtra dependências, publicação do workflow, pausa, orçamento, autorização, capacidade global, reserva de trilha e conflito de recursos. Em cada vaga compatível, ordenar por categoria de correção liberada, prioridade oficial decrescente e antiguidade. A categoria da correção acompanha seu trabalho elegível após a mudança para `PLANEJAMENTO` ou etapas posteriores. Prioridade incide na próxima admissão, sem preempção.

**Recuperação.** Se a sessão permanece viva e a identidade e o isolamento são verificáveis, reassociar. Se caiu, tentar a retomada nativa confiável do CLI. Quando indisponível ou insegura, iniciar sessão vinculada com o último checkpoint válido, plano, contrato, alterações, histórico e mensagens persistidas. Antes de retomar, reconciliar arquivos, refs, commits, estado da PR e efeitos externos incertos. Distinguir silêncio de comando longo, inatividade real e timeout absoluto. Encerrar ou isolar comprovadamente o executor anterior antes de reaproveitar a vaga. Falha persistente preserva evidências e bloqueia o trabalho afetado, liberando trabalhos independentes e seguros.

**Requisitos não funcionais.** Reserva atômica e token de geração impedem dupla execução e escrita tardia. Planejamento tem limite próprio e consome capacidade global. Reservas sobre arquivos ou recursos compartilhados incompatíveis são exclusivas. Auditar ordem de fila, decisão, saúde das rotinas e recuperação. Aplicar pausa segura e drenagem existentes ao conjunto de features em curso no instante da pausa. Uma redução de limite regula novas admissões.

**Dependências.** E10/E11/E12/E13/E16, LF-AJ-02 para guardas, LF-AJ-04 para contratos e LF-AJ-07 para adaptadores de sessão. Aceite em CT02, CT05, CT07, CT12, CT15 e CT16.

| Ticket | História ou trabalho | Depende de | Critério de aceite e teste |
|---|---|---|---|
| AJ03-01 | Conectar eventos duráveis e rotinas agendadas ao worker existente, com relógios e cursores independentes por alvo. | E10, E16 | Conclusão reavalia a fila sem aguardar varredura. Após reinício, um evento perdido é recuperado e rotinas atrasadas retomam por escopo. Falha em GitHub não suprime a leitura matinal nem a manutenção. CT02, CT15. |
| AJ03-02 | Centralizar a transição validada e os efeitos de status na Sentinela. | AJ03-01, AJ02-02 | Escrita direta do agente é recusada. Evento concorrente ou atrasado não cria avanço indevido. A mesma revisão atualiza estado, auditoria e intenção de projeção uma vez. CT02, CT03, CT15. |
| AJ03-03 | Implementar admissão por categoria, prioridade, antiguidade e disponibilidade, com reservas Frontend/Backend e planejamento. | AJ03-02, AJ04-01 | Duas trilhas independentes avançam em paralelo. Review Front ocupa a mesma vaga Front da implementação. Planejamento concorre sob limite próprio. Correção assume a próxima vaga compatível e uma correção bloqueada não impede trabalho independente. CT05, CT12. |
| AJ03-04 | Adicionar retomada nativa, fallback por checkpoint e coleta de órfãos ao supervisor existente. | AJ03-03, AJ07-01, AJ06-01 | Exercitar retomada, nova sessão, processo longo legítimo, lease perdido e escritor atrasado. Preservar patch, arquivos não rastreados autorizados, commits e operações externas já aplicadas. CT07, CT15. |
| AJ03-05 | Preparar migração aditiva, ativação por workflow publicado e reversão operacional segura. | AJ03-03, AJ05-03 | Em cópia com dados legados e tarefa ativa, rodar prévia, migração repetida e ativação em ponto seguro. Preservar histórico, IDs, vínculos e efeitos. Drenar o executor anterior, reconciliar e desabilitar novas admissões no rollback, preservando dados e efeitos já aplicados. CT16. |

<a id="lf-aj-04"></a>

### LF-AJ-04 — Contratos determinísticos de execução

**Objetivo.** Entregar a cada agente um contrato JSON verificável, híbrido e imutável por execução.

**Requisitos funcionais e regras.** O backend preenche IDs de projeto, Issue, subtarefa e execução, fase/status, prioridade oficial, identidade do repositório, branch e SHA base, worktree, versões de workflow/agente/skill, limites e permissões efetivas. O Tech Lead propõe critérios técnicos, classificação de risco, dependências, recursos compartilhados e arquivos relevantes. A Sentinela verifica schema, referências e proveniência, rejeita dependências cíclicas e caminhos fora do escopo, aplica pisos de risco e publica o JSON com ID, versão e hash. O cabeçalho de toda tarefa de agente, CLI ou API, recebe o contrato aplicável, com referência aos requisitos, critérios, plano e checkpoint quando existentes. Resultados, evidências e orientações entram como registros vinculados. Alteração válida dos insumos gera outro contrato para despacho posterior, preservando as versões anteriores e a cadeia de derivação.

**Gate PRD.** A Sentinela confere campos obrigatórios configurados no projeto, conteúdo acessível, consistência de requisitos, critérios de aceite e referências. Se incompleto, expõe pendências e permanece em `PRD-DOC`. Se válido, publica o contrato inicial com os dados disponíveis, sem exigir descobertas que só serão feitas no planejamento. A publicação precede a transição que libera o Tech Lead.

**Requisitos não funcionais.** Schema versionado e compatível com leitura histórica. Referências ao cofre no lugar de segredos. Snapshot imutável com verificação de integridade. Revogação e limites vigentes são revalidados antes dos efeitos. A entrada do agente não tem autoridade para ampliar permissões ou sobrescrever dados oficiais.

**Dependências.** E07/E12/E13, LF-AJ-02 para status e LF-AJ-03 para publicação e transições. Aceite em CT03, CT04 e CT07.

| Ticket | História ou trabalho | Depende de | Critério de aceite e teste |
|---|---|---|---|
| AJ04-01 | Definir schema, proveniência e invariantes e publicar snapshot validado pela Sentinela. | AJ02-01, E12, E13 | Proposta de elevar permissões, mudar IDs ou escapar do worktree não é publicada. Proposta válida tem versão, hash e vínculo de origem. Alteração posterior cria outra versão, mantendo leitura da anterior. CT04. |
| AJ04-02 | Validar a documentação externa e emitir contrato inicial antes de liberar o planejamento. | AJ04-01, AJ01-01, AJ03-02 | Documento incompleto deixa pendências acessíveis. Corrigir o PRD permite publicar exatamente um contrato inicial para a revisão e efetivar `PRD-DOC-CONCLUIDO`. CT03. |
| AJ04-03 | Fornecer contrato imutável no cabeçalho de cada despacho e lidar com revisão dos insumos. | AJ04-01, AJ07-01 | CLI e API recebem o contrato da sua execução. Alterar requisito ou plano em ponto seguro gera despacho com snapshot novo, sem reescrever histórico. Evidências ligam versão, execução e resultado. CT04, CT07. |

<a id="lf-aj-05"></a>

### LF-AJ-05 — Workflow padrão, planejamento técnico e subtarefas

**Objetivo.** Dar a cada projeto novo um workflow pronto para configurar e transformar Issues em trabalho executável pelo Tech Lead.

**Requisitos funcionais e regras.** O cadastro cria um rascunho estrutural com nós, conexões e condições para gate PRD, planejamento, implementação por trilha, review, consolidação, Draft PR, espera por merge e ciclo de correções. O administrador escolhe manualmente agente, skill, modelo, conta e ferramentas por etapa, pode personalizar a estrutura e publica a versão após validar configuração e capacidades. Execuções já iniciadas mantêm a versão publicada que receberam. O editor rejeita a publicação de grafos que permitam contornar guardas obrigatórias da Sentinela. Usar seleção de workflows e subfluxos existente quando aplicável.

O Tech Lead recebe contrato inicial ou contrato de entrada da correção e propõe plano versionado com objetivos, critérios verificáveis, ordem, risco, testes, dependências, escopo de arquivos e recursos compartilhados. A Sentinela valida e materializa subtarefas somente na fábrica. Classificar cada subtarefa como `FRONTEND` ou `BACKEND`, dividir tarefas mistas sempre que possível e explicitar dependências em contratos de API, schema, configurações e arquivos comuns. Para trabalho indivisível, registrar responsabilidade predominante e reservar recursos incompatíveis antes de rodar. Como política de engenharia, se exigir acesso exclusivo às duas trilhas, adquirir ambas as reservas atomicamente. Uma correção usa planejamento focalizado obrigatório, reaproveita contexto ainda válido e gera só as subtarefas, contratos e verificações afetadas.

**Requisitos não funcionais.** Provisionamento e materialização idempotentes. Plano versionado com vínculo a workflow, requisitos, revisão dos comentários quando aplicável, contratos, execuções, commits e evidências. Configuração incompleta bloqueia ativação, com diagnóstico por nó. A publicação de nova versão preserva o histórico de planos e execuções.

**Dependências.** E05/E07/E08/E09/E10, LF-AJ-04 para contrato, LF-AJ-06 para recursos e LF-AJ-09 para rodadas. Aceite em CT04, CT05, CT11 e CT14.

| Ticket | História ou trabalho | Depende de | Critério de aceite e teste |
|---|---|---|---|
| AJ05-01 | Criar o workflow estrutural no cadastro com caminhos de feature e correção. | AJ02-02, E05, E07 | Um novo projeto exibe o grafo editável com os dois caminhos. Repetir o provisionamento preserva o rascunho editado e não cria cópia. CT14. |
| AJ05-02 | Validar proposta Tech Lead e materializar plano e subtarefas apenas internas. | AJ04-01, E10 | Uma Issue mista gera unidades classificadas, com dependências, risco e critérios verificáveis. Ciclo de dependência bloqueia publicação. Captura dos adaptadores confirma ausência de criação remota de subtarefas. CT04, CT05. |
| AJ05-03 | Configurar manualmente os nós e validar publicação, ativação e fixação de versão. | AJ05-01, AJ07-01 | Projeto com bindings incompletos não inicia execuções. Publicação válida libera ativação. Mudança posterior do grafo vale para novos trabalhos. Nó customizado não contorna gate obrigatório. CT14, CT16. |
| AJ05-04 | Implementar plano focalizado a partir da rodada e da cabeça atual da PR original. | AJ05-02, AJ09-02 | Cada subtarefa de correção aponta para os comentários e critérios afetados. Plano retém referências ainda válidas, reavalia o risco e gera novos contratos. Mudança material da cabeça antes do despacho exige reconciliação. CT11, CT12. |

<a id="lf-aj-06"></a>

### LF-AJ-06 — Execução paralela Frontend/Backend e consolidação Git

**Objetivo.** Permitir concorrência segura entre as trilhas e consolidar seus commits em uma única branch de entrega por feature.

**Requisitos funcionais e regras.** Alocar worktree, ambiente e branch temporária próprios por tentativa de implementação ou correção que escreva código. Fixar a SHA de partida e controlar o conjunto de caminhos e recursos do plano. Revisores usam sessão independente inicial e o snapshot candidato correto. Trilhas independentes podem executar simultaneamente no projeto. Um contrato ou serviço exigido pelo outro lado vira dependência verificável. Conflito de acesso a recurso compartilhado impede concorrência, mesmo com duas vagas livres.

A Sentinela autoriza a consolidação serializada por feature. Partir da cabeça atual da branch de entrega, incorporar commits aprovados na ordem das dependências e registrar equivalência de commits quando a integração mudar a SHA. Em divergência, preservar ambos os lados e encaminhar conflito para resolução autorizada. Se a consolidação mudar a evidência de revisão, revalidar o escopo afetado. Rodar testes integrados na cabeça que será enviada ao GitHub. Criar ou atualizar somente a PR vinculada à feature. Se a criação tiver resultado incerto, reconciliar o vínculo pela identidade do repositório e da branch, confirmando a base e a identificação da feature antes de repetir. Remover recursos temporários após comprovar preservação de artefatos e ausência de processo escritor ativo.

**Requisitos não funcionais.** Proteção contra escrita concorrente e atualização de ref condicionada à cabeça esperada. Rastreio de base, commits originais e integrados, artefatos e revisão. Sem publicação de branches temporárias como PR da feature, força de push automática ou descarte silencioso de alterações humanas.

**Dependências.** E11/E12/E13/E15, LF-AJ-03 para reservas, LF-AJ-05 para dependências, LF-AJ-08 para gates e LF-AJ-09 para vínculo da PR. Aceite em CT05, CT06, CT07 e CT12.

| Ticket | História ou trabalho | Depende de | Critério de aceite e teste |
|---|---|---|---|
| AJ06-01 | Provisionar worktree e branch temporária por tentativa com escopo e checkpoint. | AJ04-01, E11, E12 | Front e Back têm diretórios e branches distintos. Alterações não rastreadas e checkpoints permanecem recuperáveis. Limpeza de uma tentativa não remove worktree ou artefatos da outra. CT05, CT07. |
| AJ06-02 | Usar dependências e reservas de recursos compartilhados na admissão das trilhas. | AJ03-03, AJ05-02, AJ06-01 | Tarefas independentes ocupam as duas vagas. Tarefas com arquivo, schema ou recurso incompatível aguardam. Review inicia cedo quando suas dependências estão disponíveis e ocupa a vaga da respectiva trilha. CT05, CT06. |
| AJ06-03 | Consolidar commits e registrar a cabeça candidata de entrega. | AJ06-02, AJ08-01 | Alteração concorrente da branch impede atualização obsoleta. Conflito preserva trabalho e bloqueia push até resolução. Repetição após queda não duplica commits. CT06, CT07. |
| AJ06-04 | Conectar consolidação, checks integrados e adaptador de publicação GitHub. | AJ06-03, AJ08-05 | Testes e gates referenciam a cabeça publicada. Cabeça nova após testes exige revalidação. Feature abre uma Draft PR. Correção atualiza a mesma PR e mantém os vínculos com os commits. CT06, CT12. |

<a id="lf-aj-07"></a>

### LF-AJ-07 — Monitoramento de agentes, terminais e interação humana

**Objetivo.** Acompanhar e orientar execuções CLI e API na mesma interface.

**Requisitos funcionais e regras.** Agentes CLI são preferenciais, com seleção manual conforme capacidades verificadas do cliente. Manter execução via API por adaptadores existentes, sob o mesmo contrato de tarefa, skill, permissões, orçamento e política de fallback. Acompanhar CLI com saída real, comandos quando observáveis, erros, tempo decorrido e progresso. Exibir eventos de respostas e ferramentas para API. Mostrar subagentes quando o executor os identifica. Chat aceita orientação vinculada à execução. Entregar imediatamente quando o executor suporta interação segura, ou enfileirar para o próximo ponto seguro. Exibir estado de envio, fila, entrega e reconhecimento conforme a confirmação efetiva do cliente. Quando não houver confirmação, registrar entrega incerta para reconciliação. Se a execução encerrar antes da entrega, expor esse resultado e preservar o histórico para retomada ou decisão do administrador.

**Requisitos não funcionais.** Stream com sequência, cursor de reconexão e retenção conforme políticas do MVP. Respeitar backpressure e explicitar lacunas. Sanitizar logs, payloads e saída do terminal. Chat opera por capacidade do cliente, com autenticação, escopo, registro de autoria e sem entrada bruta em shell. Mensagens não ampliam permissões, não substituem aprovação e não alteram o contrato publicado.

**Dependências.** E08/E09/E11/E12/E13/E16. Aceite em CT07, CT10 e CT14.

| Ticket | História ou trabalho | Depende de | Critério de aceite e teste |
|---|---|---|---|
| AJ07-01 | Expor capacidades CLI/API no contrato dos adaptadores de execução e recuperação. | E08, E09, E11, E12 | CLI configurável recebe instruções e skill, expõe sessão nativa quando disponível e declara capacidade de interação. API usa eventos estruturados, mantendo limites e o mesmo formato de tarefa. CT07, CT10, CT14. |
| AJ07-02 | Como administrador, acompanhar terminal, eventos, cronômetro, progresso e subagentes observáveis. | AJ07-01 | Saída aparece durante o processo. Reconexão continua pelo cursor sem duplicar eventos. Execução API exibe respostas e ferramentas. Subagente sem telemetria não recebe progresso fabricado. CT10. |
| AJ07-03 | Enviar orientações e acompanhar confirmação técnica, preservando o histórico na retomada. | AJ07-02, AJ04-03 | Testar CLI interativo e não interativo, API, queda entre envio e confirmação e tentativa de contornar permissões. Enfileiramento, entrega e reconhecimento refletem o estado comprovado. Orientação já confirmada não é reenviada. CT07, CT10. |

<a id="lf-aj-08"></a>

### LF-AJ-08 — Revisão automatizada, testes e ambiente visual

**Objetivo.** Comprovar cada entrega com testes reais e revisar a interface em navegador isolado, usando um ambiente descoberto e validado por projeto.

**Requisitos funcionais e regras.**

- **Todas as tarefas.** Criar ou atualizar testes automatizados aplicáveis e executá-los. Guardar comandos, resultados, versão do código, erros e justificativas de aplicabilidade. Falhas e regressões seguem os gates do MVP. Checks aplicáveis continuam obrigatórios para Backend de risco baixo ou médio.
- **Frontend.** Review visual obrigatório com Chrome/Chromium em container isolado por execução, via MCP autorizado. O revisor navega e interage com a aplicação executada para validar os critérios da tarefa. Registrar screenshots, logs, erros e resultados. Falha de prontidão ou do MCP deixa o gate pendente.
- **Backend.** Agente revisor por IA somente em alto risco, com classificação reavaliada sobre o diff real. Aplicar pisos obrigatórios e checks de segurança existentes. Em ambas as trilhas, revisar de forma independente na avaliação inicial. Achados acionáveis seguem para correção pelo revisor em etapa própria, com reexecução dos testes e da revisão afetada. Respeitar limites de rodadas e escalonamento existentes.
- **Convergência.** Reviews independentes podem rodar cedo quando seus pré-requisitos estiverem prontos. Testes integrados aguardam as trilhas obrigatórias e são executados na cabeça consolidada. Evidência vinculada a versão anterior fica inválida para o que mudou.

**Ambiente visual e Decisão 29.** O descobridor lê evidências do repositório na revisão examinada, como README, manifestos de pacote, scripts declarados, configurações de CI, arquivos de container e exemplos de variáveis. Registra origem dos campos, lacunas e ambiguidades. A proposta descreve diretórios e comandos, serviços e dependências, portas e URLs privadas, prontidão, preparo e limpeza de dados, credenciais por referência e acesso pelo navegador MCP. O administrador revisa e edita. A validação confere schema, dependências e política e roda em ambiente isolado para comprovar inicialização, prontidão, acesso pelo navegador e limpeza. A configuração validada e aprovada fica vinculada ao projeto e disponível para as próximas execuções.

| Estado da configuração | Regra |
|---|---|
| Proposta ou edição | Conteúdo descoberto ou editado, com indicação de origem e lacunas. A descoberta é leitura estática. Comandos só executam na validação isolada após revisão. |
| Em validação | A tentativa fixa a versão candidata e o SHA analisado. Guarda evidência de prontidão, conectividade MCP e limpeza. |
| Validada | A Sentinela registra versão e hash e aponta o projeto para a configuração aprovada. Execuções fixam essa versão e fazem preflight na revisão que será testada. |
| Inválida ou requer revalidação | Falha ou mudança relevante em comandos, serviços ou permissões gera diagnóstico e proposta revisável. O painel distingue a última versão validada e a candidata. A versão anterior só pode ser reutilizada quando aplicável à execução e confirmada pelo preflight. |

**Requisitos não funcionais.** Containers efêmeros, rede privada por execução e credenciais de teste de escopo mínimo. Bloquear acesso a produção, host, plano de controle, outras execuções e perfis compartilhados. Tratar comandos vindos do repositório como dados até validação. Restringir destinos de rede e volumes, segregar fixtures e impor quotas e timeouts. A Sentinela coleta recursos de sucesso, falha, cancelamento e queda com identificação por execução e limpeza repetível. Sanitizar evidências e guardar o suficiente para reproduzir a verificação.

**Dependências.** E04/E08/E11/E13/E14, LF-AJ-03 para supervisão, LF-AJ-06 para worktree e LF-AJ-07 para execução. Aceite em CT06, CT08, CT09 e CT17.

| Ticket | História ou trabalho | Depende de | Critério de aceite e teste |
|---|---|---|---|
| AJ08-01 | Aplicar política de testes, risco e review por trilha ao pipeline existente. | AJ03-03, AJ05-02, E14 | Front de baixo risco exige navegador. Back baixo/médio passa pelos checks sem review de IA, e Back alto exige review independente. Todos executam testes reais. Correção reaplica gates. CT17. |
| AJ08-02 | Provisionar Chrome/Chromium isolado e ferramentas MCP de navegação, evidência e limpeza. | AJ06-01, AJ03-01, E08, E11, E13 | MCP acessa a aplicação de teste autorizada. Isolamento impede acesso cruzado e a produção. Falha do navegador impede aprovação visual. Sucesso, cancelamento e queda recolhem os recursos identificados. CT08, CT09. |
| AJ08-03 | Descobrir o ambiente de teste pelo repositório e apresentar proposta editável. | AJ06-01, E04 | Fixture com frontend, API e banco gera proposta com fontes, serviços, readiness e dados. Fixture ambígua mostra lacunas. Descoberta não executa scripts nem inclui segredos nos campos. CT08. |
| AJ08-04 | Validar, versionar, persistir e reutilizar a configuração de ambiente por projeto. | AJ08-02, AJ08-03 | Administrador edita, valida e aprova. A versão reaparece após reinício e é usada por outro review. Mudança relevante exige diagnóstico e revalidação. Segredos ficam referenciados, e falha de readiness impede promover a versão. CT08, CT09. |
| AJ08-05 | Integrar evidência visual, resultados por trilha e gate de testes integrados antes da PR. | AJ08-01, AJ08-04, AJ06-03 | Head consolidado tem verificações aplicáveis rastreáveis. Um Front pode concluir review cedo, mas faltas no Back, nos checks integrados ou no review visual impedem publicação. CT06, CT17. |

<a id="lf-aj-09"></a>

### LF-AJ-09 — Draft PR, revisão humana e ciclo de correções

**Objetivo.** Transformar rodadas de revisão no GitHub em correções controladas e encerrar Issues após integração efetiva.

**Requisitos funcionais e regras.** A Sentinela consulta somente PRs gerenciadas, em frequência configurável própria e sob demanda no painel. Detecta alterações de estado da PR, merge e encerramento de rodada por revisão formal com solicitação de mudanças ou comando explícito de revisor autorizado. Agrupa comentários acionáveis da rodada em uma Issue externa `REPROVADO_PR`, com contexto, PR, Issue original, revisão, IDs de comentários e seus links. Preservar a identidade da rodada na repetição. Um comentário solto sem fechamento permanece disponível para consolidação. A criação do ticket precede a liberação humana em Jira/ClickUp, que dispara planejamento focalizado e prioridade na próxima vaga compatível. A correção usa worktrees temporários, revisão e testes aplicáveis e commits adicionais na branch e PR originais.

**Proposta de engenharia para rodadas.** Registrar o ID estável da PR, o evento de fechamento, a cabeça observada e o conjunto de comentários associados. Vincular comentários de linha à revisão formal e incluir avulsos acionáveis explicitamente abrangidos pelo comando ou pela janela da rodada ainda aberta. Usar ordenação estável de eventos e registrar atribuição dos comentários já consolidados. Se revisão formal e comando encerrarem o mesmo conjunto, tratar como a mesma rodada. Comentários posteriores pertencem à próxima rodada. Em agrupamento ambíguo, expor a pendência para revisão sem disparar correção. Comando e texto dos comentários são insumos, com autoria e permissões verificadas.

**Conclusão.** No merge, conferir repositório, PR, base/cabeça e prova de incorporação. Guardar SHA do merge, data, cabeça integrada e vínculo com cada correção. Para squash ou rebase, comprovar a inclusão pela cabeça e pelo histórico de integração registrado, sem depender de identidade literal do SHA do commit original. A Sentinela conclui a Issue principal e as correções incorporadas. Pendentes, rejeitadas, não enviadas ou comprovadamente excluídas aguardam tratamento. Aprovação da PR e saída de Draft conservam a espera pelo merge. PR fechada sem merge segue o fluxo de decisão humana da D93. A promoção de memória e a liberação de dependentes respeitam o gate original de integração.

**Requisitos não funcionais.** Leituras paginadas com cursores por PR e respeito a rate limits. Efeitos idempotentes por rodada e merge. Em timeout após criação externa, reconciliar vínculo e chave da rodada antes de repetir. Edição, exclusão ou reentrega de comentário após fechamento conserva o snapshot auditável. Revalidar o estado da PR antes de despachar ou publicar correção. Merge concorrente cancela o avanço de publicação da rodada ainda não incorporada e preserva seu trabalho para tratamento humano.

**Dependências.** E06/E13/E15, LF-AJ-01 e LF-AJ-03 para liberação e fila, LF-AJ-05/LF-AJ-06/LF-AJ-08 para a execução e os gates. Aceite em CT02, CT11, CT12, CT13 e CT15.

| Ticket | História ou trabalho | Depende de | Critério de aceite e teste |
|---|---|---|---|
| AJ09-01 | Consultar PRs vinculadas com agenda própria e ação manual. | AJ03-01, E15 | Intervalo GitHub pode ser alterado sem afetar Jira/ClickUp. Detectar revisão, fechamento e merge no escopo da fábrica, inclusive após reinício. CT11, CT13, CT15. |
| AJ09-02 | Agrupar comentários da rodada e criar a Issue `REPROVADO_PR` na origem oficial. | AJ09-01, AJ01-04 | Revisão formal ou comando autorizado fecha rodada com achados. O mesmo conjunto gera uma Issue, inclusive após timeout. Comentário solto, comando não autorizado e repetição não criam ticket. Segunda rodada cria nova Issue. CT11, CT15. |
| AJ09-03 | Despachar correção liberada com prioridade, executar plano focalizado e atualizar a PR original. | AJ01-03, AJ05-04, AJ06-04 | Correção aguarda autorização humana e vaga compatível, preserva trabalho em andamento e publica commits na mesma PR após gates. Evidência liga rodada, Issue, contrato, commits e cabeça publicada. CT02, CT12. |
| AJ09-04 | Encerrar somente Issues incorporadas após merge confirmado e tratar fechamento sem merge. | AJ09-01, AJ09-03, AJ03-02 | Aprovação ou Ready for Review não conclui. Merge conclui feature e correções incorporadas, inclusive squash/rebase verificável. Uma correção pendente fica aberta. Retry da projeção não duplica efeitos. CT13, CT15. |

## Sequência técnica e migração

1. **Contratos e coordenação.** Estabelecer catálogo e guardas (AJ02-01/02), agendamento e transição (AJ03-01/02), schema e proveniência (AJ04-01) e capacidades do executor (AJ07-01). O registro de eventos, a outbox e o controle de versão existentes sustentam a adoção.
2. **Entrada e planejamento.** Integrar Jira/ClickUp (AJ01-01 a AJ01-04), gate PRD (AJ04-02/03), cadastro estrutural e Tech Lead (AJ05-01 a AJ05-03). Acrescentar reservas e prioridade (AJ03-03) e projeção do progresso (AJ02-03).
3. **Trilhas e evidências.** Acrescentar worktrees e reservas compartilhadas (AJ06-01/02), testes e reviews (AJ08-01 a AJ08-04), recuperação e chat (AJ03-04 e AJ07-02/03). Depois consolidar commits e ligar os checks integrados à publicação (AJ06-03, AJ08-05, AJ06-04).
4. **Ciclo GitHub.** Integrar consulta, rodadas e tickets (AJ09-01/02), plano focalizado (AJ05-04), atualização da PR e merge (AJ09-03/04).
5. **Ativação e aceite integral.** Validar migração e retomada, ativar o fluxo em pontos seguros (AJ03-05) e executar os cenários complementares junto com o roteiro de regressão original. [Aceite do MVP](le-fabrique-mvp/05-ACEITE-E-PILOTO.md)

As alterações de persistência previstas são aditivas. Reutilizar entidades existentes e acrescentar, onde faltarem, mapeamento canônico por projeto, registros de reserva e geração, snapshots de contrato, configuração visual versionada, atribuição de comentários a rodadas e prova de incorporação de correções. Preservar IDs externos, versões publicadas, histórico, checkpoints e vínculos Git. Fazer prévia e backup coordenado de banco e volumes, validar mapas antes de ativar e reconciliar as Issues e PRs já conhecidas. Na troca de fluxo, concluir ou suspender com checkpoint as execuções ativas, sem transferir silenciosamente uma execução para outro workflow. A reversão operacional drena o fluxo novo, preserva registros e reconcilia efeitos externos antes de reativar qualquer outro despachante.

## Cenários de teste e evidências

Para o aceite, registrar projeto, workflow publicado, Issue/rodada, execução/tentativa, versão do contrato, SHAs e artefatos aplicáveis. Usar relógio controlável nos casos de agendamento, injeção de falhas nos adaptadores e execução real dos testes na trilha ponta a ponta. O piloto deve percorrer cadastro, PRD, feature com duas trilhas, navegador isolado, PR, correção humana e merge. Rodar testes de contrato e integração de Jira e ClickUp e validar a origem escolhida no projeto piloto. Os demais casos reproduzíveis podem usar ambiente controlado.

| Cenário | Preparação e ação | Resultado verificável |
|---|---|---|
| CT01 · Sync geral | Em Jira e ClickUp, preparar Issues novas em `PRD-DOC`, `CORRECAO_PR` e outro status, mais uma Issue conhecida alterada fora dos estados de entrada. Disparar horário matinal e botão manual com sobreposição. | Somente entradas elegíveis são descobertas. Issue conhecida é atualizada. IDs e revisões são estáveis, sync manual e matinal aparecem no painel, e repetição não duplica registros. |
| CT02 · Liberação direcionada | Criar correção em `REPROVADO_PR`, mantê-la aguardando, depois mover para `CORRECAO_PR` na origem como humano. Repetir polling e entregar observações concorrentes da mesma revisão. | Apenas IDs pendentes são consultados no ciclo direcionado. Sem alteração humana, nenhum despacho. Liberação válida gera uma admissão focalizada e o item sai do conjunto de espera. Alias legado converge para o estado canônico. |
| CT03 · Gate do PRD | Importar Issue sem critério obrigatório. Forjar um evento de conclusão de agente. Depois completar o PRD e sincronizar duas vezes. | Pendências aparecem e não há despacho de Tech Lead antes da validação. Uma versão inicial publicada precede a transição `PRD-DOC-CONCLUIDO`. Agente não altera status por conta própria. |
| CT04 · Contrato e plano | Tech Lead propõe critérios, caminhos, riscos e dependências. Incluir um ciclo, um caminho proibido e alteração de ID. Corrigir a proposta e depois alterar um insumo relevante. | Propostas inválidas são recusadas com motivo. Plano válido gera subtarefas locais, contrato com versão e hash e cabeçalho de despacho. Nova revisão preserva o JSON histórico. Sem criação remota de subtarefas. |
| CT05 · Vagas e recursos | Enfileirar Front A, Front B, Back A e planejamento. Simular dependência de API e disputa por arquivo compartilhado, depois removê-las. | Sem dependências ou conflitos, Front e Back executam em worktrees e branches distintos. Segunda tarefa Front aguarda. Com conflito real, uma espera. Planejamento respeita seu limite e a capacidade global. |
| CT06 · Reviews antecipados e convergência | Finalizar Front enquanto Back ainda implementa. Rodar review visual. Finalizar Back e consolidar. Alterar a cabeça após um check. | Front pode concluir review e o agregado permanece em `IMPLEMENTACAO`. Com ambas as implementações prontas, agregado muda a `REVIEW`. PR aguarda evidência da cabeça válida e testes integrados. Uma única Draft PR é confirmada. |
| CT07 · Recuperação | Derrubar worker com sessão reaproveitável e repetir com sessão perdida. Simular operação externa sem resposta e comando longo sem stdout. | Retomar sessão original quando suportada ou nova sessão com checkpoint. Nenhuma perda de código ou repetição de efeito. Escritor antigo não readquire autoridade. Comando saudável não é encerrado por falta de stdout. |
| CT08 · Decisão 29 e navegador | Usar repositório com frontend, API, banco e fixtures. Descobrir, editar, validar e aprovar configuração. Reiniciar e executar outro review. Testar também um repositório ambíguo. | Proposta indica fontes e lacunas, validação comprova serviços prontos e acesso MCP e a versão aprovada é reutilizada. Review real gera evidência visual. |
| CT09 · Isolamento e revalidação | Tentar usar destino de produção, perfil de outro projeto e dados fora do escopo. Alterar serviço obrigatório e simular falha de readiness, cancelamento e queda. | Tentativas indevidas são bloqueadas, segredos ficam fora dos artefatos e a mudança exige validação aplicável. A Sentinela recolhe containers, processos, redes e volumes temporários da execução correta. |
| CT10 · CLI, API e chat | Acompanhar CLI interativo, CLI sem suporte de entrada imediata e agente API. Reconectar durante a execução, enviar orientação e provocar queda após envio. | Terminal real e eventos API aparecem na mesma interface. Sequência retoma pelo cursor. Chat entrega ou enfileira conforme capacidade, conserva confirmação comprovada, sinaliza entrega incerta e preserva histórico. |
| CT11 · Rodadas de PR | Publicar comentários avulsos, revisão formal com alterações e comando autorizado sobre a mesma rodada. Repetir eventos, testar comando de autor sem autorização e encerrar rodada posterior. | Uma Issue `REPROVADO_PR` por rodada acionável. Agrupamento guarda comentários e links. Comentários soltos e comando sem autorização não criam Issue. Comando válido não autoriza execução da correção. |
| CT12 · Correção prioritária | Ocupar a vaga Front com feature normal. Liberar correções Front e Back, com prioridades e tempos distintos. Manter uma correção bloqueada por dependência. | Próxima vaga compatível atende primeiro a correção elegível. Empate usa prioridade oficial e antiguidade. Execução ativa segue. Tech Lead planeja apenas escopo afetado e a PR original recebe os commits validados. |
| CT13 · Merge e exclusões | Aprovar PR, mudar Draft para Ready e fechar sem merge. Em outro ciclo, fazer merge humano com uma correção incorporada e outra pendente. Exercitar merge comum e squash ou rebase. | Aprovação, Ready e fechamento sem merge não concluem. Após merge verificado, feature e correção incorporada vão a `CONCLUIDO`, com prova e atualização idempotente. A correção pendente fica aberta. |
| CT14 · Novo projeto | Cadastrar projeto, abrir editor, tentar ativar com nós incompletos, escolher recursos manualmente e publicar. Editar o workflow durante uma execução. | Grafo padrão inclui PRD e correções. Pré-requisitos ausentes impedem ativação. Após publicar, execução usa configuração válida e preserva versão fixada até o fim. |
| CT15 · Falhas de integração e serviço | Provocar limite de API, paginação parcial, timeout depois de efeito remoto, eventos fora de ordem e reinício entre persistência e envio. Testar falhas em cadências diferentes. | Rotinas independentes continuam. Backoff e cursor são observáveis. Retomada reconcilia efeitos e mantém último estado válido. Auditoria identifica transição, evidência, atraso de projeção e resultado final. |
| CT16 · Migração e regressão | Preparar projeto legado com alias, subtarefa externa, workflow publicado, PR vinculada e tentativa ativa. Aplicar prévia, migração, repetição e reversão operacional. | IDs, histórico e vínculos sobrevivem. Workflow fixado não muda. Nenhum trabalho adquire duas reservas nem duplica efeitos. A suíte original de MVP passa com as expectativas de delta aplicáveis ao fluxo novo. |
| CT17 · Política de qualidade | Executar Front de baixo risco, Back baixo/médio e Back alto após reclassificação pelo diff. Induzir regressão e correção pelo revisor. | Front sempre usa navegador MCP. Back alto exige review IA. Todas as tarefas têm testes reais e evidências. Correção repete verificações afetadas e reprovação impede a publicação. |

## Matriz de rastreabilidade

As linhas sintetizam as respostas da entrevista de adequação. A coluna de backlog aponta para os tickets complementares que materializam a decisão e a coluna de teste indica onde comprovar o comportamento. [Entrevista de adequação](https://chatgpt.com/c/6af502cf-88a4-83e9-ae8c-17f007ed8705)

| Decisão | Regra registrada | Épicos e tickets principais | Teste |
|---|---|---|---|
| AJ01 | Importar Issues novas `PRD-DOC` e `CORRECAO_PR`. Atualizar conhecidas em qualquer status. Sincronizar pela manhã ou manualmente. | [LF-AJ-01](#lf-aj-01) · AJ01-01, AJ01-02 | CT01 |
| AJ02 | Adotar `REPROVADO_PR` como código canônico e normalizar o alias legado com preservação do histórico. | [LF-AJ-02](#lf-aj-02) · AJ02-01 | CT02, CT16 |
| AJ03 | Uma execução Frontend e uma Backend simultâneas por projeto, sujeitas aos limites globais. | [LF-AJ-03](#lf-aj-03), [LF-AJ-06](#lf-aj-06) · AJ03-03, AJ06-02 | CT05 |
| AJ04 | Isolar worktrees e branches temporárias e integrar commits na branch única da feature. | [LF-AJ-06](#lf-aj-06) · AJ06-01, AJ06-03, AJ06-04 | CT05, CT06 |
| AJ05 | Review por trilha quando as dependências permitirem. Testes integrados e Draft PR aguardam as trilhas obrigatórias. | [LF-AJ-06](#lf-aj-06), [LF-AJ-08](#lf-aj-08) · AJ06-02, AJ08-05 | CT06 |
| AJ06 | Tentar retomar sessão original de CLI. Se inviável, iniciar outra com checkpoint e reconciliação. | [LF-AJ-03](#lf-aj-03), [LF-AJ-07](#lf-aj-07) · AJ03-04, AJ07-01 | CT07 |
| AJ07 | Terminal CLI em tempo real e chat com entrega imediata segura ou enfileirada. | [LF-AJ-07](#lf-aj-07) · AJ07-02, AJ07-03 | CT10 |
| AJ08 | Implementação e review compartilham a vaga por especialidade. Planejamento tem controle separado. | [LF-AJ-03](#lf-aj-03), [LF-AJ-06](#lf-aj-06), [LF-AJ-08](#lf-aj-08) · AJ03-03, AJ06-02, AJ08-01 | CT05, CT06 |
| AJ09 | Sentinela valida evidências, efetiva transições e sincroniza status. | [LF-AJ-02](#lf-aj-02), [LF-AJ-03](#lf-aj-03) · AJ02-02, AJ03-02 | CT03, CT15 |
| AJ10 | Contrato JSON híbrido com dados determinísticos e propostas técnicas, validado pela Sentinela e imutável por execução. | [LF-AJ-04](#lf-aj-04) · AJ04-01, AJ04-03 | CT04 |
| AJ11 | `PRD-DOC-CONCLUIDO` comprova PRD externo validado e contrato inicial publicado. Libera planejamento. | [LF-AJ-02](#lf-aj-02), [LF-AJ-04](#lf-aj-04) · AJ02-02, AJ04-02 | CT03 |
| AJ12 | Tech Lead cria subtarefas técnicas exclusivamente na fábrica. Jira/ClickUp acompanha a Issue principal. | [LF-AJ-05](#lf-aj-05), [LF-AJ-02](#lf-aj-02) · AJ05-02, AJ02-03 | CT04, CT06 |
| AJ13 | Frontend sempre tem review visual. Backend só tem review de IA em alto risco. Testes reais em todas as tarefas. | [LF-AJ-08](#lf-aj-08), [LF-AJ-03](#lf-aj-03) · AJ08-01, AJ03-02 | CT17 |
| AJ14 | Chrome/Chromium isolado por execução via MCP, com evidências e limpeza pela Sentinela. | [LF-AJ-08](#lf-aj-08) · AJ08-02 | CT08, CT09 |
| AJ15 | Uma Issue de correção `REPROVADO_PR` por rodada com comentários acionáveis agrupados. Subtarefas são internas. | [LF-AJ-09](#lf-aj-09) · AJ09-02, AJ09-03 | CT11, CT12 |
| AJ16 | Encerrar rodada com review formal solicitando alterações ou sinalização explícita autorizada no GitHub. | [LF-AJ-09](#lf-aj-09) · AJ09-02 | CT11 |
| AJ17 | Correção reutiliza contexto válido e produz commits em worktrees temporários, enviados à branch e PR originais. | [LF-AJ-06](#lf-aj-06), [LF-AJ-09](#lf-aj-09) · AJ06-01, AJ06-04, AJ09-03 | CT12 |
| AJ18 | Concluir a Issue original e as correções incorporadas após confirmar merge da PR. | [LF-AJ-09](#lf-aj-09) · AJ09-04 | CT13 |
| AJ19 | Uma Sentinela híbrida reage a eventos internos e faz checagens periódicas de segurança e recuperação. | [LF-AJ-03](#lf-aj-03) · AJ03-01, AJ03-04 | CT07, CT15 |
| AJ20 | Monitorar as PRs gerenciadas no GitHub em intervalo próprio configurável, com consulta manual. | [LF-AJ-09](#lf-aj-09) · AJ09-01 | CT11, CT13, CT15 |
| AJ21 | Toda correção liberada passa pelo Tech Lead em planejamento focalizado, com reaproveitamento seguro. | [LF-AJ-05](#lf-aj-05), [LF-AJ-09](#lf-aj-09) · AJ05-04, AJ09-03 | CT12 |
| AJ22 | Correção liberada tem prioridade na próxima vaga compatível, sem interromper trabalho ativo. | [LF-AJ-03](#lf-aj-03), [LF-AJ-09](#lf-aj-09) · AJ03-03, AJ09-03 | CT12 |
| AJ23 | Ordenar candidatas elegíveis por categoria, prioridade oficial e antiguidade. | [LF-AJ-03](#lf-aj-03) · AJ03-03 | CT12 |
| AJ24 | Tech Lead divide trabalho misto quando possível e explicita dependências e recursos compartilhados. | [LF-AJ-03](#lf-aj-03), [LF-AJ-05](#lf-aj-05), [LF-AJ-06](#lf-aj-06) · AJ03-03, AJ05-02, AJ06-02 | CT04, CT05 |
| AJ25 | Status principal segue a etapa obrigatória mais atrasada. Todas as implementações obrigatórias precedem `REVIEW` agregado. | [LF-AJ-02](#lf-aj-02), [LF-AJ-03](#lf-aj-03), [LF-AJ-06](#lf-aj-06) · AJ02-02, AJ02-03, AJ03-02, AJ06-02 | CT06 |
| AJ26 | Criar estrutura padrão do workflow no cadastro e configurar agentes, skills, modelos, contas e ferramentas manualmente antes de ativar. | [LF-AJ-05](#lf-aj-05) · AJ05-01, AJ05-03 | CT14 |
| AJ27 | CLI preferencial, modelos via API por adaptadores e acompanhamento unificado. | [LF-AJ-07](#lf-aj-07) · AJ07-01, AJ07-02, AJ07-03 | CT10, CT14 |
| AJ28 | Manter sync geral matinal ou manual e consultar periodicamente apenas Issues `REPROVADO_PR` aguardando liberação humana. | [LF-AJ-01](#lf-aj-01), [LF-AJ-03](#lf-aj-03), [LF-AJ-09](#lf-aj-09) · AJ01-03, AJ03-01, AJ09-03 | CT02, CT15 |
| AJ29 | Descobrir configuração de ambiente visual pelo repositório, permitir revisão e edição e persistir a configuração validada. | [LF-AJ-08](#lf-aj-08) · AJ08-03, AJ08-04 | CT08, CT09 |
