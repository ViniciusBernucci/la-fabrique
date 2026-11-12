# FAC-009 — Developer, checks e revisao

Status: DONE

Revisao aceita: `02819fb847e303f1a823cc2784a7326ec5e696f9`, em 2026-10-01. O commit documental posterior apenas registra o aceite da revisao apresentada.

## Objetivo

Compor os componentes aceitos de workspace, contexto, guard, runtime, checks isolados e snapshot em um coordenador executavel no worker, com sessoes separadas para Developer e Reviewer, baseline distinguivel de regressao e no maximo duas correcoes.

## Atual e esperado

Atualmente o worker executa apenas o probe sintetico de claim/checkpoint/complete. Runtime Codex, Context Builder, RuntimeGuard, WorkspaceManager, SandboxRunner e SnapshotManager existem como componentes isolados. Ao final, um fluxo sintetico deve preparar worktree, registrar baseline, executar Developer, rodar checks confiaveis, preservar snapshot, executar Reviewer somente leitura e produzir resultado validado sem considerar exit code ou mensagem do modelo como aceite suficiente.

## Escopo permitido e proibido

Permitido: contratos Zod em `packages/contracts`; coordenador e testes em `apps/worker`; pequenos ajustes de exportacao em `packages/runtime`; fixtures locais; documentacao de runtime, operacao, planejamento, contexto, handoff e lessons.

Proibido: executar cliente real nos testes; aceitar comandos sugeridos pelo modelo; habilitar API, extra usage, credito ou fallback pago; segundo provider; merge/deploy; migracao destrutiva; piloto externo; resolver clone/autenticacao de repositorio ainda nao contratado; ligar o fluxo real ao BullMQ sem perfil de execucao local confiavel.

## Criterios de aceite verificaveis

1. Request, resultado, checks e veredito do Reviewer possuem validacao runtime e limites explicitos.
2. O coordenador cria worktree na revisao exata, constroi contexto deterministico e autoriza cada chamada pelo RuntimeGuard subscription-only.
3. Developer usa permissao `WORKSPACE_WRITE`; Reviewer usa nova execucao e `READ_ONLY`; resposta de review invalida ou nao estruturada falha fechada.
4. Checks sao argv confiaveis do request, executados sequencialmente pelo SandboxRunner; nenhum output de IA vira comando.
5. Baseline e pos-alteracao ficam separados; falha existente e marcada como preexistente, e regressao nova impede aprovacao.
6. Cada rodada concluida gera snapshot verificavel antes do Reviewer; nova correcao so ocorre depois de processo encerrado e snapshot persistido.
7. Sucesso exige runtime concluido, checks sem regressao e Reviewer `APPROVE`; caso contrario corrige ate o menor limite entre guard e duas rodadas, depois pausa com diagnostico.
8. Testes usam adapter, sandbox e repositorio sinteticos; nenhuma assinatura/API e consumida.

## Baseline e comandos de checks

- Base aceita: `59ba387937467881fee1835a7d90c4b018c45805`.
- Branch/worktree: `feat/fac-009-developer-checks-review` em `/home/vinicius/le-fabrique-fac-009`.
- Checks: `npm run lint`, `npm run typecheck`, `npm test`, `npm run build` e testes direcionados do worker/contratos.
- Provider elegivel para a integracao futura: Codex CLI `0.159.2` com autenticacao ChatGPT aceita no FAC-002 e adapter aceito no FAC-005. Nesta entrega, chamadas reais ficam proibidas; fixtures exercitam a porta do adapter.

## Risco e orçamento

Risco R3 pela composicao de escrita, checks e review. Um writer, um fluxo ativo, janela maxima contratual de 30 minutos, no maximo duas correcoes e dois handoffs. RuntimeGuard exige assinatura, budget API zero, fallback e extras desligados. Dados e repositorios exclusivamente sinteticos.

## Dependencias

FAC-005, FAC-006, FAC-007 e FAC-008 aceitos. O piloto externo continua `DEFERRED` e nao bloqueia a fixture interna.

## Entregaveis e documentacao afetada

Contratos compartilhados, coordenador no worker, testes de sucesso/review invalido/regressao/limite, relato datado, READMEs de runtime/operacao/contexto/handoff, indice, changelog, backlog e lesson sem duplicacao.

## Evidencias e aceite

Registrar SHA funcional, diff sanitizado, contagens de testes, separacao baseline/regressao, snapshots e resultado das pausas. FAC-009 so muda para `DONE` apos revisao e aceite humano da revisao exata.
