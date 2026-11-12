# FAC-012D — Restringir escrita do CLI aos caminhos do projeto

Status: AWAITING_HUMAN

## Objetivo

Levar a lista `allowedPaths` congelada no snapshot do projeto até o perfil nativo de permissões usado pelo Codex Developer. O agente deve ler o workspace, gravar somente nos caminhos configurados e manter `.git`, `.codex`, host e rede externa inacessíveis.

## Critérios de aceite

1. Contratos de workflow/runtime carregam caminhos graváveis relativos, vazios por padrão, únicos, não sobrepostos e sem segmentos `.git`/`.codex`.
2. `compileWorkflowRequest` valida o snapshot imutável em runtime, obtém caminhos exclusivamente de `executionSpecification.project.definition.allowedPaths` e rejeita definições inválidas/sobrepostas a `forbiddenPaths`.
3. Developer recebe a allowlist; Reviewer e `READ_ONLY` não recebem escrita mesmo que o request tente incluí-la.
4. CodexAdapter gera um único perfil `permissions.lefabrique`: raiz do workspace read-only, somente caminhos autorizados graváveis, `.git`/`.codex` negados, rede desligada. Não combinar `default_permissions` com `--sandbox`/`sandbox_mode`.
5. Testes de contratos, compilador, argumentos/perfil do adapter e workflow demonstram os caminhos propagados e a falha fechada do default vazio; nenhum provider real é chamado.
6. Lint, typecheck, testes, build e `git diff --check` passam.
7. Não ligar consumer, não usar banco/fila, não autenticar, clonar projeto ou iniciar modelo. É permitido testar o sandbox nativo com `codex sandbox` e fixture local sem modelo/provider.

## Baseline, arquivos e limites

Base `a6fdf9ce57cc2293ddf310198204ea1093db9522`; branch `feat/fac-012d-project-write-paths`; worktree `/home/vinicius/le-fabrique-fac-012d`. Arquivos afetados: contracts, runtime CodexAdapter, worker workflow compiler/DeveloperWorkflow e documentação de runtime/operação. Nenhum provider precisa estar elegível. Codex CLI local está instalado (0.159.2), mas será usado somente para leitura de help; sem inferência/custo. Há um worker antigo ativo fora desta worktree processando a fila sintética; não interagir com ele nem com DB/Redis.

## Contexto de segurança

FAC-012C endureceu os comandos do `SandboxRunner`. Separadamente, `CodexAdapter` já usa o sandbox nativo Codex com perfil customizado e rede desabilitada. A lacuna é a permissão de escrita atual em `.` para o Developer, que ignora `allowedPaths`. O sandbox do CLI permanece distinto de `SandboxRunner`; não declarar prova operacional de isolamento de credenciais sem teste sob a identidade real do worker.

## Fora de escopo

Ativação do consumer BullMQ, roteamento/configuração de provider no READY, checkout/clonagem, preflight de cobrança/login, suporte a Claude e prova com modelo real. A decisão de piloto continua manual.

## Implementação e evidência

Implementado em `packages/contracts`, `packages/runtime` e `apps/worker`. O perfil Codex mantém leitura da raiz do workspace, libera escrita somente em paths permitidos existentes e sem symlinks e nega `.git`/`.codex`; rede permanece desligada. O Reviewer recebe `READ_ONLY` e lista vazia. Paths inválidos falham antes de iniciar o CLI. O compilador agora faz parse runtime do snapshot completo e rejeita sobreposição permitido/proibido.

Ver relatório `documentacoes/operacao/2026-10-02-FAC-012D-caminhos-escrita-cli.md`. Lint, typecheck, 175 testes, build e `git diff --check` passaram. Codex CLI 0.159.2 também aplicou o perfil em fixture nativa sem inferência: escrita autorizada passou, escrita raiz/documentação e leitura `.git`/`.codex` foram bloqueadas, e conexão loopback foi bloqueada. Evidência segue local/interativa: não foi feita inferência real, teste sob usuário do serviço, integração de Claude nem ativação do consumer. Aguarda revisão/aceite humano; não fazer merge/deploy até aceite aplicável.
