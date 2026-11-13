> Leitura: [← Anterior](00-catalogo-rotas.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](02-worker.md)

# API de controle implementada

Base: a2cc5e0. Fonte de schemas: [packages/contracts/src/index.ts](../../../packages/contracts/src/index.ts); prefixo global `/api`. Consumers: painel e operador administrativo. Auth: Bearer ADMIN_API_TOKEN validado por AdminAuthGuard; token do worker não equivale ao administrativo. Sem membership/grants/tenant implementados.

## Requests, responses e efeitos

| Operação | Request e response canônicos | Efeito / concorrência |
|---|---|---|
| POST /api/projects | createProjectSchema → projectSchema | Cria Project; não clona nem valida acesso real ao repo |
| GET /api/projects | sem body → projectListSchema | Leitura administrativa |
| PATCH /api/projects/:id/base-revision | updateProjectBaseRevisionSchema → projectSchema | expectedBaseRef + baseRevision SHA exato; comparação otimista |
| GET /api/projects/:id/definition | projectDefinitionStateSchema | definition nullable, sem default sintético |
| PUT /api/projects/:id/definition | putProjectDefinitionSchema → projectDefinitionSchema | expectedVersion=0 na criação, versão positiva no update; preserva JSON validado |
| POST/GET /api/projects/:id/tickets | createTicketSchema / ticketSchema ou ticketListSchema | Cria DRAFT/lista; title/objective/acceptanceCriteria com limites |
| POST /api/tickets/:id/ready | readyTicketSchema → ticketSchema | Ticket+outbox atômicos, snapshot ExecutionSpecification, dedupe ticket:{id}:ready |
| GET /api/runs?projectId=UUID e /api/runs/:id | runSummarySchema / runDetailSchema | Até 50 runs/tentativas recentes; não cria Run |
| POST /api/runs/:id/control | requestRunControlSchema → runControlReceiptSchema | expectedVersion/attemptId/action PAUSE ou CANCEL; pending=true é intenção, não stop |
| POST /api/runs/:id/resume | requestRunResumeSchema | expectedVersion/attemptId/artifactDigest; valida checkpoint/bundle/stop e publica outbox, nova tentativa/fence |
| POST /api/runs/:id/recover-finalization | requestRunRecoverySchema → runRecoveryReceiptSchema | FINALIZATION_ONLY; REQUESTED não significa concluído; sem novo writer/IA |
| GET /api/runs/:id/delivery | runDeliveryStateSchema | delivery nullable e reason normalizado; Markdown/digests exatos |
| POST /api/runs/:id/approve-delivery | approveRunDeliverySchema → runSummarySchema | version/attempt/deliveryDigest, gate válido em VALIDATING; grava approval e DONE |
| GET /api/projects/:id/events e events/stream | projectEventCursorSchema → páginas/SSE | Cursor decimal string persistido; Last-Event-ID, Bearer header, polling fallback |
| GET /api/operation; PUT /api/operation/scheduling | operation DTO; updateFactorySchedulingSchema | Observação read-only; pause com expectedVersion, default true no DB |
| GET/PUT /api/settings e subrotas de integrações | schemas factory/settings/onboarding/verification/PR | Config singleton versionada; states só por evidência worker; PR preparar/aprovar separado |

[Catálogo completo de controllers](00-catalogo-rotas.md) inclui health, integrações, login/challenge e protocolo interno. Os schemas linkados são fonte única dos campos: não copiar todos os DTOs para este capítulo.

## Divergências resolvidas

Não existe POST /runs nem POST /tickets/:id/runs no código. READY e claim implementam o fluxo; não há header Idempotency-Key genérico. POST /tickets da ESPEC histórica corresponde fisicamente à rota aninhada por projeto. Pause/cancel são action no endpoint control; aprovação é approve-delivery. Eventos são por projeto, não GET /runs/:id/events. Nenhum alias foi implementado por esta migração.

## Erros e versionamento

400 em payload/UUID inválido; 401 em auth ausente/incorreta; 404 quando objeto não é encontrado; 409 em conflito de estado/versão/revisão/evidência. Respostas Nest padrão não são envelope de erro versionado compartilhado; OpenAPI completo é pendência. POST usa status padrão Nest quando não há override, mas esta migração não testou HTTP real. Tenancy e convenção 403/404 são alvo posterior.

Schemas de execução usam schemaVersion=1; configuração/entidade têm version otimista distinta. Alterar código/schema exige revisar produtores/consumidores e compatibilidade legada; não aceitar documentação como migration. Efeitos externos nunca são inferidos do exit code.

## Exemplo sintético, sem chamada HTTP

```json
{"title":"Validar campo","objective":"Recusar campo vazio","acceptanceCriteria":["Campo vazio é recusado"]}
```

Body válido de criação de ticket sob projeto cadastrado; UUID e Bearer só vêm do ambiente autorizado. READY envia `{"expectedVersion":1}`; repetir quando ainda READY e outbox existe devolve ticket sem outro evento. Quando já avançou, comportamento depende do estado e pode retornar conflito.

## Testes e limites

Testes versionados: control.service.spec.ts, run-control/resume/recovery/delivery e execution-results/artifacts specs em apps/api/src; contratos em packages/contracts/src. Não executados nesta migração. Rotas e schemas foram conferidos por leitura local. Revisão semântica independente/aceite da revisão documental permanece pendente.

## API da conta do usuário — FAC-038

Rotas /api/account usam AccountGuard/cookie próprio, não ADMIN_API_TOKEN nem WORKER_API_TOKEN. Login/cadastro são públicos da camada de identidade, mas exigem X-Account-Request: 1, origem permitida e rate limit Redis. Todos os contratos são [schemas de conta](../../../packages/contracts/src/account.ts); [controller](../../../apps/api/src/account/account.controller.ts). GETs de sessão/overview/sessions e emissão de convite usam Cache-Control: no-store.

| Método e path relativo a /api/account | Contrato/efeito |
|---|---|
| POST /register; POST /login | accountRegisterSchema/accountLoginSchema → accountSessionSchema + cookie privado |
| GET /session | Perfil/conta atual, organizações ativas e CSRF; nenhum segredo de sessão |
| GET /overview | Membros autorizados, convites/auditoria somente gestores, contagens reais por escopo, storage/reservas reais e tokens observados nullable |
| PUT /profile; PUT /organization | Rascunho validado, expectedVersion, gravação/auditoria atômicas |
| PUT /session/organization | Membership ativa obrigatória; muda seleção da sessão |
| POST /invitations; DELETE /invitations/:id | Convite 72h por link privado/uso único; reemissão revoga anterior; token só na resposta de criação |
| POST /invitations/accept | Conta autenticada/email correspondente verificado; lock serializa aceite concorrente |
| PATCH /members/:id | Papel/status/version; escopo da conta, owner protegido e limite de delegação |
| GET /sessions; DELETE /sessions/:id | Sessões próprias, sem cookie/hash/CSRF; revogação por proprietário da sessão |
| PUT /password | Senha atual obrigatória; novo hash e revogação das outras sessões na transação |
| POST /logout | Remove sessão corrente e limpa cookie |

Mutações exigem X-Account-CSRF e X-Account-Organization (conta esperada). Conta mudou em outra aba → 409; UUID externo/membership ausente → 404 nos alvos; papel/origem/CSRF inválidos → 403; sessão inválida/expirada → 401; rate limit → 429; Redis/hash capacity indisponíveis → 503. Body inválido → 400. Mesmo no acesso HTTP direto não existe fallback para token administrativo. Cookies humanos não abrem rotas antigas de projeto/settings/operation. Config WEB_ORIGIN define origem exata quando presente; instalação recomendada usa frontend/API no mesmo origin.

## Recursos complementares da conta — FAC-039–041

[Controller work](../../../apps/api/src/account/account-work.controller.ts), [features](../../../apps/api/src/account/account-features.controller.ts), [canal worker](../../../apps/api/src/account/account-worker.controller.ts) e [contratos runtime](../../../packages/contracts/src/account.ts) são fontes executáveis; paths exatos em [ROTAS-OBSERVADAS.json](ROTAS-OBSERVADAS.json).

| Família | Autorização e contrato |
|---|---|
| /account/work/projects, definition, base-revision, tickets, grants, routes | AccountGuard; leitura exige grant/gestão, operação OPERATOR em conta/projeto, configuração/grants só gestores; contratos existentes de projeto/ticket/definição e accountCreateProjectSchema/accountGrantSchema |
| /account/work/runs, delivery, approval, control, resume, recovery, artifacts | AccountGuard e projeto derivado do run no servidor; aceite APPROVER/gestor, operação OPERATOR/gestor; contratos existentes de digest/versão/stop/fence |
| /account/work/projects/:id/events e /stream | Mesmo escopo; stream verifica sessão/grant a cada poll, sem confiar em autorização inicial |
| /account/integrations e /projects/:projectId, login/verify/challenge/code | Gestores; segredo somente na entrada, AES-GCM privado, grants compostos; DTOs sanitizados e contratos oficiais de onboarding existentes |
| /account/recover, /reset, /verify | Públicos com origem/cabeçalho e rate-limit; tokens de uso único, resposta genérica na recuperação |
| /account/email, /mfa | Cookie/CSRF/rate-limit; senha e segundo fator para ações sensíveis; MFA/recovery codes retornam uma vez na configuração |
| /account/notifications, /read | Usuário/organização correntes e grants atuais; leitura individual, limite de 100 |
| /account/data/settings, /data/export | Dados fiscais/retenção somente owner; exportação por seção/cursor própria, 200 registros/página, gestores para dados da equipe |
| /account/ownership, /archive | Owner/aceite do destinatário, senha/segundo fator e invariantes de owner/stop/assinatura |
| /account/billing, checkout, portal, reconcile | Consulta de plano da conta; dados fiscais/faturas/ação comercial só owner; intenção persistida/idempotency key |
| /account/billing/webhook | Sem cookie humano: raw body/Stripe-Signature obrigatório, timestamp/HMAC/dedupe e reconsulta da Stripe |
| /internal/account-repositories/:attemptId/credential | WorkerAuthGuard, worker/attempt/fence/lease atuais, job/grant ainda autorizado; no-store; credencial somente no supervisor |

Todos os endpoints protegidos da conta são no-store. Contexto de conta fornecido também é verificado em GETs; browser reconsulta /session sem contexto antigo para atualizar a seleção. Negar 404 para recurso de organização diferente e 403 para recurso próprio sem grant não expõe IDs externos. Serviços de domínio se unem à transação do check via account-transaction.ts; nenhuma commit separado entre autorização e mutação. Registrar chaves API não habilita execução API/gasto.

FAC-041: overview inclui `awaitingApproval`, `executionSeconds` nullable, `executionTimeIncomplete`/`executionTimeObservedAt` e `aiUsage.byProvider` (tokens nullable quando não observados). Exportação aceita a seção `attempts`, paginada e restrita aos projetos autorizados, com resultado público sanitizado; não inclui worker/fence/credenciais nem artefato bruto.
