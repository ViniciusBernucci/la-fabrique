# FAC-010A — Verificacao gerenciada de instalacoes

Status: DONE

## Objetivo

Permitir que o operador solicite pelo Centro de Configuracoes uma verificacao real de cada cliente oficial. A API persiste a intencao e publica job; o worker executa somente comandos de status allowlisted e grava estado/evidencia sanitizados, sem iniciar login, executar prompt ou transportar credenciais.

## Escopo

- Contratos Zod para pedido, job e resultado de verificacao.
- Persistencia PostgreSQL de verificacoes e outbox transacional/idempotente.
- Fila BullMQ separada `le-fabrique.provider-verification`.
- Worker com perfis fixos para Codex, Claude Code e Antigravity; argv explicito, shell desligado, timeout e logs limitados.
- Endpoint interno autenticado para concluir a verificacao e atualizar estado observado.
- Botao e historico resumido no painel.
- Fixtures sem chamadas reais nos testes.

## Fora do escopo

- Iniciar `codex login --device-auth`, `claude auth login` ou interface Antigravity.
- Persistir/exibir token, cookie, `auth.json`, device code ou output bruto.
- Enviar prompt, consumir cota de modelo, escolher plano ou habilitar API/extras.
- Implementar segundo adapter, handoff, GitHub OAuth, merge ou deploy.

## Criterios de aceite

1. Apenas instalacao existente pode receber verificacao; uma verificacao ativa por instalacao e deduplicada.
2. Pedido e outbox `provider.verification.requested.v1` sao gravados atomicamente.
3. Worker resolve provider para comando fixo e nunca executa `executable` arbitrario vindo do painel.
4. Saida e reduzida a estado, versao, modelos observados e mensagem limitada; output bruto/credencial nao e persistido.
5. Resultado exige identidade do worker e transicao valida; reentrega identica nao duplica efeito.
6. Update administrativo continua incapaz de forjar estado observado.
7. Painel solicita verificacao e mostra `PENDING`, `RUNNING`, `COMPLETED` ou `FAILED` sem afirmar login.
8. Lint, typecheck, testes, build e `git diff --check` passam.

## Baseline, provider e limites

- Base aceita: `f3872072b2770970814c694153c6583ad5abc057`.
- Branch/worktree: `feat/fac-010-managed-onboarding`, `/home/vinicius/le-fabrique-fac-010-onboarding`.
- Binarios observados: Codex `0.159.2`, Claude Code `2.1.285`, Antigravity `1.2.14`.
- Codex usa somente `login status`; Claude, `auth status --json`; Antigravity, `models`.
- Timeout 15 s, 64 KiB combinados, um job por vez, nenhuma inferencia e budget API zero.

## Sequenciamento

FAC-010A prepara evidencia e status. FAC-010B implementara o canal interativo efemero para device/browser login; FAC-010C implementara adapter secundario e handoff apenas depois de elegibilidade comprovada.

## Aceite

Entrega aceita explicitamente pelo responsavel em 2026-10-01 na revisao documental exata `7c8d95eecc33488c43d7bf6d6166bedd6c143341`.
