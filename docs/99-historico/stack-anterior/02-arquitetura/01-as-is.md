> Leitura: [← Anterior](00-README.md) · [Índice didático](../02-INDEX.md) · [Próximo →](02-contexto.md)

# Estado observado versus direção proposta

Revisão de código inspecionada: `a2cc5e0db0f30bf227dfc64300f74bde23a6f412`, branch de origem developer, 2026-10-07 (Europe/Berlin). Código presente não prova serviço implantado ou eficácia operacional. Nenhum teste de software foi reexecutado nesta migração documental.

| Área | AS-IS inspecionado | Direção / evidência ainda necessária |
|---|---|---|
| Web | React/Vite/TypeScript; home demonstrativa, App, painel de definição e SettingsPanel | Aceite visual exato; dados da home não são métricas operacionais |
| API | NestJS modular: Control, Settings, Orchestration, WorkerIdentity, Health, Infrastructure | Sem src/modules fictício; identidade multiusuário/tenancy não implementadas |
| Auth | AdminAuthGuard e WorkerAuthGuard usam Bearer distintos; comparação de hashes constante | Membership, grants por projeto e convenção antienumeração são propostas |
| Dados | Prisma/PostgreSQL: Project, ProjectDefinition, Ticket, Run único por ticket, Attempt, Checkpoint, OutboxEvent, FactorySettings, integrações, eventos e pausa | Sem Tenant/ProjectGrant/RLS/Step/Ledger físico; configuração/resultado JSON não são tabelas separadas |
| Dispatch | READY transacional; dispatcher publica ticket.ready.v1, run.resume.v1, run.finalization-recovery.v1 | Outbox está no banco; Redis transporta BullMQ e desafios efêmeros |
| Writer | OrchestrationService + global-writer-guard + índice parcial stopped_confirmed=false | Lease vencida ou heartbeat stale não provam parada; índice deve existir no DB efetivo |
| Worker | Consumer real condicionado a WORKER_EXECUTION_ENABLED; default false no config | Valor do ambiente ativo não lido; não afirmar que o processo ativo está habilitado/desabilitado |
| Context/checks | ContextBuilder, SandboxRunner, compilador, DeveloperWorkflow, documentação e Reviewer separado | Ensaios na identidade efetiva, medição de recursos e compatibilidade de cliente |
| Providers | CodexAdapter, ClaudeAdapter, stores por instalação e preflight Claude opt-in | Antigravity tem onboarding/verificação; não há adapter de execução operacional |
| Broker/tenant | Nenhum broker remoto, capability service ou tenancy encontrado no código versionado | Propostas LF-MT; CLI sem ponte suportada permanece incompatível com perfil futuro, sem workaround OAuth/API/auth mount |
| Backup | evidence-backup.ts e CLI offline; testes versionados para roundtrip sintético | Storage externo, agenda, retention efetiva, RPO/RTO reais não comprovados |
| CI | Nenhuma configuração .github/workflows ou pipeline versionada localizada | Validador manual instalado; CI não é afirmada por esta entrega |

## Fluxo real

Operador autentica → cadastra projeto → configura definição/checks/documentação → resolve base SHA de 40 caracteres → cria ticket → POST /api/tickets/:id/ready com expectedVersion → transação grava ticket e outbox com ExecutionSpecification → dispatcher BullMQ → worker pede claim interno → serviço cria/recupera Run e Attempt com fence → checkout/workflow se enabled → stop, resultado, bundle, checkpoint e complete → Run/Ticket VALIDATING → aceite humano exato → DONE.

AWAITING_HUMAN é resultado do workflow e estado possível de Ticket, mas o gate de aceite persistido exige VALIDATING. Não transformar automaticamente cada etapa Developer/Reviewer em um enum Run. Um Run por ticket, várias Attempts; retomada explícita gera nova tentativa/fence. Handoff interno Y troca cliente/worktree mantendo a autoridade da tentativa, diferente de resume administrativo.

## Evidência de implementação e limites

As fontes primárias locais são [AppModule](../../apps/api/src/app.module.ts), [controllers de controle](../../apps/api/src/control/control.controller.ts), [serviço de orquestração](../../apps/api/src/orchestration/orchestration.service.ts), [worker](../../apps/worker/src/main.ts), [contratos Zod](../../packages/contracts/src/index.ts) e [schema Prisma](../../apps/api/prisma/schema.prisma). Migrations preservam índice global/triggers/check constraints não expressáveis no Prisma; não aplicar migration por leitura deste manual.

Relatos antigos afirmam testes nas revisões de seus tickets. São evidências históricas, não resultado desta migração. OPS-008/009 registra DB dev divergente e duas attempts sem stop; não consultamos DB atual. CONTROLE-MVP e READMEs antigos tinham frases por incremento já superadas; referências integrais têm nota editorial. Nenhum software foi corrigido para atender proposta nesta tarefa.

## Evolução focal FAC-038 — camada de conta

Diff feat/fac-038-painel-conta sobre ee4485b, 2026-10-09. A linha Auth da tabela acima descreve a revisão histórica a2cc5e0: permanece vigente para rotas globais legadas, mas não descreve a nova API /account. AccountModule implementa sessões/cadastro/login individuais, organização, vínculos/papéis, convites, preferências e auditoria próprios. Cookie humano e token administrativo são autoridades separadas. [ADR-007](../06-decisoes/ADR-007-conta-usuario-e-sessoes.md), [dados](06-dados.md#dados-da-conta--fac-038) e [feature](../04-features/01-control-settings.md#painel-da-conta-do-usuário--fac-038).

FAC-039–041 vinculam projetos novos, grants, jobs, integrações e roteamento à organização; legados permanecem sem atribuição automática. Email verificado/MFA/recuperação, notificações/exportação e adaptadores configuráveis SMTP/Stripe estão implementados. LF-MT/broker/RLS e isolamento hostil continuam propostos; não promover essa fronteira lógica a release comercial. Testes PostgreSQL são efêmeros; nenhum banco ativo/provider/pagamento foi alterado.
