# Checklist do backlog

Acompanhamento de todos os tickets de [03-BACKLOG.md](03-BACKLOG.md). **Fonte de verdade do estado**: atualize este arquivo ao concluir ou alterar qualquer ticket (marcar só com evidência).

Legenda: `[x]` feito e verificado · `[~]` parcial (ver nota) · `[ ]` pendente.

_Última atualização: 2026-11-12 (UTC-03:00) — E10 (T10-01)._

## Resumo

**19 feitos · 26 parciais · 22 pendentes** de 67 tickets.


## E01 — Bootstrap e monorepo

- [x] **[T01-01](03-BACKLOG.md#t01-01)** — Registrar arquitetura inicial e ADRs · dep.: Nenhuma.
  - ADR-0001 a 0004 em docs/adr. Benchmark da VPS (T02-01) ainda NOT_RUN.
- [~] **[T01-02](03-BACKLOG.md#t01-02)** — Criar e configurar repositório GitHub da fábrica · dep.: T01-01.
  - Remoto existe; PR template e guia criados. Proteção de main, checks e PR de teste NOT_RUN (gh sem login; ver docs/operacao/github.md).
- [~] **[T01-03](03-BACKLOG.md#t01-03)** — Criar layout e ferramentas de desenvolvimento · dep.: T01-02.
  - Layout, Makefile, locks, .env.example e check de segredos PASS. Segundo clone limpo NOT_RUN.
- [x] **[T01-04](03-BACKLOG.md#t01-04)** — Publicar guias e padrões no repositório · dep.: T01-03.
  - README, CLAUDE.md/AGENTS.md/ANTIGRAVITY.md, convenções e índice em docs/.

## E02 — VPS, CI e recuperação da instalação

- [~] **[T02-01](03-BACKLOG.md#t02-01)** — Inventariar e medir VPS existente · dep.: T01-01.
  - Pronto: `scripts/vps-inventory.sh` e `scripts/vps-benchmark.sh` (somente leitura/não destrutivos; sintaxe e execução parcial validadas localmente) e o registro [vps-inventario.md](../operacao/vps-inventario.md) com valores "NÃO MEDIDO" identificados. **NOT_RUN:** nenhuma medida da VPS (sem acesso).
- [~] **[T02-02](03-BACKLOG.md#t02-02)** — Configurar ambientes, rede, proxy e containers · dep.: T01-03, T02-01.
  - Compose dev/prod PASS (só web publicado, rootfs RO, healthchecks). Reboot, TLS e homologação NOT_RUN; limites não medidos.
- [~] **[T02-03](03-BACKLOG.md#t02-03)** — Automatizar CI e promoção por revisão · dep.: T01-03, T02-02, T03-01.
  - Pronto: `.github/workflows/ci.yml` (Go com integração em PostgreSQL, web, segredos/dependências, build de imagens com tag = commit, smoke e evidência por commit; sem `secrets.*`) e [ci-e-promocao.md](../operacao/ci-e-promocao.md) com promoção e rollback. **NOT_RUN:** o workflow nunca executou no GitHub (sem push/autorização); checks obrigatórios na `main` pendentes; ações fixadas por tag, não por SHA.
- [~] **[T02-04](03-BACKLOG.md#t02-04)** — Documentar e ensaiar backup e restauração · dep.: T02-02, T03-02, T04-03, T12-01.
  - Só o necessário p/ dev: backup/restore do banco PASS em dev. Chave do cofre (`VAULT_KEY`) documentada; decifrar com chave certa/errada PASS em integração. Restore ponta a ponta (dump + chave) em destino isolado, volumes e snapshot da VPS pendentes.
  - **Atualização (T02-04):** `make restore-drill` (backup → restore isolado → verificação → descarte) e `cmd/verify-restore` conferem migrations, TOTP e conexões decifráveis com a chave, hashes de checkpoints/skills/contexto; PASS em integração (íntegro passa; chave errada e adulterações são apontadas sem vazar valor) e mecânica PASS em dev (a verificação reprovou de verdade um dev sem migrations e sem administrador). **Pendente:** restore do snapshot da VPS e em produção, rotação da chave. Entrega: [E02 restauração](../entregas/2026-11-12-E02-ensaio-de-restauracao.md).

## E03 — Fundação web, API e dados

- [x] **[T03-01](03-BACKLOG.md#t03-01)** — Inicializar web, API e worker · dep.: T01-03.
  - web, API e worker sobem em dev e prod; config inválida falha com diagnóstico. CI (T02-03) fora do escopo.
- [x] **[T03-02](03-BACKLOG.md#t03-02)** — Implementar banco, migrations e transações · dep.: T03-01.
  - 12 testes com PostgreSQL real PASS (restart, duplicidade, escopo, migração, concorrência).
- [x] **[T03-03](03-BACKLOG.md#t03-03)** — Criar navegação centrada em projetos · dep.: T03-01, T04-01.
  - Navegação + 401 no servidor PASS. Login real entregue no T04-01; o acesso dev provisório foi removido (ADR-0004 superada pela ADR-0005).

## E04 — Acesso e credenciais

- [x] **[T04-01](03-BACKLOG.md#t04-01)** — Implementar login, TOTP e recuperação · dep.: T03-02.
  - PASS (PostgreSQL real): senha sozinha não abre sessão; recuperação funciona exatamente uma vez sob concorrência; TOTP anti-replay; argon2id; segredo TOTP cifrado; throttle; `make create-admin`; acesso dev removido. Revisão independente NOT_RUN. Entrega: [2026-11-12-E04](../entregas/2026-11-12-E04-acesso-e-credenciais.md).
- [x] **[T04-02](03-BACKLOG.md#t04-02)** — Endurecer sessões e ações sensíveis · dep.: T02-02, T04-01.
  - PASS: revogação (individual/outras/logout), reautenticação com janela de 5 min, CSRF (token + Origin/Sec-Fetch-Site), cookies HttpOnly/Strict/Secure, HTTPS obrigatório + HSTS (simulando o proxy), trilha sem segredos. TLS real e IP real atrás do proxy do host NOT_RUN (T02-02). Revisão independente NOT_RUN.
- [~] **[T04-03](03-BACKLOG.md#t04-03)** — Criar cofre, gestão e teste de conexões · dep.: T03-02, T04-02.
  - PASS: AES-256-GCM com chave externa; criar/listar/testar/rotacionar/revogar; rotação preserva o id e descarta o valor anterior; revogação vale no próximo uso; sentinela ausente de respostas, logs e tabelas; redatores e `Vault.Resolve`. **Parcial:** prova em *prompts* NOT_RUN (sem pipeline de prompts — E09/E10); teste remoto só GitHub/ClickUp/Jira e só contra servidores falsos (reais NOT_RUN); `cli`/`api`/`mcp` sem teste remoto; rotação da chave mestra não implementada.

## E05 — Cadastro de projeto

- [x] **[T05-01](03-BACKLOG.md#t05-01)** — Cadastrar projeto em rascunho · dep.: T03-03, T04-02.
  - PASS (PostgreSQL real + vitest): projeto novo nasce em rascunho sem vínculos, persiste por etapa e retoma na etapa certa (inclusive em outra sessão); ativação bloqueada (422) sem preenchimento válido, com pendências e ação concreta; conexão revogada ou de tipo errado vira pendência; configuração travada enquanto ativo; suspensão/retomada revalidam; trilha sem segredos; projeto A não altera B; 8 gravações concorrentes → 1 vencedor; CSRF e sessão exigidos. **Não verificado nesta versão (declarado na tela):** acesso remoto ao repositório/origem (T05-02) e recursos associados (T05-03). Revisão independente NOT_RUN. Entrega: [2026-11-12-E05](../entregas/2026-11-12-E05-cadastro-de-projeto.md).
- [~] **[T05-02](03-BACKLOG.md#t05-02)** — Configurar repositório, origem e elegibilidade · dep.: T05-01, T06-01, T06-02, T15-01.
  - PASS (PostgreSQL real + fakes de Jira): origem Jira/ClickUp com marcador, estados elegíveis, mapa de prioridade e escopo; conexão incompatível/revogada, duas origens ativas e mapeamento inválido falham com motivo; verificação remota registrada devolve preview correto (marcador + estado, sem concluídos, prioridade mapeada) sem escrever na origem; ativação exige verificação bem-sucedida e atual (invalidada por mudança de config ou rotação do segredo); sem vazamento de segredo. **Parcial:** repositório GitHub não verificado (depende de T15-01); escrita na origem só provada no primeiro uso; Jira/ClickUp reais NOT_RUN. Entrega: [E05 adendo](../entregas/2026-11-12-E05-cadastro-de-projeto.md).
- [~] **[T05-03](03-BACKLOG.md#t05-03)** — Descobrir stack e associar recursos manualmente · dep.: T05-02, T07-03, T07-04, T08-04, T09-02.
  - PASS (PostgreSQL real, `-race`): descoberta segura de manifestos enviados (sugestões com origem, gerenciador pelo lockfile, só nomes de `.env.example`, `.env` nunca lido), perfil com origem por comando e segredo só por referência (literal/revogada/inexistente recusados), recursos globais associados explicitamente com sobrescrita local que não altera o global (ferramentas só reduzidas), rascunho de workflow e conta/modelo incompatível bloqueando a ativação, projeto apto iniciando execução da versão publicada. **Parcial:** sem leitura do repositório real no GitHub (T05-02/T15-01 parciais; os manifestos vêm do chamador) e sem tela do assistente; skills/ferramentas só têm existência validada. Revisão independente NOT_RUN. Entrega: [E05 perfil e recursos](../entregas/2026-11-12-E05-perfil-e-recursos.md).

## E06 — Integrações e sincronização

- [~] **[T06-01](03-BACKLOG.md#t06-01)** — Implementar adaptador Jira · dep.: T03-02, T04-03.
  - PASS (contrato com fixtures, `-race`): itens vs subtarefas, paginação por token, transições válidas (inválida não escreve), permissão (403), rate limit (`RetryAfter`), dependência/link/subtarefa, sem vazamento de segredo/URL/corpo, só `*.atlassian.net`. **Parcial:** instância real de teste do Jira NOT_RUN. Entrega: [E06](../entregas/2026-11-12-E06-adaptadores-jira-clickup.md).
- [~] **[T06-02](03-BACKLOG.md#t06-02)** — Implementar adaptador ClickUp · dep.: T03-02, T04-03.
  - PASS (mesma suíte de contrato): pai/filho, estados e categorias, prioridade, dependências, paginação, erros e rate limit. **Parcial:** lista real do ClickUp NOT_RUN; link de PR vira comentário (repetir duplica). Entrega: [E06](../entregas/2026-11-12-E06-adaptadores-jira-clickup.md).
- [x] **[T06-03](03-BACKLOG.md#t06-03)** — Projetar subtarefas e status sem duplicação · dep.: T06-01, T06-02, T13-04.
  - PASS (PostgreSQL real, `-race`): escrita na origem pelo ledger com prova de existência por operação (marcador na subtarefa, dependência, link, estado); timeout após criar não duplica (prova confirma), ausência comprovada repete, sem prova bloqueia; estado revalida o alvo antes de repetir; Draft PR → aguardando merge e conclusão só com merge confirmado (PR aberta/sem SHA/fechada não concluem). **Limites:** Jira/ClickUp reais NOT_RUN (fixtures + destino em memória); verificador de merge do GitHub é T15-03. Revisão independente NOT_RUN. Entrega: [E06 adendo](../entregas/2026-11-12-E06-adaptadores-jira-clickup.md).
- [~] **[T06-04](03-BACKLOG.md#t06-04)** — Implementar sincronização manual e preflight · dep.: T05-02, T06-01, T06-02, T13-04.
  - PASS (PostgreSQL real, `-race`): relógio ocioso de um mês sem nenhuma leitura; mudança detectada uma vez (sync e preflight não se repetem; concorrência registra uma); importação deduplicada e paginada; saída/retorno de itens; leitura que falha não altera item; elegibilidade e prioridade reavaliadas; edição humana de prioridade nunca sobrescrita (conflito, preflight bloqueia, resolução); API manual. **Parcial:** Jira/ClickUp reais NOT_RUN; GitHub (T15-03), ligação do preflight ao agendador (T10-02) e tela pendentes. Revisão independente NOT_RUN. Entrega: [E06 sync](../entregas/2026-11-12-E06-sincronizacao-manual-e-preflight.md).
- [ ] **[T06-05](03-BACKLOG.md#t06-05)** — Tratar mudança de requisitos e troca de plataforma · dep.: T06-03, T06-04, T10-01, T12-04.

## E07 — Workflows visuais

- [x] **[T07-01](03-BACKLOG.md#t07-01)** — Construir editor visual de workflows · dep.: T03-03.
  - PASS: catálogo de 14 tipos de nó, validador com nó/campo (ciclo só via loop com limite, portas, uma conexão por saída/entrada, obrigatórios), fluxo complexo salvo e reaberto idêntico em outra sessão, rascunho inválido salvo com problemas, concorrência e CSRF, editor de canvas (arrastar, conectar, propriedades, validação ao editar). Editor em navegador real NOT_RUN; sem zoom/pan/desfazer. Revisão independente NOT_RUN. Entrega: [E07](../entregas/2026-11-12-E07-editor-de-workflows.md).
- [x] **[T07-02](03-BACKLOG.md#t07-02)** — Executar nós e estruturas de controle · dep.: T07-01, T03-02.
  - PASS (`-race`): ramos e convergência (`any`/`all`), loop limitado (limite e saída antecipada), espera durável retomada de estado JSON persistido, decisão humana, `fork` sequencial, rotas de erro, expressões, subfluxos expandidos em linha com recursão/profundidade bloqueadas, determinismo. **Limite:** executores reais (agente, skill, script, Git) NOT_RUN — são de E09/E11/E13/E15; sem persistência de execuções no banco. Revisão independente NOT_RUN. Entrega: [E07 motor](../entregas/2026-11-12-E07-motor-de-workflows.md).
- [~] **[T07-03](03-BACKLOG.md#t07-03)** — Publicar e congelar configuração efetiva · dep.: T07-02, T08-01, T09-02.
  - PASS (PostgreSQL real): versão publicada imutável (gatilho), validação de agentes/contas/modelos/ferramentas/skills/subfluxos antes de publicar, snapshot sanitizado com hash, idempotência, rascunho não executa, execução presa à versão (nova publicação ou mudança no catálogo não a altera), subfluxos fixados por versão, revalidação em runtime respeitando conta revogada. **Parcial:** motor/worker ainda não consome `StartExecution` nem revalida a cada nó; sem tela. Revisão independente NOT_RUN. Entrega: [E07 publicação](../entregas/2026-11-12-E07-publicacao-e-congelamento.md).
- [x] **[T07-04](03-BACKLOG.md#t07-04)** — Selecionar fluxo e subfluxo por regras · dep.: T07-03.
  - PASS (PostgreSQL real): regras ordenadas com condições normalizadas, primeira que casa vence (as demais ficam no motivo), padrão explícito, falha explícita sem padrão, reordenação muda o vencedor, mesma entrada ×20 = mesma escolha, rastro persistido com o motivo, simulação sem gravar, configuração atômica, destino não publicado recusado. **Limite:** atributos reais da origem (Jira/ClickUp) NOT_RUN; sem tela. Revisão independente NOT_RUN. Entrega: [E07 seleção](../entregas/2026-11-12-E07-selecao-por-regras.md).

## E08 — Catálogos e MCP

- [x] **[T08-01](03-BACKLOG.md#t08-01)** — Gerenciar agentes e sobrescritas · dep.: T03-03, T04-03, T09-01.
  - PASS (PostgreSQL real + vitest): CRUD com versão; conta compatível com o provedor (revogada/inexistente/tipo errado recusadas); sobrescritas por nó (modelo, conta, instruções, tempo, ferramentas só reduzindo) sem alterar a base; dois workflows com efetivos diferentes e catálogo íntegro; efetivo sanitizado com a origem de cada valor. **Limite:** o efetivo é do rascunho salvo (congelamento por execução é T07-03); execução real NOT_RUN. Revisão independente NOT_RUN. Entrega: [E08](../entregas/2026-11-12-E08-catalogo-de-agentes.md).
- [~] **[T08-02](03-BACKLOG.md#t08-02)** — Gerenciar skills e importações · dep.: T03-02, T04-03, T13-01.
  - PASS (PostgreSQL real, Docker real, vitest): 4 modalidades (IA, script, determinística, híbrida), importação de tar.gz/zip/SKILL.md com travessia, links e limites recusados, scripts contra a política, hash/origem/arquivos, idempotência e versão, trilha auditável sem conteúdo, nada executa na importação, execução sob política e nó de skill isolado. **Parcial:** importação por **Git UNSUPPORTED** (T15-01); IA real NOT_RUN (executor de mentira). Revisão independente NOT_RUN. Entrega: [E08 skills](../entregas/2026-11-12-E08-catalogo-de-skills.md).
- [x] **[T08-03](03-BACKLOG.md#t08-03)** — Criar catálogo de ferramentas · dep.: T03-02, T13-01.
  - PASS (Docker real + PostgreSQL real + vitest): 8 nativas e personalizadas (script e integração) com esquemas, permissões e limites; ordem existe → concedida → escopo → argumentos → política → execução; ferramenta nova não herda acesso; saída tipada; parâmetros por ambiente sem injeção; limites de saída e tempo. **Limites:** `git_*` com repositório real NOT_RUN (sem git nas imagens); symlinks/socket: resolvidos no T11-02; cadastro de personalizadas só por API. Revisão independente NOT_RUN. Entrega: [E08 ferramentas](../entregas/2026-11-12-E08-catalogo-de-ferramentas.md).
- [~] **[T08-04](03-BACKLOG.md#t08-04)** — Cadastrar MCP, descobrir e autorizar ferramentas · dep.: T08-03, T04-03.
  - PASS (PostgreSQL real + servidor MCP falso, JSON e SSE): descoberta paginada, concessão por hash pinado, ferramenta nova sem acesso, mudança de esquema bloqueia até nova aprovação, argumentos validados antes da chamada, política após conceder, trilha redigida, credencial só do cofre. **Parcial:** servidor MCP real NOT_RUN, stdio UNSUPPORTED, sem tela no painel. Revisão independente NOT_RUN. Entrega: [E08 MCP](../entregas/2026-11-12-E08-mcp.md).

## E09 — Provedores de IA e orçamento

- [~] **[T09-01](03-BACKLOG.md#t09-01)** — Cadastrar contas e adaptar clientes CLI · dep.: T04-03, T11-01.
  - PASS (Docker real, `-race`, com CLI de mentira): comando por modo autorizado (sem `bypassPermissions`/`danger-full-access`), prompt por stdin, entradas validadas, eventos normalizados, duas contas do mesmo provedor com volumes distintos e isoladas, cancelamento encerra a árvore. **Parcial:** Claude Code e Codex CLI reais NOT_RUN (exigem conta do responsável); sem fluxo de login nem imagem de sandbox com os CLIs; formatos de evento não confirmados por execução. Entrega: [E09](../entregas/2026-11-12-E09-clientes-cli.md).
- [~] **[T09-02](03-BACKLOG.md#t09-02)** — Cadastrar modelos e adaptar gateways de API · dep.: T04-03, T08-03, T11-01.
  - PASS (PostgreSQL real, Docker real, `-race`, gateway de mentira): catálogo com parâmetros validados, compatibilidade com motivos (provedor, conta, gateway, ferramentas, estado), sobreposição agente/nó verificável (`model_check`), OpenRouter e Vercel rodando edição + teste pelo catálogo de ferramentas sob política, credencial só do cofre. **Parcial:** gateways reais NOT_RUN; sem tela; laço ainda não ligado ao worker; sem streaming. Revisão independente NOT_RUN. Entrega: [E09 modelos](../entregas/2026-11-12-E09-modelos-e-gateways.md).
- [~] **[T09-03](03-BACKLOG.md#t09-03)** — Impor orçamento e registrar uso · dep.: T09-01, T09-02, T03-02.
  - PASS (PostgreSQL real, `-race`, gateway de mentira): reserva atômica (40 concorrentes ⇒ exatamente os que cabem; teto por agente e global respeitados), liquidação idempotente com overrun, liberação e expiração, falha fechada sem orçamento global ou sem preço, janelas dia/mês, cota de assinatura separada do gasto em API (o banco impede misturar), cotas só observadas com fonte e horário, reconciliação (ajusta o que faltou, mantém o maior), orçamento imposto antes de cada chamada no laço do gateway, API de orçamento e uso. **Parcial:** gateways e créditos reais NOT_RUN; sem tela; limites globais de concorrência são do T10-03. Revisão independente NOT_RUN. Entrega: [E09 orçamento](../entregas/2026-11-12-E09-orcamento-e-uso.md).
- [~] **[T09-04](03-BACKLOG.md#t09-04)** — Implementar fallback e disponibilidade · dep.: T09-03, T12-02.
  - PASS (PostgreSQL real, `-race`): quatro causas com regras de retorno próprias, sinal oficial (cota observada) + histórico com backoff, cadeia explícita sem rodízio (conta fora da cadeia nunca usada), compatibilidade e orçamento checados, troca só com checkpoint íntegro e quiescência, esgotamento com espera (vaga liberada só com checkpoint seguro) enquanto outra execução avança, trilha auditável. **Parcial:** sem ligação ao worker/executor e sem classificação dos erros reais de Codex/Claude/OpenRouter/Vercel (NOT_RUN), sem rotas/tela; quiescência declarada pelo chamador. Revisão independente NOT_RUN. Entrega: [E09 fallback](../entregas/2026-11-12-E09-fallback-e-disponibilidade.md).

## E10 — Planejamento, fila e controles

- [~] **[T10-01](03-BACKLOG.md#t10-01)** — Planejar e versionar decomposição da feature · dep.: T06-03, T07-04, T12-02.
  - PASS (unitário + PostgreSQL real, `-race`): contrato do plano com critérios verificáveis (comando, teste, arquivo, revisão com evidência), risco e DAG; recusa de ciclo (mostrado), referência ausente, tarefa sem critério e critério inverificável; versão vigente preservada em plano inválido ou conflitante (tarefa já em execução não é alterada nem removida; rejeições ficam registradas); mesmo conteúdo sem versão nova, também sob concorrência; identidade estável da tarefa entre versões; publicação de uma subtarefa por tarefa e uma relação por dependência pelo ledger sem duplicar após resposta perdida nem no retry; API de proposta/leitura. **Parcial:** o agente planejador (produtor do plano) e a leitura de código/contexto não existem; `Publish` sem rota nem ligação ao scheduler (T10-02); Jira/ClickUp reais NOT_RUN; sem tela. Revisão independente NOT_RUN. Entrega: [E10 planejamento](../entregas/2026-11-12-E10-planejamento.md).
- [ ] **[T10-02](03-BACKLOG.md#t10-02)** — Implementar fila local por projeto · dep.: T10-01, T03-02.
- [ ] **[T10-03](03-BACKLOG.md#t10-03)** — Aplicar limite global, leases e justiça · dep.: T10-02, T11-03.
- [ ] **[T10-04](03-BACKLOG.md#t10-04)** — Implementar ativação, pausa e retomada operacional · dep.: T10-03, T12-01.

## E11 — Sandbox e processos

- [~] **[T11-01](03-BACKLOG.md#t11-01)** — Provisionar sandbox e perfis de execução · dep.: T02-02, T03-01.
  - PASS (Docker real, `-race`): container+volume+rede por feature, usuário sem privilégios, rootfs RO, sem capabilities/socket/montagens do host, quotas de CPU/memória/processos/disco, A não vê B (arquivos, processos, rede), seed+build sem rede, destruição idempotente e sem sobras. **Parcial:** clone remoto com credencial e rede com saída controlada não implementados; isolamento do host só provado em Docker Desktop (Linux NOT_RUN); worker ainda sem acesso ao runtime. Entrega: [E11](../entregas/2026-11-12-E11-sandbox.md).
- [~] **[T11-02](03-BACKLOG.md#t11-02)** — Executar ferramentas e skills com limites · dep.: T11-01, T08-02, T08-03.
  - PASS (Docker real, `-race`, CLI e gateway de mentira): ferramentas de arquivo confinadas por resolução de links (16 fugas recusadas, links internos funcionam), traversal e socket do runtime recusados (SafePath, política e ausência no container), modo API pelo `Registry` e modo CLI no container, chamadas brokeradas exigem identidade e registram o ator. **Parcial:** CLIs e gateways reais NOT_RUN; `git_*` em repositório real NOT_RUN; scripts não são confinados por caminho; corrida entre resolução e uso só afeta o próprio sandbox. Revisão independente NOT_RUN. Entrega: [E11 ferramentas](../entregas/2026-11-12-E11-execucao-de-ferramentas.md).
- [x] **[T11-03](03-BACKLOG.md#t11-03)** — Supervisionar processos, timeout e inatividade · dep.: T11-01, T13-01.
  - PASS (Docker real, `-race`): prazo absoluto e de inatividade independentes; atividade = saída, CPU ou I/O da árvore (processo silencioso que trabalha continua); timeout encerra a árvore (TERM→KILL) com parada confirmada por amostra; estado incerto → quarentena do sandbox (Exec/Start recusados) até `VerifyQuiescent`; relatório auditável. **Limites:** processo que só espera (sem CPU/I/O/saída) é tratado como travado; sem ligação com o worker/checkpoint (E12). Revisão independente NOT_RUN. Entrega: [E11 adendo](../entregas/2026-11-12-E11-sandbox.md).

## E12 — Contexto e recuperação

- [~] **[T12-01](03-BACKLOG.md#t12-01)** — Persistir e restaurar checkpoints íntegros · dep.: T03-02, T11-03.
  - PASS (PostgreSQL e Docker reais, `-race`): captura de arquivos/links/modos com hash, gravação atômica (queda em 3 pontos mantém o anterior, sem órfãos), restauração em container novo com `sha256sum` idêntico e divergência detectada, parada só com sandbox quieto e liberação depois do checkpoint durável, corrupção detectada na leitura, escopo por projeto e poda. **Parcial:** `git` real NOT_RUN (imagens sem git; HEAD/ramo/patch provados com git de mentira); sem API/tela nem ligação ao motor/worker; objetos em `bytea` (≤128 MiB por checkpoint). Revisão independente NOT_RUN. Entrega: [E12 checkpoints](../entregas/2026-11-12-E12-checkpoints.md).
- [~] **[T12-02](03-BACKLOG.md#t12-02)** — Compor contexto seletivo e renovar sessão · dep.: T07-02, T12-01.
  - PASS (PostgreSQL real, `-race`): contexto determinístico por contrato com manifesto (hash, tokens, omissões, redação), obrigatória nunca descartada em silêncio, segredos removidos antes do hash, decisão por métrica oficial ou estimativa com margem (método gravado), sessão só sobre checkpoint íntegro da mesma execução, troca de modelo só autorizada, retomada que recusa pacote ou checkpoint adulterado, sequência sem corrida. **Parcial:** sem ligação ao worker (nada decide renovar nem cria o checkpoint do marco), memória real (T15-05) não alimenta o pacote, sem rotas/tela. Revisão independente NOT_RUN. Entrega: [E12 contexto](../entregas/2026-11-12-E12-contexto-e-sessao.md).
- [ ] **[T12-03](03-BACKLOG.md#t12-03)** — Recuperar execução após falha ou reboot · dep.: T10-03, T12-01, T13-04.
- [ ] **[T12-04](03-BACKLOG.md#t12-04)** — Implementar retomada seletiva e reconciliação de insumos · dep.: T07-03, T06-04, T12-01, T13-04.

## E13 — Autorizações e efeitos

- [x] **[T13-01](03-BACKLOG.md#t13-01)** — Aplicar política de risco no executor · dep.: T03-02, T04-03.
  - PASS (`-race` + PostgreSQL real): baixo risco permitido, sensível encaminhado (`ErrApprovalRequired`) e proibido recusado em scripts, API e MCP (+ git, conectores, CLI), o mais restritivo vence em qualquer ordem, padrão nega, destino interno sempre proibido, anotação MCP não libera, trilha redigida sem segredo e ação não roda sem trilha. **Limites:** política padrão não calibrada; não ligada ao runtime (sem consumidor ainda); aprovação/Central é T13-02; DNS/redirecionamento NOT_RUN. Revisão independente NOT_RUN. Entrega: [E13](../entregas/2026-11-12-E13-politica-de-risco.md).
- [x] **[T13-02](03-BACKLOG.md#t13-02)** — Implementar pedidos e decisões humanas · dep.: T13-01, T03-03.
  - PASS (PostgreSQL real + vitest): pedido com problema/alternativas/impacto/evidência redigida; resolução transacional que só altera a execução do pedido (eventos + retomada); repetição idempotente, opção diferente → conflito, concorrência com um único sinal; execução obsoleta → `stale` e nada sinalizado; operação sensível exige reautenticação; Central no painel. **Limites:** nenhum fluxo real cria pedidos ainda e o escalonador não consome `queued` (E10/E12). Revisão independente NOT_RUN. Entrega: [E13 decisões](../entregas/2026-11-12-E13-decisoes-humanas.md).
- [x] **[T13-03](03-BACKLOG.md#t13-03)** — Controlar escopo, expiração e revogação · dep.: T13-02, T11-03, T12-01.
  - PASS (PostgreSQL e Docker reais, `-race`): autorização presa à ação canônica (ferramenta, operação, recurso, destino e hash dos parâmetros), prazo e usos; consumo atômico (corrida pelo último uso autoriza exatamente um, repetida 6×); divergência, prazo vencido, esgotamento e revogação recusam sem gastar uso e ficam na trilha; revogação imediata e, com uso em andamento, parada segura com checkpoint (sandbox real); falha da parada é explícita. **Limites:** não ligada ainda à API/worker (sem rotas, tela nem `Halter` do runtime); uso contado ao iniciar. Revisão independente NOT_RUN. Entrega: [E13 autorizações](../entregas/2026-11-12-E13-autorizacoes.md).
- [x] **[T13-04](03-BACKLOG.md#t13-04)** — Registrar e reconciliar efeitos externos · dep.: T03-02, T13-01.
  - PASS (PostgreSQL real, `-race`): intenção gravada antes do envio e pedido redigido; timeout após efeito confirmado não duplica (prova confirma); ausência comprovada permite repetir (limite de tentativas); recusa definitiva não verifica; inconclusivo (ou falha ao verificar) bloqueia a repetição e abre pedido na Central com evidência, e a decisão humana fecha ou libera o efeito; concorrência com um único envio; reconciliação de envio abandonado. **Limites:** adaptadores reais de prova (Jira/ClickUp/GitHub) são do T06-03/E15; compensação não implementada. Revisão independente NOT_RUN. Entrega: [E13 ledger](../entregas/2026-11-12-E13-ledger-de-efeitos.md).
- [~] **[T13-05](03-BACKLOG.md#t13-05)** — Compensar efeitos parciais por risco · dep.: T13-03, T13-04.
  - PASS (PostgreSQL real, `-race`, destino de mentira): compensação de baixo risco pelo ledger (idempotente, com prova, resposta perdida reconciliada), sensível só com autorização presa à compensação, irreversível/desconhecido preservado e recusado, tudo auditável; falha conserva o efeito e bloqueia operações agravantes da execução (leitura e compensação seguem), com retomada e resolução humana. **Parcial:** adaptadores reais de desfazer (Jira/ClickUp/GitHub) não existem; `CheckAggravating` não está ligado ao executor; sem rotas/tela. Revisão independente NOT_RUN. Entrega: [E13 compensação](../entregas/2026-11-12-E13-compensacao.md).

## E14 — Qualidade e revisão

- [ ] **[T14-01](03-BACKLOG.md#t14-01)** — Executar baseline, critérios e checks · dep.: T05-03, T11-02.
- [ ] **[T14-02](03-BACKLOG.md#t14-02)** — Classificar e reavaliar risco · dep.: T10-01, T14-01.
- [ ] **[T14-03](03-BACKLOG.md#t14-03)** — Executar revisão independente e correções · dep.: T07-03, T09-04, T14-02, T12-03.
- [ ] **[T14-04](03-BACKLOG.md#t14-04)** — Criar testes e rodar E2E por impacto · dep.: T14-01, T14-02.

## E15 — GitHub, documentação e memória

- [~] **[T15-01](03-BACKLOG.md#t15-01)** — Integrar GitHub, branches, commits e push · dep.: T04-03, T11-01, T13-04.
  - PASS (`git` real contra repositório local + PostgreSQL, `-race`; API do GitHub de mentira): conexão validada por clone real, commit por tarefa com checks aprovados e trailers, montado por `git fast-import` sem extrair arquivo no host, push restrito à branch da feature (nunca a base), prova por ref (timeout após o efeito reconcilia sem duplicar; antes do efeito repete com segurança; divergência bloqueia com evidência; ref à frente confirma), `.github/` exige aprovação, credencial só por ambiente do processo e redigida. **Parcial:** GitHub real NOT_RUN (permissão de push, proteção do ramo base e recusa de push direto só numa conta real); imagem do worker sem `git`; sem rotas/tela, PR e merge (T15-02/T15-03). Revisão independente NOT_RUN. Entrega: [E15 GitHub](../entregas/2026-11-12-E15-github-branch-commit-push.md).
- [ ] **[T15-02](03-BACKLOG.md#t15-02)** — Gerar documentação viva e publicar Draft PR · dep.: T15-01, T14-03, T14-04.
- [ ] **[T15-03](03-BACKLOG.md#t15-03)** — Confirmar merge e propagar conclusão · dep.: T15-02, T06-03, T06-04, T10-02.
- [ ] **[T15-04](03-BACKLOG.md#t15-04)** — Corrigir revisão externa e tratar PR fechada · dep.: T15-03, T13-02, T10-02.
- [ ] **[T15-05](03-BACKLOG.md#t15-05)** — Manter memória técnica pendente e oficial · dep.: T15-03, T12-02.

## E16 — Operação, uso e retenção

- [ ] **[T16-01](03-BACKLOG.md#t16-01)** — Exibir status, controle e logs em tempo real · dep.: T03-03, T10-04, T12-03.
- [ ] **[T16-02](03-BACKLOG.md#t16-02)** — Consolidar Central e auditoria operacional · dep.: T13-03, T13-05, T15-04, T16-01.
- [ ] **[T16-03](03-BACKLOG.md#t16-03)** — Exibir consumo, orçamento e provedores · dep.: T09-03, T09-04, T03-03.
- [ ] **[T16-04](03-BACKLOG.md#t16-04)** — Automatizar retenção com integridade · dep.: T12-01, T16-01.

## E17 — Piloto e aceite

- [ ] **[T17-01](03-BACKLOG.md#t17-01)** — Preparar projeto real e matriz de evidências · dep.: T01-04, T02-03, T05-03.
- [ ] **[T17-02](03-BACKLOG.md#t17-02)** — Executar cadastro do zero e feature ponta a ponta · dep.: T17-01, T15-05, T16-02, T16-03.
- [ ] **[T17-03](03-BACKLOG.md#t17-03)** — Executar ensaios de falha, segurança e integração · dep.: T17-01, T02-04, T06-05, T09-04, T12-04, T13-05, T15-04, T16-04.
- [ ] **[T17-04](03-BACKLOG.md#t17-04)** — Fechar aceite integral do MVP · dep.: T17-02, T17-03.
