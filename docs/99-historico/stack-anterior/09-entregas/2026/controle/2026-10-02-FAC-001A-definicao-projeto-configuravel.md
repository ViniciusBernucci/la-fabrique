# FAC-001A — Definicao de projeto configuravel

Data: 2026-10-02
Status: IMPLEMENTADO / ACEITO
Dominio: controle
Base SHA / revisao do codigo: `4630ce0f85ef549c520c758e6870fdb874de8c55f` / `148485ec5858756460b1db6f183c346854703981`
Branch / PR: `feat/fac-001a-project-definition` / sem PR

## Objetivo e criterios

Permitir que o operador descreva qualquer projeto externo na propria La fabrique. A definicao deve permanecer ausente ate ser cadastrada e precisa reunir descricao, stack, instrucoes, caminhos e checks antes de liberar novos tickets para READY. O piloto futuro sera apenas um projeto escolhido nessa interface; nenhuma identidade concreta foi embutida no codigo.

## Implementacao

- Contratos Zod estritos validam a definicao, caminhos normalizados e checks com executavel/argumentos separados.
- Escopos de caminho absolutos, com travessia, duplicados ou conflitantes sao recusados.
- A tabela aditiva `project_definitions` guarda configuracao JSON validada, versao otimista e timestamps, em relacao um-para-um com `Project`.
- `GET /api/projects/{projectId}/definition` retorna definicao ou `null`; `PUT` cria com `expectedVersion: 0` ou atualiza a versao corrente.
- O painel permite criar/editar todos os campos e uma lista de ate vinte checks. Argumentos usam array JSON para preservar limites de `argv`, sem interpretacao por shell.
- A promocao para READY exige SHA exato e definicao persistida. O evento novo registra `projectDefinitionVersion`, vinculando a intencao ao estado configurado.
- O retry de um ticket ja READY continua idempotente mesmo se a configuracao for alterada depois.

## Diff sanitizado

Revisao: `git diff af30aa906fcacd54ccfd8a985558964956585760..148485ec5858756460b1db6f183c346854703981`.

```text
packages/contracts       schemas runtime de definicao, paths e checks
apps/api/prisma          tabela project_definitions e migration aditiva
apps/api/src/control     GET/PUT otimistas e gate de READY
apps/web/src             formulario generico e bloqueio operacional
```

## Verificacao

- `npm ci`: 211 pacotes instalados; zero vulnerabilidades reportadas.
- `npm run db:generate`: Prisma Client 6.12.0 gerado.
- Testes focados: contratos 18, controle 9 e web 4 passaram.
- `npm run lint`: 138 arquivos, sem erro.
- `npm run typecheck`: contratos, runtime, API, worker e web passaram.
- `npm test`: 160 testes passaram — launcher 2, contratos 18, runtime 38, API 49, worker 49 e web 4.
- `npm run build`: todos os workspaces passaram; Vite transformou 118 modulos.
- Prisma validate com URL sintetica: schema valido.
- `git diff --check`: passou.
- Varredura sanitizada encontrou apenas nomes de variaveis/chaves sinteticas dos testes e documentacao existentes; nenhum segredo novo.

## Falhas e correcoes

A primeira tentativa de `db:generate` encontrou o worktree sem dependencias; `npm ci` resolveu a preparacao, sem mudanca de codigo. O typecheck isolado da API depende do build previo dos contratos, portanto foi substituido pelo pipeline raiz oficial. Duas correcoes do incremento foram feitas: export do tipo compartilhado e conformidade de lint/identidade React/CSS. A suite completa passou depois delas.

## Limites e seguranca

- A definicao e configuracao desejada; ainda nao foi ligada ao `DeveloperWorkflow` no consumer BullMQ real.
- O worker continuara responsavel por traduzir apenas checks cadastrados e validados para seu perfil confiavel; output de IA nao pode fornecer executaveis.
- A migration foi validada, mas nao aplicada a banco real.
- O Centro de Configuracoes existente continua como unica fonte de contas, providers, modelos e papeis. Nenhum login ou chamada de IA ocorreu.
- O projeto externo e o piloto permanecem indefinidos.

## Rollback

Antes de aplicar a migration, reverter `148485ec5858756460b1db6f183c346854703981`. Depois de aplicada, reverter o codigo exige migration compensatoria autorizada para remover `project_definitions`; nao executar `DROP TABLE` manualmente e preservar configuracoes existentes antes de qualquer rollback destrutivo.

## Documentacao e lesson

Foram atualizados README de controle/operacao, especificacao, plano, contrato adiado do piloto, README raiz, backlog, changelog e indices. `lessons/definicao-projeto-configuravel.md` registra a separacao entre configuracao de projeto, roteamento de IA e execucao.

## Aceite

Codigo verificado em `148485ec5858756460b1db6f183c346854703981`. O responsavel aceitou a revisao documental `094190af26c1175bad08eb46293230b2938843d4` em 2026-10-02.
