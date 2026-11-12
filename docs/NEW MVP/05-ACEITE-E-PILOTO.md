# Aceite integral e piloto real

**Estado:** planejado. O preenchimento das evidências ocorre durante o desenvolvimento e a validação conduzidos pelo responsável.

A D42 exige entrega integrada de todos os épicos. A D95 exige cadastro integral de projeto real, uma feature ponta a ponta até Draft PR e ensaios de recuperação, aprovações, isolamento e sincronização com evidências. O piloto usa um projeto real existente (D43). Os ensaios controlados complementam o piloto quando a condição exigiria degradar um serviço real, provocar efeito irreversível ou depender de cotas externas.

## Ficha de preparação

| Campo | Preencher antes do ensaio |
|---|---|
| Responsável pelo aceite | Pessoa que configura o painel, decide exceções e realiza o merge humano. |
| Revisão da fábrica | Commit, digests de imagens, schema do banco, configuração de ambiente e data. |
| VPS e ambiente | Instância existente, capacidade medida, limites de concorrência e sandbox, localização de backups, TLS, namespace/volumes de homologação. |
| Projeto e feature | Repositório existente, branch base e SHA, feature real, chave externa, prioridade, escopo, critérios verificáveis, tarefas e dependências a demonstrar. |
| Origem oficial | Jira ou ClickUp, projeto/lista, campo de elegibilidade, mapas de status e prioridade, e conexão com permissões verificadas. Preparar ambiente de teste da outra origem. |
| GitHub | Repositório, acesso restrito, branch de teste e identidades de revisão autorizadas. Provar que Draft PR é suportada no repo e que a conta da fábrica não faz merge pelo produto. |
| Configuração de desenvolvimento | Perfil da stack alvo, caminhos autorizados, comandos de instalação/checks/start e serviços auxiliares com dados de teste. Registrar baseline e falhas existentes. |
| Catálogos | Agentes, skills, ferramentas, MCP, conexões, modelos e cadeia de fallback por agente. Registre o modo e os escopos verificados. |
| Orçamento e risco | Teto de API, limite opcional por agente, uso de fixture para indisponibilidade, regras de aprovação, prazos, usos e destinos. |
| Workflows | Versões, padrão e regras dos dois níveis, revisor, condições, loops, esperas, convergências, falha e aprovação. |
| Ambientes de ensaio | Espaços isolados para segundo projeto, falhas, concorrência, revisão externa, restore e testes de revogação. |
| Evidências | Caminho de armazenamento, política de acesso, responsável, prazo e índice dos arquivos/links. |

## Roteiro do projeto real

1. Iniciar sessão com 2FA. Cadastrar o projeto do zero e confirmar ausência de agentes e fluxos associados. Conectar GitHub e escolher uma origem. Configurar marcador, campos, status, dependências, prioridade e comandos a partir do repositório.
2. Cadastrar conexões CLI e API, modelos manualmente, agentes, skills, ferramentas e MCP. Escolher uma conta de cada agente e uma cadeia de fallback explícita. Criar, validar, publicar e associar o fluxo principal e os subfluxos no editor. Registrar snapshot de configuração efetiva. Ativar após validação.
3. Marcar a feature na origem e usar o gatilho permitido pela D89. Conferir importação única, plano estruturado e subtarefas externas. Testar uma dependência técnica antes de sua liberação e uma dependência de merge. Identificar fontes de contexto e critérios usados pelo planejador.
4. Acompanhar fila, etapas, logs e tempos. Verificar ambiente isolado, baseline, código, testes, risco final e revisão quando exigida. Acionar ao menos uma decisão ou aprovação coerente com o workflow e registrar a retomada. Guardar evidência de resultado real.
5. Conferir commits por tarefa, validação consolidada, docs vivas e histórico da feature no mesmo repo. Observar Draft PR criada e status **Aguardando merge** na origem. Verificar memória pendente e dependente bloqueada.
6. Usar uma solicitação de revisão de autor autorizado e sincronização explícita para descobrir a pendência. Conferir correção na mesma branch, novos checks e documentação pertinente. Em cenário controlado, testar também fechamento sem merge conforme P15.
7. Realizar merge humano após aprovação da mudança no repositório piloto. Usar consulta autorizada para confirmar merge. Conferir SHA integrado, atualização externa, desbloqueio de dependentes, validação e promoção de conhecimento oficial.
8. Indexar logs e evidências por revisão. Registrar divergências entre observado e esperado, corrigir e repetir os cenários afetados antes do aceite.

## Casos de verificação

| Caso | Procedimento e resultado exigido | Evidência mínima | Épicos |
|---|---|---|---|
| P01 · Bootstrap e CI | Fazer clone limpo, preparar dev, abrir PR de teste com falha e depois revisão válida. Conferir proteção, build e rastreio por digest. | Logs de CI, configuração sanitizada, revisão e digest. | E01, E02 |
| P02 · VPS e restauração | Inspecionar rede e volumes, executar carga, reboot e restaurar backup em ambiente isolado. Dados, checkpoints e chaves recuperáveis mantêm integridade. | Medidas de capacidade, inventário, eventos de reboot, restore e hashes. | E02, E03, E04, E12 |
| P03 · Autenticação e cofre | Testar login sem TOTP, TOTP válido, recuperação de uso único, sessão revogada, rotação/revogação e redação de segredos. | Sessões e auditoria sanitizadas, teste negativo e resultado de busca com segredo sentinela. | E03, E04 |
| P04 · Configuração inicial | Cadastrar projeto vazio, tentar ativar com origem/conexão/workflow inválido, ajustar manualmente, ativar e comparar configurações com outro projeto. | Estado antes/depois, mapa de integrações e snapshot efetivo. | E05, E07, E08, E09 |
| P05 · Dois conectores | No projeto real usar uma origem. Em ambiente dedicado da outra, repetir importação, subtarefas, status, dependências, link, falha e retry. Simular troca de origem com mapeamento de IDs. | IDs remotos, snapshots, transições e uma única subtarefa por chave. | E05, E06, E13, E15 |
| P06 · Workflow completo | Montar fluxo visual com os nós aprovados. Executar IF/ELSE, loop, espera, erro, decisão, ramificações, convergência, variáveis e subfluxos. Publicar nova versão durante execução anterior. | Grafo publicado, saídas por caminho e versões da execução. | E07, E08, E12, E13 |
| P07 · Catálogos e MCP | Usar skill manual, arquivo e Git. Testar associação e nó isolado. Cadastrar servidor, descobrir e autorizar ferramenta individual. Testar chamada bloqueada e descoberta nova. | Proveniência e revisão da skill, chamadas e recusas sanitizadas. | E07, E08, E11, E13 |
| P08 · IA, orçamento e disponibilidade | Rodar CLI e API com os adaptadores aprovados. Configurar duas contas, testar seleção manual, sobreposição de modelo e fallback, esgotamento, reserva financeira concorrente e retorno de provedor. | Matriz de capacidades, conta/modelo efetivos, uso/limite, diff e checkpoint preservados. | E08, E09, E11, E12, E16 |
| P09 · Fila justa e controles | Três projetos, limite global dois, tarefas concorrentes no mesmo projeto, dependência pendente, correção de PR e mudança de prioridade. Testar envelhecimento e modos de pausa global e por projeto. | Timeline com vaga, lease, prioridades, pausas, espera e conclusões. | E10, E11, E16 |
| P10 · Isolamento e processos | Tentar acesso cruzado, socket, banco, volume, chave, link e rede proibida. Rodar processo legítimo silencioso. Forçar timeout com processos filhos. | Permissões, logs de recusa, árvore/processos encerrados e recursos liberados. | E04, E08, E11, E13 |
| P11 · Recuperação e sessões | Interromper worker/VPS com patch e arquivos novos. Restaurar, trocar sessão por saturação, retomar um nó, reexecutar saída alterada e invalidar dependentes afetados. Repetir com checkpoint corrompido. | Hashes, manifesto, versão fixada, tentativas, retomada e decisão em falha. | E02, E10, E12, E13 |
| P12 · Aprovação e revogação | Aprovar operação delimitada, tentar destino e argumentos diferentes, expirar, disputar último uso, revogar antes e durante a operação. Verificar continuidade independente. | Escopo sanitizado, prazos, usos, evento de revogação e checkpoint. | E04, E11, E12, E13, E16 |
| P13 · Efeitos e compensação | Simular resposta perdida depois de subtarefa, push ou Draft PR. Exercitar efeito encontrado, ausência comprovada e resultado inconclusivo. Testar compensação reversível, sensível e irreversível. | IDs remotos únicos, ledger com provas, autorização e preservação de artefatos. | E06, E12, E13, E15 |
| P14 · Qualidade | Gerar baixo, médio e alto risco. Verificar baseline e regressão, revisão independente, correção pelo revisor, severidade, limite e E2E. | Diffs, baseline, evidência dos checks por revisão, relatório de revisão e captura quando útil. | E14, E15 |
| P15 · GitHub, docs e memória | Provar branch e PR por feature, commits por tarefa, docs e testes no repo. Exercitar revisão externa, fechamento sem merge e opções humanas. Em merge confirmado, promover memória e liberar dependentes. | PR/commits, link na origem, revisão, decisões, docs, estado de memória e fila. | E06, E10, E13, E15 |
| P16 · D89 e mudanças externas | Alterar requisitos, prioridade e PR enquanto ocioso. Verificar ausência de webhook/polling de descoberta. Acionar sync manual e preflight separadamente. Testar mudança compatível, conflito e não preempção. | Contagem de chamadas por gatilho, snapshots/diffs, histórico de fila e trilha de decisão. | E06, E10, E12, E15, E16 |
| P17 · Painel e retenção | Reconectar stream pelo cursor. Investigar execução e decisão, ver custo desconhecido e bloquear gasto. Avançar relógio de teste após 30 dias e após carência. | Eventos, estado de UI, dados retidos, limpeza e restauração de checkpoint protegido. | E03, E09, E12, E16 |
| P18 · Aceite | Percorrer índice de evidências para D01–D95 e E01–E17, conferir cobertura e validar a feature real. Registrar aprovação do responsável para a revisão testada. | Relatório assinado pelo responsável ou decisão registrada, matriz e links da revisão. | E17 |

## Registro de evidências

Crie uma linha por critério. Evidências de CI, eventos ou arquivos devem usar referências estáveis e revisões correspondentes. Um teste de falha registra o efeito observado e o estado recuperado. O acesso ao pacote é limitado aos responsáveis pelo desenvolvimento e aceite. Redigir segredos, dados pessoais e detalhes sensíveis dos conectores.

| Caso e critério | Decisão / épico / ticket | Revisão e ambiente | Entrada e passos | Esperado | Observado | Evidência | Resultado e responsável |
|---|---|---|---|---|---|---|---|
| A preencher | Dxx / Exx / Txx-yy | SHA, digest e ambiente | Comando, ação, item ou fixture | Estado verificável | Estado efetivo | Link, ID ou hash | Aprovado, falhou ou pendente |

## Condição de lançamento

- Todos os épicos e tickets obrigatórios entregues na revisão candidata. Cada critério tem teste aprovado ou justificativa verificável de inaplicabilidade quando a decisão prevê condição, como E2E sem jornada de usuário.
- Projeto real configurado do zero, feature real executada e Draft PR comprovada. Cenários integrados demonstram o ciclo pós-merge, cobertura dos dois conectores e validação de recuperação, aprovações, isolamento e sincronização.
- Ausência de falha obrigatória pendente. Um fix que altera componente, configuração ou dado relevante invalida as evidências correspondentes até reteste.
- Registro de aceite do responsável, versões e links das evidências, runbooks de operação/restauração e limitações medidas. Nenhuma capacidade obrigatória pode ser adiada para uma entrega posterior.
