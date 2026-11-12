# Checklist do backlog

Acompanhamento de todos os tickets de [03-BACKLOG.md](03-BACKLOG.md). **Fonte de verdade do estado**: atualize este arquivo ao concluir ou alterar qualquer ticket (marcar só com evidência).

Legenda: `[x]` feito e verificado · `[~]` parcial (ver nota) · `[ ]` pendente.

_Última atualização: 2026-11-12 (UTC-03:00) — E04._

## Resumo

**7 feitos · 5 parciais · 55 pendentes** de 67 tickets.


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

- [ ] **[T02-01](03-BACKLOG.md#t02-01)** — Inventariar e medir VPS existente · dep.: T01-01.
- [~] **[T02-02](03-BACKLOG.md#t02-02)** — Configurar ambientes, rede, proxy e containers · dep.: T01-03, T02-01.
  - Compose dev/prod PASS (só web publicado, rootfs RO, healthchecks). Reboot, TLS e homologação NOT_RUN; limites não medidos.
- [ ] **[T02-03](03-BACKLOG.md#t02-03)** — Automatizar CI e promoção por revisão · dep.: T01-03, T02-02, T03-01.
- [~] **[T02-04](03-BACKLOG.md#t02-04)** — Documentar e ensaiar backup e restauração · dep.: T02-02, T03-02, T04-03, T12-01.
  - Só o necessário p/ dev: backup/restore do banco PASS em dev. Chave do cofre (`VAULT_KEY`) documentada; decifrar com chave certa/errada PASS em integração. Restore ponta a ponta (dump + chave) em destino isolado, volumes e snapshot da VPS pendentes.

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

- [ ] **[T05-01](03-BACKLOG.md#t05-01)** — Cadastrar projeto em rascunho · dep.: T03-03, T04-02.
- [ ] **[T05-02](03-BACKLOG.md#t05-02)** — Configurar repositório, origem e elegibilidade · dep.: T05-01, T06-01, T06-02, T15-01.
- [ ] **[T05-03](03-BACKLOG.md#t05-03)** — Descobrir stack e associar recursos manualmente · dep.: T05-02, T07-03, T07-04, T08-04, T09-02.

## E06 — Integrações e sincronização

- [ ] **[T06-01](03-BACKLOG.md#t06-01)** — Implementar adaptador Jira · dep.: T03-02, T04-03.
- [ ] **[T06-02](03-BACKLOG.md#t06-02)** — Implementar adaptador ClickUp · dep.: T03-02, T04-03.
- [ ] **[T06-03](03-BACKLOG.md#t06-03)** — Projetar subtarefas e status sem duplicação · dep.: T06-01, T06-02, T13-04.
- [ ] **[T06-04](03-BACKLOG.md#t06-04)** — Implementar sincronização manual e preflight · dep.: T05-02, T06-01, T06-02, T13-04.
- [ ] **[T06-05](03-BACKLOG.md#t06-05)** — Tratar mudança de requisitos e troca de plataforma · dep.: T06-03, T06-04, T10-01, T12-04.

## E07 — Workflows visuais

- [ ] **[T07-01](03-BACKLOG.md#t07-01)** — Construir editor visual de workflows · dep.: T03-03.
- [ ] **[T07-02](03-BACKLOG.md#t07-02)** — Executar nós e estruturas de controle · dep.: T07-01, T03-02.
- [ ] **[T07-03](03-BACKLOG.md#t07-03)** — Publicar e congelar configuração efetiva · dep.: T07-02, T08-01, T09-02.
- [ ] **[T07-04](03-BACKLOG.md#t07-04)** — Selecionar fluxo e subfluxo por regras · dep.: T07-03.

## E08 — Catálogos e MCP

- [ ] **[T08-01](03-BACKLOG.md#t08-01)** — Gerenciar agentes e sobrescritas · dep.: T03-03, T04-03, T09-01.
- [ ] **[T08-02](03-BACKLOG.md#t08-02)** — Gerenciar skills e importações · dep.: T03-02, T04-03, T13-01.
- [ ] **[T08-03](03-BACKLOG.md#t08-03)** — Criar catálogo de ferramentas · dep.: T03-02, T13-01.
- [ ] **[T08-04](03-BACKLOG.md#t08-04)** — Cadastrar MCP, descobrir e autorizar ferramentas · dep.: T08-03, T04-03.

## E09 — Provedores de IA e orçamento

- [ ] **[T09-01](03-BACKLOG.md#t09-01)** — Cadastrar contas e adaptar clientes CLI · dep.: T04-03, T11-01.
- [ ] **[T09-02](03-BACKLOG.md#t09-02)** — Cadastrar modelos e adaptar gateways de API · dep.: T04-03, T08-03, T11-01.
- [ ] **[T09-03](03-BACKLOG.md#t09-03)** — Impor orçamento e registrar uso · dep.: T09-01, T09-02, T03-02.
- [ ] **[T09-04](03-BACKLOG.md#t09-04)** — Implementar fallback e disponibilidade · dep.: T09-03, T12-02.

## E10 — Planejamento, fila e controles

- [ ] **[T10-01](03-BACKLOG.md#t10-01)** — Planejar e versionar decomposição da feature · dep.: T06-03, T07-04, T12-02.
- [ ] **[T10-02](03-BACKLOG.md#t10-02)** — Implementar fila local por projeto · dep.: T10-01, T03-02.
- [ ] **[T10-03](03-BACKLOG.md#t10-03)** — Aplicar limite global, leases e justiça · dep.: T10-02, T11-03.
- [ ] **[T10-04](03-BACKLOG.md#t10-04)** — Implementar ativação, pausa e retomada operacional · dep.: T10-03, T12-01.

## E11 — Sandbox e processos

- [ ] **[T11-01](03-BACKLOG.md#t11-01)** — Provisionar sandbox e perfis de execução · dep.: T02-02, T03-01.
- [ ] **[T11-02](03-BACKLOG.md#t11-02)** — Executar ferramentas e skills com limites · dep.: T11-01, T08-02, T08-03.
- [ ] **[T11-03](03-BACKLOG.md#t11-03)** — Supervisionar processos, timeout e inatividade · dep.: T11-01, T13-01.

## E12 — Contexto e recuperação

- [ ] **[T12-01](03-BACKLOG.md#t12-01)** — Persistir e restaurar checkpoints íntegros · dep.: T03-02, T11-03.
- [ ] **[T12-02](03-BACKLOG.md#t12-02)** — Compor contexto seletivo e renovar sessão · dep.: T07-02, T12-01.
- [ ] **[T12-03](03-BACKLOG.md#t12-03)** — Recuperar execução após falha ou reboot · dep.: T10-03, T12-01, T13-04.
- [ ] **[T12-04](03-BACKLOG.md#t12-04)** — Implementar retomada seletiva e reconciliação de insumos · dep.: T07-03, T06-04, T12-01, T13-04.

## E13 — Autorizações e efeitos

- [ ] **[T13-01](03-BACKLOG.md#t13-01)** — Aplicar política de risco no executor · dep.: T03-02, T04-03.
- [ ] **[T13-02](03-BACKLOG.md#t13-02)** — Implementar pedidos e decisões humanas · dep.: T13-01, T03-03.
- [ ] **[T13-03](03-BACKLOG.md#t13-03)** — Controlar escopo, expiração e revogação · dep.: T13-02, T11-03, T12-01.
- [ ] **[T13-04](03-BACKLOG.md#t13-04)** — Registrar e reconciliar efeitos externos · dep.: T03-02, T13-01.
- [ ] **[T13-05](03-BACKLOG.md#t13-05)** — Compensar efeitos parciais por risco · dep.: T13-03, T13-04.

## E14 — Qualidade e revisão

- [ ] **[T14-01](03-BACKLOG.md#t14-01)** — Executar baseline, critérios e checks · dep.: T05-03, T11-02.
- [ ] **[T14-02](03-BACKLOG.md#t14-02)** — Classificar e reavaliar risco · dep.: T10-01, T14-01.
- [ ] **[T14-03](03-BACKLOG.md#t14-03)** — Executar revisão independente e correções · dep.: T07-03, T09-04, T14-02, T12-03.
- [ ] **[T14-04](03-BACKLOG.md#t14-04)** — Criar testes e rodar E2E por impacto · dep.: T14-01, T14-02.

## E15 — GitHub, documentação e memória

- [ ] **[T15-01](03-BACKLOG.md#t15-01)** — Integrar GitHub, branches, commits e push · dep.: T04-03, T11-01, T13-04.
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
