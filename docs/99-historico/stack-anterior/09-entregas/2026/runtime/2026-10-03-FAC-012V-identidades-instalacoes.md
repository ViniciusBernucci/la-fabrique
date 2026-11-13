# FAC-012V — Identidades privadas por instalação

Data: 2026-10-03. Status: IMPLEMENTADO / AWAITING_HUMAN. Baseline `65be5ab`, READY `60e9397`. Código `22f6375522355bfd22150a1d8fa4c1375fced1d8`. Branch `feat/fac-012v-provider-identities`, worktree `/home/vinicius/le-fabrique-fac-012v`.

## Objetivo, implementação e funcionamento

Instalação escolhida pela configuração atual da interface agora determina identidade real do cliente, sem conta/modelo/piloto hardcoded. Login, status e runtime usam o mesmo ID/armazenamento oficial. `ProviderIdentityManager` exige Linux e WORKER_PROVIDER_ROOT absoluto, canônico, existente, privado e pertencente ao usuário efetivo. Não cria essa raiz nem muda permissões existentes. Cria filhos 0700 por provider/ID validado com home/store/cache, recusando symlinks, realpaths divergentes, donos/modos inseguros. Não lê/copia autenticação antiga nem apaga stores.

Ambiente do cliente é uma allowlist PATH/LANG/HOME/XDG e CODEX_HOME ou CLAUDE_CONFIG_DIR. Não herda tokens do controle, API keys, OAuth injetado, endpoints/cloud/NODE_OPTIONS/hooks. HOME/CODEX_HOME são configuração somente do subprocesso oficial, não mudança global. Binários são configuração operacional confiável, não paths livres do painel. Codex recebe `cli_auth_credentials_store="file"` e `forced_login_method="chatgpt"` no login/status/runtime; verificação exige indicação explícita de ChatGPT. API/saída ambígua não viram AVAILABLE. Claude usa diretório oficial distinto e mantém classificação first-party por assinatura; factory recusa WORKSPACE_WRITE antes de iniciar cliente, READ_ONLY exige preflight operacional.

`main.ts` compõe login/status/adapter com ID do job/rota; `ConfiguredAgentRouter` consulta configuração a cada chamada e aceita factory por rota, sem adapter global por provider no consumer real. Overload por mapa permanece nas bibliotecas/fixtures. Sem raiz, login/status falham fechados com resultado controlado; execução habilitada exige separação de checkout/execução. Antigravity unsupported/ERROR, sem fallback para identidade global.

## Fontes e compatibilidade

OpenAI Docs orientou CODEX_HOME/file/ChatGPT conforme [autenticação oficial](https://learn.chatgpt.com/docs/auth). Claude documenta contas por CLAUDE_CONFIG_DIR em [autenticação oficial](https://code.claude.com/docs/en/authentication); Console sem key não é elegível. Fontes consultadas em 2026-10-03. Versões anteriormente observadas no usuário interativo (`codex-cli 0.159.2`, Claude `2.1.285`) não provam identidade de serviço. Nenhum login/inferência oficial neste ticket.

## Diff e verificações

Diff sanitizado/revisável: `git show 22f6375522355bfd22150a1d8fa4c1375fced1d8 -- .env.example apps/worker/src infrastructure/systemd/worker.env.example`. Doze arquivos, 437 inserções/27 remoções; código e fixtures sintéticas, sem credenciais reais.

- `npm ci --ignore-scripts`: instalação privada da worktree, audit zero vulnerabilidades.
- `npm run db:generate`: Prisma Client local, sem conexão/migration.
- `npm run typecheck`: todos os workspaces passaram.
- `npm test`: 380 passaram (launcher 2, contracts 32, runtime 46, API 135, worker 150, web 15).
- `npm run lint`: 194 arquivos, passou sem fixes.
- `npm run build`: cinco workspaces passaram.
- `git diff --check`: passou.

Dez novos testes cobrem stores distintos/ambiente, raízes/IDs inválidos, canonicalidade/symlinks/modos, bloqueio Claude write, mudança de conta no mesmo provider e API Codex recusada. Subprocessos Node reais comprovam ambiente comum em runners status/login, sem segredos sintéticos; não são autenticação real. Testes anteriores de adapter/sandbox não foram renomeados como preflight deste ticket.

Primeiro typecheck isolado encontrou exports antigos porque o link inicial node_modules apontava ao build antigo da raiz. Diagnóstico: remover somente o link criado nesta worktree, instalar dependências independentes e reconstruir contratos/runtime; checks completos passaram. Não alterou builds/serviços da raiz. Worker inicial 148 testes; suíte final após testes extras 150. Sem rodada de correção de falha de código. Uma aplicação documental falhou por contexto de linha e foi reaplicada; nenhuma evidência de código invalidada.

## Limitações, rollback e aceite

Consumer false; sem login/cota, migração, serviço/banco/fila reais, piloto/push/deploy. Autenticar oficialmente na identidade nova é manual; stores antigos não são copiados. Claude login UI ainda não implementado e escrita granular bloqueada. Handoff/WAITING_PROVIDER integrados, docs técnicas e transporte maior continuam pendentes. Store 0700 não protege contra outros processos do mesmo UID fora do sandbox; preflight físico sob serviço obrigatório.

Rollback com worker comprovadamente parado: reverter código/config em ticket autorizado, preservando stores, sem transferir credenciais para identidade global. Nenhuma migration. Aceite da revisão exata pendente, não DONE.

## Docs, lessons e IA

Atualizados READMEs runtime/operação/infraestrutura/planejamento, ADR-002, README raiz, controle, ticket, índices/changelog/backlog e lesson de perfis. Sem chamada de IA dos providers da fábrica; modelo/tokens/custo do assistente não observáveis neste relatório. API/extras/fallback pago proibidos.
