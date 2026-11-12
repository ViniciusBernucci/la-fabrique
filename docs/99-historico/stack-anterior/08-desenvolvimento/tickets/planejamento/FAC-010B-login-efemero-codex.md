# FAC-010B — Login efemero do Codex

Status: DONE

## Objetivo

Permitir que o administrador inicie no Centro de Configuracoes o login oficial do Codex por device code. A API persiste apenas o ciclo de vida da solicitacao; o worker executa o cliente oficial e entrega URL/codigo ao painel por armazenamento Redis temporario, sem persistir segredo ou output bruto.

## Escopo

- Contratos Zod para sessao, job, desafio efemero e conclusao.
- Persistencia PostgreSQL de metadados da sessao e outbox transacional/idempotente.
- Fila BullMQ separada `le-fabrique.provider-onboarding`, com concorrencia um.
- Worker com comando fixo `codex login --device-auth`, shell desligado, ambiente sem API keys, timeout e limite de saida.
- Endpoint interno autenticado para publicar desafio e concluir a sessao.
- Redis privado com TTL para URL/codigo; leitura administrativa autenticada.
- Painel inicia, acompanha e exibe o desafio sem usar armazenamento persistente do navegador.
- Testes exclusivamente com processo e dados sinteticos; nenhum login real.

## Fora do escopo

- Login de Claude Code ou Antigravity, cuja automacao segura permanece nao comprovada.
- Receber senha, token, cookie, `auth.json`, API key ou codigo digitado pelo operador.
- Persistir URL/codigo no PostgreSQL, logs, eventos, auditoria ou documentacao.
- Executar prompt, consumir modelo, habilitar API/extras, escolher plano, copiar credenciais ou alterar cobranca externa.
- Segundo adapter, handoff, GitHub OAuth, merge ou deploy.

## Criterios de aceite

1. Somente instalacao Codex existente, habilitada e em `AUTH_REQUIRED` pode iniciar sessao; uma sessao ativa por instalacao e deduplicada.
2. Sessao e outbox `provider.onboarding.requested.v1` sao gravados atomicamente.
3. Worker usa somente o executavel allowlisted e argv fixo; caminho/comando do painel nunca e executado.
4. Desafio sintetico e publicado pelo worker no endpoint interno, guardado somente no Redis com TTL maximo de dez minutos e retornado apenas ao administrador autenticado.
5. PostgreSQL, jobs, mensagens, logs e respostas de listagem nao contem URL, codigo, token ou output bruto.
6. Conclusao exige identidade do worker, transicao valida e prova posterior via `codex login status`; reentrega identica nao duplica efeito.
7. Timeout/cancelamento encerram a arvore do processo e removem o desafio efemero; falhas retornam mensagem controlada.
8. Painel nao grava o desafio em `localStorage` ou `sessionStorage` e deixa claro que a autenticacao ocorre no site oficial.
9. Lint, typecheck, testes, build, Prisma validate e `git diff --check` passam.

## Baseline, provider e limites

- Base aceita: `d772d127d213999a46dfaedb633b5dc70875be54` (FAC-010A aceito e registrado).
- Branch/worktree: `feat/fac-010b-ephemeral-login`, `/home/vinicius/le-fabrique-fac-010-login`.
- Provider elegivel neste incremento: Codex CLI observado na versao `0.159.2`, autenticacao ChatGPT por `codex login --device-auth`.
- Timeout de sessao: dez minutos; desafio Redis expira em no maximo dez minutos; saida combinada limitada a 64 KiB; um job por vez.
- Budget de API zero; API keys, extra usage, creditos, autorecharge e fallback pago continuam proibidos.

## Verificacao e aceite

Fixtures simulam o processo sem chamar o fornecedor. Implementacao funcional verificada em `4d979aa7045022020447114c0dd125dfe5dfadde`; entrega aceita explicitamente pelo responsavel em 2026-10-01 na revisao documental exata `d39412282d9401b2a22c70eb4b353883e559be4c`. Merge, migracao e deploy nao fazem parte do ticket.
