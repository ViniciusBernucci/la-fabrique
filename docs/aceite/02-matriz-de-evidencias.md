# Matriz de evidências do aceite

> **Gerado** por `make matriz` a partir de `NEW MVP/05-ACEITE-E-PILOTO.md`, `03-BACKLOG.md` e `07-CHECKLIST.md`. Não edite à mão: um teste confere que este arquivo é igual ao gerado.

A coluna **Evidência no projeto real** fica *pendente* até alguém registrá-la ([ficha de preparação](01-ficha-de-preparacao.md)). Evidência de software (testes e entregas) **não** substitui a do piloto: o aceite exige o projeto real (D95).

**Situação dos tickets:** 19 feito(s), 45 parcial(is), 3 pendente(s), de 67.

## A. Casos de verificação (P01–P18)

| Caso | Épicos | Evidência mínima exigida | Evidência de software hoje | Evidência no projeto real |
|---|---|---|---|---|
| **P01** · Bootstrap e CI | E01, E02 | Logs de CI, configuração sanitizada, revisão e digest. | E01: 2 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T01-02, T01-03)<br>E02: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T02-01, T02-02, T02-03, T02-04) | pendente |
| **P02** · VPS e restauração | E02, E03, E04, E12 | Medidas de capacidade, inventário, eventos de reboot, restore e hashes. | E02: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T02-01, T02-02, T02-03, T02-04)<br>E03: 3 feito(s), 0 parcial(is), 0 pendente(s)<br>E04: 2 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T04-03)<br>E12: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T12-01, T12-02, T12-03, T12-04) | pendente |
| **P03** · Autenticação e cofre | E03, E04 | Sessões e auditoria sanitizadas, teste negativo e resultado de busca com segredo sentinela. | E03: 3 feito(s), 0 parcial(is), 0 pendente(s)<br>E04: 2 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T04-03) | pendente |
| **P04** · Configuração inicial | E05, E07, E08, E09 | Estado antes/depois, mapa de integrações e snapshot efetivo. | E05: 1 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T05-02, T05-03)<br>E07: 3 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T07-03)<br>E08: 2 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T08-02, T08-04)<br>E09: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T09-01, T09-02, T09-03, T09-04) | pendente |
| **P05** · Dois conectores | E05, E06, E13, E15 | IDs remotos, snapshots, transições e uma única subtarefa por chave. | E05: 1 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T05-02, T05-03)<br>E06: 1 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T06-01, T06-02, T06-04, T06-05)<br>E13: 4 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T13-05)<br>E15: 0 feito(s), 5 parcial(is), 0 pendente(s) (abertos: T15-01, T15-02, T15-03, T15-04, T15-05) | pendente |
| **P06** · Workflow completo | E07, E08, E12, E13 | Grafo publicado, saídas por caminho e versões da execução. | E07: 3 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T07-03)<br>E08: 2 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T08-02, T08-04)<br>E12: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T12-01, T12-02, T12-03, T12-04)<br>E13: 4 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T13-05) | pendente |
| **P07** · Catálogos e MCP | E07, E08, E11, E13 | Proveniência e revisão da skill, chamadas e recusas sanitizadas. | E07: 3 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T07-03)<br>E08: 2 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T08-02, T08-04)<br>E11: 1 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T11-01, T11-02)<br>E13: 4 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T13-05) | pendente |
| **P08** · IA, orçamento e disponibilidade | E08, E09, E11, E12, E16 | Matriz de capacidades, conta/modelo efetivos, uso/limite, diff e checkpoint preservados. | E08: 2 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T08-02, T08-04)<br>E09: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T09-01, T09-02, T09-03, T09-04)<br>E11: 1 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T11-01, T11-02)<br>E12: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T12-01, T12-02, T12-03, T12-04)<br>E16: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T16-01, T16-02, T16-03, T16-04) | pendente |
| **P09** · Fila justa e controles | E10, E11, E16 | Timeline com vaga, lease, prioridades, pausas, espera e conclusões. | E10: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T10-01, T10-02, T10-03, T10-04)<br>E11: 1 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T11-01, T11-02)<br>E16: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T16-01, T16-02, T16-03, T16-04) | pendente |
| **P10** · Isolamento e processos | E04, E08, E11, E13 | Permissões, logs de recusa, árvore/processos encerrados e recursos liberados. | E04: 2 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T04-03)<br>E08: 2 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T08-02, T08-04)<br>E11: 1 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T11-01, T11-02)<br>E13: 4 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T13-05) | pendente |
| **P11** · Recuperação e sessões | E02, E10, E12, E13 | Hashes, manifesto, versão fixada, tentativas, retomada e decisão em falha. | E02: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T02-01, T02-02, T02-03, T02-04)<br>E10: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T10-01, T10-02, T10-03, T10-04)<br>E12: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T12-01, T12-02, T12-03, T12-04)<br>E13: 4 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T13-05) | pendente |
| **P12** · Aprovação e revogação | E04, E11, E12, E13, E16 | Escopo sanitizado, prazos, usos, evento de revogação e checkpoint. | E04: 2 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T04-03)<br>E11: 1 feito(s), 2 parcial(is), 0 pendente(s) (abertos: T11-01, T11-02)<br>E12: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T12-01, T12-02, T12-03, T12-04)<br>E13: 4 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T13-05)<br>E16: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T16-01, T16-02, T16-03, T16-04) | pendente |
| **P13** · Efeitos e compensação | E06, E12, E13, E15 | IDs remotos únicos, ledger com provas, autorização e preservação de artefatos. | E06: 1 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T06-01, T06-02, T06-04, T06-05)<br>E12: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T12-01, T12-02, T12-03, T12-04)<br>E13: 4 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T13-05)<br>E15: 0 feito(s), 5 parcial(is), 0 pendente(s) (abertos: T15-01, T15-02, T15-03, T15-04, T15-05) | pendente |
| **P14** · Qualidade | E14, E15 | Diffs, baseline, evidência dos checks por revisão, relatório de revisão e captura quando útil. | E14: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T14-01, T14-02, T14-03, T14-04)<br>E15: 0 feito(s), 5 parcial(is), 0 pendente(s) (abertos: T15-01, T15-02, T15-03, T15-04, T15-05) | pendente |
| **P15** · GitHub, docs e memória | E06, E10, E13, E15 | PR/commits, link na origem, revisão, decisões, docs, estado de memória e fila. | E06: 1 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T06-01, T06-02, T06-04, T06-05)<br>E10: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T10-01, T10-02, T10-03, T10-04)<br>E13: 4 feito(s), 1 parcial(is), 0 pendente(s) (abertos: T13-05)<br>E15: 0 feito(s), 5 parcial(is), 0 pendente(s) (abertos: T15-01, T15-02, T15-03, T15-04, T15-05) | pendente |
| **P16** · D89 e mudanças externas | E06, E10, E12, E15, E16 | Contagem de chamadas por gatilho, snapshots/diffs, histórico de fila e trilha de decisão. | E06: 1 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T06-01, T06-02, T06-04, T06-05)<br>E10: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T10-01, T10-02, T10-03, T10-04)<br>E12: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T12-01, T12-02, T12-03, T12-04)<br>E15: 0 feito(s), 5 parcial(is), 0 pendente(s) (abertos: T15-01, T15-02, T15-03, T15-04, T15-05)<br>E16: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T16-01, T16-02, T16-03, T16-04) | pendente |
| **P17** · Painel e retenção | E03, E09, E12, E16 | Eventos, estado de UI, dados retidos, limpeza e restauração de checkpoint protegido. | E03: 3 feito(s), 0 parcial(is), 0 pendente(s)<br>E09: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T09-01, T09-02, T09-03, T09-04)<br>E12: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T12-01, T12-02, T12-03, T12-04)<br>E16: 0 feito(s), 4 parcial(is), 0 pendente(s) (abertos: T16-01, T16-02, T16-03, T16-04) | pendente |
| **P18** · Aceite | E17 | Relatório assinado pelo responsável ou decisão registrada, matriz e links da revisão. | E17: 0 feito(s), 1 parcial(is), 3 pendente(s) (abertos: T17-01, T17-02, T17-03, T17-04) | pendente |

## B. Critérios por épico e ticket

### E01 — Bootstrap e monorepo

2 feito(s), 2 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T01-01** · Registrar arquitetura inicial e ADRs | feito | ADR explica custos, critérios de viabilidade, integração e responsáveis por decisões delegadas. D40 está vinculada ao benchmark. | — | pendente |
| **T01-02** · Criar e configurar repositório GitHub da fábrica | parcial | PR de teste passa pelo fluxo previsto. Push direto sujeito à regra configurada e permissões conferidas com a identidade usada. | — | pendente |
| **T01-03** · Criar layout e ferramentas de desenvolvimento | parcial | Segundo clone reproduz setup e comandos com ferramentas fixadas. Segredos reais não aparecem em arquivo ou histórico Git. | — | pendente |
| **T01-04** · Publicar guias e padrões no repositório | feito | Pessoa nova localiza instruções e critérios pelo índice. Plano de lançamento inclui todos os épicos e decisões vigentes. | — | pendente |

### E02 — VPS, CI e recuperação da instalação

0 feito(s), 4 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T02-01** · Inventariar e medir VPS existente | parcial | Relatório registra medidas, gargalos, parâmetros e capacidade para ensaios de concorrência. Valores sem medida estão identificados. | — | pendente |
| **T02-02** · Configurar ambientes, rede, proxy e containers | parcial | Somente entrada autorizada exposta. Serviços voltam após reboot e homologação não usa credenciais nem volumes de produção. | — | pendente |
| **T02-03** · Automatizar CI e promoção por revisão | parcial | PR com falha é bloqueada. Imagens e evidências apontam ao mesmo commit. Segredo de produção não é disponibilizado a jobs de PR. | — | pendente |
| **T02-04** · Documentar e ensaiar backup e restauração | parcial | Restaurar em destino isolado e conferir sessão/configuração conforme aplicável, capacidade de desencriptar conexões, banco e hashes de artefatos. | [E02 restauração](entregas/2026-11-12-E02-ensaio-de-restauracao.md) | pendente |

### E03 — Fundação web, API e dados

3 feito(s), 0 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T03-01** · Inicializar web, API e worker | feito | Cada serviço inicia em dev e reporta falha de configuração de forma diagnosticável. Build reproduzível em CI. | — | pendente |
| **T03-02** · Implementar banco, migrations e transações | feito | Testes em PostgreSQL real persistem após restart, rejeitam duplicidade e escopo inválido e validam caminho de migração. | — | pendente |
| **T03-03** · Criar navegação centrada em projetos | feito | Links levam às rotas corretas, navegação autenticada funciona e acesso sem sessão é recusado pelo servidor. | — | pendente |

### E04 — Acesso e credenciais

2 feito(s), 1 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T04-01** · Implementar login, TOTP e recuperação | feito | Senha sem 2FA não abre sessão. Recuperação funciona uma vez mesmo com requisições concorrentes. | [2026-11-12-E04](entregas/2026-11-12-E04-acesso-e-credenciais.md) | pendente |
| **T04-02** · Endurecer sessões e ações sensíveis | feito | Sessão revogada perde acesso. Requisição forjada ou sessão sem reautenticação é recusada nas ações sensíveis. | — | pendente |
| **T04-03** · Criar cofre, gestão e teste de conexões | parcial | Rotação mantém vínculos sem expor valor anterior. Revogação é observável antes do próximo uso. Busca com segredo sentinela não acha texto em logs, respostas ou prompts. | — | pendente |

### E05 — Cadastro de projeto

1 feito(s), 2 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T05-01** · Cadastrar projeto em rascunho | feito | Projeto novo vazio de vínculos persiste e retorna ao assistente. Sem preenchimento obrigatório válido, ativação é bloqueada. | [2026-11-12-E05](entregas/2026-11-12-E05-cadastro-de-projeto.md) | pendente |
| **T05-02** · Configurar repositório, origem e elegibilidade | parcial | Conexão incompatível, duas origens ativas e mapeamento inválido falham com motivo. Origem válida devolve preview correto de itens elegíveis. | [E05 adendo](entregas/2026-11-12-E05-cadastro-de-projeto.md) | pendente |
| **T05-03** · Descobrir stack e associar recursos manualmente | parcial | Overrides preservam recurso global. Rascunho de workflow e conta incompatível bloqueiam ativação. Projeto apto pode iniciar execução. | [E05 perfil e recursos](entregas/2026-11-12-E05-perfil-e-recursos.md) | pendente |

### E06 — Integrações e sincronização

1 feito(s), 4 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T06-01** · Implementar adaptador Jira | parcial | Contrato com fixtures e instância de teste distingue itens, subtarefas, transições válidas, paginação, permissão e rate limit. | [E06](entregas/2026-11-12-E06-adaptadores-jira-clickup.md) | pendente |
| **T06-02** · Implementar adaptador ClickUp | parcial | Contrato com fixtures e lista de teste demonstra pai/filho, status, prioridade e tratamento de erros. | [E06](entregas/2026-11-12-E06-adaptadores-jira-clickup.md) | pendente |
| **T06-03** · Projetar subtarefas e status sem duplicação | feito | Timeout após criação não duplica subtarefa. Draft PR publica aguardando merge e conclusão depende de merge confirmado. | [E06 adendo](entregas/2026-11-12-E06-adaptadores-jira-clickup.md) | pendente |
| **T06-04** · Implementar sincronização manual e preflight | parcial | Relógio avança ocioso sem leitura periódica de mudanças. Sync manual ou preflight detecta a alteração uma vez. | [E06 sync](entregas/2026-11-12-E06-sincronizacao-manual-e-preflight.md) | pendente |
| **T06-05** · Tratar mudança de requisitos e troca de plataforma | parcial | Mudança compatível mantém histórico. Conflito preserva alterações e cria decisão. Troca mantém vínculos históricos e uma origem ativa. | [E06 mudança e troca](entregas/2026-11-12-E06-mudanca-de-requisitos-e-troca-de-origem.md) | pendente |

### E07 — Workflows visuais

3 feito(s), 1 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T07-01** · Construir editor visual de workflows | feito | Montar fluxo complexo, salvar e reabrir mantendo nós, conexões e propriedades. Exibir erro com nó e campo relevantes. | [E07](entregas/2026-11-12-E07-editor-de-workflows.md) | pendente |
| **T07-02** · Executar nós e estruturas de controle | feito | Testes de grafo percorrem caminhos e convergência, retomam espera persistida e bloqueiam loop sem limite ou subfluxo recursivo. | [E07 motor](entregas/2026-11-12-E07-motor-de-workflows.md) | pendente |
| **T07-03** · Publicar e congelar configuração efetiva | parcial | Rascunho não executa. Publicação nova não muda execução existente. Revalidação em runtime respeita revogação. | [E07 publicação](entregas/2026-11-12-E07-publicacao-e-congelamento.md) | pendente |
| **T07-04** · Selecionar fluxo e subfluxo por regras | feito | Casos com duas regras e nenhum match retornam a escolha determinística esperada, com motivo persistido. | [E07 seleção](entregas/2026-11-12-E07-selecao-por-regras.md) | pendente |

### E08 — Catálogos e MCP

2 feito(s), 2 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T08-01** · Gerenciar agentes e sobrescritas | feito | Dois workflows escolhem configurações diferentes e catálogo original permanece íntegro. Execução mostra configuração efetiva. | [E08](entregas/2026-11-12-E08-catalogo-de-agentes.md) | pendente |
| **T08-02** · Gerenciar skills e importações | parcial | Importações ficam reutilizáveis e auditáveis. Pacote malicioso falha sem executar script. Exercitar skill híbrida e nó isolado. | [E08 skills](entregas/2026-11-12-E08-catalogo-de-skills.md) | pendente |
| **T08-03** · Criar catálogo de ferramentas | feito | Chamada válida devolve saída tipada. Parâmetro ou escopo indevido falha antes de qualquer efeito. | [E08 ferramentas](entregas/2026-11-12-E08-catalogo-de-ferramentas.md) | pendente |
| **T08-04** · Cadastrar MCP, descobrir e autorizar ferramentas | parcial | Descoberta lista ferramentas. O agente recebe e invoca apenas a selecionada. Nova ferramenta não ganha acesso automaticamente. | [E08 MCP](entregas/2026-11-12-E08-mcp.md) | pendente |

### E09 — Provedores de IA e orçamento

0 feito(s), 4 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T09-01** · Cadastrar contas e adaptar clientes CLI | parcial | Duas contas do mesmo provedor são distintas e selecionáveis. Rodar alteração e checks no sandbox com modo autorizado comprovado. | [E09](entregas/2026-11-12-E09-clientes-cli.md) | pendente |
| **T09-02** · Cadastrar modelos e adaptar gateways de API | parcial | Modelo só é usado quando compatível. Cada gateway roda edição e teste com modelo elegível. Sobreposição modelo/agente/nó é verificável. | [E09 modelos](entregas/2026-11-12-E09-modelos-e-gateways.md) | pendente |
| **T09-03** · Impor orçamento e registrar uso | parcial | Chamadas concorrentes disputam reserva sem ultrapassar teto autorizado. Cota de assinatura é registrada separadamente de gasto em API. | [E09 orçamento](entregas/2026-11-12-E09-orcamento-e-uso.md) | pendente |
| **T09-04** · Implementar fallback e disponibilidade | parcial | Cota no principal leva só à alternativa autorizada. Esgotamento preserva tarefa e outra avança. Disponibilidade volta sem rodízio automático. | [E09 fallback](entregas/2026-11-12-E09-fallback-e-disponibilidade.md) | pendente |

### E10 — Planejamento, fila e controles

0 feito(s), 4 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T10-01** · Planejar e versionar decomposição da feature | parcial | Feature vira tarefas verificáveis, sem ciclo ou efeito duplicado no retry. Plano conflitante preserva versão anterior. | [E10 planejamento](entregas/2026-11-12-E10-planejamento.md) | pendente |
| **T10-02** · Implementar fila local por projeto | parcial | Só tarefa elegível inicia. PR corrigida ganha próxima vaga elegível. Estado bloqueado libera espaço após checkpoint seguro. | [E10 fila](entregas/2026-11-12-E10-fila-local.md) | pendente |
| **T10-03** · Aplicar limite global, leases e justiça | parcial | Com jobs concorrentes, um por projeto e total configurado. Escritor obsoleto não grava. Projeto elegível antigo consegue vaga. | [E10 scheduler](entregas/2026-11-12-E10-scheduler.md) | pendente |
| **T10-04** · Implementar ativação, pausa e retomada operacional | parcial | As duas ações têm efeitos diferentes e persistem no reboot. Projetos não afetados continuam executando. | [E10 controles](entregas/2026-11-12-E10-controles.md) | pendente |

### E11 — Sandbox e processos

1 feito(s), 2 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T11-01** · Provisionar sandbox e perfis de execução | parcial | Clone e build funcionam no ambiente. Execução de A não alcança B, host, banco ou rede administrativa. | [E11](entregas/2026-11-12-E11-sandbox.md) | pendente |
| **T11-02** · Executar ferramentas e skills com limites | parcial | Ferramentas autorizadas funcionam em ambos os modos de IA. Path traversal, symlinks e tentativa de acesso ao socket falham. | [E11 ferramentas](entregas/2026-11-12-E11-execucao-de-ferramentas.md) | pendente |
| **T11-03** · Supervisionar processos, timeout e inatividade | feito | Processo legítimo silencioso continua. Timeout com filhos encerra a árvore. Estado incerto bloqueia reutilização do workspace. | [E11 adendo](entregas/2026-11-12-E11-sandbox.md) | pendente |

### E12 — Contexto e recuperação

0 feito(s), 4 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T12-01** · Persistir e restaurar checkpoints íntegros | parcial | Restaurar em container novo e comparar hashes. Interromper gravação de snapshot mantém checkpoint anterior consistente. | [E12 checkpoints](entregas/2026-11-12-E12-checkpoints.md) | pendente |
| **T12-02** · Compor contexto seletivo e renovar sessão | parcial | Nova sessão retoma com artefatos e contexto verificáveis. Testar estimativa sem métrica oficial e ausência de segredos. | [E12 contexto](entregas/2026-11-12-E12-contexto-e-sessao.md) | pendente |
| **T12-03** · Recuperar execução após falha ou reboot | parcial | Reboot e crash retomam automaticamente quando seguros. Incerteza abre decisão e outras tarefas elegíveis seguem. | [E12 recuperação](entregas/2026-11-12-E12-recuperacao.md) | pendente |
| **T12-04** · Implementar retomada seletiva e reconciliação de insumos | parcial | Reexecutar nó altera só o conjunto afetado. Conflito preserva edições humanas e o workflow original continua fixado. | [E12 retomada](entregas/2026-11-12-E12-retomada-seletiva.md) | pendente |

### E13 — Autorizações e efeitos

4 feito(s), 1 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T13-01** · Aplicar política de risco no executor | feito | Suite prova baixo risco permitido, sensível encaminhado e proibido recusado em scripts, API e MCP. | [E13](entregas/2026-11-12-E13-politica-de-risco.md) | pendente |
| **T13-02** · Implementar pedidos e decisões humanas | feito | Resolver pendência por UI altera só execução afetada. Decisão repetida ou sobre versão obsoleta é tratada de forma consistente. | [E13 decisões](entregas/2026-11-12-E13-decisoes-humanas.md) | pendente |
| **T13-03** · Controlar escopo, expiração e revogação | feito | Corrida de último uso autoriza um. Argumento ou destino divergente, prazo expirado e autorização revogada impedem a operação. | [E13 autorizações](entregas/2026-11-12-E13-autorizacoes.md) | pendente |
| **T13-04** · Registrar e reconciliar efeitos externos | feito | Timeout após efeito confirmado não duplica. Ausência segura permite repetir. Estado inconclusivo abre bloqueio com evidência. | [E13 ledger](entregas/2026-11-12-E13-ledger-de-efeitos.md) | pendente |
| **T13-05** · Compensar efeitos parciais por risco | parcial | Compensação executada ou recusada fica auditável. Falha conserva trabalho e bloqueia nova operação agravante. | [E13 compensação](entregas/2026-11-12-E13-compensacao.md) | pendente |

### E14 — Qualidade e revisão

0 feito(s), 4 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T14-01** · Executar baseline, critérios e checks | parcial | Regressão bloqueia. Falha anterior comprovada é registrada. Origem desconhecida fica em investigação e não vira baseline por suposição. | [E14 checks](entregas/2026-11-12-E14-checks-e-baseline.md) | pendente |
| **T14-02** · Classificar e reavaliar risco | parcial | Fixture de arquivo crítico eleva nível apesar de sugestão baixa. Três níveis escolhem verificações previstas. | [E14 risco](entregas/2026-11-12-E14-risco.md) | pendente |
| **T14-03** · Executar revisão independente e correções | parcial | Crítico/relevante bloqueiam. Baixa severidade fica registrada. Correção do revisor roda checks e esgotamento suspende com checkpoint. | [E14 revisão](entregas/2026-11-12-E14-revisao-independente.md) | pendente |
| **T14-04** · Criar testes e rodar E2E por impacto | parcial | Jornada de alto risco testável tem E2E real. Caso inaplicável tem justificativa e evidência alternativa. | [E14 testes e E2E](entregas/2026-11-12-E14-testes-e-e2e.md) | pendente |

### E15 — GitHub, documentação e memória

0 feito(s), 5 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T15-01** · Integrar GitHub, branches, commits e push | parcial | Conexão valida clone. Commit corresponde a tarefa e checks. Simular timeout de push e prova por ref sem duplicidade ou alteração direta da base. | [E15 GitHub](entregas/2026-11-12-E15-github-branch-commit-push.md) | pendente |
| **T15-02** · Gerar documentação viva e publicar Draft PR | parcial | Repo real contém docs e testes coerentes. Repetição encontra a mesma Draft PR e ausência de documentação bloqueia entrega. | [E15 docs e Draft PR](entregas/2026-11-12-E15-documentacao-e-draft-pr.md) | pendente |
| **T15-03** · Confirmar merge e propagar conclusão | parcial | Draft PR aberta ou fechada sem merge não libera dependentes. Merge humano observado libera e eventual falha de status fica visível e reconciliável. | [E15 merge](entregas/2026-11-12-E15-merge.md) | pendente |
| **T15-04** · Corrigir revisão externa e tratar PR fechada | parcial | Mesmo comentário não duplica trabalho. PR fechada conserva artefatos. Cancelar, reabrir e nova tentativa controlada preservam histórico e respeitam decisão. | [E15 revisão externa](entregas/2026-11-12-E15-revisao-externa-e-pr-fechada.md) | pendente |
| **T15-05** · Manter memória técnica pendente e oficial | parcial | Projeto A não consome memória de B. PR fechada não promove. Merge promove só registros válidos para a revisão integrada. | [E15 memória](entregas/2026-11-12-E15-memoria-tecnica.md) | pendente |

### E16 — Operação, uso e retenção

0 feito(s), 4 parcial(is), 0 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T16-01** · Exibir status, controle e logs em tempo real | parcial | Desconexão/reconexão recupera eventos pelo cursor. Estado bloqueado explica causa e painel exibe controles persistidos. | [E16 execução ao vivo](entregas/2026-11-12-E16-execucao-ao-vivo.md) | pendente |
| **T16-02** · Consolidar Central e auditoria operacional | parcial | Aprovar, recusar ou revogar pelo painel produz evento e estado verificáveis sem notificação externa. | [E16 Central](entregas/2026-11-12-E16-central-e-auditoria.md) | pendente |
| **T16-03** · Exibir consumo, orçamento e provedores | parcial | Valor não observado aparece explicitamente. Teto global e opcional por agente refletem o bloqueio de novas chamadas. | [E16 consumo](entregas/2026-11-12-E16-consumo-e-provedores.md) | pendente |
| **T16-04** · Automatizar retenção com integridade | parcial | Relógio de teste remove apenas itens elegíveis. Checkpoint bloqueado ou vinculado a incidente continua íntegro e restaurável. | [E16 retenção](entregas/2026-11-12-E16-retencao.md) | pendente |

### E17 — Piloto e aceite

0 feito(s), 1 parcial(is), 3 pendente(s).

| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |
|---|---|---|---|---|
| **T17-01** · Preparar projeto real e matriz de evidências | parcial | Contrato preenchido e revisável, com ambiente, projeto, usuários, conexões e critérios rastreáveis. | [E17 matriz e ficha](entregas/2026-11-12-E17-matriz-e-ficha-de-aceite.md) | pendente |
| **T17-02** · Executar cadastro do zero e feature ponta a ponta | pendente | A PR, commits, checks, subtarefas, memória e dependência são verificáveis no ambiente real e ligados à revisão. | — | pendente |
| **T17-03** · Executar ensaios de falha, segurança e integração | pendente | Cada cenário tem esperado, observado, evidência e revisão. Falha obrigatória volta ao ticket e é retestada. | — | pendente |
| **T17-04** · Fechar aceite integral do MVP | pendente | Todos os épicos possuem evidência e aceite. Não há requisito obrigatório omitido ou pendente no lançamento. | — | pendente |

