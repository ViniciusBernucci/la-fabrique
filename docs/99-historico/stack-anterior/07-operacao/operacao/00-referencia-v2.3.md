> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../../08-desenvolvimento/00-README.md)

> Instruções de criação em documentacoes/ ou lessons/ abaixo são histórico da fonte; novas entregas/lessons seguem exclusivamente docs/00-governanca/POLITICA-DOCUMENTACAO.md.

> Nota editorial DOC-MV-001, 2026-10-07: conteúdo integral local preservado como referência da evolução v2.3. Afirmações por incremento têm a revisão/data original; trechos sobre consumer ausente, recuperação pendente e caminhos antigos não descrevem necessariamente HEAD a2cc5e0. O [AS-IS conferido](../../02-arquitetura/01-as-is.md) e a [auditoria](../../00-governanca/08-AUDITORIA.md) delimitam o estado atual; números de testes abaixo são evidência histórica, não reexecução desta migração.

# Operação da fábrica e worker

FAC-021: marca apresentada padronizada para La fabrique; rótulos de runtime/unit corrigidos, identificadores técnicos e operações preservados. Home integrada em developer/5173 conforme FAC-020A. [Relatório](../../09-entregas/2026/controle/2026-10-05-FAC-021-nome-escala-home.md).

FAC-018: Autenticação com keyring; implementação pronta, consentimento real e keyring Antigravity pendentes. [Relatório](../../09-entregas/2026/configuracao/2026-10-05-FAC-018-login-claude-antigravity.md).

FAC-017: só COMPLETED/AVAILABLE confirmado pelo cliente significa assinatura conectada; abrir aba não basta. [Passo a passo](../../09-entregas/2026/operacao/2026-10-05-FAC-017-confirmacao-login-codex.md).

FAC-016 confirma que setup exige reinício do supervisor dev para recarregar ambiente; Codex real voltou a AUTH_REQUIRED. [Procedimento](../../09-entregas/2026/operacao/2026-10-05-FAC-016-env-processo-dev.md).

FAC-015 provisiona identidades privadas com providers:setup; login Codex no painel e Claude por providers:login. Reinício dev obrigatório após alterar .env. [Procedimento](../../09-entregas/2026/operacao/2026-10-05-FAC-015-preparacao-assinaturas.md).

OPS-008: erros 500 em definição/PR e 503 em estado operacional foram associados a tabelas ausentes no banco dev; verifique `npm run db:status`. Health ready pode estar OK com migrations pendentes. Reconciliar histórico e prova de parada antes de atualizar schema; favicon fornecido. [Relatório](../../09-entregas/2026/infraestrutura/2026-10-05-OPS-008-bootstrap-migrations-telas.md).
OPS-007 consolida FAC-012Z–AH na developer por fast-forward e remove as nove worktrees funcionais após verificar limpeza/ancestry. Branches e aceites permanecem preservados; serviços não foram atualizados. [Relatório](../../09-entregas/2026/operacao/2026-10-05-OPS-007-consolidacao-worktrees.md).
FAC-012AH: writer global ocupado mantém próximos jobs delayed; somente parada comprovada libera capacidade. Não limpar jobs/índice ou usar expiração como stop. Pretest prepara pacotes internos. MVP interno verificado; contas/UID/financeiro/ativação continuam operação própria, sem piloto requerido. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AH-admissao-capacidade.md).

FAC-012AG: pausa global pela interface mantém jobs/outbox; ativos recebem PAUSE via renew e preservam snapshot/journal. Queue pausa também recovery enfileirado; finalização direta aceita evidência existente. Retomar agendamento não retoma run terminal/unknown nem ativa gate env/auth. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AG-pausa-global.md).

FAC-012AF fornece npm run backup create/verify/restore para dump/snapshots/journal congelados, chave externa privada, confirmação de serviço parado e staging novo. Sem dump/restore automático de ambiente existente. [Comandos e evidências](../../09-entregas/2026/operacao/2026-10-03-FAC-012AF-backup-restauracao.md).

FAC-012AE requer migration de project_events/cursors/triggers para SSE; painel conserva HTTP fallback se indisponível. Backup deve incluir eventos/cursors; sem retention automática. Proxy streaming sem buffering, heartbeat 1 s e idle client 15 s. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AE-eventos-projeto.md).

FAC-012AD mostra operação sem piloto: heartbeat até 180 s recente, futuro inconsistente e tentativas sem stop continuam bloqueadas mesmo com lease vencida. Índice global inválido impede novos claims; painel não fabrica prontidão de IA. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AD-painel-operacao.md).

FAC-012AC impede novo claim em qualquer run enquanto houver tentativa sem stop confirmado. Não limpar por lease/status; banco incoerente bloqueia migration e exige diagnóstico. `npm run test:postgres` prova exclusão em banco efêmero, sem serviço real. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AC-writer-global.md).

FAC-012AB verifica composição real Git/sandbox/contexto/snapshot/bundle/journal com adapters/controle sintéticos. Cinco cenários, 438 testes; nenhum gate operacional ativado. [Evidências](../../09-entregas/2026/operacao/2026-10-03-FAC-012AB-ensaio-integrado-mvp.md).
FAC-012Z conecta documentação técnica ao workflow real antes de Reviewer: Developer atualiza arquivos configurados, checks/snapshot/gate bounded verificam estrutura, revisão independente confere verdade e snapshot posterior invalida mudanças durante review. Ausência de política bloqueia compilação real; até duas correções continuam globais. Consumer/preflight não foram ativados. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012Z-gate-documentacao-tecnica.md).

FAC-012Y mantém lease/fence ao trocar cliente dentro da mesma tentativa, após parada e restauração verificadas. Novo job continua com fence novo; unknown/crash não é recuperado por suposição. Alternativas UI vazias por padrão, autenticação/preflight manuais. [Evidências](../../09-entregas/2026/handoff/2026-10-03-FAC-012Y-handoff-configurado.md).

FAC-012X conecta espera por auth/cota/busy no consumer: stop conhecido → snapshot → journal/result/artifact/checkpoint → WAITING_PROVIDER; retomada humana existente por evidência/configuração atual. Unknown não fabrica espera, nenhum retry/login/gasto automático. 399 testes/checks, migration não aplicada. [Relatório](../../09-entregas/2026/operacao/2026-10-03-FAC-012X-espera-provider.md).

FAC-012W supera o teto histórico 64 KiB: bundle até 8 MiB JSON/6 MiB conteúdo, incluindo base64/metadados no teto JSON; API aceita envelope +4096 bytes. Excesso preserva evidências e não aprova. Nenhuma retenção/cleanup automática; backup manual inclui stores/journal/snapshots/banco sem exportar tokens. [Relatório](../../09-entregas/2026/controle/2026-10-03-FAC-012W-artefatos-ampliados.md).

FAC-012V: configurar WORKER_PROVIDER_ROOT separado, existente/canônico, owner serviço e 0700 antes de login/status pela UI. Cada provider/ID tem home/store/cache privados; não copiar login antigo. Consumer usa factory por instalação sem herdar segredos do controle/API; Claude escrita bloqueada. [Relatório](../../09-entregas/2026/runtime/2026-10-03-FAC-012V-identidades-instalacoes.md).

FAC-012U publica recovery separado de jobs FAILED: identidade original → journal parado/result/artifact/checkpoint → reconcile; sem journal apenas reconcilia prova existente, sem checkout/IA/novo fence. BLOCKED_RECOVERY só conclui com stop/evidências íntegras; unknown não se libera por lease. [Relatório](../../09-entregas/2026/controle/2026-10-03-FAC-012U-recuperacao-finalizacao.md).

FAC-012T conecta retomada explícita via run.resume.v1: intenção imutável, claim fenced após stop/evidência, ArtifactReader, baseline limpo → restore → contexto reconstruído → workflow. Redelivery não cria writer; unknown/crash sem prova bloqueados. Developer FAILED conhecido preserva snapshot. [Relatório](../../09-entregas/2026/controle/2026-10-03-FAC-012T-retomada-snapshot.md).

FAC-012S liga PAUSE/CANCEL à renew fenced: abort tipado → stop → snapshot/journal/result/bundle/checkpoint → PAUSED/CANCELLED. Pedido não libera writer; unknown conserva fence. Pode aguardar lease/3; retomada/crash pendentes. [Relatório](../../09-entregas/2026/controle/2026-10-03-FAC-012S-comandos-run.md).

FAC-012R preserva interrupção de workflow já materializado/contextualizado: stop conhecido → snapshot novo/observações → journal → result/artifact → checkpoint/complete CANCELLED. Unknown sticky ou ausência/falha de evidência deixa fence bloqueado, sem novo writer. Crash abrupto/retry administrativo/retomada seguem pendentes. [Relatório](../../09-entregas/2026/operacao/2026-10-03-FAC-012R-snapshot-interrupcao.md).

FAC-012Q acrescenta estado Run DONE e approval persistido somente após aceite humano exato; worker nunca faz essa promoção e N impede regressão de estados humanos. Relatório usa evidências, não comando/check/cliente na API; sem merge/deploy. Migration não aplicada. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012Q-documentacao-aceite-entrega.md).

FAC-012P: finalização normal exige bundle do último snapshot antes de checkpoint/complete. ArtifactReader lê só UUID em snapshots privado e verifica manifesto/patch/untracked, recusa symlink/segredos conhecidos/limites. Teto JSON 64 KiB (bytes originais 48.000); oversized preserva journal e falha sem truncar. Não há cleanup automático. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012P-artefatos-painel.md).

FAC-012O: resultado normal parado é preservado em `WORKER_EXECUTION_ROOT/result-journal` (0700/0600, hash, até 128 KiB, fsync/publicação sem overwrite) antes de report/checkpoint/complete. Redelivery lê antes de claim e reenvia com fence original, sem IA. Root privado fora de workspaces; nenhuma limpeza automática. Interrupção pré-retorno e retry de jobs FAILED não estão cobertos. [Relatório](../../09-entregas/2026/operacao/2026-10-03-FAC-012O-journal-resultados.md).

FAC-012N: replay chama endpoint interno `/api/internal/orchestration/attempts/:id/reconcile` sob WorkerAuthGuard/fence atual. Só conclui com attempt/checkpoint parados; relatório COMPLETED/PAUSED validado/digest coerente. Sem prova retorna state null; sem checkout/IA/retry. Nenhuma regressão de estados humanos. [Relatório](../../09-entregas/2026/operacao/2026-10-03-FAC-012N-reconciliacao-checkpoint.md).

FAC-012M (`14ae5ba`) entrega relatório persistido antes de concluir e consultas/painel somente leitura. 240 testes e checks locais; migration apenas versionada, gate false, nenhum serviço atualizado. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012M-resultados-execucao-painel.md). Passagens históricas de probe/consumer não integrado foram superadas no código por FAC-012L; operação real continua desabilitada.

OPS-006 integrou o trabalho local OPS-005/K/L e removeu seus worktrees após prova de ancestry/limpeza. [Relatório](../../09-entregas/2026/operacao/2026-10-03-OPS-006-integracao-local-controle-mvp.md) e [controle do MVP](../../08-desenvolvimento/06-controle-mvp.md) distinguem código, aceites e operação manual. Serviços ativos não foram atualizados.

## Estado atual — FAC-012L

O consumer `le-fabrique.execution` compõe checkout confiável, compilador, workflow e LeaseGuard com `WORKER_EXECUTION_ENABLED=true`. O default é `false`, sem fixture consumindo essa fila; jobs aguardam. Instalação, modelo e limites Developer/Reviewer vêm da configuração atual da interface. Perda de lease e parada não comprovada impedem conclusão indevida. SIGINT/SIGTERM e falha do heartbeat abortam o trabalho antes de fechar a fila; consulta de systemd indisponível permanece desconhecida.

Resultado aprovado vai a VALIDATING com checkpoint anterior a complete. Configurar raízes separadas, hosts exatos e CLIs absolutos; ativação depende de OPS-005/preflight real. Transporte de artefatos ao painel, replay recuperável e handoff automático ainda são pendentes. Ver [relatório FAC-012L](../../09-entregas/2026/operacao/2026-10-03-FAC-012L-consumer-execucao-real.md).

## Histórico dos incrementos anteriores ao FAC-012L

OPS-004 consolidou FAC-012D e FAC-012E–J localmente em `developer`; evidências em `2026-10-03-OPS-004-consolidacao-worktrees.md`. OPS-005 prepara o processo host sem instalar serviço. FAC-012I/J/K inicialmente entregaram bibliotecas isoladas; FAC-012L integra essas peças sob gate desabilitado. Identidade de serviço continua sem prova operacional.

FAC-012K adiciona `LeaseGuard` isolado no worker: recebe fencing token, duração/expiração da lease, função de renovação e callback de parada; passa `AbortSignal` ao trabalho e falha sem aceitar resultado quando a renovação falha ou vence. A função `stopWriter` só confirma quiescência se retornar `true`; a guarda sempre espera a operação terminar antes de propagar a perda da lease. O helper não está ligado ao consumer, e sua integração ainda precisa provar cancelamento real das árvores de processo sob a identidade do serviço.

## Decisão obrigatória da stack - revisão 2.3
A stack da própria La fabrique está APROVADA: React + TypeScript + Vite no painel; NestJS + TypeScript na API; worker Node.js + TypeScript em processo separado; PostgreSQL; Redis + BullMQ; Docker Compose na mesma VPS. Não solicitar nova escolha ou confirmação da stack. Não iniciar a fábrica em PHP/Laravel, Angular ou .NET. Esta decisão substitui propostas anteriores.
Monorepo: `apps/web`, `apps/api`, `apps/worker`, `packages/contracts`. Contratos compartilhados precisam de validação em runtime. API não executa clientes, builds ou testes; o worker executa esses trabalhos com isolamento, limites e um writer inicial.
Ao trabalhar na própria fábrica, aplicar esta stack. A regra de preservar a stack existente aplica-se somente a projetos EXTERNOS cadastrados para desenvolvimento pela fábrica; ela não altera a stack da La fabrique. Se o repositório da fábrica contiver implementação anterior incompatível, registrar a divergência e planejar a adaptação por etapas; não apagar código existente nem reabrir a escolha tecnológica.
Versões exatas e comandos devem ser fixados conforme compatibilidade no bootstrap; isso não é uma nova decisão de stack. Repositório e funcionalidade do piloto externo permanecem pendentes quando não fornecidos.

## Inicialização
Durante a construção, confirmar políticas e usar somente fixtures sintéticas; instalar clientes de fontes oficiais com versões registradas; login humano nos clientes; verificar extras desligados nas contas; executar preflight; registrar worker com credencial própria de escopo mínimo e validade/rotação. Piloto externo é experimento opcional posterior e não bloqueia a conclusão/preparação operacional do MVP. Deploy do controle é uma tarefa futura autorizada separadamente.

FAC-002 comprovou o Codex CLI 0.159.2 no usuario atual da VPS, com autenticacao ChatGPT, chaves de API ausentes e comandos confinados a fixtures. O teste financeiro continua humano: verificar em Settings > Usage que creditos e recarga automatica nao serao usados. O FAC-005 deve repetir a prova sob a identidade de servico que executara o adapter.

FAC-005 implementou o adapter como biblioteca do `packages/runtime`: processo sem shell, prompt por stdin, ambiente sem chaves de API, eventos sanitizados, limite de logs, timeout e cancelamento do grupo. O worker ainda inicia fila de probe e filas administrativas, mas não cria consumidor para `le-fabrique.execution`; o workflow real permanece desligado.

FAC-006 adicionou Context Builder e RuntimeGuard como bibliotecas locais. O manifesto registra fontes, hashes, omissoes e revisao-base; qualquer omissao sinaliza truncamento. O guard deve autorizar a tentativa antes do adapter e seu estado precisa ser persistido pela futura orquestracao. O sanitizador comum remove nomes conhecidos de API key, mas a identidade de servico e o sandbox ainda precisam provar que o codigo executado nao alcanca o armazenamento oficial de login.

FAC-007 implementa o limite de execucao local: supervisor cria worktree e snapshot fora do namespace; comandos rodam em unidade systemd transiente, cgroup e namespaces. FAC-012H pivota o comando para uma raiz mínima, sem host root/home/`/run`/`/var`/Docker socket ou rede externa, e remove capabilities antes de executar o check. Timeout mata o cgroup e exige confirmacao. O host precisa passar preflight de systemd de usuario, user namespaces, mount e `prlimit`; falha de qualquer capacidade bloqueia jobs. O loop BullMQ ainda nao chama esses componentes.

FAC-008 implementa o dispatcher outbox/BullMQ e o protocolo interno de claim, renovação, checkpoint e conclusão. O antigo consumidor fixture de `le-fabrique.execution` foi retirado de `main.ts`: READY não será marcado `VALIDATING` sem executar trabalho real. Run/attempt persistem lease e fencing; reentrega reutiliza attempt parado e lease vencido sem quiescência confirmada bloqueia recuperação. FAC-003A exige SHA-base exato e FAC-012A snapshot imutável. `processOrchestrationFixture` permanece somente como fixture de testes. FAC-012A/B foram aceitos; workflow/compilador ainda não ligados à fila.

FAC-009 implementa `DeveloperWorkflow` como coordenador injetavel no worker. Ele cria worktree na revisao exata, constroi contexto, mede baseline, autoriza chamadas pelo guard, executa Developer com escrita, checks argv confiaveis no sandbox, snapshot e Reviewer em execucao separada somente leitura. Falha preexistente e registrada separadamente; regressao nova bloqueia aprovacao. Check sem parada confirmada encerra o fluxo antes de iniciar writer ou capturar snapshot.

FAC-012B compila snapshot FAC-012A em `DeveloperWorkflowRequest` após reconciliar projeto/URL/SHA contra perfil confiável do worker. Checks configurados precisam corresponder exatamente a uma allowlist em argv; fontes de contexto ficam em caminhos permitidos e fora dos proibidos. Limites e modelo entram por configuração validada, não por hardcode. O compilador ainda não é chamado pelo consumer.

FAC-012C torna o `SandboxRunner` read-only por padrão: o contrato aceita caminhos graváveis relativos explícitos, e o runner rejeita caminhos ausentes, fora da raiz ou com symlinks antes de iniciar. O launcher monta o workspace somente leitura e abre apenas os caminhos selecionados. Checks do workflow usam lista vazia.

Correção de escopo: o `CodexAdapter` não usa `SandboxRunner`, mas já configura o sandbox nativo do Codex em perfil customizado com rede desabilitada e raiz negada. O perfil atual dá escrita em `.`; FAC-012D leva os `allowedPaths` configurados no projeto para as permissões do CLI. Credenciais oficiais e acesso no sistema operacional ainda exigem validação sob a identidade real do worker.
FAC-012J adiciona `prepareRepositoryCheckout` como serviço confiável fora do sandbox: root/hosts são configuração local, repo URL é HTTPS sem credenciais e host allowlisted, SHA é exato e o destino combina UUID de projeto/workflow. `gh auth status` precisa provar armazenamento `keyring` antes de `gh auth token`; o token é mantido em memória e enviado ao Git por configuração ambiental host-scoped, nunca por argv, URL, log, job ou controle. Git roda sem shell, `GIT_CONFIG_NOSYSTEM`, hooks ou transferência LFS, e o checkout detached é publicado por rename de diretório privado. O módulo não está ligado ao loop BullMQ; os testes usam runner sintético e não constituem prova da VPS/keyring reais.
O coordenador ainda nao substitui o probe BullMQ. FAC-001A fornece a definicao versionada desejada de repositorio, caminhos e checks, mas o worker ainda precisa traduzi-la para um perfil local confiavel/allowlist e associar o repositorio clonado. Ativar antes disso aceitaria configuracao administrativa sem a reconciliacao do executor. O FAC-010 pode reutilizar o resultado e snapshots, mas nao deve contornar esse gate.

FAC-010 esta WAITING_PROVIDER. Claude Code e Antigravity estao instalados, mas deslogados; o worker nao deve tentar login, copiar credenciais ou cair para API. Sem segundo provider elegivel, preservar snapshots/checkpoints e aguardar intervencao humana.

FAC-011 implementa o Centro de Configuracoes antes da retomada do FAC-010. O operador pode preparar contas, modelos, atribuicoes por funcao e metadados GitHub no painel, mas salvar nao executa login ou provider. O worker confiavel ainda precisa reconciliar o estado desejado com binario, identidade e autenticacao reais. Dados de login permanecem no armazenamento oficial do cliente e nunca no browser/tabela `factory_settings`.

FAC-010A adiciona reconciliacao somente-leitura: `Verificar instalacao` cria job dedicado e o worker executa comandos de status fixos. Saida bruta nao retorna; apenas estado, versao, modelos filtrados e mensagem controlada. A fila tem concorrencia um e fecha junto ao worker. Login permanece proibido ate o canal efemero FAC-010B.

FAC-010B adiciona a fila concorrencia-um `le-fabrique.provider-onboarding`. Somente Codex esta habilitado: o administrador inicia a sessao, abre a URL oficial e informa o codigo temporario. O desafio fica no Redis privado com TTL maximo de dez minutos, nunca no PostgreSQL/log, e e removido na conclusao; indisponibilidade do Redis falha fechado. Reinicio ou sinal de parada encerra o grupo de processos, e sessao vencida permanece `EXPIRED`/`AUTH_REQUIRED`.

FAC-010C adiciona o adapter Claude e o coordenador de handoff como bibliotecas, sem liga-los ao consumer BullMQ real. O processo Claude remove credenciais/modos API e cloud herdados, usa perfil fixo e so aceita auth de assinatura first-party. Handoff restaura snapshot em nova worktree depois de quiescencia e RuntimeGuard; auth/cota no destino retorna `WAITING_PROVIDER`. Ativar exige perfil confiavel de repositorio/comandos, claim/lease/fencing persistidos e preflight real da identidade do worker.

FAC-011A adiciona a fila concorrencia-um `le-fabrique.github-verification`. O worker executa apenas `gh --version` e `gh auth status --hostname <host> --json hosts`, remove variaveis de token, limita processo a 15 s/64 KiB e persiste somente classificacao controlada. O GitHub CLI nao esta instalado no ambiente desta entrega; instalar e autenticar a identidade de servico continuam passos operacionais futuros. Login, PR, merge e deploy nao fazem parte deste consumer.

FAC-011B adiciona a fila concorrencia-um `le-fabrique.github-onboarding`. O fluxo web/device dura no maximo dez minutos; URL/codigo ficam no Redis e a arvore e encerrada se publicacao, timeout ou shutdown falhar. Somente `tokenSource=keyring` e elegivel; fallback oficial para `hosts.yml` fica `ERROR` e exige remediacao consciente sob a identidade de servico. O worker nao apaga arquivo nem desloga automaticamente. `gh` continua ausente, portanto esta entrega nao altera autenticacao real.

FAC-011C adiciona a fila concorrencia-um `le-fabrique.github-repository-verification`. Ela so recebe snapshot de uma configuracao salva `CONNECTED` e executa duas chamadas `gh api` com `GET`: metadados/permissao do repositorio e branch base. Segmentos sao codificados, tokens herdados sao removidos e resposta bruta e descartada. Mudanca de alvo durante a execucao invalida o resultado; nenhuma operacao Git, escrita ou PR pertence a este consumer.

FAC-011D adiciona `le-fabrique.github-pull-request`, concorrencia um. Job so nasce apos aprovacao humana exata. O consumer prova push, reconcilia PR aberto e cria no maximo o PR aprovado; nao cria branch, nao faz push e nao possui merge. Resultado incerto e reconciliado antes de retry humano.
VPS roda proxy HTTPS, frontend, API, scheduler, PostgreSQL e Redis privados. Supervisor interno roda como usuário dedicado e usa protocolo interno autenticado. Manter credencial da fábrica separada de credenciais dos providers. Não expor endpoint local ou montar diretórios pessoais nos worktrees.
Sandbox sem privileged, socket Docker, home completo, banco/Redis da fábrica, credenciais de Git amplas ou dados reais. Supervisor confiável prepara checkout e serviços sintéticos. Credenciais de provider inevitavelmente acessíveis ao cliente exigem isolamento de identidade/armazenamento; código executado não deve conseguir lê-las. Validar por teste de acesso negado, não só instrução escrita.
## Worker API
Implementado no FAC-004: `POST /internal/workers/register`, `POST /internal/workers/{id}/heartbeat` e `GET /workers` administrativo. O protocolo interno usa `WORKER_API_TOKEN`, não aceita a credencial administrativa e é bloqueado pelo Nginx público. O worker encerra após três heartbeats consecutivos com falha.

Implementado no FAC-008: claim, renovacao, checkpoint e complete com payloads versionados e autenticacao interna. Claim e transacional; todas as mutacoes exigem fencing token atual. Checkpoint aceita somente revisoes e hashes validados pelos contratos. Eventos detalhados e armazenamento ampliado de artefatos permanecem futuros.
Worker não escolhe arbitrariamente repo, comando ou URL fornecidos em output de IA. Controle só distribui jobs de projetos cadastrados; valida repo/ref e perfis de comando. Eventos/logs são sanitizados e limitados antes de armazenar/exibir.

## Controle administrativo atual

FAC-003 protege `GET /auth/session`, projetos e tickets com `Authorization: Bearer`. O valor de `ADMIN_API_TOKEN` tem no mínimo 32 caracteres, fica somente no ambiente da API e nunca deve entrar no bundle, banco ou logs. O painel conserva a credencial em `sessionStorage`, portanto a implantação exige HTTPS e uma origem web confiável; identidade multiusuário, expiração e revogação granular continuam futuras.

O operador pode criar/listar projetos, configurar revisao-base e definicao versionada, criar/listar tickets, promover `DRAFT` para `READY` e editar o Centro de Configuracoes. A definicao guarda contexto, stack, instrucoes, caminhos e checks separados em executavel/argumentos; nao guarda contas ou modelos. A promoção exige `expectedVersion`, SHA Git exato e definicao persistida, incrementa a versão e cria `ticket.ready.v1` com `projectDefinitionVersion` na mesma transação. `deduplication_key` único torna retries de READY idempotentes.
## Recuperação
Queda do worker: parar agendamento local, manter painel, esperar heartbeat; preservar checkpoints. Internet caiu: parar execução antes de lease expirar. Auth expirada: AUTH_REQUIRED; usuário faz login oficial local. Cota acabou: handoff seguro ou WAITING_PROVIDER. Disco cheio/OOM: PAUSED_RESOURCE, limpar somente workspaces já preservados. CLI mudou schema: bloquear adapter, testar fixture e integrar atualização em ticket próprio.
Kill switch bloqueia claims, solicita cancelamento e monitora fim das árvores de processos; nunca declarar cancelado sem evidência. Supervisor aplica timeout e limites de logs/disco, além de CPU/RAM.
## Backup e restauração
Restaurar PostgreSQL numa instância de teste; conferir tickets e versões; recuperar artefatos por hash; buscar Git/revisão; renovar credencial do worker sem restaurar tokens de provider por cópia informal. Testar cenário com execução interrompida. Registrar tempo real/RPO efetivo e lacunas.
## Observabilidade
Fila/etapa, last heartbeat, lease, processos, CPU/RAM/disco, OOM, duração, provider/modelo/versão, uso conhecido/desconhecido, handoffs, retries, checks/docs e revisão aceita. Evitar prompts completos e segredos nos logs. Browser de QA com perfil efêmero sem sessão pessoal.
## Evolução
Separar execução em outra VPS é alternativa futura, sujeita a métricas e revisão de isolamento. APIs podem prover capacidade com orçamento; execução na VPS não elimina termos dos fornecedores. Código de terceiros exige isolamento mais forte que containers compartilhando host.

## Provisionamento da VPS única no MVP
Selecionar perfil em infraestrutura/DIMENSIONAMENTO-VPS.md e registrar especificações reais. Configurar firewall: HTTPS público; SSH administrativo restrito; Redis/PostgreSQL/worker privados. Separar volumes persistentes de temporários. Instalar supervisor como serviço com usuário dedicado e recuperação após reinício.
Fazer login oficial pelo método suportado para servidor sem interface; callback/fluxo remoto apenas conforme documentação do cliente. Não desabilitar autenticação nem copiar cookies. Confirmar credenciais acessíveis ao cliente e inacessíveis a código/testes/logs.
Rodar baseline no mesmo ambiente que executará jobs. Medir recursos e painel durante build. Reiniciar worker, depois VPS em janela de teste; comprovar leases, nenhuma duplicação e retomada de checkpoint. Validar backup externo e restauração. Não abrir banco/fila para facilitar diagnóstico.

## Bootstrap executável atual

FAC-000 implementa a fundação local. `compose.dev.yaml` publica PostgreSQL e Redis apenas no loopback para desenvolvimento. A composição de controle publica Nginx em `127.0.0.1:8080`, API em `127.0.0.1:3000` e Redis em `127.0.0.1:6379`; PostgreSQL não tem porta publicada. OPS-005 fornece um unit para worker host dedicado, sem instalá-lo. `main.ts` não consome a fila de execução: `processOrchestrationFixture` é somente teste. FAC-009 fornece o coordenador, ainda sem ligação real à fila.

OPS-001 faz o launcher raiz carregar `.env` e limita o registro inicial a 30 tentativas com um segundo de intervalo. Somente falha de rede e HTTP 5xx recebem retry; 4xx/configuracao falham imediatamente. Essa espera nao altera a regra de heartbeat: depois do registro, tres falhas consecutivas encerram o worker. Mensagens históricas `orchestration probe failed` podem pertencer a uma imagem antiga do worker. Não reiniciar/recriar stack ativa automaticamente nem limpar filas durante migração de topologia; provar quiescência e seguir janela de mudança autorizada.

FAC-012D restringe o sandbox nativo do Codex aos caminhos permitidos pelo snapshot e aguarda aceite humano. A evidência não valida identidade de serviço nem habilita consumidor/provider.
FAC-012E fornece ao worker somente leitura interna da configuração da interface, com autenticação `WORKER_API_TOKEN`, schema estrito, versão e defaults não persistidos inativos. Testes usam Prisma/fetch simulados; nenhum banco ou serviço real foi consultado e o endpoint ainda não está no consumer de execução.

FAC-012F resolve a rota configurada em cada chamada e valida adapter/permissões sem executar o cliente. Testes usam snapshots e adapters sintéticos; o módulo ainda não está no consumidor real.

FAC-012G conecta o workflow à resolução independente de Developer e Reviewer por chamada. Testes usam adapters/routing sintéticos, inclusive providers e modelos diferentes; o workflow ainda não está na fila e nenhum cliente real foi iniciado.

FAC-012H troca a raiz do `SandboxRunner` antes do check, monta apenas `/usr` read-only e o workspace, desanexa a raiz host, oculta home/`/run`/`/var`/cgroup, cria devices/temporários mínimos e remove capabilities antes de `exec`. A prova que inicialmente lia `/etc/hostname` agora falha; também falham symlink para host, acesso a socket, rede e remount do workspace. Teste rodou no usuário interativo atual, não sob identidade systemd de produção. Consumer e CLI Developer continuam desligados.


## Preflight granular Claude — FAC-012AA

IMPLEMENTADO como comando opt-in; NÃO EXECUTADO no serviço. Primeiro confirmar stop físico do worker/árvore anterior, origem oficial do CLI suportado 2.1.285, UID e root privado separados dos workspaces. Instalação/modelo precisam ser os IDs escolhidos no painel. Login é oficial/humano sob o mesmo UID/store, extras/créditos/autorecharge desligados; não exportar auth/token nem herdar overrides API/cloud. Ter Node/dependências incluindo tsx, contratos/runtime buildados (`npm ci`, `npm run db:generate`, `npm run build` sem aplicar migrations).

Após essas confirmações, sob o UID do serviço e usando valores reais verificados (não valores de exemplo):

```bash
npm run preflight:claude -- \
  --provider-root "$WORKER_PROVIDER_ROOT" \
  --installation "$CLAUDE_INSTALLATION_ID" \
  --binary "$WORKER_CLAUDE_BINARY" \
  --model "$CLAUDE_MODEL_ID" \
  --confirm-subscription-only \
  --confirm-worker-stopped
```

Flags são confirmação humana, não detecção financeira/stop. Uma chamada de assinatura com teto 180 s/1 MiB, sem retry ou conta/modelo fixos; não usar conta real enquanto financeiro desconhecido. PASS exige traces nativos e bytes de canários, não finalMessage. Prova privada `claude-<id>/confinement.json` vincula UID/ID/CLI hash/política/data/casos; factory revalida antes de cada writer. Prova antiga preservada como superseded antes do ensaio de ferramentas. FAIL indica estágio controlado e mantém escrita bloqueada; não restaurar prova antiga/forjar JSON para contornar. Não salvar raw auth/prompts/events.

Lock `.claude-confinement-preflight.lock` é exclusivo no root e removido após término conhecido. Crash pode conservá-lo: verificar fisicamente árvore/parada e preservar checkpoint antes de intervenção manual; não limpar por idade. Fixtures/saída sanitizada são evidência operacional, preservar antes de limpeza autorizada. Só após prova/aceites instalar/ativar serviço em janela autorizada; implantação não faz parte deste comando. [Relatório e limites](../../09-entregas/2026/runtime/2026-10-03-FAC-012AA-confinamento-claude.md).
