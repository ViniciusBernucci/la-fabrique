> Leitura: [← Anterior](00-README.md) · [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../configuracao/00-README.md)

> Instruções de criação em documentacoes/ ou lessons/ abaixo são histórico da fonte; novas entregas/lessons seguem exclusivamente docs/00-governanca/POLITICA-DOCUMENTACAO.md.

> Nota editorial DOC-MV-001, 2026-10-07: conteúdo integral local preservado como referência da evolução v2.3. Afirmações por incremento têm a revisão/data original; trechos sobre consumer ausente, recuperação pendente e caminhos antigos não descrevem necessariamente HEAD a2cc5e0. O [AS-IS conferido](../../02-arquitetura/01-as-is.md) e a [auditoria](../../00-governanca/08-AUDITORIA.md) delimitam o estado atual; números de testes abaixo são evidência histórica, não reexecução desta migração.

# Controle administrativo

## Página inicial — FAC-020/FAC-025

Estado IMPLEMENTADO/AWAITING_HUMAN. DashboardLayout abre a home antes do login com menus persistentes e dados demonstrativos. FAC-025 retira o vídeo por pedido do responsável e restaura a imagem cartoon original, estática. Largura desktop 42% da coluna, mobile em 70% da largura anterior, proporção/enquadramento do PNG. Nenhum player ou request MP4; vídeo/poster retirados do build. Menus/atalhos preservados. [Estado atual e checks](../../09-entregas/2026/controle/2026-10-06-FAC-025-retirar-video-dashboard.md).

Indicadores, notificações, mensagens, agentes e projetos continuam simulados; home não lê API na entrada. Acessos às áreas existentes preservam autenticação administrativa; áreas futuras usam prévias. Ícones/avatares SVG locais. Histórico da ilustração está no [FAC-020](../../09-entregas/2026/controle/2026-10-05-FAC-020-central-controle-estatica.md). Nenhum contrato/backend/migration alterado nesta troca visual.

FAC-012AH: claim responde 429/WRITER_BUSY por capacidade global/rollback conhecido antes de autoridade. Worker adia job sem FAILED/attemptsMade; 409/lease/execução/ambiguidade propagam. 528 testes + 10 PostgreSQL + 3 Redis. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AH-admissao-capacidade.md).

FAC-012AG adiciona PUT /api/operation/scheduling (paused/expectedVersion) e estado no painel. Default pausado; claim usa FOR SHARE/423, renew pede PAUSE sem sobrescrever CANCEL; não afirma stop. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AG-pausa-global.md).

FAC-012AE adiciona GET /api/projects/:projectId/events e events/stream administrativos: páginas até 100, cursor bigint string, SSE Last-Event-ID e invalidação minimizada transacional; tickets/runs atualizam com polling fallback. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AE-eventos-projeto.md).

FAC-012AD adiciona GET /api/operation administrativo e painel Estado da fábrica sem projeto. DTO minimizado, heartbeat stale não vira offline por escrita; ausência de índice global válido bloqueia claim novo. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AD-painel-operacao.md).

FAC-012AC exige ausência global de tentativa sem parada antes de claim novo; índice PostgreSQL fecha corrida entre runs. Replay preservado, lease vencida não libera. 445 testes + 3 PostgreSQL; [evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AC-writer-global.md).

FAC-012Z: Definição do projeto configura arquivos Markdown/relatório/seções em `executionProfile.documentation`. Novos READY exigem política; arquivos precisam estar permitidos. Painel de resultados e relatório público mostram hashes/gate; aceite exige evidências PASS para conjunto exato e manifesto final, além de checks/revisão/bundle/stop existentes. Legados permanecem legíveis. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012Z-gate-documentacao-tecnica.md).

FAC-012Y adiciona até duas alternativas explícitas por função na configuração e histórico de handoff no resultado. Defaults vazios, catálogo/referências validados; nenhuma troca para API/extras, nenhum novo writer liberado pelo painel. [Evidências](../../09-entregas/2026/handoff/2026-10-03-FAC-012Y-handoff-configurado.md).

FAC-012X persiste WAITING_PROVIDER com checkpoint/resultado coerentes e permite resume existente somente por stop/bundle/versão/fence exatos; painel explica autenticação/cota/configuração e confirmação de novo orçamento. Nenhum retry automático. [Evidências](../../09-entregas/2026/operacao/2026-10-03-FAC-012X-espera-provider.md).

FAC-012W amplia artifact para 8 MiB JSON/6 MiB raw, sem truncar e com hashes/guard/fencing inalterados. Parser API aceita envelope bounded, painel integral/escapado informa teto. 387 testes/checks; transporte interno por loopback, nenhum upload público ampliado. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012W-artefatos-ampliados.md).

FAC-012U: POST `/api/runs/:id/recover-finalization` solicita outbox FINALIZATION_ONLY sob auth/versão/tentativa. REQUESTED não é prova; worker reenvia journal/reconcile sem claim/IA. Unknown permanece bloqueado; conclusão vista pelo status do run. [Relatório](../../09-entregas/2026/controle/2026-10-03-FAC-012U-recuperacao-finalizacao.md).

FAC-012T: POST `/api/runs/:id/resume` confirma versão/origem/digest e exige stop/resultado/bundle/checkpoint íntegros. Outbox congela objetivo original; claim cria fence novo só para intenção exata. UI confirma nova tentativa/limites e preserva histórico. [Relatório](../../09-entregas/2026/controle/2026-10-03-FAC-012T-retomada-snapshot.md).

FAC-012S: POST `/api/runs/:id/control` AdminAuthGuard/expectedVersion/attemptId/PAUSE|CANCEL registra intenção, não parada. Detail expõe pedido; worker lê na renew e conclui após stop/evidência. UI confirma por versão; PAUSED manual difere de PAUSED_LIMIT; retomada pendente. [Relatório](../../09-entregas/2026/controle/2026-10-03-FAC-012S-comandos-run.md).

FAC-012R amplia relatório público com CANCELLED/INTERRUPTED e reconcile valida relatório/snapshot de cancelamento. UI pode inspecionar resultado parcial, mas Q não o torna elegível ao aceite/DONE. Sem comandos de pausa/retomada por esse incremento. [Evidências](../../09-entregas/2026/operacao/2026-10-03-FAC-012R-snapshot-interrupcao.md).

FAC-012Q: GET `/api/runs/:id/delivery` gera Markdown/gate a partir de READY congelado/evidências; POST `/api/runs/:id/approve-delivery` AdminAuthGuard/expectedVersion/attempt/deliveryDigest registra aceite exato e só então DONE em run/ticket. Checks/review/bundle/stop precisam coincidir, replay igual não duplica; stale falha. Painel exige confirmar critérios/diff/docs e loadedVersion atual. Relatório não substitui documentação técnica no projeto. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012Q-documentacao-aceite-entrega.md).

FAC-012P: POST interno `/api/internal/orchestration/attempts/:id/artifact` (WorkerAuthGuard) persiste bundle estrito/imutável/fenced após relatório correspondente. GET `/api/runs/:runId/attempts/:attemptId/artifact` (AdminAuthGuard) confere vínculo e retorna artifact nullable. Painel carrega sob demanda com abort, diff escapado/download JSON; não executa patch. Migration não aplicada, teto 64 KiB. [Relatório](../../09-entregas/2026/controle/2026-10-03-FAC-012P-artefatos-painel.md).

FAC-012N adiciona POST interno de reconciliação por tentativa, WorkerAuthGuard e payload estrito workerId/fence, response state nullable. Não é comando administrativo de retry nem prova de parada fornecida pelo painel. Completar não pode reescrever terminal/outcome divergente ou estado humano. [Evidências](../../09-entregas/2026/operacao/2026-10-03-FAC-012N-reconciliacao-checkpoint.md).

## Atualização FAC-012M

GET `/api/runs?projectId=UUID` e `/api/runs/:runId` usam AdminAuthGuard e limitam consultas a 50 registros/tentativas recentes. POST `/api/internal/orchestration/attempts/:attemptId/result` exige WorkerAuthGuard, worker/fence atuais e relatório estrito até 64 KiB. Resultado imutável não libera writer. Painel exibe histórico/checks/revisão/metadados/chamadas, polling sequencial/cancelável; desconhecidos não são estimados. Migration versionada, não aplicada; sem aprovação/retomada automática. [Relatório](../../09-entregas/2026/controle/2026-10-03-FAC-012M-resultados-execucao-painel.md). Passagens abaixo sobre probe/bibliotecas descrevem seus incrementos históricos; estado integrado atual é FAC-012L/M, gate desligado.

Status: controle base ACEITO no FAC-003; Centro de Configuracoes ACEITO no FAC-011.

A API NestJS expõe health sem autenticação e protege as rotas administrativas com um Bearer token vindo de `ADMIN_API_TOKEN`. O painel solicita esse token ao operador, mantém o valor somente durante a sessão do navegador e chama contratos validados por Zod.

Projetos registram nome, URL do repositório e referência base. Tickets registram objetivo, critérios, estado e versão otimista. FAC-003A permite trocar a referencia por um SHA Git minusculo de 40 caracteres com comparacao otimista. A transição `DRAFT -> READY` so grava ticket e `ticket.ready.v1` quando esse SHA ja esta resolvido; falha de elegibilidade ocorre antes de qualquer mutacao. A chave `ticket:{id}:ready` impede evento duplicado em retry.

FAC-001A adiciona `ProjectDefinition`, configurada pelo painel sem hardcode de projeto ou piloto. Ela persiste descricao, stack externa, instrucoes, caminhos permitidos/proibidos e checks em `argv`, com contrato Zod e versao otimista. Definicao ausente e retornada como `null`; nenhum default sintetico finge inspecao. Novos READY exigem definicao e registram `projectDefinitionVersion`. Contas, providers, modelos e papeis continuam separados em `FactorySettings`.

FAC-012I adiciona `executionProfile` opcional à definição: o operador lista fontes de contexto individuais dentro de caminhos permitidos e marca checks configurados para execução autônoma. A API exige perfil não vazio antes de promover ticket a READY e congela o perfil no snapshot. O worker compara fontes e checks resolvidos localmente com o snapshot e compila somente checks aprovados. Editar comando/argumentos revoga a aprovação no painel. Perfil vazio continua válido para rascunho, nunca para READY.

Rotas atuais:

- `GET /api/auth/session`
- `POST|GET /api/projects`
- `PATCH /api/projects/{projectId}/base-revision`
- `GET|PUT /api/projects/{projectId}/definition`
- `POST|GET /api/projects/{projectId}/tickets`
- `POST /api/tickets/{ticketId}/ready`
- `GET|PUT /api/settings`
- `GET /api/internal/worker-settings` (somente `WORKER_API_TOKEN`)
- `POST|GET /api/settings/github/verifications`
- `POST|GET /api/settings/github/onboarding`
- `GET /api/settings/github/onboarding/{sessionId}/challenge`

FAC-011 adiciona uma configuracao administrativa singleton no PostgreSQL. O update exige `expectedVersion` e valida de forma atomica instalacoes de clientes oficiais, catalogos de modelos, atribuicoes por funcao, metadados GitHub e protecoes financeiras. Objetos sao estritos: credenciais, tokens e campos desconhecidos sao rejeitados. Os defaults nao afirmam elegibilidade — contas ficam desabilitadas, `AUTH_REQUIRED` e sem modelos ate preflight do worker. O update administrativo tambem nao pode promover estado de provider ou GitHub; essas observacoes ficam reservadas a evidencia do worker.

FAC-012E expõe snapshot da configuração somente ao worker autenticado. A resposta usa contrato Zod estrito, inclui versão e instante de observação, lê sem escrita e cai em defaults inativos/version 0 se a linha ainda não existir. O endpoint não expõe o registro Prisma nem inclui campos de credencial.

FAC-011A permite pedir/listar verificacoes GitHub. A criacao grava `GithubVerification` e outbox na mesma transacao; start/complete ficam em rotas `/api/internal/github-verifications/{id}/...` protegidas por `WORKER_API_TOKEN`. Somente a conclusao valida do worker altera o estado observado e incrementa a versao da configuracao. Credenciais e output bruto nao pertencem a esses contratos.

FAC-011B adiciona sessoes GitHub e `github.onboarding.requested.v1`. Rotas internas `/api/internal/github-onboarding/{id}/...` iniciam, publicam desafio e concluem sob `WORKER_API_TOKEN`. PostgreSQL nunca recebe o desafio; a rota administrativa le o valor efemero do Redis. Conclusao `CONNECTED` exige `SECURE_STORE`; login/verificacao ativos se excluem e host alterado invalida a sessao antiga.

FAC-011C adiciona `GET/POST /api/settings/github/repository-verifications` e as rotas internas `/api/internal/github-repository-verifications/{id}/start|complete`. Pedido e `github.repository-verification.requested.v1` sao atomicos e guardam snapshot do alvo, nunca credencial. O complete so aceita o worker autenticado, reconfere configuracao `CONNECTED` e alvo atual e persiste evidencia integral ou falha sem observacao parcial; a API nao executa `gh`.

FAC-011D adiciona `GET/POST /api/settings/github/pull-requests`, `approve` e `cancel`, alem das rotas internas `start|complete`. Somente `approve` cria outbox; `expectedVersion` e digest vinculam o gate ao payload exato. Complete reconfere alvo/opt-in e URL. API nao executa escrita GitHub.

FAC-008 conecta o evento `READY` ao BullMQ por dispatcher idempotente, oferece o protocolo interno de runs/attempts, leases, fencing e checkpoints e liga ao worker um consumidor restrito ao probe sintetico. O consumidor do runtime real ainda nao esta ligado ao loop principal. SSE, usuarios multiplos, login gerenciado e conexao GitHub pertencem aos proximos tickets.

FAC-003A tambem protege a fronteira do dispatcher: evento legado PENDING com `baseRevision: null` nunca chega a `queue.add` e converge para `FAILED` pelo limite da outbox. Eventos e jobs historicos permanecem preservados; nao existe limpeza automatica neste fluxo.

FAC-012A torna o evento READY autocontido: `executionSpecification` copia projeto, SHA, definicao completa e ticket dentro da mesma transacao. O contrato cruza IDs e versoes do snapshot com o envelope. Atualizar a definicao depois nao altera eventos anteriores, e evento legado sem snapshot nao e publicado. O responsavel aceitou FAC-012A na revisao documental `3ef3d543b98fb48226714787315f4de947fa6dd9`.

FAC-012B consome esse snapshot apenas como dado de entrada de um compilador puro no worker. Perfil local confiável e allowlists não são aceitos do job; nenhum campo administrativo inicia comando por si só. O responsável aceitou FAC-012B na revisão documental `3e28363b0642f8e05840bb649b681e3837e1e015`. O consumer continua em probe até enforcement de caminhos de escrita, binding/checkout confiável, provider elegível e integração de lease/fencing serem demonstrados.

FAC-020A verificou a home no Vite habitual (porta 5173). FAC-021 corrigiu a marca apresentada para La fabrique e centralizou a cena reduzida no desktop; mobile usa espaço disponível. [Evidências atuais](../../09-entregas/2026/controle/2026-10-05-FAC-021-nome-escala-home.md).

## Template padrão — FAC-022

Código atual `4e042a1`, integrado em developer. `DashboardLayout` é a raiz visual de home/login/controle/configurações: mantém menus e altera somente children do conteúdo administrativo (home é o conteúdo padrão). Sidebar marca a área atual; topbar sticky e busca permanecem disponíveis. Conteúdo interno usa h1 24px, h2 16px, h3/campos/botões 13px e texto/labels 12px, inclusive modais. Autenticação e contratos preservados. [Relatório e capturas](../../09-entregas/2026/controle/2026-10-05-FAC-022-template-tipografia.md).


FAC-023: Configurações → Equipes → Skills é o cadastro de skills por projeto; Agentes permite adicionar, editar e excluir múltiplos perfis com vínculos às skills. Mantém template/fontes. [Contrato, persistência e evidências](../../09-entregas/2026/configuracao/2026-10-05-FAC-023-agentes-skills-projeto.md).


FAC-027: botão sob a marca recolhe sidebar para 66px e só ícones; preferência local persiste navegação/reload desktop. Mobile recolhido por padrão, expansão sobre conteúdo. [Funcionamento/checks](../../09-entregas/2026/controle/2026-10-06-FAC-027-menu-recolhivel.md). FAC-026 apresenta agentes pré-configurados na lista com switches e explicação/configuração em dois passos; [relatório](../../09-entregas/2026/configuracao/2026-10-06-FAC-026-agentes-preconfigurados.md).


FAC-028: toggle minimalista só com seta de 16px, sem texto visível/borda/fundo; nomes acessíveis e comportamento FAC-027 preservados. [Checks e capturas](../../09-entregas/2026/controle/2026-10-06-FAC-028-seta-menu-minimalista.md).


FAC-029: seta minimalista agora na borda interna direita, nos estados expandido/recolhido. [Funcionamento/evidências](../../09-entregas/2026/controle/2026-10-06-FAC-029-seta-direita-menu.md).


FAC-030: padrão atual é menu recolhido em toda abertura/reload, independente da preferência antiga. Expansão manual mantém estado entre telas na navegação atual. Seta permanece minimalista à direita. [Funcionamento/checks](../../09-entregas/2026/controle/2026-10-06-FAC-030-menu-inicial-recolhido.md). Este pedido substitui a persistência após reload do FAC-027.


FAC-031: posição atual da seta depende do estado: central no recolhido, à direita com padding 12px no expandido. Padrão de abertura FAC-030 preservado. [Checks](../../09-entregas/2026/controle/2026-10-06-FAC-031-posicao-seta-menu.md).
