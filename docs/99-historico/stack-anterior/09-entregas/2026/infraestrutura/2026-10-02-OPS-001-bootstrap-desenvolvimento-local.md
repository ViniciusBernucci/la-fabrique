# OPS-001 — Bootstrap confiavel do desenvolvimento local

Data: 2026-10-02
Status: IMPLEMENTADO / DONE
Dominio: infraestrutura e operacao
Base SHA / revisao do codigo: `c069926cb4b43a1519b5fd10b212b8fbe36cbfaa` / `12018c3596efceb7bf4d171ca4787a81bfd9c136`
Branch / run / PR: `fix/ops-001-dev-bootstrap` / worktree `/home/vinicius/le-fabrique-dev-bootstrap` / sem PR

## Objetivo e criterios de aceite

Corrigir as falhas reproduzidas no primeiro ensaio manual: variaveis ausentes ao executar `npm run dev`, encerramento do worker quando ele registra antes da API e criacao acidental de artefato ao usar `prisma migrate dev` apenas para iniciar o ambiente. O fluxo deve continuar local, limitado, sem limpar fila, aplicar migration ou ativar provider externo.

## O que foi implementado

- `scripts/dev.mjs` carrega obrigatoriamente o `.env` da raiz com a API nativa do Node.js e inicia API, worker e web por argv fixo, sem shell.
- `scripts/dev.test.mjs` comprova carregamento do ambiente, os tres comandos, propagacao do ambiente e `shell: false`; a suite raiz agora inclui esses testes.
- O registro inicial do worker tenta novamente somente para erro de rede (`TypeError`) ou resposta 5xx. Sao no maximo 30 tentativas, com 1 segundo entre falhas; 4xx e erros de schema falham imediatamente.
- O log de espera contem somente tentativa/limite/intervalo e nao inclui URL, token nem resposta bruta.
- `.env.example` usa `VITE_API_URL=/api`, de modo que o browser utiliza o proxy same-origin do Vite ao abrir localhost ou o endereco de rede da VPS.
- O lint passou a cobrir os launchers `.mjs` em `scripts/`.

## Funcionamento

O operador copia `.env.example` para `.env`, sobe PostgreSQL e Redis, instala as dependencias, gera o Prisma Client, aplica as migrations ja versionadas e executa `npm run dev`. O launcher importa o ambiente antes de criar os filhos; todos recebem o mesmo `process.env`.

Se a API ainda estiver subindo, o worker registra novamente dentro da janela limitada. Depois do registro, a politica existente permanece: tres falhas consecutivas de heartbeat encerram o consumidor de forma conservadora. Erros permanentes de autorizacao nao entram em retry.

Jobs sinteticos historicos presentes no Redis podem emitir `orchestration probe failed` quando a revisao-base da fixture nao existe mais. Isso e resultado do job, nao falha de startup; OPS-001 nao apaga nem repete dados administrativamente. `worker ready` continua sendo a evidencia de que o consumidor iniciou.

## Alteracoes e diff

Diff revisavel: `git diff ab48aead8eec1680e7b15b5f3b7abbaa9340a04d..12018c3596efceb7bf4d171ca4787a81bfd9c136`.

Resumo sanitizado:

```text
package.json                         launcher raiz e testes
scripts/dev.mjs                     importacao do .env e spawn sem shell
scripts/dev.test.mjs                2 testes do launcher
apps/worker/src/control-client.ts   classificacao e retry limitado
apps/worker/src/control-client.spec.ts  3 cenarios de registro
.env.example                        API do painel por /api
```

## Verificacao

- `npm ci`: 211 pacotes instalados, zero vulnerabilidades reportadas.
- `npm run db:generate`: Prisma Client 6.12.0 gerado sem acessar o banco.
- `npm run typecheck`: passou nos cinco workspaces/pacotes depois de gerar o Prisma Client.
- `npm run lint`: 133 arquivos verificados, sem erro.
- `npm test`: 148 testes passaram em 36 arquivos/suites — launcher 2, contratos 16, runtime 38, API 41, worker 49 e web 2.
- `npm run build`: contratos, runtime, API, worker e painel passaram; Vite transformou 115 modulos.
- `npm run db:deploy -w @le-fabrique/api -- --help`: confirmou `prisma migrate deploy` sem abrir conexao ou aplicar migration.
- `DATABASE_URL=<url-sintetica> npm exec -w @le-fabrique/api -- prisma validate`: schema valido; a URL foi usada somente para parsing.
- `git diff --check`: passou antes do commit funcional.
- Rodada inicial do teste isolado do worker falhou porque `@le-fabrique/runtime` ainda nao tinha `dist`; a ordem oficial compilou os pacotes e removeu a falha.
- Primeira tentativa de typecheck falhou porque o Prisma Client ainda nao havia sido gerado; `npm run db:generate`, passo documentado do bootstrap, removeu a falha.
- O smoke multi-processo nao foi executado nesta worktree porque o ensaio do responsavel compartilha as mesmas portas, PostgreSQL e fila, e nao foi provisionado um plano de dados isolado para este ticket. Iniciar outro consumidor poderia reprocessar jobs historicos e alterar estado.

## Riscos e limitacoes

- A janela de aproximadamente 29 segundos tolera apenas a inicializacao normal; indisponibilidade persistente continua encerrando o worker.
- O proxy Vite e somente de desenvolvimento. Na composicao completa, Nginx continua responsavel por `/api`.
- O launcher requer `.env`; ausencia produz instrucao explicita para copiar `.env.example`.
- Jobs antigos permanecem no historico BullMQ. Qualquer politica de limpeza exige ticket e autorizacao proprios.
- Nenhuma migration foi aplicada, nenhuma fila foi limpa e nenhum dado do responsavel foi alterado.

## Rollback

Reverter o commit funcional `12018c3596efceb7bf4d171ca4787a81bfd9c136`. Como contorno temporario, exportar manualmente o `.env`, iniciar API antes do worker e usar `npm run db:deploy` para migrations versionadas. Nao remover volumes nem limpar Redis como rollback.

## Documentacao atualizada

README raiz, `documentacoes/infraestrutura/README.md`, `documentacoes/operacao/README.md`, ticket OPS-001, indice, changelog, backlog e `lessons/worker-heartbeat-quiescencia.md`. Nenhuma decisao estrutural mudou; ADR novo nao e necessario.

## Lessons

`lessons/worker-heartbeat-quiescencia.md` agora separa retry de bootstrap, heartbeat durante operacao e falha de job historico.

## Uso de IA e custos

Implementacao assistida por Codex nesta sessao; versao/modelo efetivo, tokens e cota nao foram expostos pelo ambiente. Nenhum cliente de provider do produto foi chamado, nenhum handoff ocorreu e nenhuma API, extra usage, credito ou custo incremental foi habilitado. Tempo de engenharia e custo fixo da assinatura permanecem nao medidos.

## Pendencias e aceite

A revisao funcional exata e `12018c3596efceb7bf4d171ca4787a81bfd9c136`. O responsavel aceitou explicitamente a revisao documental exata `7c1d1dcb44ab3464085aad729ae2b5dbdc473057` em 2026-10-02. Prova de VPS/Compose completo e limpeza de historico da fila nao pertencem a este ticket.
