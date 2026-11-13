# FAC-012C — Política de escrita do SandboxRunner

Status: AWAITING_HUMAN

## Objetivo

Impedir que comandos executados pelo SandboxRunner alterem o workspace inteiro por padrão. Tornar a árvore somente leitura e reabrir escrita apenas em caminhos explícitos confiáveis, mantendo isolamento de host, limites e quiescência existentes.

## Critérios de aceite

1. Contrato runtime adiciona `writablePaths`, vazio por padrão, com caminhos relativos normalizados, únicos e sem sobreposição.
2. Runner falha antes de iniciar processos se um caminho não existe, contém symlink ou sai da raiz real do workspace.
3. Launcher monta `/mnt` somente leitura e reabre apenas caminhos declarados; o comando não consegue criar/alterar arquivos fora deles.
4. Caminhos exatos de arquivo e diretório funcionam; sistema de arquivos e política de rede/limites atuais permanecem.
5. Testes de integração comprovam escrita permitida, escrita negada, symlink recusado, root readonly, timeout e parada confirmada.
6. Lint, typecheck, testes, build, `git diff --check` e validação do Compose passam.
7. Não ligar o consumer real nem executar cliente/provider.

## Baseline e limites

Base `ce27c13cace1fa81b8bffde1df98e8c3e499fe0e`; branch `feat/fac-012c-sandbox-path-policy`; worktree `/home/vinicius/le-fabrique-fac-012c`. Caminhos: `packages/contracts`, `packages/runtime`, testes e docs de infraestrutura/operação. Sem DB, Redis, checkout externo, deploy, login, migration aplicada ou gasto. Nenhum provider elegível necessário.

## Dependências e limites conhecidos

FAC-007, FAC-009, FAC-012A/B. Este ticket restringe processos do `SandboxRunner`. Correção de escopo: CodexAdapter já executa comandos do modelo sob perfil de sandbox nativo customizado, com rede desativada; naquele perfil, no entanto, a escrita está liberada em `.` em vez dos caminhos `allowedPaths` do projeto. FAC-012D trata essa lacuna. A inacessibilidade operacional às credenciais oficiais sob a identidade real do worker continua sem prova.

## Implementação

Código: `a8f7a66dfc1b5c58a0596611dc300559504709ce`. Contrato compartilhado exige `writablePaths` (vazio por padrão), valida caminhos relativos únicos/não sobrepostos e rejeita `.git`. Runner verifica existência, raiz real e ausência de symlinks antes de iniciar. Launcher monta `/mnt` read-only e reabre somente os caminhos selecionados. Checks do `DeveloperWorkflow` passam lista vazia explicitamente.

## Evidências

- `npm test -w @le-fabrique/contracts` — 20 testes passaram.
- `npm test -w @le-fabrique/runtime -- src/sandbox-runner.spec.ts` — suíte do runtime, 40 testes passaram; inclui isolamento Linux de escrita permitida/negada.
- Testes do worker — 56 passaram.
- `npm run lint` — passou, 140 arquivos.
- `npm run typecheck` — passou em todos os workspaces.
- `npm test` — 172 testes passaram (launcher 2, contracts 20, runtime 40, API 50, worker 56, web 4).
- `npm run build` — passou; build Vite incluiu 118 módulos.
- `git diff --check` — passou.

Integração Linux foi executada com fixtures; nenhum cliente/provider, repositório externo, banco, fila ou migration foi usado. Não executar comandos reais do provider nem ativar o consumer até aplicar isolamento equivalente ao CLI Developer e concluir o preflight operacional de identidade/custos.
