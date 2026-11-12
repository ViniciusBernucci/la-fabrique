# FAC-003A — Gate de revisao-base exata antes de READY

Data: 2026-10-02
Status: IMPLEMENTADO / ACEITO
Dominio: controle e operacao
Base SHA / revisao do codigo: `4e5b772a2dbeb1dd8868e3fe2bedd480c8e837c4` / `e65c88fd8f3bd15e3aa65773fedc945af3709def`
Branch / run / PR: `fix/ops-002-stale-orchestration-jobs` / worktree `/home/vinicius/le-fabrique-ops-002` / sem PR

## Objetivo e criterios de aceite

Impedir a publicacao de novos jobs de ticket sem commit-base resolvido e dar ao operador um caminho configuravel no painel para trocar `main` ou outra referencia por um SHA Git exato antes de READY. A correcao deve preservar jobs e eventos historicos, falhar antes de qualquer mutacao parcial e nao consultar Git/GitHub.

## Diagnostico

Os seis erros mostrados no ensaio continham seis `jobId` diferentes. A API anterior aceitava `baseRef: main`, promovia cada ticket e persistia um evento diferente com `baseRevision: null`; cada job falhava uma vez no worker antes do claim. Portanto, a evidencia nao indicava retry infinito do mesmo job, e sim seis intencoes distintas sem revisao executavel.

## O que foi implementado

- `gitCommitShaSchema` e `updateProjectBaseRevisionSchema` compartilham a validacao runtime do SHA hexadecimal minusculo de 40 caracteres.
- `PATCH /api/projects/{projectId}/base-revision` atualiza a referencia com comparacao otimista por `expectedBaseRef`, retornando conflito se outra escrita venceu.
- `DRAFT -> READY` valida o SHA dentro da mesma transacao antes de atualizar o ticket ou criar a outbox. O payload novo sempre recebe `baseRevision` valida e nao nula.
- O dispatcher recusa evento legado ainda `PENDING` com revisao nula antes de `queue.add`; apos o limite existente de tentativas da outbox, ele permanece `FAILED` no banco, sem ser apagado.
- O painel mostra a referencia atual, aceita o SHA exato, indica se o projeto esta liberado e desabilita `Marcar READY` enquanto nao estiver resolvido.
- O retry idempotente de ticket ja READY com evento existente continua retornando o estado atual.

## Funcionamento

O cadastro do projeto ainda pode usar uma branch para planejamento. Antes da execucao, o operador seleciona o projeto, informa o commit exato no card `Revisao-base executavel` e salva. Browser e API validam o formato; a API tambem compara a referencia previamente lida para evitar sobrescrita silenciosa.

Ao promover o ticket, a API le ticket e projeto na transacao. Referencia invalida produz conflito antes de `ticket.updateMany` e `outboxEvent.create`. A interface tambem bloqueia o botao, mas o backend permanece a autoridade. Eventos antigos e jobs falhos nao sao removidos, reabertos nem reenfileirados por este fluxo.

## Alteracoes e diff

Diff revisavel: `git diff dcc380c11775bba97c7020d5a22ca075f4feee05..e65c88fd8f3bd15e3aa65773fedc945af3709def`.

Resumo sanitizado:

```text
packages/contracts       schema de SHA e update otimista
apps/api/src/control     endpoint, gate transacional e testes
apps/api/src/orchestration  barreira para evento legado nulo
apps/web/src             configuracao da revisao e bloqueio visual
```

## Verificacao

- `npm ci`: 211 pacotes instalados; zero vulnerabilidades reportadas.
- `npm run db:generate`: Prisma Client 6.12.0 gerado sem conexao ao banco.
- Testes focados: contratos 17, controle/dispatcher 9 e web 3 passaram.
- `npm run lint`: 135 arquivos verificados, sem erro apos uma correcao de ordenacao de imports.
- `npm run typecheck`: cinco workspaces/pacotes passaram.
- `npm test`: 155 testes passaram — launcher 2, contratos 17, runtime 38, API 46, worker 49 e web 3.
- `npm run build`: contratos, runtime, API, worker e painel passaram; Vite transformou 116 modulos.
- Prisma validate com URL sintetica: schema valido, sem abrir conexao nem aplicar migration.
- `git diff --check`: passou antes do commit funcional.
- Nenhum smoke HTTP/Redis/PostgreSQL foi executado para nao alterar o plano de dados usado no ensaio do responsavel.

## Riscos e limitacoes

- O operador ainda precisa obter o SHA por fonte confiavel; resolver branch automaticamente fica fora deste ticket.
- Projetos antigos com branch permanecem cadastrados, mas seus tickets DRAFT ficam bloqueados ate a atualizacao pelo painel.
- Tickets/eventos/jobs ja publicados nao sao reescritos. O dispatcher apenas impede que evento legado ainda PENDING alcance a fila.
- `removeOnFail: false` continua preservando jobs BullMQ para auditoria; limpeza exige ticket e autorizacao separados.
- Nenhuma migration, chamada Git/GitHub, provider de IA, merge ou deploy ocorreu.

## Rollback

Reverter `e65c88fd8f3bd15e3aa65773fedc945af3709def`. Isso restaura a promocao anterior que aceitava revisao nula; nao remover jobs, eventos ou volumes como rollback. Nenhuma reversao de schema de banco e necessaria.

## Documentacao atualizada

`documentacoes/controle/README.md`, `documentacoes/operacao/README.md`, ticket FAC-003A, indice, changelog, backlog, README raiz e `lessons/outbox-idempotencia.md`. Nenhuma decisao arquitetural mudou e nao foi criado ADR.

## Lessons

`lessons/outbox-idempotencia.md` passa a registrar que deduplicacao nao substitui elegibilidade do payload: uma intencao unica ainda pode ser impossivel e deve ser bloqueada antes da outbox.

## Uso de IA e custos

Implementacao assistida por Codex nesta sessao; versao/modelo efetivo, tokens e cota nao foram expostos pelo ambiente. Nenhum provider do produto, API, extra usage, credito ou autorecharge foi usado; nenhum handoff ocorreu. Tempo e custo fixo da assinatura permanecem nao medidos.

## Pendencias e aceite

Codigo verificado em `e65c88fd8f3bd15e3aa65773fedc945af3709def`. O responsavel aceitou a revisao documental `4630ce0b9a613b937d40d550e4551fa1402d6c6a` em 2026-10-02. Resolucao automatica de branch, remediacao de tickets READY antigos e politica de retencao BullMQ ficam fora do escopo.
