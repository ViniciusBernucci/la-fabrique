> Leitura: [← Anterior](01-deploy.md) · [Índice didático](../02-INDEX.md) · [Próximo →](03-observabilidade.md)

> Conciliação DOC-MV-001: capítulo do pacote preserva direção proposta. Estado implementado em a2cc5e0: Consumer real gated, journal/result/artifact/checkpoint/complete e recovery finalization existem. Pausa global default true e exclusão PostgreSQL não são lock BullMQ. Writer unknown bloqueia; confirmar stop antes de outro. Eficácia de serviço NÃO VERIFICADA nesta migração.

# Execução, handoff e recuperação

Data: 2026-10-06 (America/Sao_Paulo). Status da arquitetura e controles: **PLANEJADO**. Implementação e eficácia: **NÃO VERIFICADAS**. Este documento especifica trabalho futuro; não comprova instalação, configuração ou execução.

## Fluxo por tentativa

1. Usuário autenticado cria ticket em projeto autorizado; controle resolve tenant e instalação elegível, versão de política e limites.
2. Outbox transacional registra dispatch. Worker autenticado faz claim idempotente; Executor revalida toda cadeia e grant atual.
3. Context Builder seleciona somente snapshot/requisitos desse projeto, exclui segredos e registra hashes/omissões/truncamentos. JobPackage não contém DB DSN, token do worker, credential_ref sensível ou endpoints gerais.
4. Adquirir writer lock e fencing; admission confirma recursos e saúde dos controles. Criar sandbox limpa, verificar perfil, rede/mounts e canários; só então READY.
5. Criar sessão isolada no Provider Runtime escolhido; resolver auth por instalação autorizada. Emitir capability de ferramentas ligada à attempt; modelo vê somente contexto do projeto e catálogo estreito.
6. Broker valida cada operação e executa apenas no sandbox. Supervisor aplica deadline/cgroup/lease; eventos limitados não mudam estado sem validação.
7. Após execução, encerrar árvore e revogar capabilities; validar diff/artefatos após quiescência, executar checks em ambiente restrito e registrar revisão exata.
8. Reviewer é nova sessão/sandbox sobre essa revisão, RO para código. Correções geram nova revisão e invalidam aceite/testes pertinentes.
9. Documentação e gate; AWAITING_HUMAN até aceite real. Coleta imutável por hash e autoria, depois destruir sandbox e dados temporários.

## Estados e ownership

Preservar DRAFT → READY → WAITING_WORKER → RUNNING → VALIDATING → REVIEW → DOCS → AWAITING_HUMAN → DONE, com WAITING_PROVIDER, PAUSED_LIMIT, PAUSED_RESOURCE, BLOCKED_RECOVERY, AUTH_REQUIRED, FAILED, CANCELLED. Introduzir reason codes POLICY_UNAVAILABLE, UNSUPPORTED_ISOLATION e SECURITY_DENIED sem inventar que já existem em código. Lifecycle sandbox tem estados próprios e não substitui estados do ticket.

Worker protocol interno mantém register/heartbeat/claim/events/checkpoint/complete com schema_version, identidade de serviço, tenant/project/run/attempt, sequence/event_id, version e fencing. Identidade worker tem escopo operacional limitado; tokens do worker não chegam ao Provider Runtime, broker ou sandbox. Evento adulterado não altera propriedade, budget, política, instalação ou aprovações. Reconciliar RESULT_UNKNOWN antes de repetir chamada externa.

## Handoff e recovery

Bloquear novas operações, cancelar árvore quando pause não for suportado, confirmar quiescência, coletar base/code SHA, patch e untracked sanitizados, hashes/checks e pendências. Checkpoint guarda escopo tenant/project/revisão e política. Criar novo attempt/sandbox/session e capability, nunca reutilizar home/transcript completo/auth. Revalidar membership, instalação e grant antes de restaurar. Novo provider é do mesmo tenant.

Heartbeat 15s/lease 90s são propostas herdadas, a ensaiar com margem de shutdown; capability curta não ultrapassa lease. Worker perde lease: auto-interrompe antes do vencimento. Controle perde prova de término: BLOCKED_RECOVERY, não iniciar outro writer. Fencing impede aceitar eventos antigos, mas não mata processo local. Reboot reconcilia cgroups, containers, claims e artefatos órfãos antes de liberar execução.

## Retenção e backup

Propostas herdadas: artefatos 30 dias, auditoria resumida 90 dias, backup diário criptografado fora da VPS com sete diários/quatro semanais; RPO 24h/RTO 4h são metas NÃO VERIFICADAS. Ownership e autorização permanecem no backup/restauração. Sem dados reais no ensaio. Restore em ambiente privado de teste, verificar cadeia/hashes e autorização; sessões oficiais reautenticadas conforme mecanismo suportado. Limpeza só após coleta comprovada e ownership confirmado; nenhum prune geral de volumes persistentes.

## Operação financeira

Subscription-first, clientes oficiais, API/fallback/extra usage/autorecharge desligados; chave API herdada bloqueada. Não escolher credencial de outro tenant por falta de quota. WAITING_PROVIDER preserva checkpoint. Uso nullable com fonte/observed_at; custo fixo não é custo API por token. Não contratar VPS, plano ou créditos neste pacote.


## Origem desta edição

[Versão original preservada](../99-historico/originais/output/le-fabrique-multitenant/documentacoes/operacao/README.md). Migração editorial de paths em 2026-10-06; conteúdo de engenharia continua proposto.

## Conteúdo local mesclado

[Referência v2.3 integral com detalhes/revisões/limites](operacao/00-referencia-v2.3.md) e [AS-IS](../02-arquitetura/01-as-is.md).
