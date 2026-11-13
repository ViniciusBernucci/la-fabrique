> Leitura: [← Anterior](07-piloto.md) · [Índice didático](../02-INDEX.md) · [Próximo →](09-backlog.md)

# Handoff para Tech Lead

## Estado atual

Atualizado pelo FAC-012AH em 2026-10-03. Controle/worker/workflow, resultados/artefatos, retomada/recovery/handoff, docs técnicas e perfis oficiais estão implementados nas revisões Z–AH; aceites exatos pendentes registrados no BACKLOG. Consumer real composto sob gate padrão false; não há fixture consumindo fila de execução. AC garante writer global no PostgreSQL e AD permite observá-lo sem projeto. 528 testes + 10 PostgreSQL + 3 Redis passaram. Piloto externo é opcional posterior, nunca bloqueio da conclusão do MVP.

## Fronteiras e contratos

- O painel consome somente contratos exportados por `packages/contracts` e acessa a API pelo proxy `/api`.
- A API controla HTTP e persistência; não executa clientes, builds ou testes.
- O worker é um processo separado e o único ponto de execução de jobs, com consumer real implementado sob gate.
- `packages/contracts` valida payloads em runtime com Zod.
- `packages/runtime` define portas e adapters oficiais; credenciais não são exportadas para controle/painel.
- PostgreSQL é a fonte de verdade; Redis/BullMQ transporta trabalho e não substitui outbox, leases ou fencing.

## Sequência recomendada

1. Escopo interno auditado implementado/verificado: followup de capacidade AH, pausa global AG, backup AF e SSE AE fechados; revisar os commits exatos.
2. Revisar incrementos exatos e preparar operação com dados sintéticos. Login/preflight do usuário do serviço e confirmação financeira dependem do responsável; prosseguir desenvolvimento independente.
3. Ativação/implantação em janela autorizada, sem inferir elegibilidade. Piloto pode ser escolhido posteriormente pela interface.

## Restrições para os próximos tickets

Manter um executor inicial e o limite global observável. Não expor PostgreSQL, Redis ou worker. Não colocar auth de providers em banco, logs, contratos do frontend ou worktrees. Não usar API de IA, extra usage, autorecharge ou fallback pago. Qualquer provisionamento de VPS, merge ou deploy requer autorização aplicável.

## Riscos abertos

Preflights reais sob UID do serviço, confirmação financeira, keyring/PR remoto, implantação/backup externo/restauração/reboot continuam não verificados. Isso não impede implementação interna. SSE implementado no AE; painel de runs, gate documental e composição do consumer já implementados, contrariando o handoff antigo. Multiusuário/rotação avançada não foram incluídos implicitamente no MVP de administrador único. [Estado detalhado](06-controle-mvp.md).


## Proposta focal — painel administrativo da conta

Data: 2026-10-09 (Europe/Berlin). Base inspecionada: `ee4485bffb517247cd7b584653ad9ec1423f9202`, branch developer, árvore inicialmente limpa. Pedido: planejar perfil, dados, planos, usuários e permissões com a skill arquiteto-de-software. Esclarecimento explícito do responsável: somente a camada da conta do usuário; administração global da plataforma será feita depois. Status: **PROPOSTA**, implementação **PLANEJADA**. Este complemento preserva os estados históricos acima e não cria tickets READY nem aprova decisões LF-MT.

### Objetivo e estado observado

Dar ao usuário controle sobre sua própria conta, equipe, projetos, integrações e limites, preservando os gates de execução e aceite da fábrica. Gestão de todas as contas clientes, catálogo/preços globais de planos, workers, pausa global, infraestrutura e suporte administrativo da plataforma ficam fora deste escopo.

- [AdminAuthGuard](../../apps/api/src/control/admin-auth.guard.ts) usa um ADMIN_API_TOKEN compartilhado; não identifica cada pessoa. [App](../../apps/web/src/App.tsx) mantém esse token em sessionStorage.
- [Schema Prisma](../../apps/api/prisma/schema.prisma) não contém User, Tenant, Membership, ProjectGrant, Session ou plano comercial. [SettingsService](../../apps/api/src/settings/settings.service.ts) usa FactorySettings global/versionado; não é configuração por organização.
- [Navegação](../../apps/web/src/DashboardLayout.tsx) tem Usuários como prévia. SettingsPanel já oferece Contas de IA, Integrações IA e Equipes de agentes; pessoas e agentes são entidades distintas.
- FAC-037 permite cadastrar chaves API criptografadas e assinaturas CLI, com seleção de IA/modelo. Execução agentiva por API continua indisponível; esta proposta não amplia essa autorização. [Estado da configuração](../03-modulos/configuracao/00-README.md#cadastro-api-ou-assinatura-cli--fac-037).
- Tenant/RLS/broker continuam propostas [LF-MT](11-plano-lf-mt-proposto.md). Um painel multiusuário não comprova isolamento para clientes nem execução segura de código de terceiros.

### Estrutura de navegação e produto

Minha conta entra pelo menu do avatar. A área Administração da conta agrupa organização/equipe, pessoas, acesso, integrações, uso e auditoria apenas da conta atual. Conta individual pode ter um único membro; a estrutura de organização permite convidar outras pessoas sem confundir perfil pessoal com recursos compartilhados. O painel global da plataforma será tratado em trabalho posterior. Reutilizar o tema Centro de Comando, formulários com rascunho e salvar/cancelar, estados vazios e mensagens acessíveis.

| Área | Conteúdo proposto | Prioridade arquitetural |
|---|---|---|
| Minha conta | Nome, avatar opcional, email verificado, idioma, fuso, preferências e notificações; segurança e sessões | Inicial |
| Organização | Nome, responsável, identidade visual simples, fuso padrão e membros; dados fiscais quando houver cobrança | Inicial; fiscal posterior |
| Usuários e acesso | Convite, aceite, reenvio/revogação, papel, projetos permitidos, suspensão e remoção de vínculo | Inicial |
| Segurança | Login individual, recuperação, revogação de sessões, proteção de ações sensíveis; MFA para perfis privilegiados como requisito proposto | Inicial |
| Visão da conta | Membros ativos/convites, projetos, tarefas aguardando aceite, execuções/esperas e alertas reais apenas da conta | Após identidade e escopo |
| Integrações | Reaproveitar IA/GitHub existentes; definir dono, projetos autorizados e responsáveis pela manutenção | Inicial; ownership exige adaptação |
| Uso e limites | Execuções, tempo, armazenamento, uso observado por provider e limites internos | Segunda etapa |
| Plano da La fabrique | Plano vigente, direitos/limites, ciclo e histórico de mudanças | Condicional à oferta comercial |
| Cobrança | Dados fiscais, faturas, pagamento, troca/cancelamento e inadimplência | Etapa comercial posterior proposta |
| Auditoria e dados | Quem alterou acessos/configurações, iniciou/cancelou/aceitou; exportação e retenção por escopo | Auditoria inicial; exportação posterior |

Notificações úteis: entrega pronta para aceite, execução bloqueada, integração precisando de autenticação, limite observado, convite e mudança de acesso. Preferências pessoais podem reduzir alertas comuns; eventos de segurança devem ter política própria.

### Papéis e permissões

Começar com papéis fixos e acesso explícito por projeto, evitando editor de papéis personalizados na primeira versão. Nome de papel não substitui verificação da permissão no servidor.

| Papel humano | Capacidade padrão proposta | Limites |
|---|---|---|
| Proprietário | Organização, usuários, integrações, plano, dados e transferência de titularidade | Proteger último proprietário ativo; transferência exige confirmação do destinatário |
| Administrador | Equipe e configurações da organização, dentro da delegação recebida | Sem transferência de titularidade ou cobrança por padrão; não pode conceder autoridade superior |
| Operador de projeto | Criar/editar tickets, solicitar execução e controlar runs dos projetos concedidos | Sem gestão de pessoas, planos ou credenciais; retomada respeita stop/fence |
| Aprovador de projeto | Consultar evidências e aceitar a revisão exata nos projetos concedidos | Aceite é permissão específica; não concede publicação/merge/deploy |
| Leitor de projeto | Consultar tickets, status e evidências autorizadas | Sem mutações nem acesso a segredos |

Um membro pode acumular operador e aprovador conforme política; separação obrigatória entre execução e aceite deve ser decisão explícita. Permissões de PR/publicação são independentes. Papéis desta matriz valem somente dentro da conta. Nenhum concede administração global da La fabrique; esse painel e sua matriz serão definidos depois.

Fluxo de convite: PENDING → ACCEPTED, EXPIRED ou REVOKED. Email do convite deve corresponder à identidade verificada; token de uso único fica armazenado como hash, com validade. Reenvio invalida convite anterior. Suspensão revoga sessões/acessos futuros e dispara tratamento conservador dos jobs afetados; nunca marca writer como parado sem prova. Não apagar autorias históricas ao remover vínculo.

### Domínios, dados e contratos propostos

Preservar React/Vite, NestJS modular, worker Node/TS, Postgres, Redis/BullMQ e VPS única conforme ADR-003 aceito. Nenhum serviço novo é necessário nesta proposta.

- Identidade: User, Session e convites/recuperação. Recomendação inicial: sessões revogáveis no servidor e cookie HttpOnly/Secure, com proteção CSRF e política de origem; método de login local ou identidade externa permanece a decidir. WorkerAuthGuard continua uma fronteira independente.
- Organização: reutilizar o conceito Tenant já proposto, exibido como Organização, com Membership e ProjectGrant. Não criar Account e Tenant concorrentes para o mesmo ownership. Perfil/preferências são do usuário; integrações/limites compartilhados são da organização; projeto tem grants próprios.
- Política de acesso: autorização por ação, membership ativa e projeto; negar por padrão. Plano habilita um recurso, papel autoriza a ação e gates de execução validam sua realização. Nenhuma dessas camadas substitui as outras.
- Plano: PlanVersion/Entitlement e vínculo da organização. Subscription/Invoice/Payment são condicionais à cobrança; não criar essas tabelas apenas para mostrar assinatura CLI. Versões do plano preservam direitos já contratados.
- Uso: UsageObservation com fonte, observedAt e valor desconhecido permitido. Quotas internas precisam de medição/reserva transacional antes da admissão, para impedir ultrapassagem concorrente. Não prometer quota de tokens que o provider não informa.
- Auditoria: AuditEvent vincula ator humano, organização/projeto, ação, alvo, data, versão e resultado, sem tokens/chaves/PII desnecessária. Ações sensíveis gravam alteração e auditoria na mesma transação. Notificações usam outbox existente quando necessário.

Famílias de contratos a especificar no Tech Lead: identidade/sessões, meu perfil, organização/membros/convites, grants de projeto, auditoria, uso e direitos do plano. DTOs Zod em packages/contracts; dados do usuário não entram em prompts por padrão. Escritas administrativas usam versão esperada e conflito de atualização; convite/reenvio e futuros webhooks comerciais exigem deduplicação definida.

Autorização deve cobrir também listagens, contagens, busca, exportações, artefatos, SSE e jobs. Na execução, preservar a cadeia organização → projeto → run/attempt → instalação autorizada, com revalidação no claim. Revogação encerra streams e acessos futuros; revogação de jobs exige reconciliação e stop real.

### Planos da fábrica e assinaturas de IA

Nesta camada, o usuário consulta e eventualmente troca seu plano da La fabrique; não cadastra planos, preços ou limites globais da plataforma. Plano comercial da La fabrique pode diferenciar membros ativos, projetos ativos, retenção, armazenamento e recursos administrativos. Exemplos conceituais: Individual, Equipe e Organização; nomes, preços, quantidades e política de excedente permanecem a definir. Não oferecer mais writers/concurrency como benefício antes de rever a capacidade e o invariante atual de um executor global.

Assinatura de IA representa provedor, instalação, modo CLI/API, titularidade, estado de autenticação, modelos e limite observado com fonte/data. Renovação/reset não informado permanece desconhecido. Custo fixo de assinatura, cobrança da fábrica e eventual custo API ficam separados; tokens CLI não viram preço API. Novo plano da fábrica não ativa execução API, fallback ou gastos.

### Alternativas e sequência de evolução

Alternativa A: equipe interna primeiro, uma organização explícita, papéis/grants e sessões individuais; implementar ownership para os recursos expostos sem prometer isolamento comercial. Menor migração inicial, mas exige gate adicional antes de receber clientes.

Alternativa B: organizações clientes desde o início. Requer ownership completo e isolamento ponta a ponta (incluindo instalação, fila, artifacts, SSE, backup e configuração), além dos gates LF-MT aplicáveis. Mais trabalho inicial; evita uma segunda migração dos mesmos dados. Criar somente tenant_id em Project é insuficiente.

Escopo confirmado pelo responsável: painel da camada de usuário primeiro; painel administrativo global depois. Hipótese ainda não aceita: evolução da conta/equipe sem cobrança automática inicialmente. A disponibilidade inicial para contas clientes e o significado de planos permanecem a esclarecer; os ramos comerciais seguem condicionais. Ambas as alternativas acima tratam a conta do usuário, não um painel global.

Ordem por dependência, sem tickets/sprints:

1. Confirmar público e planos; consolidar matriz de acesso, login e titularidade de integrações. Registrar ADR proposto relevante antes da implementação estrutural, relacionando as propostas LF-MT existentes sem alterar ADRs aceitos.
2. Identidade, sessões e organização inicial; migração explícita do administrador/dados existentes. Inventário e dono confirmado, sem atribuição arbitrária de órfãos; descontinuar token compartilhado após transição verificável.
3. Convites, papéis, grants, auditoria e migração das rotas existentes para a autorização individual. Cobrir downloads/SSE/jobs e impedir rotas antigas de contornar o controle.
4. Telas Minha conta/Organização/Usuários e resumo da própria conta com dados reais; reaproveitar integrações preservando seu escopo e gates. Nenhuma tela de gestão global da plataforma.
5. Uso observado, notificações e quotas internas consistentes; exportação/retention conforme ownership.
6. Se clientes forem objetivo inicial, os gates de isolamento antecedem onboarding desses clientes. Cobrança automatizada vem após plano/medição/regras comerciais definidos.

### Migração, critérios e pendências para Tech Lead

Migrações aditivas, backfill com owner confirmado e restrições após reconciliação; preservar revisão, authoria histórica, checkpoints e writer global. Reversão exige plano que não reabra token global para contas clientes; desabilitar novas admissões quando não for possível preservar fronteiras. Nenhuma migration/ativação foi realizada nesta análise.

Critérios propostos: usuário sem grant não vê projeto nem suas contagens/artefatos/eventos; acesso via HTTP direto não contorna UI; convite expirado/reutilizado é negado; último proprietário não é removido; suspensão revoga sessão e acesso de streams; job revogado não inicia nova attempt; transição para parado exige prova; alterações sensíveis têm autoria; aceite continua por digest exato; métricas desconhecidas não aparecem como zero; quota não excede sob concorrência; mudança de plano não remove projetos/dados silenciosamente.

Decisões abertas: equipe própria versus clientes imediatos; plano comercial versus assinatura IA; login escolhido; obrigatoriedade de MFA; integrações pertencem à organização ou ao usuário; separação operador/aprovador; cobrança e excedentes; retenção/exportação/encerramento. Sem resposta, especificar alternativas no planejamento e manter dependências comerciais pendentes. Tech Lead deve converter a opção escolhida em tickets locais com critérios e gates; não há implementação autorizada por este handoff.

Registro: [PLAN-ADM-001](../09-entregas/2026/2026-10-09-PLAN-ADM-001-painel-administrativo-conta.md).


## Execução FAC-038 — primeira versão da conta

Pedido “Ok pode executar” autoriza a implementação focal da camada de usuário. [FAC-038](tickets/FAC-038.md) especifica o incremento executável. Implementados cadastro/login individual, perfil/preferências, organização, membros/papéis, convites privados, segurança/sessões e auditoria; plano/uso mostram apenas o disponível. [ADR-007](../06-decisoes/ADR-007-conta-usuario-e-sessoes.md) delimita a conta frente ao token administrativo legado. Revisão/aceite/implantação pendentes. Nenhuma sessão independente foi iniciada; não inferir aprovação de code review/QA.

Dependências posteriores: FAC-039 (ownership/grants projetos, artefatos/SSE/jobs e transição das rotas globais), FAC-040 (ownership integrações/uso real da conta) e FAC-041 (verificação/MFA/recovery e requisitos comerciais). Permanecem PLANEJADAS, sem tickets READY ou autorização de cobrança/gastos. Novas contas não possuem os projetos/settings globais por atribuição automática. Matrizes globais de administração ficam para depois, conforme pedido explícito.

## Execução dos complementos autorizados

O pedido “termine todas as outras partes” tornou READY os [tickets FAC-039–041](09-backlog.md#fac-039041--complemento-da-camada-de-usuário). Implementação de software segue a [feature atual](../04-features/01-control-settings.md#painel-da-conta-do-usuário--fac-038). Configuração de catálogo/serviço SMTP/cobrança foi consultada ao responsável e não respondida; adaptadores não ativados e sem preços inventados. Não descontinuar token operacional legado nem atribuir seus projetos sem dono confirmado. Gates de review/aceite/produção/LF-MT permanecem próprios.

## Handoff arquitetural — memória e decisões (2026-10-09)

[Planos separados](planos-memoria-e-decisoes/00-README.md) e [requisitos/evidências/pendências](planos-memoria-e-decisoes/01-decisoes-e-dependencias.md) constituem o handoff desta análise. Objetivo: memória persistente e contexto da própria La fabrique; base inspecionada `896ef0d2c804b80711d5ad00ad96695eaaf9f2d7`; main de referência ainda requer conferência, pois a ref local aponta ao bootstrap. Stack e VPS única preservadas; recursos declarados 2 vCPUs/8 GB.

Estado alvo: memória versionada, índice/grafo, packs/expansão/continuidade, decisões auditadas, atualização/conflitos e painel. Modelos locais, integração JEV e treinamento ficam para tarefas futuras. [Contratos comuns](../05-contratos/memoria-e-decisoes/00-contratos-propostos.md) e contratos por plano são propostas, não software existente. [Matriz](planos-memoria-e-decisoes/10-matriz-fluxo.md) conserva 145/56/20 itens, sem provar implementação individual.

Sequência por dependências: fontes/autoridade → índice → pacotes → integração com decisões/workflow/painel e avaliação → modelos/treino futuros. Riscos: revisão main divergente, lacunas dinâmicas do grafo, dados/índices obsoletos, limites host inadequados e consumo externo desconhecido. Migrações devem ser aditivas/versionadas, preservar contratos atuais e permitir fallback somente quando cumprir obrigatórios.

Tech Lead deverá gerar backlog/tickets somente após avaliação das propostas técnicas relevantes; não reutilizar FAC ilustrativos da transcrição, não reabrir stack nem reimplementar tickets aceitos por leitura histórica. Preencher orçamento JEV pelo sistema e benchmark antes de ativação/modelos. Exceção específica [ADR-008](../06-decisoes/ADR-008-excecao-jev.md) não libera APIs de LLM. Sem implementação/push/deploy neste handoff.
