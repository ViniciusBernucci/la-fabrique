> Leitura: [← Anterior](01-direcao-proposta.md) · [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../worker/00-README.md)

> Instruções de criação em documentacoes/ ou lessons/ abaixo são histórico da fonte; novas entregas/lessons seguem exclusivamente docs/00-governanca/POLITICA-DOCUMENTACAO.md.

> Nota editorial DOC-MV-001, 2026-10-07: conteúdo integral local preservado como referência da evolução v2.3. Afirmações por incremento têm a revisão/data original; trechos sobre consumer ausente, recuperação pendente e caminhos antigos não descrevem necessariamente HEAD a2cc5e0. O [AS-IS conferido](../../02-arquitetura/01-as-is.md) e a [auditoria](../../00-governanca/08-AUDITORIA.md) delimitam o estado atual; números de testes abaixo são evidência histórica, não reexecução desta migração.

# Agent Runtime Gateway e Provider Manager

FAC-021: marca apresentada padronizada para La fabrique; rótulos de runtime/unit corrigidos, identificadores técnicos e operações preservados. Home integrada em developer/5173 conforme FAC-020A. [Relatório](../../09-entregas/2026/controle/2026-10-05-FAC-021-nome-escala-home.md).

FAC-018: Autorização oficial; implementação pronta, consentimento real e keyring Antigravity pendentes. [Relatório](../../09-entregas/2026/configuracao/2026-10-05-FAC-018-login-claude-antigravity.md).

FAC-017 remove ANSI do desafio Codex e preserva código completo/caso. [Contrato e evidências](../../09-entregas/2026/runtime/2026-10-05-FAC-017-desafio-ansi-codex.md).

FAC-016 identifica raiz privada ausente com mensagem fixa, preservando sanitização de erros desconhecidos. [Diagnóstico](../../09-entregas/2026/runtime/2026-10-05-FAC-016-diagnostico-raiz-privada.md).

FAC-015 consulta metadados model/list por identidade privada Codex, valida conta ChatGPT e não envia prompts. [Catálogo e evidências](../../09-entregas/2026/runtime/2026-10-05-FAC-015-catalogo-clientes-oficiais.md).

FAC-012AF exporta create/verify/restoreEvidenceBackup: offline, AES-256-GCM, dados bounded e arquivo privado/staging exclusivo, sem cliente/processo DB dentro do módulo. [Evidências](../../09-entregas/2026/operacao/2026-10-03-FAC-012AF-backup-restauracao.md).

FAC-012AA usa dontAsk + allow Edit absoluta por caminho, metadata deny, sem Bash/MCP/customizações; factory Claude writer exige prova privada current do UID/instalação/CLI 2.1.285/fingerprint/política, sete dias e 14 negações. `npm run preflight:claude` prepara ensaio oficial opt-in; fixtures não comprovam permissões reais e nenhuma instalação real foi desbloqueada. [Evidências](../../09-entregas/2026/runtime/2026-10-03-FAC-012AA-confinamento-claude.md).

FAC-012Y integra alternativas UI ao workflow: chamada terminal indisponível → stop → snapshot → nova worktree/restore/contexto → rota revalidada. Máximo dois handoffs sem reset de budgets; Reviewer read-only, factory segura não vira fallback. [Evidências](../../09-entregas/2026/handoff/2026-10-03-FAC-012Y-handoff-configurado.md).

FAC-012X trata indisponibilidade tipada da instalação e AUTH_REQUIRED/RATE_LIMITED/PROVIDER_BUSY em Developer/Reviewer como WAITING_PROVIDER depois de stop/snapshot. Não gasta correções repetindo IA; config/segurança/unknown não viram espera falsa. Handoff automático separado. [Evidências](../../09-entregas/2026/operacao/2026-10-03-FAC-012X-espera-provider.md).

FAC-012W amplia entrega para 8 MiB JSON/6 MiB raw e valida totais/tamanhos antes de leituras bounded; mantém hashes e journal. Excesso preservado, não aprovado. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012W-artefatos-ampliados.md).

FAC-012V liga login/status/runtime a stores privados por instalação escolhida na UI. Factory consulta rota atual, ambiente allowlisted sem credenciais do controle/API; Codex file/ChatGPT, Claude escrita bloqueada até prova granular. 380 testes/checks; preflight real não executado. [Evidências](../../09-entregas/2026/runtime/2026-10-03-FAC-012V-identidades-instalacoes.md).

FAC-012T retoma por estado externo, nunca sessão privada: baseline antes de restore, contexto depois, regressão preservada não preexistente. Restore recusa symlinks/controle/dirty/base divergente; falha não inicia cliente. Developer com falha terminal conhecida captura progresso. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012T-retomada-snapshot.md).

FAC-012R mantém término unknown sticky mesmo após callbacks saírem do Set. AbortSignal conhecido gera CANCELLED/INTERRUPTED com observações/snapshot posterior à parada; não inicia outra chamada. Falha de captura não fabrica evidência nem liberação. [Evidências](../../09-entregas/2026/operacao/2026-10-03-FAC-012R-snapshot-interrupcao.md).

FAC-012P conecta exportação de snapshot ao consumer/journal, sem chamar IA para transportar artefatos. Falha de transporte não é aprovação; resultado/artefatos permanecem preservados para recuperação. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012P-artefatos-painel.md).

FAC-012O adiciona journal ao consumer: somente resultado retornado com stop conhecido; sem prompt/workspace/credentials. Recupera finalização em redelivery sem iniciar adapter; não retoma sessão ou writer. [Evidências](../../09-entregas/2026/operacao/2026-10-03-FAC-012O-journal-resultados.md).

FAC-012N recupera apenas a conclusão após checkpoint parado persistido; replay não inicia adapter, checkout ou workflow. Retomar execução e preservar resultado ainda não persistido continuam pendentes. [Evidências](../../09-entregas/2026/operacao/2026-10-03-FAC-012N-reconciliacao-checkpoint.md).

FAC-012M registra observações por chamada (papel, instalação configurada, provider, configuração/versionamento, modelo solicitado/efetivo, status, uso e timestamps) sem prompt/session/stdout. Null permanece desconhecido. Identidade autenticada por instalação não está comprovada por esse registro. Relatório público omite caminhos e é persistido antes de checkpoint/complete; falha de persistência impede conclusão. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012M-resultados-execucao-painel.md).

## Estado atual — FAC-012L

O workflow é composto pelo consumer real com gate desabilitado por padrão. Developer/Reviewer resolvem instalação, modelo, timeout e tentativas da configuração atual a cada chamada. AbortSignal de lease/shutdown alcança clientes e checks; cancelamento aguarda término. Falha de observação de systemd nunca confirma parada. Testes Linux verificam cancelamento de descendente; clientes usam fakes. Identidade de serviço e Claude continuam sem validação operacional. Recuperação, handoff e transporte de artefatos ainda são lacunas; ver [FAC-012L](../../09-entregas/2026/operacao/2026-10-03-FAC-012L-consumer-execucao-real.md).

## Histórico e componentes

FAC-012C define escrita explícita e vazia por padrão para processos no `SandboxRunner`. FAC-012D restringe o perfil nativo do Codex: leitura geral do workspace, escrita apenas nos `allowedPaths` do snapshot imutável, `.git`/`.codex` negados e rede desligada. Paths autorizados precisam existir e não podem conter symlinks; paths vazios mantêm o workspace read-only. Reviewer usa lista vazia e `READ_ONLY`. Além de contratos/fixtures, o perfil passou smoke test real sem inferência pelo `codex sandbox` 0.159.2: escrita fora da allowlist (`EROFS`), leitura de `.git`/`.codex` (`EACCES`) e conexão local (`EPERM`) foram bloqueadas; detalhes sanitizados no relatório FAC-012D. Isso ainda não comprova acesso/autenticação sob a identidade real do serviço. Claude não aplica essa allowlist granular e não está coberto por FAC-012D.
Perfis de projeto não selecionam contas, modelos ou providers. A cada execução, o worker deve continuar resolvendo essas escolhas pela configuração atual do Centro de Configurações e comparando-as com os adapters instalados/elegíveis; FAC-012I não liga o consumer nem altera autenticação.

FAC-012C define escrita explícita e vazia por padrão para processos no `SandboxRunner`. A política não é confinamento do processo Developer CLI; ver relatório operacional e manter consumer desativado até essa lacuna ser fechada.

## Decisão obrigatória da stack - revisão 2.3
A stack da própria La fabrique está APROVADA: React + TypeScript + Vite no painel; NestJS + TypeScript na API; worker Node.js + TypeScript em processo separado; PostgreSQL; Redis + BullMQ; Docker Compose na mesma VPS. Não solicitar nova escolha ou confirmação da stack. Não iniciar a fábrica em PHP/Laravel, Angular ou .NET. Esta decisão substitui propostas anteriores.
Monorepo: `apps/web`, `apps/api`, `apps/worker`, `packages/contracts`. Contratos compartilhados precisam de validação em runtime. API não executa clientes, builds ou testes; o worker executa esses trabalhos com isolamento, limites e um writer inicial.
Ao trabalhar na própria fábrica, aplicar esta stack. A regra de preservar a stack existente aplica-se somente a projetos EXTERNOS cadastrados para desenvolvimento pela fábrica; ela não altera a stack da La fabrique. Se o repositório da fábrica contiver implementação anterior incompatível, registrar a divergência e planejar a adaptação por etapas; não apagar código existente nem reabrir a escolha tecnológica.
Versões exatas e comandos devem ser fixados conforme compatibilidade no bootstrap; isso não é uma nova decisão de stack. Repositório e funcionalidade do piloto externo permanecem pendentes quando não fornecidos.

Status: componentes ate FAC-011, FAC-010A/B/C e FAC-012A/B aceitos. Claude real permanece `WAITING_PROVIDER` para validacao operacional; o consumer autonomo ainda nao esta ligado.
## Adapters
Codex implementado em `packages/runtime`: `codex exec --json` com perfil nomeado que nega o host, libera somente runtime minimo/workspace necessario e desliga rede de comandos; prompt via stdin, execucao efemera e configuracao do usuario ignorada. Claude implementado por `claude -p --output-format stream-json`, `safe-mode`, `restricted`, allowlist de ferramentas, prompts negados e sessao nao persistida; autenticacao precisa ser assinatura explicita first-party. Antigravity permanece planejado; validar flags de saída/permissões pela versão instalada antes de assumir JSON.
Exemplos acima descrevem invocação, não autorizam executar código com privilégios. Usar spawn/execFile com array de argumentos e stdin, jamais interpolar prompt em shell. Fixar versão/binário; validar origem oficial e registrar checksums quando disponíveis.
Clientes CLI executam ferramentas autonomamente. Adaptador coleta eventos e controla processo; o host impõe recursos, mounts, rede e identidade. Não fingir que tool_requests da API continua idêntico ao loop interno do CLI.
## Preflight por instalação
Registrar OS/CPU, versão, binário, método oficial de login e plano declarado, modelos realmente acessíveis, modo não interativo, leitura das regras, sandbox, permissão de ferramentas, formato de eventos, cancelamento e comportamento de limite. Desabilitar API keys herdadas e verificar extras desligados. Rodar um ticket sintético de leitura e depois alteração pequena com checks.
Testar credenciais no serviço real: login num terminal não prova acesso pelo usuário systemd/container. Usar armazenamento oficial suportado; não copiar sessões entre máquinas como atalho. Se containerização não suporta login seguro, manter adapter bloqueado até ADR de isolamento compatível.

O FAC-002 validou `codex-cli 0.159.2` no usuario atual da VPS com autenticacao ChatGPT, JSONL e execucao efemera. O perfil aplicado nega leitura do host por padrao, libera somente runtime minimo e a raiz ativa, desliga rede de comandos e bloqueia `~/.codex/auth.json`. O script `scripts/fac-002-codex-preflight.sh` tambem recusa `OPENAI_API_KEY` e `CODEX_API_KEY`. O FAC-005 deve executar novamente o preflight sob a identidade real do servico; a prova no usuario atual nao substitui essa verificacao.
Cada capability carrega verified_at, cli_version e evidence_id. Recursos desconhecidos são unsupported; getUsage pode retornar UNKNOWN. Um provider não passa a AVAILABLE apenas porque existe no catálogo.
## Interface atual
`execute(request, eventSink)` produz eventos sanitizados e resultado validado; `getStatus()` verifica versao/auth; `getCapabilities()` retorna capacidades comprovadas; `getUsage()` retorna observacao `UNKNOWN`; `cancel(execution_id)` confirma processo encerrado. Resume permanece planejado e deve retornar unsupported ate implementacao explicita.

`ContextBuilder.build(request)` produz conteudo e manifesto deterministico com hashes e omissoes explicitas. `RuntimeGuard` aplica tentativas, tempo, trocas de provider e repeticao de falha; `sanitizeSubscriptionEnvironment` remove chaves de API herdadas antes de iniciar o cliente.

`WorkspaceManager.create` prepara worktree detached; `SandboxRunner.execute` controla unidade/namespaces/limites; `SnapshotManager.capture/restore` preserva patch binario e untracked com verificacao de integridade. O `DeveloperWorkflow` do FAC-009 coordena essas portas com o adapter: baseline, Developer, checks, snapshot e Reviewer separado. Após OPS-005, `main.ts` não registra consumidor em `le-fabrique.execution`; a fixture `processOrchestrationFixture` é usada apenas por testes, evitando transição simulada para `VALIDATING`.

FAC-012A faz o job carregar `executionSpecification`, copia estrita e autocontida do projeto, definicao e ticket no instante de READY. Envelope e snapshot precisam concordar em IDs, SHA e versoes. Isso preserva a intencao, mas nao torna URL/comando automaticamente confiavel: o worker ainda deve reconciliar checkout e allowlist antes de substituir a fixture pelo `DeveloperWorkflow`.

FAC-012B adiciona `compileWorkflowRequest`: perfil validado e injetado pelo worker precisa casar projeto/URL/SHA; comandos são comparados por nome, executável e argv exatos; fontes de contexto respeitam caminhos permitidos/proibidos. Limites e modelo nullable também vêm do perfil, sem hardcode. Esta biblioteca pura não está conectada à fila. Caminhos de escrita ainda não são impostos pelo sandbox, então execução real permanece bloqueada.

FAC-012D faz o parse runtime do snapshot recebido e encaminha somente `project.definition.allowedPaths` à execução Developer. Codex usa rules nomeadas por subpath no perfil `permissions.lefabrique`; requests read-only não elevam escrita mesmo se trouxerem uma lista. Handoff sem manifesto de paths permanece read-only. O worker/consumer real não está ligado e a configuração de escrita granular para Claude permanece pendente.
FAC-012E adiciona `ControlClient.getWorkerConfiguration()` para buscar `GET /internal/worker-settings`, conferir versão/observedAt e validar `FactoryConfiguration` no contrato compartilhado. O endpoint requer token do worker e retorna defaults inativos sem persistir se ainda não houver settings. Ainda não existe conexão entre esse método, resolução de rota e consumer de execução.

FAC-012F adiciona `ConfiguredAgentRouter.resolve(role)`, que consulta o snapshot a cada chamada, reutiliza `resolveAgentRoute`, associa apenas o adapter registrado para o provider escolhido e devolve versão/instante da configuração junto à rota. Provider, instalação indisponível, adapter ausente/desalinhado ou Reviewer configurado com escrita falham sem fallback. Essa é uma biblioteca pura; o consumer ainda não a chama e nenhum adapter foi executado.

FAC-012G passa esse router ao `DeveloperWorkflow`: cada tentativa Developer, cada Reviewer e cada rodada de correção resolve a configuração corrente, usando adapter/modelo/permissão daquela função. O guard registra o provider real da rota. Falta de rota encerra antes do adapter, e Reviewer com permissão de escrita é recusado. A composição continua sem ligação ao consumer BullMQ; perfil/checkout confiável e política operacional de trocas ainda faltam.

FAC-012H faz o launcher do `SandboxRunner` montar um rootfs temporário, bindar `/usr` somente leitura e o workspace em `/mnt`, mover o `/proc` do PID namespace e executar `pivot_root`. O old root é desanexado antes do processo, e `setpriv` remove capabilities, bloqueia ganho de privilégios e então aplica `prlimit`. O rootfs contém apenas devices mínimos e `/tmp`, `/home`, `/run`, `/var`, `/sys` temporários; `/etc/hostname`, home host, Docker socket, `/var/lib` e cgroup do host não aparecem. Workspace permanece read-only exceto mounts graváveis previamente validados. Checks precisam depender de runtime disponível em `/usr`; outras árvores de toolchain não são expostas ainda.

O preflight FAC-010 observou Claude Code deslogado e Antigravity incapaz de listar modelos sem login. Claude e candidato preferencial somente depois de login Claude App por assinatura e confirmacao de extras desligados; Console/API nao e elegivel. Ate la, roteamento e handoff permanecem planejados.

FAC-010C implementa `ClaudeAdapter`, rota por funcionario e `ProviderHandoff` como bibliotecas testadas. O handoff recebe apenas evidencia terminal minima, objetivo/aceite e snapshot; exige parada confirmada, snapshot posterior ao processo e mesma base, autoriza a troca no RuntimeGuard e restaura em nova worktree. O consumidor real permanece no probe ate existir perfil allowlisted de repositorio/comandos e provider autenticado.

FAC-010B conecta somente Codex por fluxo oficial de device code. O worker mantem o processo em memoria por ate dez minutos, limita output e publica apenas URL/codigo validados no canal efemero. Saida bem-sucedida nao basta: `codex login status` precisa confirmar autenticacao antes de `AVAILABLE`. Encerramento do worker mata o grupo de login ativo; nenhum desafio ou output bruto vira evento persistente.

O FAC-011 introduz `ProviderInstallation` e atribuicoes administrativas por funcao. Essa configuracao expressa intencao, nao evidencia: habilitar uma linha ou digitar um modelo nao muda o estado real do cliente. O worker futuro precisa detectar binario, autenticar pelo fluxo oficial, observar modelos e gravar evidencia antes de o router considerar a instalacao elegivel. Segredos nao pertencem ao contrato nem ao PostgreSQL de controle.

O Reviewer deve devolver JSON validado com `APPROVE` ou `REQUEST_CHANGES`; texto livre, falha do runtime ou schema invalido falha fechado. `APPROVE` produz somente `AWAITING_HUMAN`, nunca `DONE`.

## Contrato ampliado planejado
Request inclui run_id, attempt_id, fencing_token, provider_installation_id, model_requested nullable, workspace_id, base_sha/code_sha, context_manifest, instruction_hashes, command_profile, limites e policy_version.
Resultado inclui schema_version, status, exit_code, model_effective nullable, provider_session_id nullable, revisão, patch/artifact hashes, checks, summary, usage nullable, billing_mode, timestamps e erro normalizado. Não capturar raciocínio interno privado.
Eventos: started, progress, checkpoint, usage_observed, limit_observed, artifact_ready, process_exited e finished. A fábrica valida sequência/schema; output do agente é dado não confiável e não pode alterar orçamento/política.
## Erros normalizados
AUTH_REQUIRED: usuário autentica no fluxo oficial. RATE_LIMITED: cooldown/fallback elegível. TRANSIENT: backoff com limite. CONTEXT_TOO_LARGE: reduzir contexto e registrar mudança. TOOL_DENIED: corrigir perfil/escopo, não ampliar automaticamente. TIMEOUT: cancelar árvore, checkpoint e revisão de estado. RESULT_UNKNOWN: reconciliar antes de repetir. UNSUPPORTED: manter adapter desativado para a capacidade.
Uso e cotas têm unidade/fonte/observed_at/reset_at/confiança. Estado de quota, crédito de assinatura e tarifa API são dados distintos. Não usar endpoints privados ou scraping do site para medir cota.
## Roteamento
Filtrar por política/capacidade e disponibilidade; ordenar pela atribuicao configurada no FAC-011 e desempenho recente; adquirir lock do writer; executar. Cada funcao pode selecionar instalacao e modelo permitido, mas o router ainda deve exigir estado/evidencia elegivel. Não usar percentuais fictícios de sucesso; aprender com ensaio registrado.
Cache de roteamento não deve ignorar observações novas de limite. Provider indisponível não implica mudar modelo/conta silenciosamente. API só entra no catálogo quando habilitada explicitamente em política futura.


FAC-023: contrato de FactoryConfiguration inclui projectSkills/digitalAgents opcionais e relações por projeto. Snapshot interno valida os registros; router mantém atribuições operacionais existentes, sem executar automaticamente os novos perfis/instruções. [Relatório](../../09-entregas/2026/configuracao/2026-10-05-FAC-023-agentes-skills-projeto.md).
