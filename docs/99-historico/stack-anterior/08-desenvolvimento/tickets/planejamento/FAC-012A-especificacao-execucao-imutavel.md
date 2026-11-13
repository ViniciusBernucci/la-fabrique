# FAC-012A — Especificacao imutavel de execucao

Status: DONE

## Objetivo

Converter projeto, definicao e ticket configurados no controle em um snapshot runtime imutavel no momento de `DRAFT -> READY`. Esse contrato prepara a ligacao do worker ao fluxo autonomo sem cadastrar ou hardcodar um projeto externo.

## Atual e esperado

FAC-001A associa somente `projectDefinitionVersion` ao evento. Como a definicao atual e atualizada no mesmo registro, a versao isolada nao permite reconstruir com seguranca o conteudo usado quando o ticket ficou READY.

O esperado e que cada novo `ticket.ready.v1` carregue uma especificacao validada com identidade/revisao do projeto, copia da definicao e objetivo/criterios do ticket. Alteracoes posteriores no painel nao podem mudar um job ja criado.

## Escopo permitido e proibido

- Permitido: contratos runtime compartilhados, snapshot no payload da outbox, invariantes cruzadas, dispatcher, probe do worker, testes e documentacao.
- Permitido: falhar fechado para evento legado ainda PENDING que nao possua especificacao.
- Proibido: clonar repositorio, executar `DeveloperWorkflow`, checks ou cliente de IA.
- Proibido: escolher provider/modelo, alterar contas, criar projeto/piloto sintetico, aplicar migration, limpar fila ou reescrever evento historico.
- Proibido: merge, deploy ou qualquer escrita externa.

## Criterios de aceite

1. `executionSpecificationSchema` estrito inclui projeto, SHA, versao e conteudo integral da definicao, alem de identidade, versao, titulo, objetivo e criterios do ticket.
2. `orchestrationJobSchema` valida correspondencia entre IDs, versoes e SHA do envelope e do snapshot.
3. `markReady` monta e valida o snapshot dentro da mesma transacao antes de atualizar ticket/outbox.
4. Evento novo persiste a especificacao integral; edicao posterior da definicao nao altera o JSON ja salvo.
5. Falha de schema ocorre antes de mutacao parcial.
6. Retry idempotente de ticket ja READY retorna o estado existente sem recriar snapshot.
7. Dispatcher nao publica evento legado sem especificacao e preserva sua politica limitada de tentativas.
8. Probe do worker recusa job sem especificacao, mas nao executa repo/check/provider.
9. Nenhum dado de conta, modelo, credencial ou piloto entra no contrato.
10. Lint, typecheck, testes, build, Prisma validate e `git diff --check` passam.

## Baseline, caminhos e limites

- Base: `43043c98ca4b105daf1f2e9986882b26da6f2ec3`.
- Branch/worktree: `feat/fac-012a-execution-spec`, `/home/vinicius/le-fabrique-fac-012a`.
- Caminhos: `packages/contracts`, controle/orquestracao da API, processor do worker, testes e documentacao afetada.
- Um writer; ate duas rodadas de correcao; nenhum banco/Redis real; custo adicional zero.

## Provider elegivel

Nenhum provider do produto. O incremento e deterministico e nao inicia cliente oficial, login ou roteamento.

## Checks

`npm ci`, `npm run db:generate`, testes focados, `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`, Prisma validate com URL sintetica e `git diff --check`.

## Entregaveis e aceite

Ticket READY: `c658df97cf745df87c5895b037b2c95d7815e7ca`. Codigo verificado: `3b6f3e4624b5935e4fe6ea067406c631a6b0ec32`. Revisao documental aceita: `3ef3d543b98fb48226714787315f4de947fa6dd9`. Aceite explicito do responsavel em 2026-10-02; sem piloto ou consumer real.
