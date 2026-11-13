# Relatório de aceite integral do MVP

> **Gerado** por `make matriz` a partir de `07-CHECKLIST.md`, `03-BACKLOG.md` e `05-ACEITE-E-PILOTO.md`. Não edite à mão: um teste confere que este arquivo é igual ao gerado. O relatório **não aceita nada**: mostra o que está provado e o que falta; a decisão é do responsável.

**Situação do aceite integral:** NÃO FECHADO.

**Tickets:** 19 feito(s), 48 parcial(is), 0 pendente(s), de 67. Todo ticket `[x]` ou `[~]` cita a verificação real feita e o que ficou NOT_RUN na própria nota.

O aceite exige o **projeto real** (D95): evidência de software (testes, ensaios e entregas) **não** a substitui. Nenhum dos 18 casos P01–P18 tem evidência no projeto real registrada ([matriz](02-matriz-de-evidencias.md)).

## Artefatos do aceite

| Artefato | Situação |
|---|---|
| [Ficha de preparação](01-ficha-de-preparacao.md) (F01–F12 a preencher pelo responsável) | presente |
| [Matriz de evidências](02-matriz-de-evidencias.md) (gerada) | presente |
| [Ensaios de falha, segurança e integração](03-ensaios-de-falha-e-seguranca.md) (gerado por `make ensaios`) | presente |

## Épicos

| Épico | Feito | Parcial | Pendente | Situação |
|---|---|---|---|---|
| **E01** · Bootstrap e monorepo | 2 | 2 | 0 | com pendências |
| **E02** · VPS, CI e recuperação da instalação | 0 | 4 | 0 | com pendências |
| **E03** · Fundação web, API e dados | 3 | 0 | 0 | software verificado |
| **E04** · Acesso e credenciais | 2 | 1 | 0 | com pendências |
| **E05** · Cadastro de projeto | 1 | 2 | 0 | com pendências |
| **E06** · Integrações e sincronização | 1 | 4 | 0 | com pendências |
| **E07** · Workflows visuais | 3 | 1 | 0 | com pendências |
| **E08** · Catálogos e MCP | 2 | 2 | 0 | com pendências |
| **E09** · Provedores de IA e orçamento | 0 | 4 | 0 | com pendências |
| **E10** · Planejamento, fila e controles | 0 | 4 | 0 | com pendências |
| **E11** · Sandbox e processos | 1 | 2 | 0 | com pendências |
| **E12** · Contexto e recuperação | 0 | 4 | 0 | com pendências |
| **E13** · Autorizações e efeitos | 4 | 1 | 0 | com pendências |
| **E14** · Qualidade e revisão | 0 | 4 | 0 | com pendências |
| **E15** · GitHub, documentação e memória | 0 | 5 | 0 | com pendências |
| **E16** · Operação, uso e retenção | 0 | 4 | 0 | com pendências |
| **E17** · Piloto e aceite | 0 | 4 | 0 | com pendências |

## Pendências por ticket (o que impede o aceite integral)

| Ticket | Estado | Aceite do ticket | Limites declarados |
|---|---|---|---|
| **T01-02** · Criar e configurar repositório GitHub da fábrica | parcial | PR de teste passa pelo fluxo previsto. Push direto sujeito à regra configurada e permissões conferidas com a identidade usada. | sem limite declarado na nota |
| **T01-03** · Criar layout e ferramentas de desenvolvimento | parcial | Segundo clone reproduz setup e comandos com ferramentas fixadas. Segredos reais não aparecem em arquivo ou histórico Git. | sem limite declarado na nota |
| **T02-01** · Inventariar e medir VPS existente | parcial | Relatório registra medidas, gargalos, parâmetros e capacidade para ensaios de concorrência. Valores sem medida estão identificados. | sem limite declarado na nota |
| **T02-02** · Configurar ambientes, rede, proxy e containers | parcial | Somente entrada autorizada exposta. Serviços voltam após reboot e homologação não usa credenciais nem volumes de produção. | sem limite declarado na nota |
| **T02-03** · Automatizar CI e promoção por revisão | parcial | PR com falha é bloqueada. Imagens e evidências apontam ao mesmo commit. Segredo de produção não é disponibilizado a jobs de PR. | sem limite declarado na nota |
| **T02-04** · Documentar e ensaiar backup e restauração | parcial | Restaurar em destino isolado e conferir sessão/configuração conforme aplicável, capacidade de desencriptar conexões, banco e hashes de artefatos. | sem limite declarado na nota |
| **T04-03** · Criar cofre, gestão e teste de conexões | parcial | Rotação mantém vínculos sem expor valor anterior. Revogação é observável antes do próximo uso. Busca com segredo sentinela não acha texto em logs, respostas ou … | prova em *prompts* NOT_RUN (sem pipeline de prompts — E09/E10); teste remoto só GitHub/ClickUp/Jira e só contra servidores falsos (reais NOT_RUN); `cli`/`api`/`mcp` sem teste remoto; rotação da chave mestra não implementada. |
| **T05-02** · Configurar repositório, origem e elegibilidade | parcial | Conexão incompatível, duas origens ativas e mapeamento inválido falham com motivo. Origem válida devolve preview correto de itens elegíveis. | repositório GitHub não verificado (depende de T15-01); escrita na origem só provada no primeiro uso; Jira/ClickUp reais NOT_RUN. |
| **T05-03** · Descobrir stack e associar recursos manualmente | parcial | Overrides preservam recurso global. Rascunho de workflow e conta incompatível bloqueiam ativação. Projeto apto pode iniciar execução. | sem leitura do repositório real no GitHub (T05-02/T15-01 parciais; os manifestos vêm do chamador) e sem tela do assistente; skills/ferramentas só têm existência validada. |
| **T06-01** · Implementar adaptador Jira | parcial | Contrato com fixtures e instância de teste distingue itens, subtarefas, transições válidas, paginação, permissão e rate limit. | instância real de teste do Jira NOT_RUN. |
| **T06-02** · Implementar adaptador ClickUp | parcial | Contrato com fixtures e lista de teste demonstra pai/filho, status, prioridade e tratamento de erros. | lista real do ClickUp NOT_RUN; link de PR vira comentário (repetir duplica). |
| **T06-04** · Implementar sincronização manual e preflight | parcial | Relógio avança ocioso sem leitura periódica de mudanças. Sync manual ou preflight detecta a alteração uma vez. | Jira/ClickUp reais NOT_RUN; GitHub (T15-03), ligação do preflight ao agendador (T10-02) e tela pendentes. |
| **T06-05** · Tratar mudança de requisitos e troca de plataforma | parcial | Mudança compatível mantém histórico. Conflito preserva alterações e cria decisão. Troca mantém vínculos históricos e uma origem ativa. | Jira/ClickUp reais NOT_RUN; sem rotas/tela; o sync (T06-04) e o planejador não chamam essas funções sozinhos; itens da origem antiga permanecem lá. |
| **T07-03** · Publicar e congelar configuração efetiva | parcial | Rascunho não executa. Publicação nova não muda execução existente. Revalidação em runtime respeita revogação. | motor/worker ainda não consome `StartExecution` nem revalida a cada nó; sem tela. |
| **T08-02** · Gerenciar skills e importações | parcial | Importações ficam reutilizáveis e auditáveis. Pacote malicioso falha sem executar script. Exercitar skill híbrida e nó isolado. | importação por **Git UNSUPPORTED** (T15-01); IA real NOT_RUN (executor de mentira). |
| **T08-04** · Cadastrar MCP, descobrir e autorizar ferramentas | parcial | Descoberta lista ferramentas. O agente recebe e invoca apenas a selecionada. Nova ferramenta não ganha acesso automaticamente. | servidor MCP real NOT_RUN, stdio UNSUPPORTED, sem tela no painel. |
| **T09-01** · Cadastrar contas e adaptar clientes CLI | parcial | Duas contas do mesmo provedor são distintas e selecionáveis. Rodar alteração e checks no sandbox com modo autorizado comprovado. | Claude Code e Codex CLI reais NOT_RUN (exigem conta do responsável); sem fluxo de login nem imagem de sandbox com os CLIs; formatos de evento não confirmados por execução. |
| **T09-02** · Cadastrar modelos e adaptar gateways de API | parcial | Modelo só é usado quando compatível. Cada gateway roda edição e teste com modelo elegível. Sobreposição modelo/agente/nó é verificável. | gateways reais NOT_RUN; sem tela; laço ainda não ligado ao worker; sem streaming. |
| **T09-03** · Impor orçamento e registrar uso | parcial | Chamadas concorrentes disputam reserva sem ultrapassar teto autorizado. Cota de assinatura é registrada separadamente de gasto em API. | gateways e créditos reais NOT_RUN; sem tela; limites globais de concorrência são do T10-03. |
| **T09-04** · Implementar fallback e disponibilidade | parcial | Cota no principal leva só à alternativa autorizada. Esgotamento preserva tarefa e outra avança. Disponibilidade volta sem rodízio automático. | sem ligação ao worker/executor e sem classificação dos erros reais de Codex/Claude/OpenRouter/Vercel (NOT_RUN), sem rotas/tela; quiescência declarada pelo chamador. |
| **T10-01** · Planejar e versionar decomposição da feature | parcial | Feature vira tarefas verificáveis, sem ciclo ou efeito duplicado no retry. Plano conflitante preserva versão anterior. | o agente planejador (produtor do plano) e a leitura de código/contexto não existem; `Publish` sem rota nem ligação ao scheduler (T10-02); Jira/ClickUp reais NOT_RUN; sem tela. |
| **T10-02** · Implementar fila local por projeto | parcial | Só tarefa elegível inicia. PR corrigida ganha próxima vaga elegível. Estado bloqueado libera espaço após checkpoint seguro. | sem scheduler, lease/heartbeat, limite global, envelhecimento nem controles (T10-03); portas de provedor/recursos não ligadas; sem rotas/tela. |
| **T10-03** · Aplicar limite global, leases e justiça | parcial | Com jobs concorrentes, um por projeto e total configurado. Escritor obsoleto não grava. Projeto elegível antigo consegue vaga. | nenhum worker chama `Claim`/`Heartbeat` ainda (mecanismo e prova, não a ligação); só itens reclamados pelo scheduler exigem o token; sem controles (T10-04) nem recuperação de tentativa interrompida (T12-03). |
| **T10-04** · Implementar ativação, pausa e retomada operacional | parcial | As duas ações têm efeitos diferentes e persistem no reboot. Projetos não afetados continuam executando. | o `Suspender` real (cancelar processo + `CaptureAndRelease`) não está ligado ao worker, sem rotas/tela; recuperação após crash é o T12-03. |
| **T11-01** · Provisionar sandbox e perfis de execução | parcial | Clone e build funcionam no ambiente. Execução de A não alcança B, host, banco ou rede administrativa. | clone remoto com credencial e rede com saída controlada não implementados; isolamento do host só provado em Docker Desktop (Linux NOT_RUN); worker ainda sem acesso ao runtime. |
| **T11-02** · Executar ferramentas e skills com limites | parcial | Ferramentas autorizadas funcionam em ambos os modos de IA. Path traversal, symlinks e tentativa de acesso ao socket falham. | CLIs e gateways reais NOT_RUN; `git_*` em repositório real NOT_RUN; scripts não são confinados por caminho; corrida entre resolução e uso só afeta o próprio sandbox. |
| **T12-01** · Persistir e restaurar checkpoints íntegros | parcial | Restaurar em container novo e comparar hashes. Interromper gravação de snapshot mantém checkpoint anterior consistente. | `git` real NOT_RUN (imagens sem git; HEAD/ramo/patch provados com git de mentira); sem API/tela nem ligação ao motor/worker; objetos em `bytea` (≤128 MiB por checkpoint). |
| **T12-02** · Compor contexto seletivo e renovar sessão | parcial | Nova sessão retoma com artefatos e contexto verificáveis. Testar estimativa sem métrica oficial e ausência de segredos. | sem ligação ao worker (nada decide renovar nem cria o checkpoint do marco), memória real (T15-05) não alimenta o pacote, sem rotas/tela. |
| **T12-03** · Recuperar execução após falha ou reboot | parcial | Reboot e crash retomam automaticamente quando seguros. Incerteza abre decisão e outras tarefas elegíveis seguem. | o `Inspector` real (Docker, Git, artefatos, políticas) e a varredura no `cmd/worker` após o reboot não existem; a classe da falha é informada por quem chama; sem rotas/tela. |
| **T12-04** · Implementar retomada seletiva e reconciliação de insumos | parcial | Reexecutar nó altera só o conjunto afetado. Conflito preserva edições humanas e o workflow original continua fixado. | sem ligação ao worker (nada chama a retomada nem grava o novo checkpoint), sem rotas/tela; a convenção de chave de efeito `exec:<execução>:<nó>:` ainda não é seguida pelos executores; fast-forward real em Git NOT_RUN (função injetada). |
| **T13-05** · Compensar efeitos parciais por risco | parcial | Compensação executada ou recusada fica auditável. Falha conserva trabalho e bloqueia nova operação agravante. | adaptadores reais de desfazer (Jira/ClickUp/GitHub) não existem; `CheckAggravating` não está ligado ao executor; sem rotas/tela. |
| **T14-01** · Executar baseline, critérios e checks | parcial | Regressão bloqueia. Falha anterior comprovada é registrada. Origem desconhecida fica em investigação e não vira baseline por suposição. | sem preparo automático do baseline (checkout da revisão anterior), sem montar os checks a partir do perfil do projeto, sem rotas/tela; assinatura é heurística textual. |
| **T14-02** · Classificar e reavaliar risco | parcial | Fixture de arquivo crítico eleva nível apesar de sugestão baixa. Três níveis escolhem verificações previstas. | sem geração do diff (git) nem ligação ao ciclo de execução, sem rotas/tela, checks de segurança só como nomes (execução em T14-01/T14-03). |
| **T14-03** · Executar revisão independente e correções | parcial | Crítico/relevante bloqueiam. Baixa severidade fica registrada. Correção do revisor roda checks e esgotamento suspende com checkpoint. | o agente revisor (que lê o diff e produz os achados), o checkpointer real e a escolha de disponibilidade ligada a T09-04 não existem (injetados); sem rotas/tela. |
| **T14-04** · Criar testes e rodar E2E por impacto | parcial | Jornada de alto risco testável tem E2E real. Caso inaplicável tem justificativa e evidência alternativa. | o navegador real (Playwright/Chromium) NOT_RUN — nenhuma imagem de sandbox traz navegador, e a execução real no Docker isolado foi provada com uma jornada simulada; heurística de testes por nome/diretório; sem comandos E2E do perfil, rotas ou tela. |
| **T15-01** · Integrar GitHub, branches, commits e push | parcial | Conexão valida clone. Commit corresponde a tarefa e checks. Simular timeout de push e prova por ref sem duplicidade ou alteração direta da base. | GitHub real NOT_RUN (permissão de push, proteção do ramo base e recusa de push direto só numa conta real); imagem do worker sem `git`; sem rotas/tela, PR e merge (T15-02/T15-03). |
| **T15-02** · Gerar documentação viva e publicar Draft PR | parcial | Repo real contém docs e testes coerentes. Repetição encontra a mesma Draft PR e ausência de documentação bloqueia entrega. | GitHub real NOT_RUN; o `git` real (commit/push, T15-01) não foi exercitado junto da PR; quem commita precisa chamar `RecordCommit`; a coerência semântica da documentação não é avaliada; sem rotas/tela (merge e correções externas são T15-03/T15-04). |
| **T15-03** · Confirmar merge e propagar conclusão | parcial | Draft PR aberta ou fechada sem merge não libera dependentes. Merge humano observado libera e eventual falha de status fica visível e reconciliável. | GitHub real NOT_RUN; nada aciona `Observe` a partir do worker/painel (ponto D89 ainda sem ligação), sem rotas/tela; dependência entre features e promoção de memória (T15-05) e as ações do fechamento sem merge (T15-04) pendentes. |
| **T15-04** · Corrigir revisão externa e tratar PR fechada | parcial | Mesmo comentário não duplica trabalho. PR fechada conserva artefatos. Cancelar, reabrir e nova tentativa controlada preservam histórico e respeitam decisão. | GitHub real NOT_RUN; a execução da correção pelo agente/worker e o acionamento da descoberta (D89) não estão ligados; sem rotas/tela. |
| **T15-05** · Manter memória técnica pendente e oficial | parcial | Projeto A não consome memória de B. PR fechada não promove. Merge promove só registros válidos para a revisão integrada. | sem ligação ao montador de contexto/worker, sem rotas/tela; extração só de achados de revisão; validade é checagem por revisão (sem avaliação semântica do conteúdo); GitHub real NOT_RUN. |
| **T16-01** · Exibir status, controle e logs em tempo real | parcial | Desconexão/reconexão recupera eventos pelo cursor. Estado bloqueado explica causa e painel exibe controles persistidos. | nenhum executor grava `execution_events` ainda (o painel mostra "Nenhum evento ainda"); o painel só EXIBE os controles (as ações são T10-04, sem rotas); não mostra agente, tentativas, checkpoints e artefatos; sem navegador real com a API ao vivo. |
| **T16-02** · Consolidar Central e auditoria operacional | parcial | Aprovar, recusar ou revogar pelo painel produz evento e estado verificáveis sem notificação externa. | navegador real NOT_RUN; parada segura na revogação com uso em andamento depende do executor (sem ele, `halt_failed`); ordem da sequência é por horário, não causal; sem filtro por projeto/paginação; recusa exige que o pedido declare a alternativa. |
| **T16-03** · Exibir consumo, orçamento e provedores | parcial | Valor não observado aparece explicitamente. Teto global e opcional por agente refletem o bloqueio de novas chamadas. | somente leitura (sem editar teto nem período), sem verificação em navegador real, provedores reais NOT_RUN. |
| **T16-04** · Automatizar retenção com integridade | parcial | Relógio de teste remove apenas itens elegíveis. Checkpoint bloqueado ou vinculado a incidente continua íntegro e restaurável. | nada agenda a retenção (sem laço no worker), sem rotas/tela; o resumo guarda contagens, não o conteúdo; carência conta pela última atualização da execução. |
| **T17-01** · Preparar projeto real e matriz de evidências | parcial | Contrato preenchido e revisável, com ambiente, projeto, usuários, conexões e critérios rastreáveis. | o contrato NÃO está completo — projeto e feature reais, VPS, origem, GitHub, contas e política (F01–F12) dependem do responsável e de credenciais. |
| **T17-02** · Executar cadastro do zero e feature ponta a ponta | parcial | A PR, commits, checks, subtarefas, memória e dependência são verificáveis no ambiente real e ligados à revisão. | sem limite declarado na nota |
| **T17-03** · Executar ensaios de falha, segurança e integração | parcial | Cada cenário tem esperado, observado, evidência e revisão. Falha obrigatória volta ao ticket e é retestada. | sem limite declarado na nota |
| **T17-04** · Fechar aceite integral do MVP | parcial | Todos os épicos possuem evidência e aceite. Não há requisito obrigatório omitido ou pendente no lançamento. | sem limite declarado na nota |

## Decisão do responsável

| Campo | Valor |
|---|---|
| Responsável | **A PREENCHER** (F01) |
| Decisão | **A PREENCHER** (aceitar / aceitar com ressalvas / recusar) |
| Data | **A PREENCHER** |
| Revisão do código aceita | **A PREENCHER** (commit e digests, F02) |
| Ressalvas | **A PREENCHER** |

Enquanto a decisão e a evidência do projeto real estiverem em branco, o MVP **não** está aceito. O agente não decide o aceite, não faz merge, push nem deploy.
