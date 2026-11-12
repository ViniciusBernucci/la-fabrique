# FAC-010 — Segundo provider e handoff automatico

Status: IN_PROGRESS — dividido em FAC-010A/B/C

## Objetivo

Validar um segundo cliente oficial em modo subscription-only, implementar seu adapter e rotear handoff por checkpoint/snapshot depois de quiescencia confirmada, sem transferir sessao privada nem habilitar API ou cobranca extra.

Execucao incremental: FAC-010A verifica instalacoes pelo worker; FAC-010B fornece login interativo efemero iniciado no painel; FAC-010C valida o segundo adapter e handoff. A divisao reduz o risco R3 sem alterar os criterios finais deste ticket.

## Atual e esperado

Codex e o unico provider elegivel aceito. Na VPS, Claude Code `2.1.285` e Antigravity CLI `1.2.14` estao instalados, mas ambos estao deslogados: `claude auth status --json` informou `loggedIn=false`, e `agy models` recusou a listagem pedindo login. Nenhum nome de variavel de API Anthropic/Google/Gemini/OpenAI foi encontrado no ambiente. O ticket aguarda autenticacao oficial e verificacao humana de plano/extras antes de qualquer adapter.

## Escopo permitido e proibido

Permitido apos o gate: preflight sintetico do cliente autenticado; adapter em `packages/runtime`; contratos Zod; router/handoff no worker; restauracao do snapshot FAC-007; fencing/checkpoint FAC-008; testes com fixtures sem chamadas reais; documentacao afetada.

Proibido: usar Anthropic Console/API, chave API, Bedrock, Vertex, gateway ou SDK como atalho; copiar OAuth/token; alterar login automaticamente; habilitar extra usage, creditos, autorecharge ou fallback pago; escolher modelo nao observado; iniciar segundo writer sem quiescencia; merge/deploy; piloto real.

## Criterios de aceite verificaveis

1. Um segundo cliente oficial autenticado por assinatura e comprovado sob a identidade real do worker, com versao, auth, modelos acessiveis e gate financeiro registrados.
2. Adapter valida request/eventos/resultados, usa argv/stdin sem shell, limita logs/tempo e confirma cancelamento.
3. Rate limit/auth indisponivel produz checkpoint e `WAITING_PROVIDER`; nao ativa API nem troca silenciosamente modelo/plano.
4. Handoff exige processo anterior parado, snapshot verificavel restaurado na mesma base e novo fencing token; evento antigo nao escreve.
5. Outro provider recebe somente estado externo sanitizado, nunca sessao privada do anterior.
6. Limites de duas tentativas e dois handoffs sao aplicados; sem provider elegivel o fluxo aguarda.
7. Testes simulam cota, auth e handoff sem consumir assinatura/API.

## Baseline e comandos de checks

- Base aceita: `485ae88a772a499f6fc8034f5ea3df0e80c32ceb`.
- Branch/worktree: `feat/fac-010-second-provider-handoff` em `/home/vinicius/le-fabrique-fac-010`.
- Clientes observados: Claude Code `2.1.285`; Antigravity CLI `1.2.14`; Codex CLI `0.159.2` continua provider primario.
- Checks futuros: `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`, preflight sintetico e `git diff --check`.

## Risco e orçamento

Risco R3. Um executor, um writer, no maximo duas tentativas, duas trocas e 30 minutos por janela. Budget API zero, subscription-only, fallback e extras pagos desligados. O preflight real so pode ocorrer depois da verificacao humana da conta.

## Dependencias

FAC-005, FAC-008 e FAC-009 aceitos; FAC-011 precisa ser aceito para fornecer a configuracao administrativa. Bloqueio externo atual: nenhum segundo provider autenticado/elegivel. O responsavel nao fornecera configuracao fora do software; o onboarding oficial precisa ser iniciado pelo Centro de Configuracoes e executado pelo worker confiavel em incremento posterior, sem segredo no browser.

## Entregaveis e documentacao afetada

Quando desbloqueado: adapter, router/handoff, contratos/testes, relato datado, READMEs de runtime/handoff/operacao/economia, indice, changelog, backlog e lesson pertinente. O checkpoint atual documenta somente o gate, sem afirmar implementacao.

## Evidencias e aceite

Preflight de 2026-10-01 em `documentacoes/runtime/2026-10-01-FAC-010-preflight-segundo-provider.md`. Retomar depois do aceite FAC-011 e de um fluxo de onboarding no software que permita ao responsavel escolher conta/plano sem entregar credenciais nesta conversa. O worker entao executa login oficial interativo e registra somente estado/evidencia sanitizada. DONE exige revisao funcional exata e aceite humano posterior.
