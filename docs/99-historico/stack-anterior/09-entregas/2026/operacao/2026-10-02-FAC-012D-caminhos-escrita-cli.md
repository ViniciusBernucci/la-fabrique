# FAC-012D — Escrita do Codex limitada aos caminhos do projeto

## Resultado

Implementação no commit `04e5f30` sobre a base `a6fdf9ce57cc2293ddf310198204ea1093db9522`, branch `feat/fac-012d-project-write-paths`. Ticket aguarda revisão/aceite humano; não houve merge, deploy, push, migration, acesso ao banco/fila, execução de provider ou seleção de piloto.

O snapshot validado pelo worker agora transporta `project.definition.allowedPaths` até o Developer. `projectWritablePathsSchema` exige caminhos relativos, únicos, sem sobreposição, `.git` ou `.codex`; o compilador revalida o `executionSpecification` em runtime e rejeita interseção com caminhos proibidos. A política comum valida diretório, existência e ausência de symlinks. Caminhos inexistentes/symlinks falham fechados antes de iniciar o CLI.

CodexAdapter gera o perfil nomeado `permissions.lefabrique` com workspace legível, escrita somente nos subpaths declarados, `.git`/`.codex` negados e rede desabilitada. `READ_ONLY` ignora qualquer allowlist recebida. O DeveloperWorkflow passa os caminhos ao Developer e força lista vazia para Reviewer. ProviderHandoff, que não recebe manifesto confiável de paths, também usa lista vazia.

## Evidências

- `npm run lint`: passou (Biome, 141 arquivos).
- `npm run typecheck`: passou em contracts, runtime, API, worker e web. Antes, `npm ci` deixou Prisma Client sem geração local; `npx prisma generate` regenerou somente artefatos locais do client. Nenhuma migration foi aplicada.
- `npm test`: passou: scripts 2, contracts 20, runtime 42, API 50, worker 57, web 4 (175 no total). Inclui fixture Codex e integração Linux do SandboxRunner; não inicia modelo/provider real.
- `npm run build`: passou nos cinco workspaces, incluindo Vite production build.
- `git diff --check`: passou.
- Verificação das regras geradas por fixture sintética confirma somente subpaths específicos como write, raiz como read, ausência de API keys herdadas e rede desabilitada. Isso comprova argumentos/configuração emitidos, não comportamento efetivo do CLI instalado durante uma inferência real.
- Verificação nativa adicional em `codex-cli 0.159.2`: `codex sandbox -P lefabrique` executou processos locais sintéticos usando o mesmo formato de perfil, sem inferência, login ou requisição ao provider. Confirmou escrita autorizada em `packages/runtime/src` (`WRITE_ALLOWED`); tentativas em `documentacoes/` e na raiz foram `EROFS`; leitura de `.git` e de sentinel sintético `.codex` foi `EACCES`. Um servidor TCP local, aberto exclusivamente pela fixture, foi inacessível pelo sandbox (`EPERM`) com `network.enabled=false`. Os arquivos de prova foram removidos e a worktree permaneceu limpa. Isso verifica aplicação pelo sandbox nativo no usuário interativo, mas não autenticação/isolamento da identidade systemd do worker.

## Diff sanitizado

- Contratos compartilhados: allowlist gravável comum para requests runtime/workflow e proteção de `.git`/`.codex` na definição do projeto.
- Runtime: política reutilizada por SandboxRunner/CodexAdapter; profile Codex restrito a subpaths; fixtures/testes cobrindo read-only, caminhos declarados e falha fechada.
- Worker: validação runtime do snapshot, propagação exclusivamente da configuração congelada do projeto; Reviewer read-only.
- Nenhum prompt/segredo, conteúdo de credencial, dado real de projeto ou saída bruta de provider foi registrado neste relatório.

## Limitações e próximos passos

1. O perfil foi exercitado pelo sandbox nativo sob o usuário interativo, mas precisa de prova controlada sob a identidade real do serviço antes de ligar o consumer. A sessão Codex do usuário interativo não prova autenticação/acesso do worker.
2. ClaudeAdapter não aplica ainda a mesma allowlist granular; FAC-012D só restringe Codex. O consumer segue desligado, portanto a rota elegível e seus controles operacionais permanecem pendentes.
3. A pasta gravável é verificada antes do spawn, mas processos concorrentes poderiam alterar paths depois da verificação. Worktrees precisam permanecer sob writer único/lease/fencing já definidos; ainda requer integração do consumer real.
4. Piloto externo e credenciais/modelos continuam configurados pelo operador na interface, sem valores hardcoded. O piloto é escolha manual posterior.

## Rollback e aceite

Rollback lógico: reverter o commit `04e5f30` após interromper/impedir qualquer execução que dependa do novo contrato; não é preciso rollback de banco, pois não houve mudança de schema/migration. Não fazer rollback destrutivo automático.

Estado: `AWAITING_HUMAN`. A evidência permite revisão de código/testes, mas não autoriza `DONE`, execução com modelo, ativação do consumer, deploy ou push.
