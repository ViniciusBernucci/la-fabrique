# FAC-012A — Especificacao imutavel de execucao

Data: 2026-10-02
Status: IMPLEMENTADO / ACEITO
Dominio: operacao, controle e contratos
Base SHA / revisao do codigo: `43043c98ca4b105daf1f2e9986882b26da6f2ec3` / `3b6f3e4624b5935e4fe6ea067406c631a6b0ec32`
Branch / PR: `feat/fac-012a-execution-spec` / sem PR

## Objetivo

Congelar, no evento READY, os dados configurados que uma execucao autonoma futura consumira. O snapshot elimina a dependencia do estado mutavel da definicao do projeto sem inventar piloto, provider ou resultado de execucao.

## Implementacao

- `executionSpecificationSchema` e estrito e contem identidade, URL, SHA, versao e conteudo integral da definicao do projeto.
- O mesmo snapshot guarda identidade/versao, titulo, objetivo e criterios do ticket.
- `orchestrationJobSchema` e o contrato de claim verificam que IDs, versoes e SHA do envelope correspondem ao snapshot.
- `markReady` le e valida a definicao dentro da transacao e monta a especificacao antes da primeira mutacao; ticket e outbox continuam atomicos.
- O JSON completo segue no evento `ticket.ready.v1`, portanto edicoes posteriores no painel nao alteram a intencao ja registrada.
- Dispatcher e probe recusam eventos sem especificacao. O limite existente de tentativas leva eventos legados PENDING a FAILED sem apagar historico.
- O probe ainda executa apenas claim/checkpoint/complete sintetico; nao clona repositorio, roda check ou inicia IA.

## Diff sanitizado

Revisao: `git diff c658df97cf745df87c5895b037b2c95d7815e7ca..3b6f3e4624b5935e4fe6ea067406c631a6b0ec32`.

```text
packages/contracts              snapshot e invariantes cruzadas
apps/api/src/control            montagem transacional no READY
apps/api/src/orchestration      bloqueio de evento legado incompleto
apps/worker/src                 validacao do probe antes do claim
```

## Verificacao

- `npm ci`: 211 pacotes instalados; zero vulnerabilidades reportadas.
- `npm run db:generate`: Prisma Client 6.12.0 gerado; nenhuma migration nova.
- Focados: contratos 19, controle/dispatcher 13 e processor do worker 3 passaram.
- `npm run lint`: 138 arquivos, sem erro depois de uma correcao de formatacao.
- `npm run typecheck`: cinco workspaces/pacotes passaram.
- `npm test`: 163 testes passaram — launcher 2, contratos 19, runtime 38, API 50, worker 50 e web 4.
- `npm run build`: contratos, runtime, API, worker e web passaram; Vite transformou 118 modulos.
- Prisma validate com URL sintetica: schema valido.
- `git diff --check`: passou.
- Varredura sanitizada encontrou somente nomes de chaves sinteticas/documentais preexistentes; nenhum segredo novo.

## Limites e riscos

- A especificacao ainda nao e traduzida em `DeveloperWorkflowRequest`; ligar o consumer real pertence ao proximo ticket.
- URL e checks continuam dados administrativos. Antes da execucao, o worker deve reconciliar checkout local e allowlist de executaveis; persistencia validada nao substitui isolamento.
- O payload da outbox fica maior, mas permanece limitado pelos contratos atuais: ate 100 caminhos por lista, 20 checks e 20 criterios.
- Eventos legados sem snapshot nao sao reconstruidos automaticamente, pois isso usaria estado atual possivelmente diferente do original.
- Nenhum banco/Redis real, provider, login, repo externo, merge ou deploy foi acessado.

## Rollback

Reverter `3b6f3e4624b5935e4fe6ea067406c631a6b0ec32`. Nao ha migration. Eventos ja gravados com o campo adicional permanecem JSON valido no PostgreSQL; nao remove-los nem reescreve-los como parte do rollback.

## Documentacao e lesson

Foram atualizados READMEs de controle/operacao/runtime, especificacao, plano, backlog, changelog e indice. A lesson `definicao-projeto-configuravel.md` passa a distinguir referencia de versao de snapshot autocontido.

## Aceite

Codigo verificado em `3b6f3e4624b5935e4fe6ea067406c631a6b0ec32`; revisao documental aceita `3ef3d543b98fb48226714787315f4de947fa6dd9`. Aceite explicito do responsavel em 2026-10-02. O piloto e o consumer real continuam fora desta entrega.
