# FAC-012C — Política de escrita do SandboxRunner

Status: AWAITING_HUMAN

Branch: `feat/fac-012c-sandbox-path-policy`

Base: `ce27c13cace1fa81b8bffde1df98e8c3e499fe0e`

Código: `a8f7a66dfc1b5c58a0596611dc300559504709ce`

## O que mudou

O contrato runtime agora oferece `writablePaths`, vazio por padrão, com validação de caminhos relativos, duplicatas, sobreposição e `.git`. Antes de iniciar um processo, `SandboxRunner` confere que cada caminho existe, permanece dentro da raiz real e não atravessa symlinks. O launcher monta o workspace em `/mnt` somente leitura e reabre apenas os caminhos autorizados. Checks do `DeveloperWorkflow` declaram explicitamente lista vazia.

Isso limita subprocessos executados pelo `SandboxRunner`; não limita o processo do CLI Developer. Logo, o consumer real continua desligado. O próximo incremento deve aplicar confinamento equivalente ao CLI writer, assegurar que credenciais oficiais não são acessíveis ao código executado e provar essas propriedades antes de conectar o fluxo real.

## Verificações

- Contracts: 20 testes passaram.
- Runtime: 40 testes passaram, incluindo integração Linux de escrita permitida/negada, caminho inválido/symlink e timeout/parada.
- Worker: 56 testes passaram.
- `npm run lint`: passou (140 arquivos).
- `npm run typecheck`: passou em todos os workspaces.
- `npm test`: 172 testes passaram no total (launcher 2, contracts 20, runtime 40, API 50, worker 56, web 4).
- `npm run build`: passou (Vite: 118 módulos).
- `git diff --check`: passou.

As provas foram feitas com fixtures locais; não houve uso de provider, login, projeto piloto, DB/fila, migration ou deploy. Nenhuma conta/modelo foi hardcoded. Não se tentou gastar créditos nem alterar configuração de cobrança.

## Limitações, rollback e aceite

Caminhos graváveis devem existir antes da execução; diretórios de saída requerem perfil confiável que os crie sem ampliar permissões. Checks atualmente recebem `[]` e podem falhar se tentarem gravar outputs no workspace. Rollback: reverter `a8f7a66dfc1b5c58a0596611dc300559504709ce`; não existe migration nem dado persistido para reverter.

Aceite pendente da revisão exata deste relatório e código. A decisão de piloto continua manual e fora do código.
