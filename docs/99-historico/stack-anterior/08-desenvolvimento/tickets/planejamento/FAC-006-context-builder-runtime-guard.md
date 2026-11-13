# FAC-006 — Context Builder e RuntimeGuard

Status: DONE

Revisao aceita: `ea8cf7afb0e55840722f306262b5334bb408b51a`, em 2026-09-30. O commit documental posterior apenas registra o aceite da revisao apresentada.

## Objetivo

Implementar a montagem deterministica de contexto local e um guard de execucao por assinatura que aplique limites de tentativas, tempo e trocas de provider antes de integrar o runtime ao orquestrador.

## Escopo

- Contratos Zod de request, manifesto, omissao, politica, estado e decisao em `packages/contracts`.
- `ContextBuilder` em `packages/runtime`, limitado a fontes relativas explicitamente selecionadas dentro do workspace.
- `RuntimeGuard` em `packages/runtime`, com pausa conservadora e ambiente sem chaves de API herdadas.
- Fixtures e testes locais, sem chamada a provider.
- Documentacao dos dominios de runtime, economia, operacao e planejamento.

Ficam fora: selecao automatica por imports, embeddings, persistencia, dispatcher BullMQ, claim/lease/fencing, sandbox/cgroups, roteamento real entre providers, deploy e piloto externo.

## Criterios de aceite

1. Fontes aceitas sao ordenadas de modo estavel e registram caminho, papel, tamanho e SHA-256; o manifesto possui hash reproduzivel e revisao-base explicita.
2. Caminho absoluto, traversal, symlink, dependencia, diretorio gerado, binario, arquivo grande e fonte com segredo detectavel nao entram no contexto e recebem motivo normalizado.
3. Limites de quantidade e bytes sao aplicados antes de produzir o contexto; omissao e truncamento nunca ficam silenciosos.
4. RuntimeGuard pausa ao atingir tentativas, tempo ou trocas de provider e ao repetir a mesma falha duas vezes consecutivas.
5. Politica subscription-only exige budget de API zero, fallback e extras desligados; configuracao contraditoria e rejeitada em runtime.
6. Chaves de API herdadas conhecidas sao removidas do ambiente entregue ao cliente oficial e os nomes removidos podem ser auditados sem registrar valores.
7. Testes usam somente fixtures locais e nao consomem assinatura, API, credito ou extra usage.

## Baseline, provider e limites

- Base: `developer` em `7044a0a932cb38f2b3c2f70c07fb0be34523ab13`.
- Branch/worktree: `feat/fac-006-context-builder-runtime-guard` em `/home/vinicius/le-fabrique-fac-000-accepted`.
- Provider elegivel futuro: Codex CLI aceito no FAC-005; esta entrega nao o executa.
- Limites iniciais: duas tentativas, 30 minutos por tentativa, duas trocas de provider, repeticao identica igual a duas e contexto de ate 200 arquivos/2 MiB.
- Um writer comprovado pelo worktree limpo; nenhuma arvore de provider foi iniciada para este ticket.

## Rollback

Reverter os commits do FAC-006 remove os contratos e modulos locais e devolve o FAC-005 sem alterar banco, migrations, login, provider ou deploy.
