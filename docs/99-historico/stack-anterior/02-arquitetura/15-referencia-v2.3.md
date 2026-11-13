> Leitura: [← Anterior](14-dados-alvo.md) · [Índice didático](../02-INDEX.md) · [Próximo assunto →](seguranca/00-threat-model.md)

> Instruções de criação em documentacoes/ ou lessons/ abaixo são histórico da fonte; novas entregas/lessons seguem exclusivamente docs/00-governanca/POLITICA-DOCUMENTACAO.md.

> Nota editorial DOC-MV-001, 2026-10-07: conteúdo integral local preservado como referência da evolução v2.3. Afirmações por incremento têm a revisão/data original; trechos sobre consumer ausente, recuperação pendente e caminhos antigos não descrevem necessariamente HEAD a2cc5e0. O [AS-IS conferido](01-as-is.md) e a [auditoria](../00-governanca/08-AUDITORIA.md) delimitam o estado atual; números de testes abaixo são evidência histórica, não reexecução desta migração.

# Software Factory - Arquitetura e operação

## Decisão obrigatória da stack - revisão 2.3
A stack da própria La fabrique está APROVADA: React + TypeScript + Vite no painel; NestJS + TypeScript na API; worker Node.js + TypeScript em processo separado; PostgreSQL; Redis + BullMQ; Docker Compose na mesma VPS. Não solicitar nova escolha ou confirmação da stack. Não iniciar a fábrica em PHP/Laravel, Angular ou .NET. Esta decisão substitui propostas anteriores.
Monorepo: `apps/web`, `apps/api`, `apps/worker`, `packages/contracts`. Contratos compartilhados precisam de validação em runtime. API não executa clientes, builds ou testes; o worker executa esses trabalhos com isolamento, limites e um writer inicial.
Ao trabalhar na própria fábrica, aplicar esta stack. A regra de preservar a stack existente aplica-se somente a projetos EXTERNOS cadastrados para desenvolvimento pela fábrica; ela não altera a stack da La fabrique. Se o repositório da fábrica contiver implementação anterior incompatível, registrar a divergência e planejar a adaptação por etapas; não apagar código existente nem reabrir a escolha tecnológica.
Versões exatas e comandos devem ser fixados conforme compatibilidade no bootstrap; isso não é uma nova decisão de stack. Repositório e funcionalidade do piloto externo permanecem pendentes quando não fornecidos.


Versão 2.3 - 2026-09-29

## 1. Decisão e escopo v2.3
Decisão do usuário em 29/09/2026: toda a fábrica na mesma VPS Linux, incluindo painel/API, PostgreSQL, Redis, supervisor, clientes oficiais, autenticação, sandboxes, worktrees, builds, testes e documentação. MacBook não é necessário para executar o MVP. Assinaturas primeiro; APIs e créditos extras desativados. Nenhum serviço foi implementado nesta entrega.
Stack aprovada: React + TypeScript + Vite no painel, NestJS + TypeScript na API, worker Node.js + TypeScript separado, PostgreSQL e Redis + BullMQ. Monorepo e backend modular. O piloto mantém sua stack. Ver ADR-003-stack-typescript.md.
Inferência permanece nos fornecedores por internet, sem GPU ou modelo local na VPS. Autenticar cada cliente pelo fluxo oficial na identidade de execução dedicada; validar headless, plano, compatibilidade e cobrança no ambiente real.
Uso pessoal de desenvolvimento. Multiusuário comercial, revenda e compartilhamento de credenciais ficam fora do MVP e exigem avaliação das condições aplicáveis.

## 2. Arquitetura numa VPS
Navegador -> HTTPS/proxy -> React/NestJS -> Orchestrator/Context Builder -> PostgreSQL/outbox/Redis -> worker interno -> Agent Runtime Gateway -> Codex/Claude/Antigravity oficiais -> sandbox/worktree -> checks/review/docs -> painel/aceite.
Separar serviços de controle, supervisor confiável, cliente de IA e execução de código por identidades, redes e volumes. Redis/PostgreSQL internos, sem portas públicas. Worker usa protocolo interno autenticado para claim/heartbeat/events/checkpoint/complete; preservar contrato que permite execução remota futura sem torná-la requisito.
Credenciais ficam nos mecanismos oficiais do cliente na VPS, fora do Git, logs, artefatos, contexto e mounts do código do piloto. A fábrica não coleta senhas nem reutiliza token OAuth como API. Login humano inicial e renovação quando solicitada; não prometer sessão permanente.
Agent Runtime Gateway gerencia lifecycle e resultados. CLI pode executar ferramentas autonomamente, exigindo limites no host. Validar que código/testes não leem credenciais; se isso não for possível no modo instalado, bloquear adapter e registrar ADR de isolamento compatível.
Um writer por worktree, revisão em sessão independente e revisão exata. Controle/executor compartilham host: falha da VPS afeta ambos. Backups ficam fora dela. Modelo cloud não transforma sandbox em isolamento de VM.

## 3. Dimensionamento e operação
Perfis de engenharia para este MVP, não requisitos oficiais de fornecedores nem benchmark. Detalhes em documentacoes/infraestrutura/DIMENSIONAMENTO-VPS.md.
| Perfil | vCPU | RAM | SSD/NVMe útil | Execução inicial |
|---|---|---|---|---|
| Mínimo | 4 | 8 GB | 120 GB | 1 job leve, etapas sequenciais |
| Bom - recomendado para começar | 8 | 16 GB | 200 GB | 1 job completo, testes e browser sequenciais |
| Ideal para amadurecer o MVP | 8 | 32 GB | 300 GB | 1 job inicial; até 2 em worktrees distintos após ensaio |
Linux 64 bits mantido, preferir x86_64 para reduzir incerteza de binários; ARM exige prova de compatibilidade. Sem GPU. CPU sustentada e I/O importam; vCPU compartilhada não equivale a núcleo dedicado. Disco precisa acomodar SO, banco, imagens, caches, worktrees e artefatos com 20% de margem livre.
No perfil Bom, reservar envelope inicial de 6 GB para SO/controle/banco/cache/supervisor; até 6 GB para execução de um job, incluindo seus processos; manter 4 GB de margem. Limite agregado precisa ser imposto via cgroups/supervisor, não só por container individual.
Começar sempre com MAX_CONCURRENT_EXECUTORS=1. Não executar build pesado, review editável ou E2E em paralelo por padrão. Promover concorrência só após medir picos, latência do painel, OOM, swap/I/O e disco.
Pausar novos jobs com disco abaixo de 20% ou memória sustentada acima de 85%. Abort/timeout conservador, preservar checkpoint e logs limitados. Worker parado gera WAITING_WORKER; reboot/rede/serviço são cenários de recuperação, não dependência de laptop doméstico.
Backup diário PostgreSQL criptografado fora da VPS; proposta sete diários/quatro semanais. Git remoto e artefatos relevantes externos. RPO 24h/RTO 4h são metas até restauração comprovada. Retenção inicial artefatos 30 dias/auditoria resumida 90 dias; limpeza sem atingir volumes persistentes.
Mensalidade/renovação, backup, armazenamento externo, assinaturas, impostos/câmbio e manutenção compõem TCO. Energia do servidor normalmente integra a hospedagem; não somar energia de worker doméstico nesta arquitetura.

## 4. Funcionamento de um ticket

Entrada obrigatória: objetivo, comportamento atual e esperado, critérios de aceite verificáveis, escopo permitido, caminhos proibidos, comandos de teste e limites de execução e gasto. Tickets vagos retornam à triagem antes de gastar com implementação.
Estados: DRAFT -> READY -> RUNNING -> VALIDATING -> REVIEW -> DOCS -> AWAITING_HUMAN -> DONE. WAITING_WORKER, WAITING_PROVIDER, PAUSED_LIMIT, PAUSED_BUDGET, BLOCKED e FAILED são estados explícitos; cancelamento interrompe novas chamadas e encerra o sandbox com evidências preservadas.
1. Inspeção determinística: branch base e SHA, instruções do repositório, dependências, testes existentes e arquivos relevantes. 2. Plano curto de implementação. 3. Cliente oficial implementa no workspace autorizado. 4. Supervisor impõe isolamento/lifecycle e coleta resultados; comandos internos do CLI não são interceptados apenas pelo prompt. 5. Lint/testes/scanners. 6. Revisão independente de diff e resultados. 7. Correções limitadas. 8. Documentação. 9. Diff/PR para aceite humano.
Revisor pode usar o mesmo provedor em sessão independente; isso não assegura ausência de vieses compartilhados. Segurança começa com ferramentas determinísticas e recebe revisão especializada conforme risco.
Cada job guarda ticket_id, run_id, attempt_id, base_sha, etapa, lease e heartbeat. Transição usa controle otimista ou lock transacional. Lease expirado recupera o job; side effects exigem idempotência. Uma chamada de IA com resultado desconhecido não deve ser reexecutada automaticamente sem reconciliar efeitos e possível consumo duplicado de cota.
No piloto, produzir branch e diff local é suficiente. Publicação de PR exige configurar a integração. Merge e deploy continuam manuais. Um gate documental pode aprovar arquivos estruturalmente válidos sem assegurar a correção técnica do conteúdo; a revisão deve conferi-los.

## 5. Agentes e responsabilidades

Planner/Tech Lead: decompor objetivo, classificar risco e selecionar estratégia. Developer: implementar dentro do escopo. Reviewer: procurar falhas de comportamento, regressões e incoerências no diff. QA: relacionar critérios de aceite a evidências. Documentation: registrar somente o que foi efetivamente implementado. Security: revisar achados e áreas sensíveis.
Os papéis são perfis de instrução e tarefas, não seis servidores nem seis chamadas obrigatórias a cada ticket. No MVP, usar Developer e Reviewer; o executor realiza verificações e a documentação faz parte da entrega. Planejamento extenso, QA separado e segurança por IA entram quando necessários.
R0: documentação e alterações cosméticas. R1: lógica local e validações comuns. R2: integrações e mudanças em mais de um domínio. R3: autenticação, autorização, pagamentos, segredos ou dados pessoais. R4: produção, infraestrutura e migrações irreversíveis. R3/R4 não pertencem ao primeiro piloto.
Ferramentas expostas: leitura limitada, busca, patch, comandos de teste cadastrados e diff. O modelo não recebe shell irrestrito, credenciais de produção ou Docker socket. Instruções encontradas em arquivos, logs e páginas são dados não confiáveis quando solicitam ultrapassar a política do job.

## 6. Context Builder e economia

Primeiro reduzir trabalho, depois escolher modelo. Ticket pequeno, critério claro e baseline de testes evitam ciclos caros. Não enviar todo o repositório, node_modules, vendor, binários, dumps ou logs integrais.
TicketContext contém: objetivo, critérios, regras aplicáveis, trechos da especificação, ADR relevante, arquivos selecionados, testes e resumo dos achados. Registrar hashes e base_sha. Começar com orçamento de 20-40 mil tokens de entrada por chamada; adaptar ao modelo e tarefa. É um teto proposto, não uma necessidade mínima.
Seleção inicial por caminhos, imports, dependências e busca textual. Indexação semântica/embeddings só após demonstrar benefício. Resumo não substitui fonte: incluir referências e permitir abrir o trecho original. Configurar limites de tamanho e sinalizar truncamento.
Gates baratos primeiro: lint, testes, type check, secret scan e dependency scan conforme stack. Fornecer à IA apenas os achados úteis, diff e contexto. Evitar pedir a um modelo que redescubra o que uma ferramenta detecta diretamente.
Prefixo estável pode favorecer cache administrado pelo cliente quando suportado; a análise financeira seguinte aplica-se somente à extensão futura de APIs. Não assumir que qualquer trecho repetido vira cache ou que cache funciona entre provedores/modelos. Medir tokens efetivamente em cache. Não omitir invalidação de regras ou contexto para preservar desconto.
Como extensão futura de API, para relatórios e auditorias não urgentes, avaliar Batch, observando elegibilidade, SLA e tarifas. Não colocar uma etapa interativa bloqueante em lote apenas pelo desconto. Batch e cache têm condições próprias; não somar descontos sem conferir a tabela do provedor.
Respostas estruturadas e patches reduzem saída desnecessária. Limitar raciocínio quando houver controle suportado, sem cortar capacidade necessária. Caching de resultados internos só para entradas equivalentes e mesma revisão; nunca reutilizar aprovação de diff alterado.

## 7. Agent Router, cotas e limites
Seleção por capacidade exigida, elegibilidade do modo de autenticação, disponibilidade do worker, saúde do cliente, cota observada, risco e desempenho medido. Preferências Claude para análise/review, Codex para implementação e Antigravity para UI/QA são hipóteses iniciais; não representam ranking comprovado.
Estados do provider: UNCONFIGURED, AUTH_REQUIRED, AVAILABLE, BUSY, RATE_LIMITED, COOLDOWN, ERROR e DISABLED. Estado de uso inclui value nullable, unidade, fonte, observed_at e reset_at nullable. Saldo desconhecido não significa disponível ilimitadamente; o painel mostra desconhecido. Não inventar percentuais ou horários de reset.
Uma assinatura pode ter várias janelas e limites por modelo/modo. O adapter deve guardar as restrições distintas. AVAILABLE significa último teste elegível, não garantia de próxima chamada. Reativar após horário informado com uma sondagem barata; se horário ausente, backoff limitado e verificação posterior.
Atingiu cota: parar writer, preservar checkpoint, registrar motivo e escolher outro provider elegível. Se nenhum disponível, WAITING_PROVIDER. Não criar contas, alternar identidades ou usar endpoints privados para contornar limites. Trocar entre assinaturas próprias não aumenta a cota de nenhuma delas.
Limites propostos: duas rodadas de correção, uma escalada, dois handoffs por ticket e 30 minutos por tentativa. Falha repetida idêntica pausa com diagnóstico. Auth exige ação humana no cliente oficial; 5xx transitório tem retry limitado; limite de uso não recebe retry imediato.
UsageMeter guarda métricas realmente expostas: tokens/modelo/tempo/ferramentas, quando disponíveis. Custo teórico de tokens apresentado pelo CLI não equivale a cobrança de assinatura. Ledger distingue assinatura fixa, extra medido, API estimada/reconciliada e desconhecido.
MVP: monthly_api_budget=0, fallback_api.enabled=false, créditos/extra usage/autorecharge desativados em cada conta. Bloquear também variáveis de API key herdadas no processo. Configuração da fábrica não controla cobrança habilitada no site do fornecedor: o preflight precisa verificá-la com o usuário no onboarding.
BudgetGuard monetário permanece preparado para futura API, com reservas atômicas e reconciliação; não é usado para fingir reservar tokens de assinatura sem suporte do fornecedor. RuntimeGuard aplica tempo, tentativas, processos, concorrência e limites de troca.

## 8. Economia e comparação financeira
Retirar a simulação por tokens da arquitetura principal: o modelo agora é assinatura/capacidade. Não prometer quantidade de tickets por plano nem economia percentual antes do experimento.
Custo mensal = VPS + backup + assinaturas + extras autorizados + armazenamento externo + manutenção. Custo incremental = novas despesas causadas pela fábrica. Custo por aceito = custo atribuído ao experimento / número de tickets aceitos, incluindo falhas no numerador. Quando não houver aceitos, informar indefinido.
Para atribuir custo fixo compartilhado, registrar método: por exemplo fração do tempo de uso ou alocação definida pelo responsável. Não somar preço teórico do CLI à assinatura como se fosse cobrança real. Mostrar separadamente custo total e incremental.
Economia principal: reaproveitar planos existentes; poucos papéis por ticket; reduzir contexto; ferramentas determinísticas antes de IA; instruções estáveis; tickets pequenos; handoffs curtos; revisão proporcional ao risco; evitar loops e paralelismo que esgotam cota.
Cache administrado pelo cliente pode ajudar uso/latência, mas não assumimos desconto em assinatura. Batch permanece extensão de API futura, não requisito nem desconto disponível nos CLIs. Modelos econômicos devem ser selecionados apenas quando expostos e elegíveis no plano.
Quando todas as cotas acabarem, esperar é o padrão com gasto adicional zero autorizado. APIs futuras só após mudança explícita de política, configuração de credencial, tarifas oficiais e limites de cobrança. A ausência de API não impede consumo de créditos extras do CLI se o usuário os habilitar: por isso o onboarding verifica ambos.
Comparar 10 tickets pequenos: aceite, regressões, tempo ativo, espera por cota, correções, handoffs, tokens quando observáveis e custo atribuído. A economia só estará demonstrada quando confrontada com baseline equivalente.

## 9. Isolamento e confiabilidade

Sandbox é workspace descartável com usuário sem privilégios, limites de CPU/memória/processos/tempo/disco, filesystem mínimo, capabilities reduzidas e rede restrita. Dependências podem exigir saída à internet via destinos controlados; o sandbox não deve alcançar banco/Redis da fábrica ou serviços internos.
Não montar /var/run/docker.sock no sandbox; não usar privileged; não montar pasta de segredos. O Executor Manager confiável prepara ambientes. Se o projeto requer containers de integração, usar instâncias pré-provisionadas ou mecanismo isolado definido em ADR; não dar controle do host ao agente.
No worker, containers compartilham kernel e uma falha do host afeta controle e execução. Restrição de recursos reduz interferência, não elimina risco de escape. O piloto usa código de um repositório próprio/confiável e dados sintéticos. Execução de código arbitrário de terceiros exige reavaliar o isolamento.
Revisão humana permanece para merge/deploy, mudança de credenciais e operações destrutivas. Logs não devem vazar PII ou segredos. Remover segredos antes de enviar arquivos aos provedores; avaliar retenção e uso de dados da modalidade contratada escolhida.
Recuperação: reiniciar executor em teste, comprovar lease expirado e recuperação sem duplicar efeitos; pausar ao exceder orçamento; testar falha do cliente, cota indisponível e timeout; restaurar backup. Nunca marcar DONE apenas porque o modelo declarou sucesso.

## 10. MVP incremental
V0 sintética: construir controle e worker, validar um cliente oficial por vez no ambiente real da VPS e confirmar autenticação, cobrança e isolamento usando fixtures controladas. Depois do núcleo pronto, preencher o piloto e executar os tickets reais e o handoff como parte do ensaio operacional; resultado sintético não substitui esse aceite.
V1: controle web, worker por HTTPS, lease/heartbeat, um adapter oficial, sandbox, Context Builder, Developer, checks e gate documental. Nenhuma API paga. Revisor em sessão independente, com revisão humana final.
V2: segundo adapter, Provider Manager e handoff automático após writer encerrado, ensaio de falha/cota e dez tickets. Terceiro adapter pode entrar após prova de compatibilidade. UI/QA só exige browser se o critério precisar dele.
V3: expandir projetos, workers ou workflow engine quando métricas justificarem. Temporal, autoscaling, APIs e inferência local seguem como opções futuras.
Fora MVP: merge/deploy automático, multiusuário comercial, credenciais compartilhadas, GPUs, execução de código arbitrário de terceiros e features de risco R3/R4.
Metas propostas: 8/10 aceitos em até duas correções; 100% com docs e evidências; zero extras/API não autorizados; zero writers simultâneos no mesmo worktree; recuperação de interrupção comprovada. Não são resultados medidos.

## 11. Documentação como parte da entrega

Toda feature, correção, refatoração, configuração ou migração exige relatório datado em documentacoes/<dominio>/AAAA-MM-DD-TICKET-titulo.md. O relatório registra objetivo, funcionamento, alterações, diff sanitizado, testes e resultados reais, riscos, rollback e limitações.
Além do histórico, atualizar a documentação de estado atual: documentacoes/<dominio>/README.md, arquitetura/ADRs quando necessário, contratos/API, índice, CHANGELOG e status do backlog. Não basta acumular relatos de entrega e deixar a descrição atual obsoleta.
Lessons: criar ou atualizar lessons/<conceito>.md com explicação, motivo, exemplo real, cuidados e links. Reutilizar documento existente; não inventar aprendizado ou gerar cópias por ticket. Atualizar o índice.
Documento de entrega vincula base_sha/head_sha ou diff de uma revisão identificada. Não tentar incluir no próprio documento o hash do commit que ainda vai conter esse documento: usar revisão do código anterior ao commit documental ou link final do PR. Remover segredos e dumps dos diffs.
Definition of Done: critérios verificados, checks relevantes aprovados, revisão resolvida, custo reconciliado ou explicitamente pendente, documentação técnica atualizada, relatório de entrega, lessons quando aplicável, diff revisável e aceite humano. Gate de CI exige arquivos e seções; revisão humana avalia a veracidade.
Os arquivos AGENTS.md, CLAUDE.md e .agents/rules/documentacao.md remetem à política comum. Antigravity exige confirmar que a versão instalada lê a pasta de regras; se não, inserir a política explicitamente no contexto. Esses MD são orientações de repositório, não instalação de skills ou garantia de leitura automática.

## 12. Contratos e painel
Detalhamento em ESPEC-MVP.md, documentacoes/runtime/README.md, documentacoes/handoff/README.md e documentacoes/operacao/README.md. Entidades: Project, Ticket, Run, Step, Worker, ProviderInstallation, UsageObservation, AgentExecution, Checkpoint, ContextManifest, Artifact, LedgerEntry e Approval.
Runtime request: ticket/run/step/attempt, workspace e revisão, provider/modelo solicitado, prompt/context manifest, ferramentas e limites, regras documentais. Result: status normalizado, exit code, revisão/diff, artefatos/checks, uso nullable, versão do cliente, modelo efetivo e checkpoints. Nunca armazenar cadeia de raciocínio privada.
Interface: execute, getStatus, getCapabilities, getUsage, cancel, resume. Capacidade ausente deve retornar unsupported, não simular equivalência. Resume nativo só no mesmo provider/versão compatível; troca de provider é nova execução com handoff. Cancel deve encerrar árvore de processos e confirmar quiescência.
Painel mobile: tickets, etapa, writer, worker online/offline, health dos providers, última observação de uso e sua fonte, espera por cota, tentativas, diff, checks e docs. O FAC-011 implementa a fundacao de configuracao: contas/instalacoes, catalogos, provider/modelo e limites por funcao, GitHub e travas financeiras. Cadastro e estado real ficam distintos; somente preflight do worker pode promover disponibilidade. Percentual de uso apenas se fornecedor expõe e adapter validou. API/extra desligados visíveis. Aceite associado a revisão exata; novo código invalida aprovação.
CLI encerrou com zero não prova sucesso do ticket. Validar schemas, diff, checks, critérios e documentação antes de AWAITING_HUMAN. Publicação de PR é integração opcional autenticada com escopo mínimo.

## 13. Fontes e pendências
Fontes oficiais verificadas em 29/09/2026, listadas integralmente em FONTES.md. Confirmado: Codex possui exec não interativo e reutiliza autenticação do CLI; Claude Code possui -p; Google documenta agy -p para automação. Isso comprova mecanismos técnicos, não capacidade ilimitada ou elegibilidade automática de qualquer uso.
Autenticação oficial, plano contratado, modo de execução, créditos extras, termos e compatibilidade precisam ser validados no worker antes de ativar o adapter. No Claude, diferenciar cliente oficial de SDK/serviço usando credenciais de assinatura; não capturar/intermediar tokens OAuth. Documentação de assinatura não substitui análise de modo de cobrança real.
Pendências para o piloto externo: repo/branch, feature e baseline. Pendências de infraestrutura e providers: perfil contratado da VPS, compatibilidade dos clientes, assinaturas e modelos acessíveis, forma oficial de login e capacidade semanal. A primeira lista não bloqueia os tickets internos marcados como independentes no backlog; a segunda deve ser resolvida nos tickets que a exigem. A stack da fábrica está aprovada e não deve ser questionada no bootstrap.
Esta v2.3 substitui a topologia VPS + MacBook da v2 e a execução por APIs da v1. Mantém qualidade, isolamento, documentação por domínio, lessons, evidências, backlog e aceite. Nenhum cliente foi instalado ou testado nesta entrega.
