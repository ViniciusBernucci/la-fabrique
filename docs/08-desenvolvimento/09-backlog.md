> Leitura: [← Anterior](08-tech-lead-handoff.md) · [Índice didático](../02-INDEX.md) · [Próximo →](10-plano-mvp.md)

> Nota editorial DOC-MV-001, 2026-10-07: backlog local integral preservado. As entradas acumuladas registram estado na revisão do incremento, com atualizações posteriores no mesmo arquivo. FAC-013 atual é modal de contas IMPLEMENTADO/AWAITING_HUMAN (54475ba e relato), enquanto “FAC-013 terceiro adapter PLANEJADO” é intenção histórica com ID reutilizado, não próximo ticket ativo. FAC-012/piloto/dependências antigas não tornam o piloto requisito do MVP: contrato local atual o mantém DEFERRED/opcional. Código de recovery/handoff/consumer integra incrementos posteriores; frases antigas “não integrado/pendente” não negam implementação atual descrita no AS-IS. Aceites por SHA continuam exatamente como registrados, sem promoção por esta migração.

# Backlog v2

- FAC-034: AWAITING_HUMAN — Centro de Comando aplicado a todo apps/web no diff de developer sobre de5dbfa. Tokens, núcleo SVG CSS/reduced-motion, frame e áreas preservam APIs/contratos. Commits impedidos por .git somente leitura; patches separados. Browser/revisão independente pendentes. [Ticket](tickets/controle/FAC-034-centro-comando.md) · [Entrega](../09-entregas/2026/2026-10-07-FAC-034-centro-comando.md).

- FAC-022: AWAITING_HUMAN — template persistente e fontes compactas em `4e042a1`, 43 testes web PASS; navegação/DOM/fontes verificadas em browser na porta 5173. [Relatório](../09-entregas/2026/controle/2026-10-05-FAC-022-template-tipografia.md).


- FAC-021: AWAITING_HUMAN — nome La fabrique e cena central 60% em `cde0cc6`, 322 testes PASS, browser na porta 5173 PASS. [Relatório](../09-entregas/2026/controle/2026-10-05-FAC-021-nome-escala-home.md).
- FAC-020A: AWAITING_HUMAN — home integrada em developer por autorização explícita, funcionamento na porta habitual verificado e worktree FAC-020 removida após preservar commits. [Relatório](../09-entregas/2026/controle/2026-10-05-FAC-020A-home-em-developer.md).


- FAC-020: AWAITING_HUMAN — central de controle estática em `904d0b5`, 42 testes web e browser desktop/mobile passaram. Dados simulados; aceite visual/integração pendentes. [Relatório](../09-entregas/2026/controle/2026-10-05-FAC-020-central-controle-estatica.md).


- FAC-017: AWAITING_HUMAN — desafio ANSI Codex corrigido e aba oficial/fallback implementados; confirmação humana da assinatura e visual pendentes. [Relatório](../09-entregas/2026/configuracao/2026-10-05-FAC-017-login-device-codex.md).

- FAC-016: AWAITING_HUMAN — feedback de verificação no modal; 35 testes web/209 worker passaram, pedido Codex real COMPLETED/AUTH_REQUIRED. [Relatório](../09-entregas/2026/configuracao/2026-10-05-FAC-016-feedback-verificacao-codex.md).

- FAC-015: AWAITING_HUMAN — login oficial por conta, catálogo Codex integrado e modelos compartilhados; 537 testes passaram. Login real e verificação visual pendentes. [Relatório](../09-entregas/2026/configuracao/2026-10-05-FAC-015-contas-autenticadas-modelos.md).

- FAC-014: AWAITING_HUMAN — listas e modais de funcionários digitais/GitHub implementados, validação visual humana pendente. [Relatório](../09-entregas/2026/configuracao/2026-10-05-FAC-014-modais-funcionarios-github.md).
- FAC-013: AWAITING_HUMAN — lista compacta de contas IA/configuração em modal implementada em `54475ba`; 33 testes web, typecheck/lint/build passaram. Validação manual do visual pendente. [Relatório](../09-entregas/2026/configuracao/2026-10-05-FAC-013-modal-contas-ia.md).
- OPS-009: AWAITING_HUMAN — P3018 reconciliado após comprovar schema, zero migrations em falha ativa. Nove migrations pendentes; duas attempts sem stop preservadas. [Evidências](../09-entregas/2026/infraestrutura/2026-10-05-OPS-009-reconciliar-migration-historica.md).
- OPS-008: AWAITING_HUMAN — comandos Prisma com `.env` raiz e favicon implementados. Falhas de telas dependem de atualizar banco dev: histórico divergente e duas tentativas sem parada confirmada; nenhuma liberação presumida. [Relatório](../09-entregas/2026/infraestrutura/2026-10-05-OPS-008-bootstrap-migrations-telas.md).
- OPS-007: AWAITING_HUMAN — FAC-012Z–AH integrados localmente em developer, nove worktrees funcionais removidas, branches preservadas. [Relatório](../09-entregas/2026/operacao/2026-10-05-OPS-007-consolidacao-worktrees.md).
- FAC-012AH: AWAITING_HUMAN — código `f2d80f1`, 528 testes + 10 PostgreSQL + 3 Redis. Lacunas internas auditadas implementadas; revisão humana/ativação operacionais separadas do software e do piloto. [Relatório](../09-entregas/2026/controle/2026-10-03-FAC-012AH-admissao-capacidade.md).

- FAC-012AG: AWAITING_HUMAN — pausa global/claim/renew/queue/UI em `881ce3d`; 519 testes + 10 PostgreSQL + 2 Redis. Followup de capacidade fechado pelo FAC-012AH. [Relatório](../09-entregas/2026/controle/2026-10-03-FAC-012AG-pausa-global.md).

- FAC-012AF: AWAITING_HUMAN — backup/restauração em `7de9041`, 501 testes internos + 8 PostgreSQL; sem storage externo/serviço real. Pausa global implementada no FAC-012AG. [Evidências](../09-entregas/2026/operacao/2026-10-03-FAC-012AF-backup-restauracao.md).

- FAC-012AE: AWAITING_HUMAN — SSE/cursor/triggers/reconnect em `9173532`, 485 testes + 7 PostgreSQL; migration apenas efêmera. Backup/restauração implementados no FAC-012AF. [Evidências](../09-entregas/2026/controle/2026-10-03-FAC-012AE-eventos-projeto.md).

- FAC-012AD: AWAITING_HUMAN — painel/endpoint operacional e guard fail-closed em `6277a81`; 461 testes + 4 PostgreSQL. SSE implementado no FAC-012AE; piloto independente. [Relatório](../09-entregas/2026/controle/2026-10-03-FAC-012AD-painel-operacao.md).

- FAC-012AC: AWAITING_HUMAN — exclusão global em `143f3d6`, 445 testes + 3 PostgreSQL; migration somente efêmera. Piloto não bloqueia MVP. [Relatório](../09-entregas/2026/controle/2026-10-03-FAC-012AC-writer-global.md).

- FAC-012AB: AWAITING_HUMAN — ensaio integrado interno em `1714347`, 438 testes; providers/controle externos sintéticos, sem operação real. [Evidências](../09-entregas/2026/operacao/2026-10-03-FAC-012AB-ensaio-integrado-mvp.md).
- FAC-012AA: AWAITING_HUMAN — perfil Claude granular/preflight/gate de prova privada em `56f2e38`; 433 testes/checks. Validação de serviço pendente, sem conta real desbloqueada; piloto opcional separado. [Evidências](../09-entregas/2026/runtime/2026-10-03-FAC-012AA-confinamento-claude.md).

- FAC-012Z: AWAITING_HUMAN — gate documental técnico implementado em `9c48dd9`, configuração UI/READY/compilador, revisão semântica e hashes exatos; 422 testes/checks. Sem provider/piloto/deploy. [Evidências](../09-entregas/2026/controle/2026-10-03-FAC-012Z-gate-documentacao-tecnica.md).

- FAC-012Y: AWAITING_HUMAN — alternativas UI/handoff no workflow real, código `f66a923`, 411 testes/checks; limites/stop/snapshot/restore/lease preservados, sem operação real. [Relatório](../09-entregas/2026/handoff/2026-10-03-FAC-012Y-handoff-configurado.md).

- FAC-012X: AWAITING_HUMAN — espera provider com stop/snapshot e retomada explícita, código `345e8ef`, 399 testes/checks; migration não aplicada. [Relatório](../09-entregas/2026/operacao/2026-10-03-FAC-012X-espera-provider.md).

- FAC-012W: AWAITING_HUMAN — entrega íntegra ampliada 8 MiB JSON/6 MiB raw, código `2a1ae96`, 387 testes/checks. [Relatório](../09-entregas/2026/controle/2026-10-03-FAC-012W-artefatos-ampliados.md).

- FAC-012V: AWAITING_HUMAN — identidade privada por instalação UI no login/status/runtime, código `22f6375`, 380 testes/checks. Claude escrita bloqueada; [relatório](../09-entregas/2026/runtime/2026-10-03-FAC-012V-identidades-instalacoes.md).

- FAC-012U: AWAITING_HUMAN — recuperação de journal/checkpoint sem IA/novo writer, código `9d4a4f1`, 370 testes/checks. [Relatório](../09-entregas/2026/controle/2026-10-03-FAC-012U-recuperacao-finalizacao.md).

- FAC-012T: AWAITING_HUMAN — retomada explícita parada/íntegra, objetivo congelado, novo fence e baseline pré-restore; código `42e7037`, 355 testes/checks. [Relatório](../09-entregas/2026/controle/2026-10-03-FAC-012T-retomada-snapshot.md).

- FAC-012S: AWAITING_HUMAN — pausa/cancelamento no painel com stop/snapshot/journal, código `be4e2ff`, 336 testes/checks. Migration não aplicada; [relatório](../09-entregas/2026/controle/2026-10-03-FAC-012S-comandos-run.md).

- FAC-012R: AWAITING_HUMAN — snapshot/observações de interrupção conhecida e término unknown sticky, código `abfb033`, 315 testes/checks; [relatório](../09-entregas/2026/operacao/2026-10-03-FAC-012R-snapshot-interrupcao.md).

- FAC-012Q: AWAITING_HUMAN — relatório determinístico de entrega/aceite exato no painel, código `bfd8532`, 306 testes/checks. Migration não aplicada; [relatório](../09-entregas/2026/controle/2026-10-03-FAC-012Q-documentacao-aceite-entrega.md).

- FAC-012P: AWAITING_HUMAN — diff e bundle do último snapshot no painel, até 64 KiB, código `739cff0`, 289 testes/checks. Migration não aplicada; [relatório](../09-entregas/2026/controle/2026-10-03-FAC-012P-artefatos-painel.md).

- FAC-012O: AWAITING_HUMAN — journal privado de resultado normal antes da API, replay sem IA; código `016a5ef`, 266 testes/checks. [Relatório](../09-entregas/2026/operacao/2026-10-03-FAC-012O-journal-resultados.md).

- FAC-012N: AWAITING_HUMAN — reconciliação de checkpoint parado no replay sem checkout/IA; código `94b5d64`, 257 testes/checks locais. [Relatório](../09-entregas/2026/operacao/2026-10-03-FAC-012N-reconciliacao-checkpoint.md).

- FAC-012M: AWAITING_HUMAN — resultados persistidos/fenced e histórico no painel, código `14ae5ba`, 240 testes/checks locais. Migration não aplicada; diff completo/recuperação pendentes. [Relatório](../09-entregas/2026/controle/2026-10-03-FAC-012M-resultados-execucao-painel.md).

Estado atual consolidado em [CONTROLE-MVP.md](06-controle-mvp.md). OPS-006 integrou OPS-005/FAC-012K/L localmente e retirou worktrees comprovadamente integrados; pendências de software permanecem explícitas.

## Decisão obrigatória da stack - revisão 2.3
A stack da própria La fabrique está APROVADA: React + TypeScript + Vite no painel; NestJS + TypeScript na API; worker Node.js + TypeScript em processo separado; PostgreSQL; Redis + BullMQ; Docker Compose na mesma VPS. Não solicitar nova escolha ou confirmação da stack. Não iniciar a fábrica em PHP/Laravel, Angular ou .NET. Esta decisão substitui propostas anteriores.
Monorepo: `apps/web`, `apps/api`, `apps/worker`, `packages/contracts`. Contratos compartilhados precisam de validação em runtime. API não executa clientes, builds ou testes; o worker executa esses trabalhos com isolamento, limites e um writer inicial.
Ao trabalhar na própria fábrica, aplicar esta stack. A regra de preservar a stack existente aplica-se somente a projetos EXTERNOS cadastrados para desenvolvimento pela fábrica; ela não altera a stack da La fabrique. Se o repositório da fábrica contiver implementação anterior incompatível, registrar a divergência e planejar a adaptação por etapas; não apagar código existente nem reabrir a escolha tecnológica.
Versões exatas e comandos devem ser fixados conforme compatibilidade no bootstrap; isso não é uma nova decisão de stack. Repositório e funcionalidade do piloto externo permanecem pendentes quando não fornecidos.


FAC-000, FAC-001A e FAC-002 a FAC-009 estão DONE. FAC-011A/B/C/D foram aceitos. FAC-012A/B foram aceitos em 2026-10-02; OPS-003 registra os aceites e consolida deltas autorizados. FAC-012C/D/E/F/G/H/I/J estão implementados e aguardam aceite humano. OPS-005 removeu o consumidor fixture do código e preparou um unit host, ainda não instalado; a execução real continua desligada. FAC-012D restringe o Codex, mas não cobre Claude nem comprova isolamento sob identidade real. FAC-010 está WAITING_PROVIDER. O projeto externo permanece indefinido.

- FAC-000: DONE — Bootstrap TypeScript e Docker aceito no SHA `15c2198076a6a6afcbf2f15212ae1eaf53822b9f`; dependências: ADR-002 e ADR-003.
- FAC-001: DEFERRED — Definir contrato do piloto externo após o núcleo da plataforma; dependências: FAC-011.
- FAC-001A: DONE — Definicao generica e versionada de projeto no painel e gate de READY aceitos na revisao `094190af26c1175bad08eb46293230b2938843d4`; sem hardcode do piloto ou de contas de IA; dependências: FAC-003, FAC-003A e FAC-011.
- FAC-002: DONE — Codex validado e aceito na revisão `1666ee108343563e35edb6971bea11234a62d47e`; dependências: FAC-004.
- FAC-003: DONE — Controle web e persistência aceitos no SHA `1807330bcf7b1374fa626d9fcbfc47dd8002f433`; dependências: FAC-000.
- FAC-003A: DONE — Gate de revisao-base exata antes de READY, barreira no dispatcher e atualizacao configuravel no painel aceitos na revisao `4630ce0b9a613b937d40d550e4551fa1402d6c6a`; dependências: FAC-003 e FAC-008.
- FAC-004: DONE — Worker interno e identidade de serviço aceitos no SHA `6d73041bdcbd50215b6a018c9937475c29f542b1`; dependências: FAC-003.
- FAC-005: DONE — Runtime Gateway e adapter Codex aceitos na revisão `9102fcc614739fd7bd7ca4992b16617db92ca66f`; dependências: FAC-002, FAC-004.
- FAC-006: DONE — Context Builder e RuntimeGuard aceitos na revisão `ea8cf7afb0e55840722f306262b5334bb408b51a`; dependências: FAC-003.
- FAC-007: DONE — Sandbox e snapshots aceitos na revisão `2bf14f35de64f118ec8224fee6151e60027dedb0`; dependências: FAC-004, FAC-005.
- FAC-008: DONE — Orquestrador e checkpoints aceitos na revisao `29061a911e0f6bc5122e9e53511f2f475caf8dce`; dependências: FAC-006, FAC-007.
- FAC-009: DONE — Coordenador de Developer, checks e Reviewer aceito na revisao `02819fb847e303f1a823cc2784a7326ec5e696f9`; dependências: FAC-008.
- FAC-010: WAITING_PROVIDER — FAC-010A DONE na revisao `7c8d95eecc33488c43d7bf6d6166bedd6c143341`, FAC-010B DONE na revisao `d39412282d9401b2a22c70eb4b353883e559be4c` e FAC-010C DONE na revisao `0748f029a8a62ce891486dc5751c207bdd6e56db`; codigo do segundo adapter/handoff aceito, validacao operacional aguarda login/preflight Claude posterior.
- FAC-011: DONE — Centro de Configuracoes aceito na revisao `e41ec1e45273aba3205f266e8753dfca305c5952`; dependências: FAC-003 e FAC-009.
- FAC-011A: DONE — Verificacao GitHub CLI aceita na revisao `dce2e676a91b5ffaeb246afeea2cc699e60aff9d`; dependência: FAC-011.
- FAC-011B: DONE — Login GitHub web/device aceito na revisao `3ac8b39de9024576df5c5709022bb4a01e5ea6a5`; dependência: FAC-011A.
- FAC-011C: DONE — Verificacao somente-leitura de repositorio/branch aceita na revisao `8fd38781652fac187d0f5a419eb336c15e94b816`; dependência: FAC-011B. `gh` e integracao real permanecem nao verificados.
- FAC-011D: DONE — Gate em duas etapas, prova de push, reconciliacao e criacao de PR aceitos na revisao `76df60bbad4eac78b5d87fad8c2e79282355bf8c`; dependência: FAC-011C. Prova real e merge permanecem pendentes.
- FAC-012: PLANEJADO — Piloto, dez tickets e operação; dependências: FAC-001, FAC-011.
- FAC-012K: AWAITING_HUMAN — `LeaseGuard` no worker renova com o fencing token, aborta na perda/expiração da lease e só relata quiescência quando o callback confirma. Implementação `8f254e8`; não integrado ao consumer.
- FAC-012L: AWAITING_HUMAN — consumer real implementado em `54bf483`, com checkout/workflow configurado, lease/fencing, cancelamento e checkpoint; default desabilitado. Recuperação, artefatos no painel e handoff pendentes. Relatório `documentacoes/operacao/2026-10-03-FAC-012L-consumer-execucao-real.md`.
- FAC-012A: DONE — Snapshot imutavel de projeto/definicao/ticket no evento READY; codigo `3b6f3e4624b5935e4fe6ea067406c631a6b0ec32`, revisao documental aceita `3ef3d543b98fb48226714787315f4de947fa6dd9`; sem piloto.
- FAC-012B: DONE — Compilador de snapshot para DeveloperWorkflowRequest com perfil confiável injetado; código `e7b9bf60e03089c502255d19edbb5d1e31725d2c`, revisao documental aceita `3e28363b0642f8e05840bb649b681e3837e1e015`; consumer permanece desligado.
- FAC-012C: AWAITING_HUMAN — SandboxRunner read-only por padrão, caminhos de escrita explícitos e prova de isolamento; código `a8f7a66dfc1b5c58a0596611dc300559504709ce`, relatório `documentacoes/operacao/2026-10-02-FAC-012C-politica-escrita-sandbox.md`; sem ligar o consumer ou iniciar provider. CodexAdapter já aplica sandbox nativo; o escopo de escrita do CLI ainda é amplo e está em FAC-012D.
- FAC-012D: AWAITING_HUMAN — `allowedPaths` validado em runtime e encaminhado ao perfil Codex; leitura geral, escrita apenas nos caminhos existentes autorizados e `.git`/`.codex` negados. Código `04e5f30`; relatório `documentacoes/operacao/2026-10-02-FAC-012D-caminhos-escrita-cli.md`. Testes sintéticos, sem consumer/provider; Claude e identidade real do worker não verificados.
- FAC-012E: AWAITING_HUMAN — endpoint `GET /api/internal/worker-settings` fornece ao worker snapshot de configuração versionado/validado sem credenciais; defaults não persistidos têm versão 0. Código `4aca1e10085f047712766f51ca7e4e749508a7a9`, relatório `documentacoes/operacao/2026-10-02-FAC-012E-configuracao-runtime-worker.md`. Não altera consumer, DB ou autenticação.
- FAC-012F: AWAITING_HUMAN — router sem cache usa cada snapshot da interface para selecionar papel, instalação/modelo e adapter; sem fallback e Reviewer obrigatoriamente `READ_ONLY`. Código `950f3b62651b1e918bae09d89996af0d95e9fbee`, relatório `documentacoes/operacao/2026-10-02-FAC-012F-rota-agente-configurada.md`. Biblioteca não conectada ao consumer nem executada.
- FAC-012G: AWAITING_HUMAN — `DeveloperWorkflow` resolve adapter/modelo e permissão por função em cada chamada, inclusive correções; rota ausente e Reviewer com escrita falham sem executar cliente. Implementado em `e1e9828` e `4bed978`; relatório `documentacoes/operacao/2026-10-02-FAC-012G-workflow-rotas-por-funcao.md`. Consumer segue no probe.
- FAC-012H: AWAITING_HUMAN — `SandboxRunner` agora pivota para rootfs mínimo e remove capabilities antes de checks; teste antes/depois prova que host `/etc`, home, `/var`, cgroup e mount escape não ficam acessíveis, sem perder workspace escopado, temporários, runtime `/usr`, rede bloqueada e quiescência. Nenhum provider ou consumer foi usado; relatório operacional FAC-012H.
- FAC-012I: AWAITING_HUMAN — perfil versionado por projeto configura fontes de contexto e checks aprovados; READY exige ambos e o compilador cruza o snapshot com perfil confiável. Evidências e limites em `documentacoes/operacao/2026-10-02-FAC-012I-perfil-execucao-projeto.md`.
- FAC-012J: AWAITING_HUMAN — preparador de checkout isolado usa root/hosts locais, credencial oficial efêmera com origem `keyring` e SHA detached verificado; não está conectado ao consumer. Relatório: `documentacoes/infraestrutura/2026-10-02-FAC-012J-checkout-confiavel-worker.md`.
- FAC-013: PLANEJADO — Terceiro adapter e QA UI; dependências: FAC-012.
- OPS-001: DONE — `.env` carregado no comando raiz, registro inicial com retry transitorio limitado e bootstrap usando migrations versionadas; aceito na revisao `7c1d1dcb44ab3464085aad729ae2b5dbdc473057`, dependências: FAC-000 e FAC-004.
- OPS-003: AWAITING_HUMAN — Aceites FAC-012A/B registrados; bind loopback e migration corretiva integrados em `e09204c5205af047d79bcd7d530a1acd1db563ca`, migration não aplicada.
- OPS-004: AWAITING_HUMAN — merges locais D e E–J e limpeza pós-merge documentados em `documentacoes/operacao/2026-10-03-OPS-004-consolidacao-worktrees.md`; sem push/deploy.
- OPS-005: AWAITING_HUMAN — worker host systemd preparado, APIs/Redis somente em loopback e consumidor fixture retirado do `main.ts`. Implementação `8f3dbda`; relatório `documentacoes/infraestrutura/2026-10-03-OPS-005-worker-host-sandbox.md`. Não instalado; worker container antigo continua ativo até janela manual.

Infraestrutura VPS única incluída em FAC-002/003/004/007/012; perfil Bom recomendado.

- OPS-006: AWAITING_HUMAN — integração local OPS-005/K/L, correção do env/user manager e controle atual do MVP; 227 testes/checks combinados passaram. Código `26e412d`; sem push/deploy. Relatório `documentacoes/operacao/2026-10-03-OPS-006-integracao-local-controle-mvp.md`.

## Stack obrigatória da fábrica - revisão 2.3
React + TypeScript + Vite no painel; NestJS + TypeScript na API; worker Node.js + TypeScript em processo separado; PostgreSQL e Redis + BullMQ. Monorepo apps/web, apps/api, apps/worker e packages/contracts. Ler documentacoes/arquitetura/ADR-003-stack-typescript.md.
Compartilhar esquemas/DTOs e validar dados em runtime; impedir import de segredos/código servidor no painel. Outbox, idempotência, leases e fencing seguem obrigatórios: lock BullMQ não substitui exclusão do writer. API não executa builds/clientes. Executar typecheck, lint, builds e testes relevantes. Preservar a stack somente de pilotos externos; a própria fábrica segue a stack aprovada.

FAC-018 AWAITING_HUMAN: login oficial Claude e Google/Antigravity no painel; keyring VPS e consentimento real pendentes. [Evidências](../09-entregas/2026/configuracao/2026-10-05-FAC-018-login-claude-antigravity.md).

FAC-019 AWAITING_HUMAN: abas Contas, Integrações IA, Equipes. [Evidências](../09-entregas/2026/configuracao/2026-10-05-FAC-019-abas-configuracoes.md).

- FAC-023: AWAITING_HUMAN — agentes e skills por projeto; 562 testes, typecheck/lint/build e browser fixture PASS. Código a25638acde42ba50576908f1bf6725570a360a99; registro não executa novos perfis automaticamente. [Relatório](../09-entregas/2026/configuracao/2026-10-05-FAC-023-agentes-skills-projeto.md).

- FAC-024: AWAITING_HUMAN — vídeo decorativo em loop e área compacta na home. Código eaf90013e8978a6562f930b98754f28d9a68c5d7; 46 testes web/typecheck/lint/build e browser com reprodução real PASS. [Relatório](../09-entregas/2026/controle/2026-10-05-FAC-024-video-dashboard.md).

- FAC-025: AWAITING_HUMAN — vídeo retirado/imagem original restaurada, área compacta mantida. Código f264747e10a489c84338ae85f3d6c92e914524b2; 46 testes web e browser/typecheck/lint/build PASS. [Relatório](../09-entregas/2026/controle/2026-10-06-FAC-025-retirar-video-dashboard.md).

- FAC-026 / FAC-027: AWAITING_HUMAN — agentes pré-configurados com switch/modal e menu recolhível. Código bb0ad15 / b1d794035342128b9ed832d5fe9b74c887eb7075; 46 testes web/typecheck/lint/build e dois fluxos browser PASS. Aceite exato pendente.

- FAC-028: AWAITING_HUMAN — seta minimalista no toggle de menu; código efbcb86631da70e6afe40ce33265a97a69f62847, typecheck/build/lint/browser PASS. [Relatório](../09-entregas/2026/controle/2026-10-06-FAC-028-seta-menu-minimalista.md).

- FAC-029: AWAITING_HUMAN — seta à direita; código ec98b7dc2f2ea700857810fe5df708bae91a720a, build/lint/browser PASS. [Relatório](../09-entregas/2026/controle/2026-10-06-FAC-029-seta-direita-menu.md).

- FAC-030: AWAITING_HUMAN — menu inicia recolhido; código 3608f1766c50ee49071a6430d1087fa2fd538749, 46 testes web/build/lint/browser PASS. [Relatório](../09-entregas/2026/controle/2026-10-06-FAC-030-menu-inicial-recolhido.md).

- FAC-031: AWAITING_HUMAN — seta por estado do menu; código 3b7e02067884378b7775ebf48af81d61c82187a8, build/lint/browser PASS. [Relatório](../09-entregas/2026/controle/2026-10-06-FAC-031-posicao-seta-menu.md).

## Migração documental — DOC-MV-001

READY autorizado pelo pedido de 2026-10-07; implementação documental executada em branch isolada docs/manual-vivo-inicial. Estado AWAITING_HUMAN após checks registrados na entrega; não altera status/aceites FAC/OPS ou implementa LF-MT. [Entrega](../09-entregas/2026/2026-10-07-DOC-MV-001-migracao-manual-vivo.md), [plano LF-MT posterior PROPOSTO](11-plano-lf-mt-proposto.md). Colisão histórica FAC-013 (plano terceiro adapter versus modal contas entregue) é explicitada na auditoria, sem renumerar passado.

### DOC-MV-001 — Integração autorizada (2026-10-07)

Revisão documental 607d6ed integrada em developer por fast-forward autorizado pelo usuário. Docs atuais em docs/; worktree de migração removida sem alterações pendentes. Permanecem os limites de revisão independente/render/autoload registrados na entrega; nenhum ticket de software promovido a DONE.

### DOC-MV-002 — Organização da raiz

IMPLEMENTADO / AWAITING_HUMAN. Dez Markdown de apoio retirados da raiz; backlog/changelog/guia realocados, atalhos consolidados, documentacoes removida após conferir 11 bridges e 138 evidências. Guias/políticas/links/validador atualizados. [Entrega](../09-entregas/2026/2026-10-07-DOC-MV-002-organizacao-raiz.md). Não promove software a DONE.

### DOC-MV-003 — Ordem didática

IMPLEMENTADO / AWAITING_HUMAN: manual numerado e navegável, regra/check para novos capítulos; históricos/IDs preservados. [Entrega](../09-entregas/2026/2026-10-07-DOC-MV-003-ordem-didatica.md). Nenhum DONE de software.

### DOC-MV-004 — Skills do projeto

Fontes correntes em skills/ e guias adaptados, AWAITING_HUMAN. Instalação em .agents impedida por read-only; descoberta nativa não verificada. [Entrega](../09-entregas/2026/2026-10-07-DOC-MV-004-skills-la-fabrique.md). Nenhum DONE de software.

### DOC-MV-005 — Commit e limpeza autorizados

BLOCKED_READ_ONLY: git add falhou criando index.lock; nenhum commit/merge/limpeza Git realizado. Alterações DOC-MV-003/004 preservadas em developer. Somente a worktree principal; candidates de branches integradas inventariadas. Retomar quando Git puder ser escrito, sem exigir autorização novamente. [Checkpoint](../09-entregas/2026/2026-10-07-DOC-MV-005-checkpoint-integracao.md).

## FAC-032 — Menu inicial reduzido

READY autorizado pelo pedido de 2026-10-07 e confirmação “pode prosseguir”. Escopo: somente a navegação principal de `apps/web/src/DashboardLayout.tsx`, mantendo seis itens nesta ordem: Painel, Projetos, Agentes de IA, Escritório, Usuários e Configurações. Preservar destinos existentes, prévias das áreas futuras, acessibilidade e recolhimento. Risco baixo; sem dependências novas, API, persistência, deploy ou alteração dos atalhos da home. Checks: testes web, typecheck, build e lint do arquivo afetado. Trabalho documental prévio preservado; continuação no workspace autorizada pelo responsável.

FAC-032: AWAITING_HUMAN — implementação local concluída, 46 testes web/typecheck/build/lint PASS; revisão independente e aceite exato pendentes. [Entrega](../09-entregas/2026/2026-10-07-FAC-032-menu-inicial-reduzido.md).

## FAC-033 — Painel de tarefas

IMPLEMENTADO como demonstração local / AWAITING_HUMAN. Menu Tarefas, quadro/lista, edição, filtros, dependências e atribuição por agentes do projeto. Onboarding simulado cria equipe antes de tickets; integração real com onboarding/PostgreSQL/worker PLANEJADA. [Ticket](tickets/FAC-033.md), [entrega](../09-entregas/2026/2026-10-07-FAC-033-painel-tarefas.md). Revisão independente e aceite exato pendentes.

## FAC-035 — Nome e cargo dos agentes

IMPLEMENTADO / AWAITING_HUMAN em worktree isolada sobre 9786910: nomes opcionais para pré-configurados/personalizados, cargos preservados, persistência versionada existente. Código bc38715; commit e integração local à developer autorizados em seguida pelo responsável. Revisão independente, browser e aceite exato pendentes. [Ticket](tickets/configuracao/FAC-035-nomes-agentes.md) · [Entrega](../09-entregas/2026/2026-10-08-FAC-035-nomes-agentes.md).

Adendo FAC-035 (2026-10-09): corrigida preparação de contratos compilados no desenvolvimento para evitar rejeição de nickname; checks locais PASS, ambiente específico do usuário não validado. AWAITING_HUMAN. [Correção](../09-entregas/2026/2026-10-09-FAC-035-correcao-salvamento-nomes.md).

## FAC-037 — Cadastro API/assinatura CLI e escolha de IA/modelo

IMPLEMENTADO cadastro/atribuição / AWAITING_HUMAN em worktree isolada feat/fac-037-api-cli-agents sobre 36e48f9. Seletor inicial, chave API criptografada separada da configuração, IA/modelo separados para ambos os tipos de agente. Execução agentiva por API permanece PLANEJADA; cadastro não ativa gastos. Migration preparada, não aplicada; implantação, revisão independente/browser e aceite exato pendentes. [Ticket](tickets/configuracao/FAC-037-api-cli-agentes.md) · [Entrega](../09-entregas/2026/2026-10-09-FAC-037-api-cli-agentes.md).

FAC-037, adendo 2026-10-09: commit `67ce0a09272966b7b1234663d087e79d12b4381f` integrado localmente à developer por fast-forward autorizado. Worktrees limpas FAC-035/FAC-037 e 72 branches locais integradas removidas; FAC-036 com trabalho não commitado preservada. AWAITING_HUMAN e limites de execução/implantação mantidos.


## PLAN-ADM-001 — Arquitetura do painel administrativo da conta

PROPOSTA DOCUMENTADA / implementada em FAC-038–041, aceite pendente. Pedido de 2026-10-09 para analisar perfil pessoal, organização, usuários/permissões, dados e planos com arquiteto-de-software. [Proposta e handoff](08-tech-lead-handoff.md#proposta-focal--painel-administrativo-da-conta) · [Registro](../09-entregas/2026/2026-10-09-PLAN-ADM-001-painel-administrativo-conta.md). Escopo confirmado: camada da conta do usuário; administração global posterior. Implementação autorizada posteriormente pelo pedido explícito do responsável; tickets FAC-038–041 abaixo. Dados comerciais/ativação externa aguardam configuração e não constituem aceite estrutural.


## FAC-038 — Painel da conta do usuário

IMPLEMENTADO / AWAITING_HUMAN no diff feat/fac-038-painel-conta sobre ee4485b. [Ticket](tickets/FAC-038.md), [entrega](../09-entregas/2026/2026-10-09-FAC-038-painel-conta-usuario.md), [ADR-007](../06-decisoes/ADR-007-conta-usuario-e-sessoes.md). Identidade/profile/organização/equipe/convites/sessões/auditoria, complementados por FAC-039–041 abaixo. Migration apenas em DB efêmero; browser verificado no complemento FAC-041; revisão independente/aceite/implantação pendentes.

## FAC-039–041 — Complemento da camada de usuário

IMPLEMENTADO em `1e6274aa6f4d5b8d9fbf6e28590c37fdafb8244e` / checks automatizados e browser fixture PASS / revisão independente e aceite pendentes. READY autorizado pelo pedido explícito para concluir todas as partes. [FAC-039](tickets/FAC-039.md), [FAC-040](tickets/FAC-040.md), [FAC-041](tickets/FAC-041.md) e [entrega](../09-entregas/2026/2026-10-09-FAC-041-complemento-painel-conta.md). Projetos/grants/execução/integrações privados, uso/quota, notificações/exportação/dados/transferência/encerramento, MFA/email/recuperação e planos/Stripe configuráveis. Catálogo comercial/SMTP/Stripe reais não definidos nem ativados; migrations somente em DB efêmero, nenhuma propriedade legada inferida. Não promover a DONE ou release comercial por esses checks.

## FAC-036 — Instruções por arquivo Markdown

READY autorizado pelo pedido de 2026-10-09; IMPLEMENTADO / AWAITING_HUMAN em worktree isolada feat/fac-036-instrucoes-md. Upload/substituição/download de instruções .md para agentes e skills; descrição apenas resumo, persistência JSON versionada compatível com registros anteriores. Revisão independente, browser, integração e aceite exato pendentes. [Ticket](tickets/configuracao/FAC-036-instrucoes-markdown.md) · [Entrega](../09-entregas/2026/2026-10-09-FAC-036-instrucoes-markdown.md).

FAC-036, adendo 2026-10-24: integrado localmente à developer no merge `82bcfc7` (2026-10-24), com conflitos resolvidos sobre FAC-037: descrição e arquivo .md junto ao seletor IA/modelo. Browser em fixture descartável PASS (7 verificações, zero exceções) e checks pós-integração PASS. Sessão distinta da implementação revisou sem achados bloqueantes, mas também resolveu o merge: revisão independente formal continua NOT_RUN. AWAITING_HUMAN para aceite da revisão exata. [Revisão e verificação](../09-entregas/2026/evidencias/FAC-036/revisao-integracao.md).

## Planejamento arquitetural — memória e decisões (2026-10-09)

Pedido do responsável: oito planos e documentação didática, sem backlog detalhado nesta etapa. [Planos](planos-memoria-e-decisoes/00-README.md), [requisitos e pendências](planos-memoria-e-decisoes/01-decisoes-e-dependencias.md), [matriz](planos-memoria-e-decisoes/10-matriz-fluxo.md) e [entrega documental](../09-entregas/2026/2026-10-09-ARQ-MEM-001-planos-memoria-decisoes.md). Estado: DOCUMENTADO / AWAITING_HUMAN para revisão exata; nenhum ticket de software promovido ou novo ID FAC criado. ARQ-MEM-001 é identificador documental desta análise, não ticket de implementação. Tech Lead define tickets posteriores após avaliação conjunta.

ARQ-MEM-001, adendo 2026-10-24: documentação integrada localmente à developer no merge `05a9877`. Colisão de número com ADR-007 da conta (FAC-038) mantida por nome de arquivo, sem renumerar; decisão de numeração pendente do responsável. Estado DOCUMENTADO / AWAITING_HUMAN inalterado.

## FAC-042 — Escritório virtual isométrico

READY por pedido do responsável em 2026-10-24; IMPLEMENTADO / AWAITING_HUMAN no diff local sobre `d923802`. Demonstração visual com dados simulados e contrato `OfficeFeed` para eventos reais; sem API, dependências ou ações. Browser no app completo e revisão independente NOT_RUN; aceite exato pendente. [Ticket](tickets/controle/FAC-042-escritorio-virtual.md) · [Entrega](../09-entregas/2026/2026-10-24-FAC-042-escritorio-virtual.md).

## FAC-043 — Visual sóbrio com seletor Novo/Antigo

READY por pedido do responsável em 2026-10-25; IMPLEMENTADO / AWAITING_HUMAN no diff local sobre `d923802`, junto com FAC-042. Tema por tokens + home nova + renomeação para Núcleos; HUD mantido como opção. Pendentes: decisão final de visual, página de lista de projetos, verificação no app completo, revisão independente e aceite. [Ticket](tickets/controle/FAC-043-visual-sobrio.md) · [Entrega](../09-entregas/2026/2026-10-25-FAC-043-visual-sobrio.md).
