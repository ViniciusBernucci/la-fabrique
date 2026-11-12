# FAC-012B — Compilador da especificação para workflow

Data: 2026-10-02  
Status: IMPLEMENTADO / ACEITO
Domínio: operação e runtime  
Base SHA: `f3ca9f2550b3d83535dffbe99f6bc7d2d07aee90`  
Revisão do código: `e7b9bf60e03089c502255d19edbb5d1e31725d2c`  
Branch: `feat/fac-012b-workflow-compiler`

## Objetivo e critérios

Preparar a fronteira entre o snapshot imutável do ticket READY e `DeveloperWorkflow`, sem ativar a fila real nem incluir projeto, piloto, provider ou modelo fixo. Perfil confiável de checkout, fontes/limites e allowlist são dados fornecidos pelo worker; o compilador não os obtém de um job nem executa operações.

## Implementação

`apps/worker/src/workflow-compiler.ts` adiciona `trustedWorkflowProfileSchema` e `compileWorkflowRequest`. O perfil é validado em runtime e associa `projectId`, URL, SHA-base e caminho local. Esses identificadores precisam corresponder exatamente ao snapshot. Os comandos configurados são aceitos apenas com correspondência exata de nome, executável e argv em uma allowlist confiável; o ambiente enviado ao workflow fica vazio.

Objetivo, stack, instruções, caminhos declarados e critérios do ticket são transferidos para o pedido. Fontes de contexto vêm do perfil e precisam ser caminhos relativos, normalizados, dentro de `allowedPaths` e fora de `forbiddenPaths`. Limites, guard subscription-only e `modelRequested` nullable também vêm do perfil validado. Nenhum provider é invocado.

## Diff sanitizado

`git diff f3ca9f2..b24e75c1ef5f3863a05fe5fc11bc844cb913e3f9`:

```text
apps/worker/src/workflow-compiler.ts      schema e compilação pura
apps/worker/src/workflow-compiler.spec.ts  fixtures de vínculo, allowlist, paths e policy
```

## Verificação

- `npm ci`: 211 pacotes; zero vulnerabilidades reportadas.
- Teste focalizado do compilador: 6 testes passaram.
- `npm run db:generate`: Prisma Client 6.12.0 gerado; nenhuma conexão/migration.
- `npm run lint`: 140 arquivos sem erro.
- `npm run typecheck`: contratos, runtime, API, worker e painel passaram.
- `npm test`: 169 testes passaram (launcher 2, contracts 19, runtime 38, API 50, worker 56, web 4).
- `npm run build`: todos os workspaces passaram; Vite transformou 118 módulos.
- `git diff --check`: passou.

## Limitações

- O compilador não está ligado ao consumidor BullMQ; jobs continuam no probe.
- Nenhum checkout, comando, cliente de IA, banco ou Redis real foi acessado.
- O caminho local precisa ser resolvido por configuração confiável futura do worker; esta entrega não clona nem cadastra um repositório local.
- `allowedPaths`/`forbiddenPaths` orientam o pedido e restringem as fontes de contexto, mas ainda não são uma política de escrita imposta pelo sandbox ao agente. Não iniciar workflow real até existir enforcement de escrita e prova de isolamento.
- Provider/modelo efetivo e elegibilidade continuam responsabilidade do Centro de Configurações + reconciliação do worker, não deste compilador.

## Rollback

Reverter os commits `e7b9bf60e03089c502255d19edbb5d1e31725d2c` e `b24e75c1ef5f3863a05fe5fc11bc844cb913e3f9`; não há migration, alteração de dados, fila ou integração externa.

## Documentação e lessons

Atualizados READMEs de operação/runtime/controle, planejamento, backlog, índice, changelog e `lessons/definicao-projeto-configuravel.md`.

## Uso de IA e custos

Implementação assistida nesta sessão, sem chamar provider do produto, sem tokens/uso de conta reportado e sem custo adicional. Provider/modelo do produto: não aplicável.

## Aceite

Revisão técnica concluída nos checks listados. Revisão documental `3e28363b0642f8e05840bb649b681e3837e1e015` aceita explicitamente pelo responsável em 2026-10-02. Estado `DONE`; nenhum consumer real ou piloto foi ativado.
