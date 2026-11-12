# Arquitetura e contratos do novo MVP

Leia a [matriz](04-RASTREABILIDADE.md) para as decisões de produto e os [épicos](02-EPICOS.md) para seus critérios. Os detalhes de arquitetura abaixo são propostas de engenharia para materializar as decisões delegadas e dar contratos aos tickets. Os caminhos de API são internos.

## Repositório e estrutura

**Monorepo privado no GitHub para a fábrica.** Frontend, API, worker, contratos, migrations e implantação mudam juntos e podem ser revisados numa PR. API e worker têm pontos de entrada e permissões distintos, compartilham módulos Go e são publicados como imagens separadas da mesma revisão. Os repositórios de software atendidos continuam independentes. O bootstrap cria a estrutura da fábrica e documenta como vincular um repositório existente ao projeto piloto.

```text
le-fabrique/
  .github/
    workflows/           # CI, empacotamento e publicação de imagens
    pull_request_template.md
  apps/
    web/                 # React, TypeScript, Vite, editor React Flow
  cmd/
    api/                 # API Go, autenticação e painel
    worker/              # agenda, leases, execução e recuperação
    migrate/             # migrações controladas
  internal/
    projects/            # cadastro e configuração efetiva
    workflows/           # grafo, publicação e execução
    orchestration/       # planejamento, fila e estados
    execution/           # sandbox, processos e ferramentas
    providers/           # adaptadores de IA e disponibilidade
    integrations/        # GitHub, Jira e ClickUp
    security/            # credenciais, autorização e auditoria
    persistence/         # persistência e operações externas
  contracts/             # OpenAPI e schemas versionados
  db/migrations/
  infra/
    compose/             # dev, homologação e produção
    images/              # web/API, worker, perfis de execução
    proxy/
    runbooks/
  tests/
    contract/
    integration/
    e2e/
    fixtures/
  docs/
    adr/
    operacao/
    aceite/
  README.md
  AGENTS.md
  CLAUDE.md
  ANTIGRAVITY.md
  .env.example
```

Um repositório privado com `main` protegida, PRs e checks obrigatórios adequados ao plano escolhido. Documentar permissões de manutenção e procedimento de exceção do proprietário quando a configuração de revisão exigir mais de uma identidade. Usar dependências e imagens com versões fixadas e registro de digests nas entregas. Manter convenções de revisão, versionamento, migrations e evidências no repositório da fábrica. Os guias por agente devem apontar para a especificação vigente e para as convenções locais.

## Serviços e ambientes

| Componente | Contrato |
|---|---|
| Web | Aplicação React/TypeScript com Vite e React Flow. Usa API autenticada. Validação visual é complementada por validação de domínio no servidor. |
| API Go | Configuração, autenticação, catálogo, publicação de workflows, comandos operacionais e leitura de estado. Stream de eventos por SSE autenticado com cursor para reconexão. |
| Worker Go | Reclama trabalho, controla um escritor por projeto, invoca adaptadores e prepara ambientes isolados. Persiste progresso e checkpoints. Recupera tarefas após reinício e executa efeitos externos pela política. |
| PostgreSQL | Estado transacional, migrations, fila durável, controle de concorrência, eventos, outbox e auditoria. A escolha é proposta para concentrar transações e evitar uma fila externa na VPS. |
| Volume de artefatos | Workspaces em armazenamento reservado e checkpoints/artefatos duráveis identificados por conteúdo. Banco guarda manifesto, versão, projeto, integridade e retenção. |
| Executor isolado | Container temporário por feature quando alocada, com workspace durável reconstituível. Recursos, rede, identidade e montagens limitados. Supervisor separado do código e das CLIs executados. |
| Proxy | TLS para painel/API, roteamento de origem única, limites de requisição e cabeçalhos de segurança. Banco, worker e supervisor privados. |

Dev local com serviços e fixtures reproduzíveis. Homologação com banco, volumes, credenciais, prefixos/integrações e capacidade reservada próprios. Pode compartilhar a VPS com produção se houver folga medida e segregação comprovada. Produção na VPS existente, com supervisão e reinício, TLS, firewall, rotação de logs, backup/snapshot do provedor e runbook. CI testa PRs com segredos fictícios e sem implantação de branches de contribuição. Artefato publicado é promovido entre ambientes por digest. O responsável executa a promoção de produção e controla migrations.

O bootstrap deve medir RAM, CPU, disco, I/O, quantidade de processos, estabilidade de reinício e duração de build, checks e navegador usando um workload representativo. Reservar capacidade para API, banco, proxy e sistema. Definir limite global inicial e envelopes dos perfis com evidências. A funcionalidade de paralelismo deve ser validada com pelo menos dois projetos em um ambiente de teste com capacidade para isso. Configurações de ambiente devem ser reproduzíveis com segredos injetados fora de Git.

## Modelo de dados e invariantes

| Entidade ou agregado | Campos e invariantes essenciais |
|---|---|
| Admin, sessão e recuperação | Um administrador, hash de senha, segredo TOTP protegido, hashes de códigos de uso único, sessão revogável e reautenticação com validade. |
| Projeto, origem e conexão | ID, slug, repo GitHub, branch base, plataforma/contêiner externo, credenciais por referência, marcador de elegibilidade, mapas de status/prioridade, regras de seleção, comandos/perfil, prioridade e estado de ativação. Uma origem de tarefas ativa por projeto. |
| Catálogos e versões | Agente, skill, ferramenta, servidor MCP, conta, modelo e workflow. Referências estáveis, versões ou snapshots imutáveis conforme a execução. Conexões revogáveis permanecem sujeitas à política atual. |
| Feature e tarefa técnica | Projeto, ID externo estável e revisão observada, plano versionado, critérios, dependências, prioridade, subtarefa remota, branch e PR. Distinguir dependência técnica de dependência de merge. |
| Execução e tentativa | Feature/tarefa, workflow e subfluxo publicados, snapshot efetivo, etapa, sequência de eventos, resultado, lease, versão e chave de exclusão. Reexecução gera tentativa própria com ligação à anterior. |
| Checkpoint, artefato e manifesto | SHA base/cabeça, patch e não rastreados necessários, referências, estado de processos, nós, insumos/saídas, aprovação/efeitos por referência, hash e versão de schema. |
| Operação externa | Chave de negócio única por efeito lógico, destino, pedido sanitizado, estado, prova remota, timestamps e tentativas. Efeito lógico independe do número da tentativa. |
| Decisão humana e autorização | Pedido, alternativas, ator, recurso e argumentos canônicos, ação, revisão aplicável, validade, usos restantes e estado. Revogação tem precedência sobre configuração fixada. |
| Uso e orçamento | Conta, agente, modalidade, modelo, moeda, valor decimal, reserva, observação, fonte, janela e reconciliação. Cota de assinatura em unidade própria. Dados ausentes têm estado explícito. |
| Memória do projeto | Chave semântica normalizada, tipo, conteúdo, fonte, evidência, revisão, confiança, vigência e vínculo com branch/feature/PR. Estado pendente, oficial, invalidado ou lição identificada. |

IDs internos opacos e estáveis. Unicidade por projeto e identidade externa. Eventos monotônicos por execução e mutações com controle de versão. Datas em UTC no armazenamento. Queries e ações sempre recebem escopo de projeto validado. O banco registra o vínculo ao arquivo após gravação íntegra, por escrita temporária e renomeação, com verificação de checksum. Backups e restauração cobrem banco e volumes coordenadamente.

## Estados, fila e exclusão mútua

| Unidade | Transições e condições |
|---|---|
| Feature | Importada → Planejamento → Desenvolvimento → Validação final → Aguardando merge. Após merge observado, registra SHA e fica pendente de sincronizar conclusão se a atualização externa falhar. Concluída após confirmação do estado remoto. PR fechada sem merge vai para decisão humana. |
| Tarefa técnica | Planejada → Elegível → Em execução → Validando/Revisando → Commit registrado → Concluída. Laterais explícitos para dependência, indisponibilidade, orçamento, aprovação, pausa, falha e recuperação incerta. A retomada cria nova tentativa vinculada. |
| Operação externa | Intenção registrada → Em envio → Confirmada ou Falha comprovada ou Resultado incerto. Após verificação, confirmar efeito encontrado, repetir se ausência comprovada e seguro ou pedir decisão. Compensação é uma nova operação auditável. |
| Pedido humano | Pendente → Aprovado, Negado ou Expirado. Uma autorização concedida pode se esgotar, expirar ou ser revogada. |
| Workflow | Rascunho editável → Versão publicada imutável. Projetos e execuções retêm vínculos efetivos. Políticas vivas de segurança e orçamento continuam valendo. |

O escalonador transacional reclama uma vaga por projeto e uma vaga global. Trabalhos de planejamento, correção de PR, desenvolvimento, revisão, docs e consolidação que possam escrever no projeto passam pela mesma exclusão. Uma tarefa possui lease, heartbeat e token de geração, validado em gravações e efeitos. Quando o lease expira, o novo escritor só começa após confirmação de parada ou isolamento efetivo do anterior. Em caso de incerteza, manter o projeto bloqueado para novas escritas e permitir progresso dos outros.

Dentro do projeto, filtrar dependências e elegibilidade, preferir correções de PR e depois aplicar prioridade de tarefa, data de elegibilidade e ID estável. Para a disputa global, proposta determinística: prioridade do projeto numa faixa finita de 0 a 4 e promoção de um nível por janela de espera contínua, até o nível máximo. Entre projetos no mesmo nível efetivo, ordenar pela espera mais antiga e depois por ID. O contador reinicia após atendimento, e projetos sem trabalho elegível saem da disputa. Testar com chegadas contínuas, relógio controlado e mudança de prioridade durante execução.

Pausar após a feature atual impede novas admissões e permite finalizar apenas a feature que ocupa a vaga no momento da ação, até entrega ou suspensão segura. A pausa global aplica o mesmo critério a cada projeto em andamento. Suspender agora solicita parada controlada, checkpoint e liberação confirmada. Reduzir o limite global impede novas admissões até que o número ativo caiba no limite. Prioridade ou capacidade alterada afeta a próxima decisão de fila.

## Sincronização e fonte de verdade

1. O projeto usa **uma** origem oficial de tarefas. GitHub determina conteúdo das refs e estado do PR. O banco local determina estado da execução. Ler requisitos e prioridades do item remoto e guardar ID, revisão/versão ou hash canônico e data observada.
2. O comando explícito **Sincronizar agora** ou o preflight autorizado de uma nova execução, inclusive retomada, captura alterações externas relevantes. O preflight reavalia candidatas caso as novas informações mudem elegibilidade, dependências ou prioridade. A reconciliação de uma operação conhecida consulta apenas o alvo desse efeito. Uma fila local vazia não dispara descoberta contínua. A interface mostra última leitura bem-sucedida e erro mais recente.
3. Processar os itens elegíveis encontrados. Em andamento, publicar transições, subtarefas, dependências e links mapeados pela plataforma oficial usando a outbox. Retentativas de um efeito pendente seguem seu próprio registro. Preservar campos mantidos pelo usuário e verificar conflitos contra a última revisão lida.
4. Uma nova revisão de requisito gera comparação e análise de impacto. Mudanças compatíveis entram em um marco seguro com plano/contexto versionados. Mudanças significativas replanejam o trecho afetado quando seguro ou pedem decisão humana. Preservar commits, alterações manuais e histórico. Prioridade isolada vale no próximo agendamento.
5. Verificação de merge é leitura explícita da PR vinculada. Conferir repo/head/base/merge observado e integrar memória oficial após validar a revisão resultante. A dependência de código usa o fato confirmado de integração, mesmo quando a projeção de status ainda está pendente. Fechamento sem merge abre decisão e mantém dependentes bloqueadas. Correções solicitadas usam a branch atual, sob revalidação e limite de rodadas.

## Publicação e execução de workflows

O grafo publicado tem ponto de entrada, nós tipados e portas de entrada/saída com schema. Condições determinísticas usam variáveis tipadas. Loops exigem condição e limite. Esperas duráveis persistem prazo/condição e devolvem a vaga quando possível. Ramificações podem ser representadas sem concorrência de escrita dentro do projeto. Convergências definem quais caminhos ativos precisam entregar resultado e o tratamento de caminho não percorrido. Falhas seguem rota de erro ou estado de bloqueio. Subfluxos publicados fixam a referência de versão e têm limite de profundidade. A publicação rejeita ciclos de chamadas, órfãos críticos e contratos incompatíveis. Nós podem invocar agente, skill, script, checks, Git, documentação e decisão humana.

A seleção de workflow de feature e subfluxo técnico usa tabelas ordenadas, regra mais alta primeiro e padrão explícito sem correspondência. Congelar workflow, subfluxos, referências, configurações efetivas, parâmetros e política de workflow no momento de criação da execução. Aplicar modelo → agente → nó, validar a conta selecionada e guardar evidência sem segredos. A revogação de credencial, ferramenta ou aprovação e o orçamento atual são verificados novamente antes de operar. Mudanças de requisitos atualizam contexto e planos sem trocar silenciosamente o workflow publicado já fixado.

## Limites de segurança e efeitos

- Rodar como usuário sem privilégios, com quotas de CPU, RAM, processos, disco e duração. Impedir acesso ao socket do runtime, diretórios administrativos, rede de controle e metadata do host. Validar caminhos e links, incluindo arquivos de pacote e Git. Projetos recebem volume e identidade próprios. Dependências, scripts, builds e testes executam no sandbox.
- Operações externas e segredos passam por adaptador com identidade mínima e acesso por destino. Autenticação CLI segue mecanismo oficial e usa identidade por conexão sob supervisão. Proteger artefatos de autenticação contra processos do projeto e medir o isolamento no preflight. Para APIs, o código e o modelo recebem ferramentas de escopo restrito. Credenciais brutas ficam no broker/cliente autorizado.
- O motor aplica política por ação, parâmetros canônicos, recurso e ambiente antes de cada chamada. Operações proibidas são recusadas. Solicitações sensíveis vinculam escopo, versão quando aplicável, expiração e usos. Consumo de autorização é atômico. Revogação impede novos usos e aciona parada segura quando possível. Auditar sem carregar segredos.
- Registrar a intenção do efeito antes do envio. Chave de negócio identifica subtarefa, push, PR, comentário ou transição. Adaptadores procuram prova verificável de ocorrência, mesmo quando a API não oferece chave idempotente. Divergência ou ausência de prova conclusiva bloqueia repetição automática. Compensações reversíveis de baixo risco exigem verificação prévia do estado. Compensações sensíveis passam pela mesma política de aprovação.
- Timeout absoluto e inatividade são independentes. Observar árvore de processos, heartbeat, I/O ou eventos do adaptador e distinguir silêncio legítimo. Em timeout, suspensão ou troca de sessão, parar a árvore, preservar progresso e confirmar liberação. Se a operação externa é incerta, reconciliá-la antes de repetir.

## Interfaces e padrões de verificação

| Área | Contratos propostos |
|---|---|
| Sessão | `POST /auth/login`, `POST /auth/totp`, `POST /auth/recover`, `POST /auth/logout` e revogação de sessão. |
| Projetos | CRUD de configuração, `POST /projects/{id}/validate`, `POST /projects/{id}/activate`, `POST /projects/{id}/sync` e comandos de pausa. |
| Catálogos | Recursos de agentes, skills, ferramentas, servidores MCP, conexões e modelos. Ações de testar, descobrir ferramentas, versionar, rotacionar e revogar. |
| Workflows | CRUD de rascunho, validação, publicação e consulta de versões. Exibir a seleção efetiva na execução. |
| Execução | Consulta de features, tarefas, eventos e artefatos. Comandos de suspender, retomar e reexecutar nó, com versão esperada e chave idempotente quando aplicável. |
| Decisões | Consulta de pendências, aprovação, recusa, revogação e decisão de conflito/PR fechada. |
| Operação | Saúde, estado global, limite de concorrência, uso, disponibilidade e histórico. SSE com cursor, paginação para histórico e reconexão autenticada. |

Contratos versionados e testes de contrato entre web, API e worker. Erros com código legível por máquina, mensagem, ação sugerida e ID de correlação. Usar rejeição de conflito para estado/revisão obsoletos. Validar schemas, autenticação, escopo e limites de payload antes de aplicar mutações. Logs estruturados por projeto, feature, tarefa, execução, tentativa e nó, com redação na origem. Cuidar de segredos em diff, prompts, parâmetros, screenshots e logs de CI.

Propostas de parâmetros ajustáveis em desenvolvimento: janela de envelhecimento de 15 minutos, timeout inicial de 45 minutos para agente, janela de suspeita de inatividade de 10 minutos baseada em sinais adicionais, recuperação de infraestrutura com três tentativas e backoff, carência de sete dias para checkpoints concluídos. São valores iniciais a medir e registrar. Duas rodadas de correção por workflow e 30 dias de logs detalhados vêm da entrevista.

## Contratos de integração e parâmetros de execução

### Credenciais, contas e orçamento de IA

Cada adaptador recebe `execution_id`, `attempt_id`, modalidade, referência de conexão, ID do modelo, parâmetros efetivos, manifesto de contexto, ferramentas autorizadas e limites. Retorna eventos estruturados, evidências sanitizadas e resultado normalizado. Registrar identidade e versão do cliente, modelo observado quando disponível, sinais de autenticação, política de cobrança e capacidades verificadas. Os estados de saída propostos incluem sucesso, falha da tarefa, cancelado, timeout, inatividade, falha temporária de provedor, cota atingida, autenticação inválida, orçamento bloqueado, configuração incompatível e resultado externo incerto. Erros de código entram na recuperação da tarefa. Cota e indisponibilidade elegíveis acionam a cadeia de fallback. Falha de autenticação é visível e invalida aquela conexão até revalidação. Resultado externo incerto passa primeiro pela reconciliação.

Cada alternativa da cadeia indica explicitamente conta, modelo e modalidade. A elegibilidade exige conexão vigente, capacidade para o nó, autorização e capacidade financeira quando aplicável. O adaptador só registra cotas que observou, com fonte e horário. Backoff por conexão com limite e dispersão evita picos. Condições conhecidas de recuperação podem informar o próximo horário. Sem alternativa, persistir motivo, checkpoint e data da próxima tentativa aplicável. Verificação de disponibilidade de IA é distinta da descoberta de mudanças de Jira, ClickUp ou GitHub.

A proposta financeira usa orçamento global em janela mensal na moeda configurada e limites opcionais por agente na mesma janela. Fixar preço/limite conhecido na autorização da chamada, com teto de entrada, saída e recursos pagos aplicáveis. Uma transação reserva o máximo autorizado nos dois limites. Considerar uso já confirmado e reservas abertas. Uma chamada com custo máximo desconhecido fica pendente de configuração. Conciliar reserva com consumo reportado quando disponível, conservar reserva conservadora em resultado incerto e registrar ajustes. Guardar moeda, fonte e vigência da tarifa usada. Separar quantidade observada de assinatura, estimativa de API e valor reportado, inclusive quando o fornecedor não informar um deles.

### MCP e ferramentas

Implementar cliente MCP básico com `stdio` em subprocesso isolado e Streamable HTTP para endpoints cadastrados. Negociar versão, testar conexão, paginar `tools/list` e chamar `tools/call` com limites e validação de schema. Identificar ferramenta pela conexão e nome, com revisão do schema. Mudança incompatível exige nova validação antes de uso. O executor checa a allowlist por agente, versão, argumentos, destino e política a cada chamada. Tratar descrição, anotação e resultado de ferramenta como dados. Acesso remoto passa por allowlist e controle de redirecionamentos e destinos, inclusive endereços internos protegidos. Processos de projeto só recebem recursos aprovados. [MCP — transportes](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports), [MCP — ferramentas](https://modelcontextprotocol.io/specification/2025-11-25/server/tools)

### Evidência de entrega Git

Chave de publicação proposta: `project_id + feature_id + delivery_attempt`. Gerar branch a partir da identidade estável da feature e vincular a PR a head/base do repositório correto. Atualizações de revisão da mesma tentativa reutilizam branch e PR. Uma nova tentativa por D93 exige decisão humana, reconciliação de refs e PRs existentes e encerramento da tentativa ativa anterior no controle local. Preservar refs, SHAs e artefatos da tentativa anterior. Uma branch distinta pode ser usada quando a nova tentativa precisar de nova base e o responsável escolher essa alternativa. O histórico liga todas as tentativas à mesma feature.

Ferramentas de agente recebem Git de escopo local. O broker da fábrica autoriza push apenas à branch da feature e restringe operações remotas ao fluxo aprovado. Validar a branch base protegida, a identidade da fábrica sem bypass de merge direto e a disponibilidade de Draft PR no repositório do projeto durante preflight. Uma confirmação de PR aberta deve demonstrar URL, número, head, base e estado draft. A confirmação de merge deve demonstrar a PR vinculada e a revisão integrada. Para PR fechada sem merge, registrar data de observação e decisão pendente.

Cada check grava comando sanitizado, revisão, ambiente, hora de início/fim, código de saída, classificação de resultado e artefato de evidência. O histórico da feature referencia o plano, os critérios e os commits. Documentos vivos descrevem o comportamento atual dos módulos afetados, inclusive dados, APIs, configuração e regras quando aplicáveis. O gate confere pertinência e coerência com o diff, e verifica que os resultados de teste relatados correspondem aos registros da fábrica.
