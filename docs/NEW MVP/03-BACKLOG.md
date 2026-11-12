# Backlog de features e tickets

Todos os tickets são planejados e pertencem ao primeiro lançamento. O responsável determina o calendário e distribui o trabalho. Cada feature abaixo corresponde a um [épico](02-EPICOS.md). A ordem por dependência admite desenvolvimento independente e integração posterior. O menor nível indica um ticket sem pré-requisitos dentro deste backlog.

## Ordem técnica por dependências

| Nível | Tickets que podem começar quando seus pré-requisitos estiverem disponíveis |
|---|---|
| 1 | [T01-01](#t01-01) |
| 2 | [T01-02](#t01-02) · [T02-01](#t02-01) |
| 3 | [T01-03](#t01-03) |
| 4 | [T01-04](#t01-04) · [T02-02](#t02-02) · [T03-01](#t03-01) |
| 5 | [T02-03](#t02-03) · [T03-02](#t03-02) · [T11-01](#t11-01) |
| 6 | [T04-01](#t04-01) |
| 7 | [T03-03](#t03-03) · [T04-02](#t04-02) |
| 8 | [T04-03](#t04-03) · [T05-01](#t05-01) · [T07-01](#t07-01) |
| 9 | [T06-01](#t06-01) · [T06-02](#t06-02) · [T07-02](#t07-02) · [T09-01](#t09-01) · [T13-01](#t13-01) |
| 10 | [T08-01](#t08-01) · [T08-02](#t08-02) · [T08-03](#t08-03) · [T11-03](#t11-03) · [T13-02](#t13-02) · [T13-04](#t13-04) |
| 11 | [T06-03](#t06-03) · [T08-04](#t08-04) · [T09-02](#t09-02) · [T11-02](#t11-02) · [T12-01](#t12-01) · [T15-01](#t15-01) |
| 12 | [T02-04](#t02-04) · [T05-02](#t05-02) · [T07-03](#t07-03) · [T09-03](#t09-03) · [T12-02](#t12-02) · [T13-03](#t13-03) |
| 13 | [T06-04](#t06-04) · [T07-04](#t07-04) · [T09-04](#t09-04) · [T13-05](#t13-05) |
| 14 | [T05-03](#t05-03) · [T10-01](#t10-01) · [T12-04](#t12-04) · [T16-03](#t16-03) |
| 15 | [T06-05](#t06-05) · [T10-02](#t10-02) · [T14-01](#t14-01) · [T17-01](#t17-01) |
| 16 | [T10-03](#t10-03) · [T14-02](#t14-02) |
| 17 | [T10-04](#t10-04) · [T12-03](#t12-03) · [T14-04](#t14-04) |
| 18 | [T14-03](#t14-03) · [T16-01](#t16-01) |
| 19 | [T15-02](#t15-02) · [T16-04](#t16-04) |
| 20 | [T15-03](#t15-03) |
| 21 | [T15-04](#t15-04) · [T15-05](#t15-05) |
| 22 | [T16-02](#t16-02) · [T17-03](#t17-03) |
| 23 | [T17-02](#t17-02) |
| 24 | [T17-04](#t17-04) |

## Critério de fechamento

Cada ticket fecha com revisão identificada, critérios conferidos, evidência de teste, documentação do contrato alterado e estado registrado no backlog do responsável. Os ensaios integrados e o projeto real são obrigatórios para o aceite da entrega.

## Feature E01 — Bootstrap e monorepo

[Especificação do épico](02-EPICOS.md#e01)

### T01-01

**Registrar arquitetura inicial e ADRs**

- **Dependências** · Nenhuma.
- **Trabalho** · Comparar monorepo e repositórios separados para web, API e worker. Registrar stack, PostgreSQL, interfaces, medição da VPS e fronteira com repositórios atendidos.
- **Aceite do ticket** · ADR explica custos, critérios de viabilidade, integração e responsáveis por decisões delegadas. D40 está vinculada ao benchmark.

### T01-02

**Criar e configurar repositório GitHub da fábrica**

- **Dependências** · [T01-01](#t01-01).
- **Trabalho** · Definir proprietário e visibilidade, inicializar main, configurar PRs, proteção, checks, acesso de manutenção e identidade de entrega. Registrar suporte do plano escolhido.
- **Aceite do ticket** · PR de teste passa pelo fluxo previsto. Push direto sujeito à regra configurada e permissões conferidas com a identidade usada.

### T01-03

**Criar layout e ferramentas de desenvolvimento**

- **Dependências** · [T01-02](#t01-02).
- **Trabalho** · Estruturar diretórios, toolchains e locks. Criar comandos de bootstrap, lint, teste, build, Compose e ambiente de exemplo sanitizado.
- **Aceite do ticket** · Segundo clone reproduz setup e comandos com ferramentas fixadas. Segredos reais não aparecem em arquivo ou histórico Git.

### T01-04

**Publicar guias e padrões no repositório**

- **Dependências** · [T01-03](#t01-03).
- **Trabalho** · Criar README, guias Codex/Claude/Antigravity, convenções de PR, revisão, migrations, logs, docs e evidências. Adotar os IDs desta especificação.
- **Aceite do ticket** · Pessoa nova localiza instruções e critérios pelo índice. Plano de lançamento inclui todos os épicos e decisões vigentes.

## Feature E02 — VPS, CI e recuperação da instalação

[Especificação do épico](02-EPICOS.md#e02)

### T02-01

**Inventariar e medir VPS existente**

- **Dependências** · [T01-01](#t01-01).
- **Trabalho** · Registrar capacidade, processos vizinhos, serviços do provedor, backup e acesso. Medir envelopes de jobs, containers, navegador e serviços de controle. Propor reserva e limite inicial.
- **Aceite do ticket** · Relatório registra medidas, gargalos, parâmetros e capacidade para ensaios de concorrência. Valores sem medida estão identificados.

### T02-02

**Configurar ambientes, rede, proxy e containers**

- **Dependências** · [T01-03](#t01-03), [T02-01](#t02-01).
- **Trabalho** · Criar imagens e configuração de dev/homologação/produção, TLS, redes privadas, volumes, limites, health checks e reinício. Injetar segredos fora de Git.
- **Aceite do ticket** · Somente entrada autorizada exposta. Serviços voltam após reboot e homologação não usa credenciais nem volumes de produção.

### T02-03

**Automatizar CI e promoção por revisão**

- **Dependências** · [T01-03](#t01-03), [T02-02](#t02-02), [T03-01](#t03-01).
- **Trabalho** · Configurar lint, teste Go/TS, contratos, integração com banco, smoke, build, análise de dependência/segredo e artefatos imutáveis. Documentar promoção e rollback.
- **Aceite do ticket** · PR com falha é bloqueada. Imagens e evidências apontam ao mesmo commit. Segredo de produção não é disponibilizado a jobs de PR.

### T02-04

**Documentar e ensaiar backup e restauração**

- **Dependências** · [T02-02](#t02-02), [T03-02](#t03-02), [T04-03](#t04-03), [T12-01](#t12-01).
- **Trabalho** · Selecionar recursos nativos da VPS. Registrar volumes, consistência, estratégia de chave, procedimento de restore e testes de integridade.
- **Aceite do ticket** · Restaurar em destino isolado e conferir sessão/configuração conforme aplicável, capacidade de desencriptar conexões, banco e hashes de artefatos.

## Feature E03 — Fundação web, API e dados

[Especificação do épico](02-EPICOS.md#e03)

### T03-01

**Inicializar web, API e worker**

- **Dependências** · [T01-03](#t01-03).
- **Trabalho** · Subir frontend React/TS, API Go e worker Go com contratos de configuração, saúde e ciclo de vida. Adicionar fixtures reproduzíveis.
- **Aceite do ticket** · Cada serviço inicia em dev e reporta falha de configuração de forma diagnosticável. Build reproduzível em CI.

### T03-02

**Implementar banco, migrations e transações**

- **Dependências** · [T03-01](#t03-01).
- **Trabalho** · Criar migrations, repositórios, chaves, unicidade, versões, outbox, eventos e metadados de artefatos. Documentar backup/upgrade e migrações compatíveis.
- **Aceite do ticket** · Testes em PostgreSQL real persistem após restart, rejeitam duplicidade e escopo inválido e validam caminho de migração.

### T03-03

**Criar navegação centrada em projetos**

- **Dependências** · [T03-01](#t03-01), [T04-01](#t04-01).
- **Trabalho** · Adicionar lista inicial, detalhes, biblioteca global e Central de Decisões. Padronizar erros, carregamento, vazio e autorização.
- **Aceite do ticket** · Links levam às rotas corretas, navegação autenticada funciona e acesso sem sessão é recusado pelo servidor.

## Feature E04 — Acesso e credenciais

[Especificação do épico](02-EPICOS.md#e04)

### T04-01

**Implementar login, TOTP e recuperação**

- **Dependências** · [T03-02](#t03-02).
- **Trabalho** · Criar conta única, hash de senha, onboarding TOTP, códigos de uso único, limitação de tentativa e sessão segura.
- **Aceite do ticket** · Senha sem 2FA não abre sessão. Recuperação funciona uma vez mesmo com requisições concorrentes.

### T04-02

**Endurecer sessões e ações sensíveis**

- **Dependências** · [T02-02](#t02-02), [T04-01](#t04-01).
- **Trabalho** · Adicionar revogação de sessões, reautenticação, CSRF, cookies seguros e trilha de autenticação.
- **Aceite do ticket** · Sessão revogada perde acesso. Requisição forjada ou sessão sem reautenticação é recusada nas ações sensíveis.

### T04-03

**Criar cofre, gestão e teste de conexões**

- **Dependências** · [T03-02](#t03-02), [T04-02](#t04-02).
- **Trabalho** · Criptografar por chave externa ao banco. Gerenciar referências, teste, rotação e revogação. Implementar redatores e acesso de serviço.
- **Aceite do ticket** · Rotação mantém vínculos sem expor valor anterior. Revogação é observável antes do próximo uso. Busca com segredo sentinela não acha texto em logs, respostas ou prompts.

## Feature E05 — Cadastro de projeto

[Especificação do épico](02-EPICOS.md#e05)

### T05-01

**Cadastrar projeto em rascunho**

- **Dependências** · [T03-03](#t03-03), [T04-02](#t04-02).
- **Trabalho** · Criar assistente, lista de campos, persistência por etapa e estado de validação. Começar sem agentes ou workflows associados.
- **Aceite do ticket** · Projeto novo vazio de vínculos persiste e retorna ao assistente. Sem preenchimento obrigatório válido, ativação é bloqueada.

### T05-02

**Configurar repositório, origem e elegibilidade**

- **Dependências** · [T05-01](#t05-01), [T06-01](#t06-01), [T06-02](#t06-02), [T15-01](#t15-01).
- **Trabalho** · Vincular GitHub/base e uma origem Jira ou ClickUp. Validar escopos, item, status, prioridade, critérios de marcação e escopo de leitura/escrita.
- **Aceite do ticket** · Conexão incompatível, duas origens ativas e mapeamento inválido falham com motivo. Origem válida devolve preview correto de itens elegíveis.

### T05-03

**Descobrir stack e associar recursos manualmente**

- **Dependências** · [T05-02](#t05-02), [T07-03](#t07-03), [T07-04](#t07-04), [T08-04](#t08-04), [T09-02](#t09-02).
- **Trabalho** · Inspecionar manifestos do repo, configurar comandos e segredos por referência, associar workflows/agentes/skills/ferramentas/modelos, validar e ativar.
- **Aceite do ticket** · Overrides preservam recurso global. Rascunho de workflow e conta incompatível bloqueiam ativação. Projeto apto pode iniciar execução.

## Feature E06 — Integrações e sincronização

[Especificação do épico](02-EPICOS.md#e06)

### T06-01

**Implementar adaptador Jira**

- **Dependências** · [T03-02](#t03-02), [T04-03](#t04-03).
- **Trabalho** · Criar operações de leitura, descoberta de tipos, transições, campos, hierarquia, vínculos e escrita usando conexão de escopo reduzido.
- **Aceite do ticket** · Contrato com fixtures e instância de teste distingue itens, subtarefas, transições válidas, paginação, permissão e rate limit.

### T06-02

**Implementar adaptador ClickUp**

- **Dependências** · [T03-02](#t03-02), [T04-03](#t04-03).
- **Trabalho** · Criar operações de leitura, listas, itens filhos, status, dependências, campos e links, com paginação e validação de escopos.
- **Aceite do ticket** · Contrato com fixtures e lista de teste demonstra pai/filho, status, prioridade e tratamento de erros.

### T06-03

**Projetar subtarefas e status sem duplicação**

- **Dependências** · [T06-01](#t06-01), [T06-02](#t06-02), [T13-04](#t13-04).
- **Trabalho** · Associar IDs externos e registrar efeitos por tarefa/transição/link. Criar subtarefas, dependências e estado mapeado e revalidar alvo antes de repetir.
- **Aceite do ticket** · Timeout após criação não duplica subtarefa. Draft PR publica aguardando merge e conclusão depende de merge confirmado.

### T06-04

**Implementar sincronização manual e preflight**

- **Dependências** · [T05-02](#t05-02), [T06-01](#t06-01), [T06-02](#t06-02), [T13-04](#t13-04).
- **Trabalho** · Ler Jira/ClickUp/GitHub nos pontos D89. Registrar snapshot, alterações, horário, origem e conflitos. Deduplicar importação e reavaliar prioridade/elegibilidade.
- **Aceite do ticket** · Relógio avança ocioso sem leitura periódica de mudanças. Sync manual ou preflight detecta a alteração uma vez.

### T06-05

**Tratar mudança de requisitos e troca de plataforma**

- **Dependências** · [T06-03](#t06-03), [T06-04](#t06-04), [T10-01](#t10-01), [T12-04](#t12-04).
- **Trabalho** · Comparar revisão, classificar impacto, atualizar em marco seguro ou replanejar/solicitar decisão. Criar procedimento de mapear IDs ao trocar origem.
- **Aceite do ticket** · Mudança compatível mantém histórico. Conflito preserva alterações e cria decisão. Troca mantém vínculos históricos e uma origem ativa.

## Feature E07 — Workflows visuais

[Especificação do épico](02-EPICOS.md#e07)

### T07-01

**Construir editor visual de workflows**

- **Dependências** · [T03-03](#t03-03).
- **Trabalho** · Criar canvas, nós, portas, conexões, validação visual e editor de propriedades. Persistir e recarregar rascunhos.
- **Aceite do ticket** · Montar fluxo complexo, salvar e reabrir mantendo nós, conexões e propriedades. Exibir erro com nó e campo relevantes.

### T07-02

**Executar nós e estruturas de controle**

- **Dependências** · [T07-01](#t07-01), [T03-02](#t03-02).
- **Trabalho** · Definir engine e schema para agentes, skills, scripts, checks, Git e docs. Implementar condições, loops, espera, erros, decisão, ramificação, convergência, variáveis e subfluxos.
- **Aceite do ticket** · Testes de grafo percorrem caminhos e convergência, retomam espera persistida e bloqueiam loop sem limite ou subfluxo recursivo.

### T07-03

**Publicar e congelar configuração efetiva**

- **Dependências** · [T07-02](#t07-02), [T08-01](#t08-01), [T09-02](#t09-02).
- **Trabalho** · Versionar workflows e subfluxos, validar referências, resolver parâmetros, guardar snapshot sanitizado e mostrar versão efetiva.
- **Aceite do ticket** · Rascunho não executa. Publicação nova não muda execução existente. Revalidação em runtime respeita revogação.

### T07-04

**Selecionar fluxo e subfluxo por regras**

- **Dependências** · [T07-03](#t07-03).
- **Trabalho** · Criar regra ordenada, valores normalizados, workflow e subfluxo padrão, simulação de seleção e rastreio.
- **Aceite do ticket** · Casos com duas regras e nenhum match retornam a escolha determinística esperada, com motivo persistido.

## Feature E08 — Catálogos e MCP

[Especificação do épico](02-EPICOS.md#e08)

### T08-01

**Gerenciar agentes e sobrescritas**

- **Dependências** · [T03-03](#t03-03), [T04-03](#t04-03), [T09-01](#t09-01).
- **Trabalho** · Cadastrar instruções, provedor, conexão, modelo, skills, ferramentas e limites. Resolver overrides por nó com referência à base.
- **Aceite do ticket** · Dois workflows escolhem configurações diferentes e catálogo original permanece íntegro. Execução mostra configuração efetiva.

### T08-02

**Gerenciar skills e importações**

- **Dependências** · [T03-02](#t03-02), [T04-03](#t04-03), [T13-01](#t13-01).
- **Trabalho** · Importar manualmente, por arquivo/pacote e Git. Validar origem, caminhos, revisão e manifesto. Preparar executor IA, script e determinístico.
- **Aceite do ticket** · Importações ficam reutilizáveis e auditáveis. Pacote malicioso falha sem executar script. Exercitar skill híbrida e nó isolado.

### T08-03

**Criar catálogo de ferramentas**

- **Dependências** · [T03-02](#t03-02), [T13-01](#t13-01).
- **Trabalho** · Modelar ferramentas nativas e personalizadas, parâmetros, resultados, permissões, limites e integração com executor.
- **Aceite do ticket** · Chamada válida devolve saída tipada. Parâmetro ou escopo indevido falha antes de qualquer efeito.

### T08-04

**Cadastrar MCP, descobrir e autorizar ferramentas**

- **Dependências** · [T08-03](#t08-03), [T04-03](#t04-03).
- **Trabalho** · Configurar servidor, transporte, conexão, descoberta e allowlist por agente. Normalizar schemas, revalidar cada chamada e auditar.
- **Aceite do ticket** · Descoberta lista ferramentas. O agente recebe e invoca apenas a selecionada. Nova ferramenta não ganha acesso automaticamente.

## Feature E09 — Provedores de IA e orçamento

[Especificação do épico](02-EPICOS.md#e09)

### T09-01

**Cadastrar contas e adaptar clientes CLI**

- **Dependências** · [T04-03](#t04-03), [T11-01](#t11-01).
- **Trabalho** · Gerenciar conexões nomeadas. Adaptar Codex CLI e Claude Code com autenticação oficial, isolamento, capacidade, eventos e cancelamento. Validar uso efetivo da conta.
- **Aceite do ticket** · Duas contas do mesmo provedor são distintas e selecionáveis. Rodar alteração e checks no sandbox com modo autorizado comprovado.

### T09-02

**Cadastrar modelos e adaptar gateways de API**

- **Dependências** · [T04-03](#t04-03), [T08-03](#t08-03), [T11-01](#t11-01).
- **Trabalho** · Cadastrar modelos manualmente, validar parâmetros, conta e capacidade. Adaptar OpenRouter e Vercel AI Gateway com interface de ferramentas controladas.
- **Aceite do ticket** · Modelo só é usado quando compatível. Cada gateway roda edição e teste com modelo elegível. Sobreposição modelo/agente/nó é verificável.

### T09-03

**Impor orçamento e registrar uso**

- **Dependências** · [T09-01](#t09-01), [T09-02](#t09-02), [T03-02](#t03-02).
- **Trabalho** · Implementar cotas/estimativas, reserva de custo, limites globais e opcionais por agente, reconciliação e fontes.
- **Aceite do ticket** · Chamadas concorrentes disputam reserva sem ultrapassar teto autorizado. Cota de assinatura é registrada separadamente de gasto em API.

### T09-04

**Implementar fallback e disponibilidade**

- **Dependências** · [T09-03](#t09-03), [T12-02](#t12-02).
- **Trabalho** · Registrar estados e falhas, combinar sinais oficiais e histórico, seguir cadeia explícita após quiescência e checkpoint e reagendar com backoff.
- **Aceite do ticket** · Cota no principal leva só à alternativa autorizada. Esgotamento preserva tarefa e outra avança. Disponibilidade volta sem rodízio automático.

## Feature E10 — Planejamento, fila e controles

[Especificação do épico](02-EPICOS.md#e10)

### T10-01

**Planejar e versionar decomposição da feature**

- **Dependências** · [T06-03](#t06-03), [T07-04](#t07-04), [T12-02](#t12-02).
- **Trabalho** · Definir contrato do planejador, critérios, risco inicial, DAG e identidades estáveis. Validar plano e relacionar subtarefas externas.
- **Aceite do ticket** · Feature vira tarefas verificáveis, sem ciclo ou efeito duplicado no retry. Plano conflitante preserva versão anterior.

### T10-02

**Implementar fila local por projeto**

- **Dependências** · [T10-01](#t10-01), [T03-02](#t03-02).
- **Trabalho** · Aplicar pré-requisitos, prioridade, correção de PR e exclusão por projeto. Representar espera por merge, provedor, decisão e recursos.
- **Aceite do ticket** · Só tarefa elegível inicia. PR corrigida ganha próxima vaga elegível. Estado bloqueado libera espaço após checkpoint seguro.

### T10-03

**Aplicar limite global, leases e justiça**

- **Dependências** · [T10-02](#t10-02), [T11-03](#t11-03).
- **Trabalho** · Reclamar vagas em transação, com heartbeat e token de geração. Aplicar prioridade e envelhecimento e atualização sem preempção.
- **Aceite do ticket** · Com jobs concorrentes, um por projeto e total configurado. Escritor obsoleto não grava. Projeto elegível antigo consegue vaga.

### T10-04

**Implementar ativação, pausa e retomada operacional**

- **Dependências** · [T10-03](#t10-03), [T12-01](#t12-01).
- **Trabalho** · Persistir controles globais e por projeto. Drenar feature em curso ou suspender agora com checkpoint, auditar e retomar.
- **Aceite do ticket** · As duas ações têm efeitos diferentes e persistem no reboot. Projetos não afetados continuam executando.

## Feature E11 — Sandbox e processos

[Especificação do épico](02-EPICOS.md#e11)

### T11-01

**Provisionar sandbox e perfis de execução**

- **Dependências** · [T02-02](#t02-02), [T03-01](#t03-01).
- **Trabalho** · Criar container/workspace efêmero por feature, montagem e rede restritas, perfis de stacks, quotas e identidade de supervisor.
- **Aceite do ticket** · Clone e build funcionam no ambiente. Execução de A não alcança B, host, banco ou rede administrativa.

### T11-02

**Executar ferramentas e skills com limites**

- **Dependências** · [T11-01](#t11-01), [T08-02](#t08-02), [T08-03](#t08-03).
- **Trabalho** · Executar operações de leitura/escrita, Git, scripts, testes e chamadas brokeradas com identidade e política por operação.
- **Aceite do ticket** · Ferramentas autorizadas funcionam em ambos os modos de IA. Path traversal, symlinks e tentativa de acesso ao socket falham.

### T11-03

**Supervisionar processos, timeout e inatividade**

- **Dependências** · [T11-01](#t11-01), [T13-01](#t13-01).
- **Trabalho** · Monitorar tempo e sinais de atividade. Encerrar a árvore de processos com escalonamento, confirmar parada e retornar estado auditável.
- **Aceite do ticket** · Processo legítimo silencioso continua. Timeout com filhos encerra a árvore. Estado incerto bloqueia reutilização do workspace.

## Feature E12 — Contexto e recuperação

[Especificação do épico](02-EPICOS.md#e12)

### T12-01

**Persistir e restaurar checkpoints íntegros**

- **Dependências** · [T03-02](#t03-02), [T11-03](#t11-03).
- **Trabalho** · Capturar patch, arquivos não rastreados, SHAs, estado dos nós, referências e manifesto. Gravar duravelmente e liberar recursos após parada confirmada.
- **Aceite do ticket** · Restaurar em container novo e comparar hashes. Interromper gravação de snapshot mantém checkpoint anterior consistente.

### T12-02

**Compor contexto seletivo e renovar sessão**

- **Dependências** · [T07-02](#t07-02), [T12-01](#t12-01).
- **Trabalho** · Montar contexto de artefatos, requisitos, critérios e memória por contrato. Registrar fontes e criar checkpoint em marcos e saturação.
- **Aceite do ticket** · Nova sessão retoma com artefatos e contexto verificáveis. Testar estimativa sem métrica oficial e ausência de segredos.

### T12-03

**Recuperar execução após falha ou reboot**

- **Dependências** · [T10-03](#t10-03), [T12-01](#t12-01), [T13-04](#t13-04).
- **Trabalho** · Detectar tentativas interrompidas, validar estado e quiescência, diagnosticar falhas e aplicar limites separados de infraestrutura/IA/revisão.
- **Aceite do ticket** · Reboot e crash retomam automaticamente quando seguros. Incerteza abre decisão e outras tarefas elegíveis seguem.

### T12-04

**Implementar retomada seletiva e reconciliação de insumos**

- **Dependências** · [T07-03](#t07-03), [T06-04](#t06-04), [T12-01](#t12-01), [T13-04](#t13-04).
- **Trabalho** · Selecionar nó, validar checkpoints, permissões e efeitos. Comparar refs, requisitos e configs. Incorporar mudanças compatíveis e invalidar dependentes quando saída mudar.
- **Aceite do ticket** · Reexecutar nó altera só o conjunto afetado. Conflito preserva edições humanas e o workflow original continua fixado.

## Feature E13 — Autorizações e efeitos

[Especificação do épico](02-EPICOS.md#e13)

### T13-01

**Aplicar política de risco no executor**

- **Dependências** · [T03-02](#t03-02), [T04-03](#t04-03).
- **Trabalho** · Classificar ação e alvo e aplicar allowlist, proibição e decisão conforme parâmetros e escopo. Redigir trilha.
- **Aceite do ticket** · Suite prova baixo risco permitido, sensível encaminhado e proibido recusado em scripts, API e MCP.

### T13-02

**Implementar pedidos e decisões humanas**

- **Dependências** · [T13-01](#t13-01), [T03-03](#t03-03).
- **Trabalho** · Modelar solicitação, evidência, opções, resolução, histórico e sinal de retomada. Ligar Central no painel.
- **Aceite do ticket** · Resolver pendência por UI altera só execução afetada. Decisão repetida ou sobre versão obsoleta é tratada de forma consistente.

### T13-03

**Controlar escopo, expiração e revogação**

- **Dependências** · [T13-02](#t13-02), [T11-03](#t11-03), [T12-01](#t12-01).
- **Trabalho** · Vincular parâmetros e recursos canônicos, validade e usos. Consumir atomicamente. Revogar e disparar parada segura/checkpoint quando aplicável.
- **Aceite do ticket** · Corrida de último uso autoriza um. Argumento ou destino divergente, prazo expirado e autorização revogada impedem a operação.

### T13-04

**Registrar e reconciliar efeitos externos**

- **Dependências** · [T03-02](#t03-02), [T13-01](#t13-01).
- **Trabalho** · Criar ledger de intenção e chave de negócio. Definir provas e classificação confirmado, ausente ou inconclusivo.
- **Aceite do ticket** · Timeout após efeito confirmado não duplica. Ausência segura permite repetir. Estado inconclusivo abre bloqueio com evidência.

### T13-05

**Compensar efeitos parciais por risco**

- **Dependências** · [T13-03](#t13-03), [T13-04](#t13-04).
- **Trabalho** · Implementar compensações idempotentes de baixo risco, autorização sensível e preservação de efeitos irreversíveis.
- **Aceite do ticket** · Compensação executada ou recusada fica auditável. Falha conserva trabalho e bloqueia nova operação agravante.

## Feature E14 — Qualidade e revisão

[Especificação do épico](02-EPICOS.md#e14)

### T14-01

**Executar baseline, critérios e checks**

- **Dependências** · [T05-03](#t05-03), [T11-02](#t11-02).
- **Trabalho** · Executar checks no baseline viável e revisão candidata, guardar evidências e classificar falhas.
- **Aceite do ticket** · Regressão bloqueia. Falha anterior comprovada é registrada. Origem desconhecida fica em investigação e não vira baseline por suposição.

### T14-02

**Classificar e reavaliar risco**

- **Dependências** · [T10-01](#t10-01), [T14-01](#t14-01).
- **Trabalho** · Combinar estimativa da IA e pisos determinísticos. Reavaliar diffs e justificar nível final.
- **Aceite do ticket** · Fixture de arquivo crítico eleva nível apesar de sugestão baixa. Três níveis escolhem verificações previstas.

### T14-03

**Executar revisão independente e correções**

- **Dependências** · [T07-03](#t07-03), [T09-04](#t09-04), [T14-02](#t14-02), [T12-03](#t12-03).
- **Trabalho** · Configurar revisor/fallback, separar revisão inicial, classificar achados, permitir correção e revalidar. Definir limite padrão de duas rodadas.
- **Aceite do ticket** · Crítico/relevante bloqueiam. Baixa severidade fica registrada. Correção do revisor roda checks e esgotamento suspende com checkpoint.

### T14-04

**Criar testes e rodar E2E por impacto**

- **Dependências** · [T14-01](#t14-01), [T14-02](#t14-02).
- **Trabalho** · Orientar criação/atualização de testes e configurar navegador isolado. Associar execuções, fixtures e capturas à revisão.
- **Aceite do ticket** · Jornada de alto risco testável tem E2E real. Caso inaplicável tem justificativa e evidência alternativa.

## Feature E15 — GitHub, documentação e memória

[Especificação do épico](02-EPICOS.md#e15)

### T15-01

**Integrar GitHub, branches, commits e push**

- **Dependências** · [T04-03](#t04-03), [T11-01](#t11-01), [T13-04](#t13-04).
- **Trabalho** · Separar interface de conexão/clone e interface de efeitos. Configurar branch por feature, commits por tarefa validada e push restrito. Confirmar refs.
- **Aceite do ticket** · Conexão valida clone. Commit corresponde a tarefa e checks. Simular timeout de push e prova por ref sem duplicidade ou alteração direta da base.

### T15-02

**Gerar documentação viva e publicar Draft PR**

- **Dependências** · [T15-01](#t15-01), [T14-03](#t14-03), [T14-04](#t14-04).
- **Trabalho** · Criar gate de docs por módulo e histórico de feature, incluir evidências, consolidar commits e abrir Draft PR idempotente.
- **Aceite do ticket** · Repo real contém docs e testes coerentes. Repetição encontra a mesma Draft PR e ausência de documentação bloqueia entrega.

### T15-03

**Confirmar merge e propagar conclusão**

- **Dependências** · [T15-02](#t15-02), [T06-03](#t06-03), [T06-04](#t06-04), [T10-02](#t10-02).
- **Trabalho** · Consultar estado em ponto D89, confirmar PR integrada, registrar SHA, projetar status na origem e liberar dependentes com evidência.
- **Aceite do ticket** · Draft PR aberta ou fechada sem merge não libera dependentes. Merge humano observado libera e eventual falha de status fica visível e reconciliável.

### T15-04

**Corrigir revisão externa e tratar PR fechada**

- **Dependências** · [T15-03](#t15-03), [T13-02](#t13-02), [T10-02](#t10-02).
- **Trabalho** · Descobrir comentários autorizados na consulta prevista, preferir próxima vaga, corrigir branch, atualizar docs e checks. Em fechamento sem merge, oferecer três ações.
- **Aceite do ticket** · Mesmo comentário não duplica trabalho. PR fechada conserva artefatos. Cancelar, reabrir e nova tentativa controlada preservam histórico e respeitam decisão.

### T15-05

**Manter memória técnica pendente e oficial**

- **Dependências** · [T15-03](#t15-03), [T12-02](#t12-02).
- **Trabalho** · Extrair conhecimento de entrega, deduplicar e vincular evidências. Disponibilizar pendente por branch e promover informação validada no merge.
- **Aceite do ticket** · Projeto A não consome memória de B. PR fechada não promove. Merge promove só registros válidos para a revisão integrada.

## Feature E16 — Operação, uso e retenção

[Especificação do épico](02-EPICOS.md#e16)

### T16-01

**Exibir status, controle e logs em tempo real**

- **Dependências** · [T03-03](#t03-03), [T10-04](#t10-04), [T12-03](#t12-03).
- **Trabalho** · Adicionar resumo e detalhe de projeto, etapas, estados, agentes, tempos, histórico, logs SSE e reconexão.
- **Aceite do ticket** · Desconexão/reconexão recupera eventos pelo cursor. Estado bloqueado explica causa e painel exibe controles persistidos.

### T16-02

**Consolidar Central e auditoria operacional**

- **Dependências** · [T13-03](#t13-03), [T13-05](#t13-05), [T15-04](#t15-04), [T16-01](#t16-01).
- **Trabalho** · Expor pendências, decisão, expiração, revogação, evidências e sequência do incidente e da retomada.
- **Aceite do ticket** · Aprovar, recusar ou revogar pelo painel produz evento e estado verificáveis sem notificação externa.

### T16-03

**Exibir consumo, orçamento e provedores**

- **Dependências** · [T09-03](#t09-03), [T09-04](#t09-04), [T03-03](#t03-03).
- **Trabalho** · Mostrar limites, reservas, valores estimados/reportados, cota, disponibilidade e desconhecidos.
- **Aceite do ticket** · Valor não observado aparece explicitamente. Teto global e opcional por agente refletem o bloqueio de novas chamadas.

### T16-04

**Automatizar retenção com integridade**

- **Dependências** · [T12-01](#t12-01), [T16-01](#t16-01).
- **Trabalho** · Aplicar 30 dias para logs detalhados, carência para concluídos e preservação de resumo e checkpoints protegidos. Auditar limpeza.
- **Aceite do ticket** · Relógio de teste remove apenas itens elegíveis. Checkpoint bloqueado ou vinculado a incidente continua íntegro e restaurável.

## Feature E17 — Piloto e aceite

[Especificação do épico](02-EPICOS.md#e17)

### T17-01

**Preparar projeto real e matriz de evidências**

- **Dependências** · [T01-04](#t01-04), [T02-03](#t02-03), [T05-03](#t05-03).
- **Trabalho** · Selecionar projeto existente e feature real, levantar baseline, revisão, critérios, dependências e acessos. Preparar cenários e evidência por épico.
- **Aceite do ticket** · Contrato preenchido e revisável, com ambiente, projeto, usuários, conexões e critérios rastreáveis.

### T17-02

**Executar cadastro do zero e feature ponta a ponta**

- **Dependências** · [T17-01](#t17-01), [T15-05](#t15-05), [T16-02](#t16-02), [T16-03](#t16-03).
- **Trabalho** · Cadastrar todos os recursos manualmente. Importar, planejar, implementar, testar, revisar, documentar e entregar Draft PR real. Validar ciclo pós-merge humano.
- **Aceite do ticket** · A PR, commits, checks, subtarefas, memória e dependência são verificáveis no ambiente real e ligados à revisão.

### T17-03

**Executar ensaios de falha, segurança e integração**

- **Dependências** · [T17-01](#t17-01), [T02-04](#t02-04), [T06-05](#t06-05), [T09-04](#t09-04), [T12-04](#t12-04), [T13-05](#t13-05), [T15-04](#t15-04), [T16-04](#t16-04).
- **Trabalho** · Executar os cenários de recuperação, idempotência, conflitos, isolamento, concorrência, fallback, orçamento, aprovações e ambos os conectores.
- **Aceite do ticket** · Cada cenário tem esperado, observado, evidência e revisão. Falha obrigatória volta ao ticket e é retestada.

### T17-04

**Fechar aceite integral do MVP**

- **Dependências** · [T17-02](#t17-02), [T17-03](#t17-03).
- **Trabalho** · Verificar todos os critérios, dependências, decisões vigentes e sobreposição de testes. Consolidar relatório e obter decisão do responsável.
- **Aceite do ticket** · Todos os épicos possuem evidência e aceite. Não há requisito obrigatório omitido ou pendente no lançamento.
