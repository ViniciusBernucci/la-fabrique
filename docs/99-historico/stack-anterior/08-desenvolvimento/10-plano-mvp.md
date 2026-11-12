> Leitura: [← Anterior](09-backlog.md) · [Índice didático](../02-INDEX.md) · [Próximo →](11-plano-lf-mt-proposto.md)

> Nota editorial DOC-MV-001, 2026-10-07: conteúdo integral local preservado como referência da evolução v2.3. Afirmações por incremento têm a revisão/data original; trechos sobre consumer ausente, recuperação pendente e caminhos antigos não descrevem necessariamente HEAD a2cc5e0. O [AS-IS conferido](../02-arquitetura/01-as-is.md) e a [auditoria](../00-governanca/08-AUDITORIA.md) delimitam o estado atual; números de testes abaixo são evidência histórica, não reexecução desta migração.

# Plano do MVP v2

Estado 2026-10-03/FAC-012AH: escopo interno MVP implementado e verificado, 528 testes + 10 PostgreSQL + 3 Redis. [CONTROLE-MVP.md](06-controle-mvp.md) separa software, aceite e ativação. Piloto/dez tickets externos são experimento opcional posterior, jamais requisito para desenvolvimento ou conclusão da fábrica.

## Decisão obrigatória da stack - revisão 2.3
A stack da própria La fabrique está APROVADA: React + TypeScript + Vite no painel; NestJS + TypeScript na API; worker Node.js + TypeScript em processo separado; PostgreSQL; Redis + BullMQ; Docker Compose na mesma VPS. Não solicitar nova escolha ou confirmação da stack. Não iniciar a fábrica em PHP/Laravel, Angular ou .NET. Esta decisão substitui propostas anteriores.
Monorepo: `apps/web`, `apps/api`, `apps/worker`, `packages/contracts`. Contratos compartilhados precisam de validação em runtime. API não executa clientes, builds ou testes; o worker executa esses trabalhos com isolamento, limites e um writer inicial.
Ao trabalhar na própria fábrica, aplicar esta stack. A regra de preservar a stack existente aplica-se somente a projetos EXTERNOS cadastrados para desenvolvimento pela fábrica; ela não altera a stack da La fabrique. Se o repositório da fábrica contiver implementação anterior incompatível, registrar a divergência e planejar a adaptação por etapas; não apagar código existente nem reabrir a escolha tecnológica.
Versões exatas e comandos devem ser fixados conforme compatibilidade no bootstrap; isso não é uma nova decisão de stack. Repositório e funcionalidade do piloto externo permanecem pendentes quando não fornecidos.

Um projeto, um worker, um writer. Durante a construção, fixtures e repositórios sintéticos validam o núcleo; um projeto externo real pode entrar posteriormente em um experimento separado, sem bloquear a conclusão do MVP. Sprints por objetivos de 1-2 semanas sugeridas, sem datas contratuais. Somatório estimado: 23-35 dias de engenharia para FAC-001 a FAC-012; calendário depende de disponibilidade e compatibilidade. FAC-013 é evolução opcional.
V1 valida o núcleo e um runtime; V2 testa handoff; Sprint 3 conclui o software e verifica a operação com dados sintéticos; piloto externo é experimento posterior opcional. FAC-000, FAC-002 a FAC-009, FAC-010A/B/C e FAC-011A/B/C estão implementados e aceitos. FAC-010 aguarda validacao operacional; FAC-011D implementa PR sob gate e aguarda aceite. FAC-001 foi adiado até a plataforma estar pronta.
## Épicos
Sprint 1: controle, worker, preflight sintético e runtime. Sprint 2: execução recuperável e providers. Sprint 3: documentação, verificação integrada e preparação operacional sem dependência de piloto. Experimento externo: projeto escolhido posteriormente. Evolução: terceiro adapter/capacidades novas.

## FAC-001 — Contratar piloto
Sprint: Experimento externo opcional. Dependências: FAC-011. Esforço estimado: 1-2 dias. Status: DEFERRED.
Descrição: Após o núcleo da plataforma estar pronto, o responsável escolhe o projeto externo pela interface; então repo, baseline, regras, caminhos e tickets pequenos ficam registrados no software para o ensaio operacional. Nenhuma escolha concreta entra no codigo da fabrica.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Contrato verificável e baseline reproduzido; não editar lógica fora do ticket.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## FAC-001A — Definicao configuravel de projeto
Sprint: Sprint 3 antecipado como fundacao interna. Dependências: FAC-003, FAC-003A, FAC-011. Esforço estimado: 1 dia. Status: DONE.
Descrição: Persistir e editar no painel a descricao, stack externa, instrucoes, caminhos e checks de qualquer projeto, mantendo contas/modelos no Centro de Configuracoes.
Aceite: Contratos runtime, versao otimista, gate de READY e ausencia explicita de definicao; sem hardcode do piloto.
Entregáveis: migration aditiva, API, painel, testes e documentacao vinculados ao codigo `148485ec5858756460b1db6f183c346854703981`.

## FAC-002 — Validar clientes e baseline assistido
Sprint: Sprint 1. Dependências: FAC-004. Esforço estimado: 2-3 dias. Status: DONE.
Descrição: Fazer preflight de login, cobrança e isolamento na VPS; medir recursos com fixture/repositório sintético controlado e confirmar providers um a um. A execução de tickets reais e o handoff sobre o piloto ficam para FAC-012.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Um cliente oficial elegível comprovado em cenário sintético, autenticação por assinatura verificada, extras/API bloqueados e limites desconhecidos explícitos.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## FAC-003 — Controle web e persistência
Sprint: Sprint 1. Dependências: FAC-000. Esforço estimado: 2-3 dias. Status: DONE.
Descrição: Implementar autenticação administrativa mínima, cadastro e consulta de projetos/tickets e transição idempotente para `READY` com outbox transacional, mantendo Redis/PostgreSQL privados. Provisionamento, HTTPS e backup externo permanecem em tarefas operacionais autorizadas separadamente.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Ticket autenticado persiste após reinício e dispatch idempotente; sem segredo em Git.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## FAC-004 — Worker interno e identidade de serviço
Sprint: Sprint 1. Dependências: FAC-003. Esforço estimado: 2-3 dias. Status: DONE.
Descrição: Estabelecer identidade própria do worker Node.js separado, registro restrito, heartbeat persistido, encerramento conservador após perda do controle e rede interna autenticada. Claim, leases, fencing e execução real permanecem nos tickets de orquestração/runtime.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Worker registra identidade/capacidades, heartbeat avança, credencial inválida é rejeitada, perda repetida do controle encerra o processo e worker/banco/fila não possuem portas públicas.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## FAC-005 — Runtime Gateway e primeiro adapter
Sprint: Sprint 1. Dependências: FAC-002, FAC-004. Esforço estimado: 2-3 dias. Status: DONE.
Descrição: Interface, execução segura por argv/stdin, fixtures/eventos e cliente oficial escolhido por preflight.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Execute/cancel/schema comprovados; auth/cota normalizados; modelo/uso desconhecido não inventados.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## FAC-006 — Context Builder e RuntimeGuard
Sprint: Sprint 1. Dependências: FAC-003. Esforço estimado: 1-2 dias. Status: DONE.
Descrição: Fontes determinísticas/hashes; limites de tentativas/tempo/troca; API/extra bloqueados.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Sem segredo/dependência no contexto; falha repetida pausa; keys herdadas não ativam API.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## FAC-007 — Sandbox e snapshots recuperáveis
Sprint: Sprint 2. Dependências: FAC-004, FAC-005. Esforço estimado: 2-3 dias. Status: DONE.
Descrição: Worktree isolado, recursos, serviços sintéticos, patches/untracked e revisão exata.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Sem host socket/home/segredos; timeout mata árvore; snapshot restaura arquivos rastreados e untracked.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## FAC-008 — Orquestrador e checkpoints
Sprint: Sprint 2. Dependências: FAC-006, FAC-007. Esforço estimado: 2-3 dias. Status: DONE.
Descrição: Orquestração NestJS, PostgreSQL/outbox, Redis/BullMQ e leases, writer lock, pausa/cancel/resume e recovery conservador.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Lease expirado sem quiescência bloqueia novo writer; eventos duplicados não repetem efeitos; retorno seguro após crash.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## FAC-009 — Developer, checks e revisão
Sprint: Sprint 2. Dependências: FAC-008. Esforço estimado: 2-3 dias. Status: DONE.
Descrição: Executar ticket, checks do repo, reviewer separado e correções limitadas.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Critérios comprovados na revisão exata; falhas anteriores separadas; até duas correções sem loop.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## FAC-010 — Segundo provider e handoff automático
Sprint: Sprint 2. Dependências: FAC-005, FAC-008, FAC-009, FAC-011. Esforço estimado: 2-3 dias. Status: WAITING_PROVIDER (FAC-010A/B/C DONE; preflight Claude real pendente).
Descrição: Preflight segundo adapter, roteamento por capacidade e troca após término confirmado.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Simular cota, preservar patch/untracked, outro cliente continua sem writer concorrente; sem provider aguarda.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## FAC-011 — Centro de Configuracoes e Provider Manager
Sprint: Sprint 2. Dependências: FAC-003, FAC-009. Esforço estimado: 2-3 dias. Status: DONE.
Descrição: Painel React responsivo, persistencia e contratos para contas/modelos/funcionarios, GitHub e protecoes financeiras. FAC-011A/B/C/D foram aceitos; PR permanece sob gate e sem merge automatico.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Configuracao versionada rejeita segredo e relacoes invalidas; unknown/auth required permanece visivel; API/extras/fallback/merge nao podem ser ativados; docs atuais coerentes com código.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## FAC-012A — Especificacao imutavel de execucao
Sprint: Sprint 3, fundacao interna. Dependências: FAC-001A, FAC-003A, FAC-008. Esforço estimado: 1 dia. Status: DONE.
Descrição: Congelar projeto, definicao e ticket no evento READY para que a execucao futura nao dependa de configuracao mutavel.
Aceite: Snapshot estrito, invariantes cruzadas, montagem transacional e bloqueio de legado incompleto, sem iniciar piloto/provider.
Entregáveis: contratos, controle, dispatcher, probe, testes e documentacao vinculados ao codigo `3b6f3e4624b5935e4fe6ea067406c631a6b0ec32`.

Aceite registrado pelo OPS-003 na revisão documental `3ef3d543b98fb48226714787315f4de947fa6dd9`. FAC-012B também está DONE na revisão `3e28363b0642f8e05840bb649b681e3837e1e015`; C–L aguardam aceite humano. O estado detalhado atual do MVP está em [CONTROLE-MVP.md](06-controle-mvp.md), sem incluir piloto/implantação no escopo automático.

## FAC-012 — Experimento externo com dez tickets
Sprint: Experimento externo opcional, fora do aceite do MVP de software. Dependências: FAC-001, FAC-011. Esforço estimado: 2-3 dias. Status: PLANEJADO.
Descrição: Executar o projeto real selecionado e configurado pelo responsavel no software, incluindo dez tickets, falhas, backup/restauração, métricas de custo/espera/qualidade e retrospectiva.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Dez tickets contabilizados; restauração comprovada; metas avaliadas; zero cobrança extra/API não autorizada.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## FAC-013 — Terceiro adapter e QA UI
Sprint: Evolução. Dependências: FAC-012. Esforço estimado: 1-3 dias. Status: PLANEJADO.
Descrição: Validar provider restante e browser efêmero se piloto precisar.
Subtarefas: registrar contrato e evidência de início; implementar menor incremento; verificar cenários de sucesso/falha descritos; revisar diff/limites; atualizar relatório por domínio, README atual, lessons pertinentes e backlog.
Aceite: Capacidades/termos/login/uso comprovados; critérios UI com evidências; sem sessão pessoal.
Entregáveis: código quando pertinente, checks reais, artefatos por revisão e documentação.

## Definition of Done
Aceite/critério com evidência, checks da revisão final, revisão resolvida, diff recuperável, estado de uso/custo honesto, docs atuais e relato datado, lessons pertinentes, índice/backlog atualizados e aceite humano. Sem merge/deploy automático.
## Experimento
Registrar todos os dez tickets, inclusive fracassos. Meta 8/10 em até duas correções; 100% documentados; zero extras/API não autorizados e zero writers concorrentes. Medir duração ativa/espera, retrabalho, handoffs, regressões, custos atribuídos e recursos. Decidir expandir/corrigir/pausar com dados; não adicionar segundo worker ou Temporal por antecipação.

## Infraestrutura obrigatória do MVP
Toda execução fica na mesma VPS; MacBook não é dependência. Assinaturas autenticadas nos clientes oficiais da VPS. Estimativas: Mínimo 4 vCPU/8 GB/120 GB; Bom recomendado 8 vCPU/16 GB/200 GB; Ideal 8 vCPU/32 GB/300 GB. Sem GPU. Um executor inicial em todos os perfis.
FAC-002 valida compatibilidade, login, extras desativados e baseline sintético na VPS. FAC-003 preserva a separação de redes/volumes na aplicação. O perfil contratado e os limites reais serão registrados antes de qualquer deploy autorizado. FAC-004 testa serviço/credenciais após reboot. FAC-007 prova que código não lê credenciais/controle. A validação operacional mede painel sob build, OOM/I/O/disco e restauração externa com dados sintéticos; FAC-012 mede projeto externo apenas no experimento opcional.
Aceite de infraestrutura: specs reais documentadas; checks sintéticos da fábrica cabem no envelope; sem OOM no ensaio; bancos privados; segredo inacessível ao código; nenhum efeito duplicado após reinício; backup externo restaurável. Na ocorrência de falha de capacidade, otimizar/subir perfil antes de ampliar concorrência.
Perfil Bom é recomendação de engenharia, não contratação autorizada. Preços, fornecedor e acesso ainda precisam ser informados para provisionar. Detalhamento em documentacoes/infraestrutura/DIMENSIONAMENTO-VPS.md.

## Stack obrigatória da fábrica - revisão 2.3
React + TypeScript + Vite no painel; NestJS + TypeScript na API; worker Node.js + TypeScript em processo separado; PostgreSQL e Redis + BullMQ. Monorepo apps/web, apps/api, apps/worker e packages/contracts. Ler documentacoes/arquitetura/ADR-003-stack-typescript.md.
Compartilhar esquemas/DTOs e validar dados em runtime; impedir import de segredos/código servidor no painel. Outbox, idempotência, leases e fencing seguem obrigatórios: lock BullMQ não substitui exclusão do writer. API não executa builds/clientes. Executar typecheck, lint, builds e testes relevantes. Preservar a stack somente de pilotos externos; a própria fábrica segue a stack aprovada.
