> Leitura: [← Anterior](00-README.md) · [Índice didático](../02-INDEX.md) · [Próximo assunto →](schemas/00-entidades.md)

> Nota editorial DOC-MV-001, 2026-10-07: conteúdo integral local preservado como referência da evolução v2.3. Afirmações por incremento têm a revisão/data original; trechos sobre consumer ausente, recuperação pendente e caminhos antigos não descrevem necessariamente HEAD a2cc5e0. O [AS-IS conferido](../02-arquitetura/01-as-is.md) e a [auditoria](../00-governanca/08-AUDITORIA.md) delimitam o estado atual; números de testes abaixo são evidência histórica, não reexecução desta migração.

# Especificação implementável v2

## Decisão obrigatória da stack - revisão 2.3
A stack da própria La fabrique está APROVADA: React + TypeScript + Vite no painel; NestJS + TypeScript na API; worker Node.js + TypeScript em processo separado; PostgreSQL; Redis + BullMQ; Docker Compose na mesma VPS. Não solicitar nova escolha ou confirmação da stack. Não iniciar a fábrica em PHP/Laravel, Angular ou .NET. Esta decisão substitui propostas anteriores.
Monorepo: `apps/web`, `apps/api`, `apps/worker`, `packages/contracts`. Contratos compartilhados precisam de validação em runtime. API não executa clientes, builds ou testes; o worker executa esses trabalhos com isolamento, limites e um writer inicial.
Ao trabalhar na própria fábrica, aplicar esta stack. A regra de preservar a stack existente aplica-se somente a projetos EXTERNOS cadastrados para desenvolvimento pela fábrica; ela não altera a stack da La fabrique. Se o repositório da fábrica contiver implementação anterior incompatível, registrar a divergência e planejar a adaptação por etapas; não apagar código existente nem reabrir a escolha tecnológica.
Versões exatas e comandos devem ser fixados conforme compatibilidade no bootstrap; isso não é uma nova decisão de stack. Repositório e funcionalidade do piloto externo permanecem pendentes quando não fornecidos.

Status: IMPLEMENTACAO INCREMENTAL. Controle, runtime e configuracoes descritos no backlog existem parcialmente; nenhum contrato abaixo é API de fornecedor.
## Dados e invariantes
Project: repo/ref e uma ProjectDefinition versionada com descricao, stack externa, instrucoes, caminhos e checks cadastrados. READY copia projeto, definicao e ticket para uma ExecutionSpecification imutavel na outbox; a definicao nao contem conta/modelo de IA. Ticket: objetivo/aceite, risco, runtime_limits, budget opcional, status. Run: revisão/base, branch, estado, policy_version e version para lock otimista. Step/Attempt: papel, tentativa, provider, lease_owner/expires, fencing_token, processo/session IDs quando observáveis.
Worker: ID, capacidades verificadas, OS/arch/versão, last heartbeat, disponibilidade e resource limits. ProviderInstallation: worker/provider, cli_version, auth_method, billing_mode, models_verified, capabilities, status e evidência de preflight; sem tokens pessoais.
UsageObservation: instalação, janela/modelo/unidade, valor nullable, reset nullable, fonte e observed_at. AgentExecution: attempt, input/context hashes, result status, modelo efetivo nullable, usage nullable, limites e timestamps. Checkpoint: code SHA, patch/untracked artifacts, stopped_confirmed, next actions e schema_version.
ContextManifest: revisão, fontes/hashes, tamanho estimado, omissões/truncamentos e instruction hashes. Artifact: caminho validado/hash/tamanho/tipo/retention. Approval: actor, run e revisão exata. LedgerEntry: fixed_subscription/extra/api, moeda, valor decimal nullable, alocação/estimativa/reconciliação e fonte.
Constraints: event_id e sequence únicos por attempt; um writer por workspace; transições versionadas; aprovação vinculada ao code SHA; dispatch via outbox; artefatos imutáveis por hash. Nunca dinheiro em float.
## APIs do controle
POST /projects; GET/PUT /projects/{id}/definition; POST /tickets; POST /tickets/{id}/runs com Idempotency-Key; GET /runs/{id}/events; POST /runs/{id}/pause, cancel, resume, approvals. GET /workers; GET /providers; PATCH /providers/{id}/policy. Provider policy não aceita senha/token OAuth nem pode habilitar gasto via evento de agente.
Worker protocol em documentacoes/operacao/README.md. Autorização por projeto/worker, limites de upload, sanitização e audit log. 409 para estado/revisão obsoleta; 422 contrato incompleto; 401/403 identidade/permissão.
## Máquina de estados
DRAFT -> READY -> WAITING_WORKER -> RUNNING -> VALIDATING -> REVIEW -> DOCS -> AWAITING_HUMAN -> DONE. Estados laterais WAITING_PROVIDER, PAUSED_LIMIT, PAUSED_RESOURCE, BLOCKED_RECOVERY, AUTH_REQUIRED, FAILED e CANCELLED. Resume é nova tentativa baseada em checkpoint persistido; não salto direto a DONE.
Cada transição guarda ator/evento/revisão/motivo. DONE requer aceite; cancelamento pode terminar como resultado desconhecido até confirmar fim. Uso indisponível não pode impedir relatório honesto, mas precisa constar desconhecido.
## Fluxo
Inspeção determinística -> contexto -> lock writer -> adapter execute -> checks reais -> reviewer separado -> correção limitada -> documentação -> gate -> aceite. Cliente oficial executa tools dentro do isolamento; supervisor controla lifecycle e valida outputs. Não tratar evento de IA como comando confiável.
## Gate documental
Comparar base/code revision, exigir relato por domínio, diff/patch associado, checks com resultados e revisão, README atual e atualização de índice/changelog/backlog. Validar links/estrutura, evitar placeholder em relatório IMPLEMENTADO e segredos. Revisão semântica confere conteúdo; presença de arquivo não prova verdade.
## Checks essenciais do MVP
Idempotência de claim e eventos; worker desconectado interrompe writer; processos antigos não continuam após handoff; arquivos untracked recuperáveis; auth/cota tratados sem loop; API key herdada bloqueada; extras desligados verificados; segredo inacessível ao código do piloto; doc gate rejeita entrega incompleta; aceite invalidado por novo diff.
Mocks/fixtures sanitizadas para adapters; integração real opt-in no preflight dentro da assinatura validada. Medir ensaio de falha por simulação de adapter, sem desperdiçar cota real de propósito.

## Stack obrigatória da fábrica - revisão 2.3
React + TypeScript + Vite no painel; NestJS + TypeScript na API; worker Node.js + TypeScript em processo separado; PostgreSQL e Redis + BullMQ. Monorepo apps/web, apps/api, apps/worker e packages/contracts. Ler documentacoes/arquitetura/ADR-003-stack-typescript.md.
Compartilhar esquemas/DTOs e validar dados em runtime; impedir import de segredos/código servidor no painel. Outbox, idempotência, leases e fencing seguem obrigatórios: lock BullMQ não substitui exclusão do writer. API não executa builds/clientes. Executar typecheck, lint, builds e testes relevantes. Preservar a stack somente de pilotos externos; a própria fábrica segue a stack aprovada.
