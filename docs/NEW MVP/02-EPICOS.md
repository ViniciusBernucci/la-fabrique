# Épicos do MVP

Todos os épicos são obrigatórios para o lançamento. A sequência abaixo expressa dependência técnica. O [backlog](03-BACKLOG.md) detalha a ordem por ticket e permite antecipar contratos que serão integrados posteriormente. A [arquitetura](01-ARQUITETURA.md) define os contratos compartilhados. A [matriz](04-RASTREABILIDADE.md) liga decisões aos épicos e o [piloto](05-ACEITE-E-PILOTO.md) define evidência de aceite.

| Ordem | Trabalho | Resultado esperado |
|---|---|---|
| 1 · Fundação | E01–E04 | Repositório e ambientes, base de CI, dados, serviços e acesso seguro. |
| 2 · Configuração | E05–E09 | Projeto, conectores, editor, catálogos e adaptadores configuráveis. |
| 3 · Execução | E10–E13 | Planejamento, fila, sandbox, retomada, autorizações e reconciliação. |
| 4 · Entrega e operação | E14–E16 | Qualidade, entrega Git, documentação, memória e operação auditável. |
| 5 · Aceite | E17 | Cobertura completa e validação em projeto real, com evidências. |

## E01

**Bootstrap, monorepo e GitHub · Fundação**

**Objetivo e contexto.** Dar ao responsável um ponto de partida reproduzível para implementar a fábrica. A D40 delega a escolha técnica e a D42 exige que a entrega integre todos os épicos.

**Escopo e requisitos.**

- Registrar ADR da stack inicial React/TypeScript/Vite/React Flow e Go, da escolha PostgreSQL e do monorepo. Explicar fronteiras de web, API, worker, sandbox e repositórios de software atendidos. Vincular benchmark E02.
- Criar o repositório privado da fábrica quando o desenvolvimento começar. Definir proprietário, acesso de manutenção, `main`, proteção de branch, checks de PR, gestão de releases, identidade de deploy e procedimento de revisão compatível com um administrador.
- Criar estrutura de diretórios, toolchains fixadas, arquivos de exemplo sem segredos, comandos de desenvolvimento e convenções de migrations, testes, documentos e revisão.
- Criar README, guias de uso para agentes, índice e ADRs, com a especificação da entrevista como referência e controle de mudanças de escopo.

**Regras.** Repositórios dos projetos atendidos são vinculados pelo onboarding. Estrutura e ordem técnica seguem D42. O responsável administra o GitHub e o desenvolvimento.

**Dependências e entregáveis.** Início sem dependência técnica. Entrega repositório, estrutura, convenções, ADRs e baseline de CI em conjunto com E02.

**Aceite e testes.** A partir de clone limpo, reproduzir preparação e smoke dos componentes com as ferramentas fixadas. Verificar proteção/checks configurados, escopo das identidades e ausência de credenciais commitadas. Conferir links dos guias e a presença de todo o backlog no plano do primeiro lançamento.

## E02

**Infraestrutura VPS, ambientes, containers, CI e restauração · Fundação**

**Objetivo e contexto.** Operar continuamente na VPS existente, com capacidade mensurável, configuração repetível e recuperação comprovada (D02, D34, D38–40).

**Escopo e requisitos.**

- Inventariar CPU, RAM, disco, I/O, SO, arquitetura, cargas vizinhas, DNS, acessos e backup do provedor. Medir builds, checks, navegador, worker e recuperação com cargas representativas.
- Versionar imagens e configuração de dev, homologação e produção. Separar volumes, redes e segredos. Provisionar proxy/TLS, supervisão após reboot, serviços privados, limites e sondas de saúde.
- CI de PR executa formatação/lint, testes unitários e contratos, integração com banco, build das imagens, verificações de segredos e smoke de composição. Publicar artefatos imutáveis de referência e definir promoção/rollback por revisão.
- Usar backups e snapshots nativos da VPS. Incluir banco, volumes, skills, checkpoints, configuração e caminho de recuperação das chaves. Documentar consistência, recuperação e limites do provedor.

**Regras.** Limite global inicial vem da capacidade medida. Homologação e produção têm identidades e dados distintos. O produto usa backup do provedor e runbook. Alteração estrutural e migração acompanham instrução de restauração.

**Dependências e entregáveis.** E01. Entregar ADR de capacidade, composição, CI, procedimento de promoção, diagnóstico, recuperação e evidência de restore.

**Aceite e testes.** PR com falha bloqueia integração. Em instalação limpa, a revisão sobe saudável com portas internas fechadas. Reboot recupera serviços. Restaurar snapshot em destino isolado, conferir dados, hashes, credenciais e checkpoints. Sob carga representativa, painel e banco preservam capacidade operacional.

## E03

**Fundação da aplicação web, API, worker e banco · Fundação**

**Objetivo e contexto.** Sustentar configuração e execução com estado durável, contratos claros e navegação centrada em projetos (D02, D27–28, D40).

**Escopo e requisitos.**

- Inicializar web React, API e worker Go, PostgreSQL, migrations e persistência de artefatos por referência. Definir transações, índices, restrições e versões das entidades.
- Expor contratos internos versionados, validação de payload, filtros/paginação e erros correlacionáveis. Diferenciar usuário administrativo e identidades internas. Criar outbox/eventos e protocolo de armazenamento.
- Montar navegação inicial com projetos, biblioteca global, Central de Decisões e detalhes de projeto. Separar shell e permissões das funcionalidades entregues pelos demais épicos.

**Regras.** Estados de execução são locais. Banco e armazenamento permanecem inacessíveis ao código executado. Integridade referencial impede vínculo entre projetos errados. Migrations executam com identidade própria.

**Dependências e entregáveis.** E01 e contrato de ambiente E02. Entregar aplicações, schema inicial e migrations, contratos, fixtures e testes de persistência.

**Aceite e testes.** Subir do zero e migrar schema anterior com fixture. Persistir configuração e eventos após reinício. Rejeitar versões concorrentes e acessos fora do escopo. Smoke de autenticação, navegação, página vazia e estado de erro após E04.

## E04

**Autenticação administrativa e gestão de credenciais · Fundação**

**Objetivo e contexto.** Restringir o painel a um administrador e permitir conexões seguras para desenvolvimento contínuo (D01, D36–37).

**Escopo e requisitos.**

- Bootstrap de uma conta, senha com hash adequado, TOTP obrigatório, códigos de recuperação de uso único e substituição segura do segundo fator. Limitação de tentativas e proteção de sessões.
- Encerrar e revogar sessões. Exigir reautenticação para gestão sensível e aplicar HTTPS, proteção contra requisições forjadas e cookies seguros.
- Cofre com criptografia e chave separada do banco. Interface para criar, testar, substituir e revogar conexões GitHub, Jira, ClickUp, CLI, API e MCP. Usar mecanismos oficiais de CLI e escopos mínimos por destino.
- Controlar injeção por referência e redigir logs, prompts, documentação, relatórios e respostas da API.

**Regras.** Sem TOTP ou recuperação válida, sessão administrativa não abre. Códigos usados são invalidados atomicamente. Revogação é efetiva em novos acessos, inclusive execução com configuração anterior.

**Dependências e entregáveis.** E02 e E03. Entregar autenticação, cofre, UI de conexões, auditoria, runbook de recuperação e testes de segurança.

**Aceite e testes.** Validar login, código errado e expirado, recovery concorrente, revogação e reautenticação. Testar rotação com referência persistente e revogação em execução. Demonstrar que logs, banco restaurado sem chave e sandbox não fornecem segredo em claro.

## E05

**Cadastro e configuração manual de projetos · Produto**

**Objetivo e contexto.** Cadastrar um projeto real existente, orientar configuração e impedir ativação incompleta (D06, D22, D29–30, D58, D60, D91, D94).

**Escopo e requisitos.**

- Assistente determinístico para dados, repo/base GitHub, origem Jira ou ClickUp, escopos, seleção de features, mapa de status, regras, perfil técnico e validação.
- Identificar configurações úteis no repo, como manifests, Compose e testes, por inspeção segura. Apresentar origem do valor e permitir sobrescritas de instalação, build, checks, start, serviços e referências de segredo.
- Escolher manualmente workflow e subfluxo padrão, regras alternativas, agentes, skills, ferramentas, modelos e conexões que serão utilizados. Personalizar localmente e validar publicação, permissões e compatibilidade.
- Registrar ativação, suspensão e alterações de origem. Mostrar pendências com ação concreta e conservar rascunho.

**Regras.** Projeto novo vazio de atribuições. Uma origem oficial por projeto. Descoberta técnica pode sugerir comandos encontrados, cuja execução segue política e perfil. Associar recurso global é escolha explícita.

**Dependências e entregáveis.** E03, E04 e contratos de E06–E09. Entregar assistente, configuração efetiva, mapa de integração e relatório de validação.

**Aceite e testes.** Cadastrar do zero sem atribuição automática. Impedir ativação com acesso inválido, origem ambígua, grafo em rascunho ou modelo incompatível. Ativar quando configuração válida. Alterar projeto A e demonstrar que projeto B e catálogo global preservam suas configurações.

## E06

**Jira, ClickUp e sincronização explícita · Produto**

**Objetivo e contexto.** Tratar Jira e ClickUp como fontes oficiais alternativas e manter a execução local reconciliada (D03, D05–06, D50, D85, D88–92).

**Escopo e requisitos.**

- Adaptadores simétricos para validar conexão, ler campos, listar itens com paginação, mapear prioridade/status, criar subtarefas, relacionar dependências e escrever links de PR. Tratar campos/transições indisponíveis, permissões e limites da plataforma.
- Deduplicar importação e subtarefas por identificador de negócio. Gravar tentativa e resultado das escritas. Confirmar estado antes de repetir criação ou transição ambígua.
- Expor sincronização manual e preflight de nova execução. Normalizar ID, versão/hash, conteúdo, estado e prioridade. Exibir última atualização e diferenças. Aplicar D89 às mudanças de requisitos e PRs.
- Comparar snapshots e classificar impacto de novos requisitos. Atualizar em marco seguro, replanejar ou abrir decisão. Troca de origem preserva histórico, mapeia identidades e só ativa nova origem após validação.

**Regras.** Uma origem por projeto e uma identidade remota por item. Escritas do ciclo de vida seguem D50. A descoberta segue D89. Rejeitar atualização que sobrescreva silenciosamente edição humana. Conclusão externa depende de merge confirmado.

**Dependências e entregáveis.** E03–E04, contratos E05 e E13, integração com E10 e E15. Entregar adaptadores, normalizador, outbox, UI de sincronização e testes de contrato.

**Aceite e testes.** Repetir sincronização e simular perda de resposta de criação em cada plataforma. Verificar uma subtarefa por tarefa, status correto, link e dependências. Alterar origem remota com fila ociosa e provar que só é detectado na próxima consulta permitida. Validar requisitos compatíveis, conflitantes e mudança de prioridade sem preempção.

## E07

**Editor visual, publicação e execução de workflows · Produto**

**Objetivo e contexto.** Permitir construir e reutilizar fluxos completos e determinísticos no painel (D09, D13, D44–45, D58–60, D64, D94).

**Escopo e requisitos.**

- Canvas de nós e conexões, edição de propriedades, variáveis, subfluxos e validação com localização do problema. Criar, editar, salvar e reutilizar workflows. Tipos de execução incluem IA, skills, scripts, testes, Git e documentação.
- Engine com IF/ELSE, loops limitados, espera durável, tratamento de erro, decisões humanas, ramificação e convergência. Portas e saídas tipadas. Workflow de feature coordena subfluxos técnicos reutilizáveis.
- Rascunho e versão publicada, validação server-side, dependências versionadas, agente revisor configurável, parâmetros e seleção efetiva fixados na execução.
- Seleção determinística de fluxo principal e subfluxo por regra ordenada, com padrões explícitos e explicação da regra selecionada.

**Regras.** Rascunho não inicia execução. Caminhos do grafo respeitam o escritor único do projeto. Evitar recursão de subfluxos e loops sem limite. Atualização de configuração publicada afeta novas execuções. Políticas e revogações atuais continuam em vigor.

**Dependências e entregáveis.** E03–E04, contratos dos catálogos E08–E09 e motor E10–E13. Entregar canvas, schema, validador, engine e histórico de publicação.

**Aceite e testes.** Montar e publicar manualmente um fluxo com todos os tipos de controle, percorrer ambos os ramos, executar loop limitado e convergir. Rejeitar ciclos indevidos, portas inválidas e referências revogadas na execução. Editar e publicar nova versão durante execução e provar que ela usa a anterior ao retomar.

## E08

**Catálogos de agentes, skills, ferramentas e MCP · Produto**

**Objetivo e contexto.** Reutilizar recursos controlados em projetos e workflows, com origem, contratos e privilégio mínimo (D10–13, D74, D78–80).

**Escopo e requisitos.**

- CRUD de agentes com instruções, conta/modelo, fallback, skills, ferramentas e timeout. Permitir overrides por nó sem editar base. Exibir efetivo sanitizado.
- Skills manuais, `SKILL.md`/pacote e Git público/privado autorizado. Guardar origem, revisão, hash, recursos, permissões e tipo de executor. Validar pacote, caminhos e scripts antes de uso. Suportar skill de IA, executável, determinística ou híbrida.
- Ferramentas nativas de leitura, edição, comando e consulta Git. Cadastro de scripts e integrações com schemas de entrada/saída, limite de execução e política.
- Cadastro e teste MCP, transporte local e remoto conforme contrato aprovado no preflight, descoberta de ferramentas, allowlist individual por agente e execução auditada.

**Regras.** Ferramenta descoberta nasce sem autorização. O modelo recebe descrições apenas das permitidas e necessárias. Chamadas são revalidadas no executor. Recursos importados são tratados como dados e não iniciam execução automaticamente.

**Dependências e entregáveis.** E03–E04, E07 e contratos E09, E11, E13. Entregar catálogos, importadores, integração MCP e seleção por workflow.

**Aceite e testes.** Cadastrar uma skill por cada modalidade e usá-la com agente e de forma independente. Rejeitar travessia de caminho e script fora da política. Descobrir duas ferramentas, autorizar uma, bloquear outra e provar que ferramenta recém-descoberta não herda acesso.

## E09

**Contas, modelos, adaptadores de IA, orçamento e fallback · Produto**

**Objetivo e contexto.** Dar o mesmo suporte funcional a assinatura/CLI e API, com seleção explícita, gasto limitado e continuidade da execução (D14–17, D37, D71, D73–77).

**Escopo e requisitos.**

- Adaptadores para Codex CLI, Claude Code, OpenRouter e Vercel AI Gateway. Interface de capacidade, autenticação, validação, execução, cancelamento, eventos, consumo e falhas. Confirmar elegibilidade, isolamento e cobrança com contas reais no preflight.
- Múltiplas conexões nomeadas por provedor, configuração manual por agente/nó. Modelos cadastrados manualmente, parâmetros comuns e específicos, validação de compatibilidade e registro de configuração efetiva.
- Chamadas de desenvolvimento via CLI e via API. Usar ferramentas controladas para leitura, edição, comando e Git. Em APIs, fornecer loop de chamada/retorno de ferramentas com limites e validação.
- Controlar orçamento global e opcional por agente com reserva transacional e reconciliação de uso. Exibir estimativa, reportado, desconhecido e cota de assinatura em unidades próprias.
- Aplicar cadeia explícita de fallback e status híbrido de disponibilidade, com retentativa controlada e preservação antes de troca.

**Regras.** Seleção manual exclui rodízio. Fallback percorre somente alternativas autorizadas, compatíveis e dentro do orçamento. Sem alternativa, tarefa aguarda sem ocupar a vaga quando houver checkpoint seguro. Diferenciar autenticação, cota, indisponibilidade e bloqueio financeiro.

**Dependências e entregáveis.** E03–E04, contratos de E08, E11–E13. Entregar adaptadores, catálogo de modelos, painel financeiro e registros de preflight.

**Aceite e testes.** Executar uma edição/teste isolados pelos dois modos e provar handoff preservando diff. Forçar cota e indisponibilidade com fixtures, testar duas contas do mesmo provedor e garantir escolha explícita. Disputar teto com chamadas concorrentes e provar que a reserva impede exceder autorização.

## E10

**Planejamento, dependências, fila e controles de execução · Execução**

**Objetivo e contexto.** Converter requisitos em trabalho técnico verificável e agendar projetos continuamente dentro da capacidade disponível (D04–08, D21, D32 substituída, D52–57, D59, D90, D92).

**Escopo e requisitos.**

- Agente planejador lê requisitos, código e contexto relevante e produz plano estruturado com tarefas, critérios, risco inicial, dependências e estimativa de ordem. Validador recusa ciclos, referências ausentes e critérios que não possam ser verificados. Registrar versão e gerar subtarefas na origem.
- Scheduler transacional considera dependências técnicas, merge, elegibilidade, pausa, disponibilidade, orçamento e permissões. Fila preferencial para correções de PR e ordem por prioridade para o restante.
- Uma tarefa técnica ativa por projeto, paralelismo entre projetos e limite global persistente. Recurso de prioridade por projeto e envelhecimento determinístico. Reclamar vaga com lease, heartbeat e geração.
- Controles globais e por projeto: ativar, pausar após feature atual, suspender agora com checkpoint, retomar e ajustar limite. Mudança de prioridade afeta próxima seleção. Tarefa bloqueada devolve vaga após liberação segura.

**Regras.** D55 substitui D32. Tarefas dependentes aguardam evidência do pré-requisito. Corrigir PR na vaga seguinte elegível. Seleção após preflight D89 considera novas informações. Scheduler respeita escritor único mesmo com dois workers reclamando o mesmo trabalho.

**Dependências e entregáveis.** E03, E05–E07, contratos E09 e E11–E13. Entregar planejador, máquina de estados, scheduler, leases e APIs operacionais.

**Aceite e testes.** Rodar três projetos com limite dois e duas tarefas prontas no mesmo projeto. Medir sobreposição entre projetos e exclusão dentro deles. Cobrir corrida de claim, bloqueio de dependência, fila de correção, mudança de prioridade, projeto antigo finalmente servido, reboot e dois modos de pausa.

## E11

**Worker, sandbox e controle de processos · Execução**

**Objetivo e contexto.** Executar agentes e ferramentas com autonomia dentro do escopo de desenvolvimento, mantendo isolamento por feature (D02, D30–31, D35, D54, D70, D77–78, D84).

**Escopo e requisitos.**

- Provisionar container temporário por feature, branch/workspace e volumes próprios, com ferramentas da stack alvo e serviços auxiliares isolados. Aplicar CPU, memória, processos, disco, rede e política de montagens.
- Supervisionar clientes CLI, ferramenta de API, scripts, builds, checks e navegador. Aplicar as mesmas fronteiras ao subprocesso. Fornecer operações de arquivos, comando e Git com validação de caminho/escopo.
- Monitorar duração máxima por agente, heartbeat, árvore de processos, sinais de atividade e estado do runtime. Diferenciar tarefa silenciosa legítima de travamento.
- Em pausa, timeout, revogação ou falha, interromper com segurança, confirmar quiescência, registrar resultado e coordenar checkpoint e liberação de recursos. Tratar operação externa não cancelável com estado explícito.

**Regras.** Um escritor autorizado por projeto. Host, cofre, banco, outros projetos e credenciais permanecem fora do alcance do código. Política avalia rede e alvo. Aprovação não libera ação proibida.

**Dependências e entregáveis.** E02–E04, contratos E08–E10, E12–E13. Entregar worker, supervisor, imagens/perfis de sandbox, políticas de recursos e testes adversariais.

**Aceite e testes.** Executar install/build/teste e escrita no workspace. Tentar ler volumes de outro projeto, acessar socket/host, escapar com symlink e atingir rede proibida. Forçar timeout com filhos e verificar fim da árvore, checkpoint e ausência de escritor antigo. Preservar execução legítima que esteja ativa sem stdout.

## E12

**Contexto estruturado, checkpoints e retomada seletiva · Execução**

**Objetivo e contexto.** Continuar trabalho sem depender da sessão de um fornecedor e recuperar falhas com estado íntegro e rastreável (D15, D17–18, D31, D33–34, D41, D45–46, D66, D71–72, D83, D87–88, D90).

**Escopo e requisitos.**

- Checkpoint versionado e persistente com revisão Git, alterações rastreadas e não rastreadas, artefatos, estado de nós, entradas/saídas, processo parado, efeitos registrados e próximas ações. Verificar integridade antes de disponibilizar.
- Montar contexto seletivo a partir de requisitos, critérios, decisões, memória relevante, arquivos, evidências e saídas estruturadas. Registrar manifesto de fontes, hashes, limites e omissões. Remover segredos.
- Checkpoints em planejamento, implementação, testes, revisão, suspensão e mudança de sessão. Renovar preventivamente com métricas disponíveis ou estimativas auditáveis, inclusive troca autorizada de modelo.
- Em reinício, verificar lease/processo, estado Git, artefatos, políticas e efeitos. Diagnosticar, reprocessar com limite por classe de falha, preservar e bloquear apenas o afetado se houver incerteza.
- Retomar nó específico com validação de dependências, autorizações e reconciliação de efeitos. Preservar workflow publicado e invalidar descendentes quando insumos mudam. Incorporar alterações externas compatíveis e abrir decisão em conflito.

**Regras.** O último checkpoint íntegro governa a restauração. Etapas confirmadas são reutilizadas quando válidas. Aprovações expiradas/revogadas exigem novo tratamento. Repetição e compensação obedecem E13. Limite de revisão é independente de infraestrutura e provedores.

**Dependências e entregáveis.** E03, E07, E10–E11 e contratos E13. Entregar armazenamento, manifesto de contexto, mecanismos de recuperação e UI/API de retomada.

**Aceite e testes.** Interromper antes e depois de um checkpoint, reiniciar VPS/worker, restaurar patch e arquivos novos com hashes iguais e retomar sem efeitos duplicados. Reexecutar etapa com saída alterada e observar invalidação dos descendentes. Simular alteração externa conflitante e contexto próximo do limite sem métricas oficiais.

## E13

**Políticas, decisões humanas, autorização e efeitos externos · Execução**

**Objetivo e contexto.** Autorizar operações pelo risco e tornar repetição e compensação seguras (D07, D26, D34–35, D80–87, D93).

**Escopo e requisitos.**

- Classificar ação por ferramenta, operação, parâmetros, recurso e destino. Ações autorizadas de baixo risco seguem automaticamente, sensíveis permitidas exigem aprovação e proibidas são bloqueadas. Aplicar a scripts, CLI, API, MCP e conectores.
- Central no painel com problema, alternativas, impacto e evidência. Resposta registrada e integrada a fila/retomada. Suportar conflito, falta de requisito, operação sensível, resultado incerto e PR fechada.
- Autorizações de escopo explícito, argumentos canônicos, validade, limite de usos e vínculo à execução/revisão quando aplicável. Expiração sem autorização e revogação atômica. Interrupção segura e checkpoint em coordenação com E11–E12.
- Ledger de intenção, identidade de operação e confirmação remota. Em resposta perdida, reconciliar antes de repetir. Em caso inconclusivo, solicitar decisão. Compensar reversíveis de baixo risco, aprovar sensíveis e preservar irreversíveis.

**Regras.** Agentes e comentários externos não concedem permissão. Revogação e política atual se aplicam à próxima operação, inclusive em execução versionada. Uma aprovação de operação é distinta do parecer de revisão de código.

**Dependências e entregáveis.** E03–E04, contratos E06, E08, E11–E12. Entregar motor de política, central, autorizações, ledger e trilha de auditoria.

**Aceite e testes.** Validar uso dentro do escopo, destino diferente, argumento adulterado, expiração, corrida no último uso, revogação antes/durante operação, proibição e comportamento de tarefas independentes. Simular efeito remoto aplicado seguido de timeout, efeito comprovadamente ausente e resultado inconclusivo. Testar compensação segura, sensível e irreversível.

## E14

**Qualidade, risco, revisão e testes automatizados · Integração**

**Objetivo e contexto.** Condicionar commits e entrega a critérios e evidências proporcionais ao risco (D23, D61–69).

**Escopo e requisitos.**

- Configurar comandos e critérios por projeto/tarefa. Registrar baseline quando viável, revisão do código e ambiente. Executar testes, lint, build e checks aplicáveis. Comparar resultado para distinguir regressão, falha anterior e origem incerta.
- Estimar risco no planejamento, aplicar pisos determinísticos a áreas críticas e reavaliar diff após implementação/correção. Classificar baixo, médio e alto com justificativa. Alto requer checks adicionais de segurança aplicáveis.
- Revisor configurável pelo workflow, execução inicial independente, achados estruturados e níveis crítico, relevante e baixo. O revisor pode corrigir em etapa posterior, revalidar com evidências e respeitar duas rodadas por padrão configurável.
- Criar ou atualizar testes quando aplicável. E2E de navegador conforme impacto/risco, com dados de teste, revisão e evidências. Justificar casos em que teste automatizado não é aplicável e registrar alternativa.

**Regras.** Falha crítica/relevante ou regressão bloqueia commit e entrega. Falha anterior comprovada é documentada e não bloqueia automaticamente. Origem incerta exige investigação. Uma alteração após check invalida evidência aplicável à revisão anterior.

**Dependências e entregáveis.** E05, E07–E12. Entregar classificação, executor de checks, revisão, correções, testes de navegador e evidências consumidas no Git.

**Aceite e testes.** Exercitar os três níveis, incluindo elevação de risco por arquivo crítico. Mostrar revisão separada e correção pelo próprio revisor. Forçar falha preexistente, regressão e inconclusiva. Esgotar limite e preservar checkpoint. Validar evidência E2E associada à revisão e mudança relevante.

## E15

**Entrega GitHub, revisão externa, documentação e memória · Integração**

**Objetivo e contexto.** Consolidar código, testes, documentação e conhecimento numa entrega rastreável por feature (D19–25, D47–52, D61, D67, D85, D89, D93).

**Escopo e requisitos.**

- Clone e branch por feature, commits por tarefa validada, push restrito e Draft PR única no fluxo normal. Guardar SHAs, referências, identificação de feature e prova de efeitos. Rejeitar alteração direta na branch base.
- Gate de Markdown no repositório do projeto: documentação viva dos módulos e histórico por feature com decisões, arquivos, APIs/dados/configuração, comandos, resultados e limites reais. Incluir docs e testes na mesma PR.
- Capturar estado do GitHub nos momentos D89. Confirmar merge para concluir, sincronizar plataforma e liberar dependentes. Corrigir solicitações de revisores autorizados na mesma branch, preferir próxima vaga, reexecutar validações e atualizar docs/PR.
- Em fechamento sem merge, preservar histórico e abrir decisão para cancelar, reabrir ou nova tentativa controlada. Validar refs/PRs antes de qualquer efeito. Somente uma tentativa ativa de entrega por feature.
- Produzir memória técnica automaticamente com evidência e deduplicação. Manter pendente por branch. Após merge, validar contra revisão integrada e promover fatos aplicáveis. Permitir consulta seletiva, atualização e invalidação.

**Regras.** Merge humano. Link e estado da PR são auditáveis. Comentários são insumos e passam pelo controle de risco. Conhecimento pendente não é fato oficial. Fechamento sem merge mantém dependentes bloqueadas e artefatos protegidos.

**Dependências e entregáveis.** E05–E06, E10–E14. Entregar adaptador GitHub, fluxo de branch/commit/PR, gate documental e memória de projeto.

**Aceite e testes.** Feature com várias tarefas produz commits e Draft PR única com código, testes e docs. Perda de resposta após push/criação é reconciliada. Descobrir pedido de revisão por sync, corrigir na mesma branch e verificar limite. Testar três decisões após PR fechada, merge humano, atualização de status, desbloqueio e promoção da memória.

## E16

**Painel operacional, histórico, uso e retenção · Integração**

**Objetivo e contexto.** Permitir supervisão, decisão e investigação pelo painel com evidências disponíveis e armazenamento controlado (D16, D26–28, D41, D53–54, D73, D83–84).

**Escopo e requisitos.**

- Lista de projetos com resumo de status e tela de detalhe com features, tarefas, fluxo, configurações, duração, estágio, agente, estado e motivo do bloqueio. Controles operacionais e disponibilidade da última sincronização.
- Stream de logs/eventos em tempo real com reconexão, cursor, histórico pesquisável e trilha de auditoria. Visualizar tentativa, checkpoint e artefatos com autorização e escopo do projeto.
- Central global com fila de decisões, contexto, validade, decisão, revogação e resultado. Mostrar progresso da retomada e evidências de operações.
- Visão de orçamento, cotas, estimado/reportado/desconhecido e disponibilidade. Limpeza automática de logs detalhados aos 30 dias e checkpoints concluídos após carência. Conservar histórico resumido permanente e checkpoints necessários.

**Regras.** Estados e resultados vêm do servidor. Redigir dados sensíveis. Exclusão de logs por idade não elimina o resumo de auditoria exigido. Checkpoint ativo, bloqueado ou referenciado por incidente/decisão é protegido.

**Dependências e entregáveis.** E03–E05, E09–E15. Entregar interface operacional, stream, retenção e runbooks de diagnóstico.

**Aceite e testes.** Conectar, derrubar e reconectar stream sem lacuna ou duplicação na interface. Ver estados e tempos de execução real, aprovar/revogar pelo painel, confirmar continuidade independente e proteger dados sensíveis. Com relógio controlado, provar retenção de 30 dias e proteção de checkpoints recuperáveis.

## E17

**Validação integral, piloto real e aceite do lançamento · Aceite**

**Objetivo e contexto.** Demonstrar todos os épicos do MVP integrado e a operação da fábrica em projeto real (D42, D43, D95).

**Escopo e requisitos.**

- Selecionar repositório real existente e uma feature com critérios, dependências e evidências. Cadastrar projeto do zero pela interface, configurar integrações, workflows/subfluxos, catálogos, modelos, ferramentas, MCP e orçamento.
- Executar da leitura de requisitos até Draft PR real. Exercitar revisão, testes, documentação, memória, controles, recuperação, aprovações e sincronização. Confirmar merge humano e efeitos a jusante na validação de ciclo.
- Cobrir os dois conectores com ensaios de integração e as regras de concorrência com múltiplos projetos em ambiente isolado, complementando a feature do piloto. Demonstrar adapters, fallback, backup/restauração e incidentes de efeito externo.
- Registrar evidências por critério, revisão, configuração e ambiente. Consolidar relatório de aceite, pendências e procedimento operacional.

**Regras.** Aceite somente quando todos os épicos e critérios obrigatórios tiverem evidência válida. O cenário real é acompanhado de testes controlados para falhas, concorrência e segurança. O responsável aprova a entrega e gerencia o desenvolvimento.

**Dependências e entregáveis.** E01–E16. Entregar [contrato e roteiros de piloto](05-ACEITE-E-PILOTO.md) preenchidos na execução, pacote de evidências e aceite explícito do responsável.

**Aceite e testes.** Revisar a cobertura D01–D95, configurar o projeto do zero e executar a feature real até a Draft PR. Comprovar recuperação, retomada seletiva, isolamento, aprovação e revogação, sincronização e idempotência. Verificar decisões substituídas pelos testes correspondentes. A apresentação de uma execução isolada é insuficiente para o aceite.
