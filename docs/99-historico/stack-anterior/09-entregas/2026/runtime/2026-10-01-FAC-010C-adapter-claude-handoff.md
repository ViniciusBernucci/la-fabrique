# Entrega FAC-010C — Adapter Claude e handoff seguro

Data: 2026-10-01. Status: DONE; aceita na revisao `0748f029a8a62ce891486dc5751c207bdd6e56db`.

## Revisoes e funcionamento

Base aceita e registro do FAC-010B: `0b1544dba3d8c2fef96fd61ac0ad816d312fd94c`; ticket READY: `8a3c1d660feb360883902286da04087657edc8c0`; codigo verificado: `3420e241644fa64d7c525939c3cfebf407518681`.

Os contratos de runtime agora aceitam `codex` e `claude`. `ClaudeAdapter` executa o cliente oficial por argv fixo e stdin, com `--output-format stream-json`, `--safe-mode`, `--restricted`, MCP estrito, sessao nao persistida, browser desligado, prompts de permissao negados e allowlist de ferramentas distinta para leitura e escrita. O adapter limita output/tempo, mantem uma execucao por vez, encerra a arvore em cancelamento e reduz JSONL a eventos, uso e resultado sanitizados. IDs de sessao de execucoes com falha nao sao retornados.

O ambiente remove chaves/tokens e seletores conhecidos de API, gateway, Bedrock, Vertex, Foundry e AWS. O classificador compartilhado so aceita Claude `loggedIn=true`, metodo explicitamente de assinatura e `apiProvider=firstParty`; status ambiguo, Console/API ou terceiro provider resulta `ERROR`. A mesma regra foi aplicada ao verificador FAC-010A, impedindo que um login pago por API seja marcado `AVAILABLE`.

`resolveAgentRoute` usa a conta, o modelo e a permissao escolhidos para cada funcionario no Centro de Configuracoes, exigindo instalacao habilitada e `AVAILABLE`. `ProviderHandoff` exige parada confirmada, evidencia terminal minima, snapshot posterior ao termino e mesma revisao-base; o RuntimeGuard autoriza a troca, uma nova worktree e criada e o snapshot e restaurado por hash antes do provider de destino. O prompt recebe objetivo, criterios e hash, nunca ID de sessao, mensagem bruta ou raciocinio do provider anterior.

## Fontes e evidencias

A implementacao foi confrontada com a referencia oficial da CLI Claude Code (<https://docs.anthropic.com/en/docs/claude-code/cli-usage>) e o guia oficial de setup/autenticacao (<https://docs.anthropic.com/en/docs/claude-code/getting-started>). A CLI local `2.1.285` confirmou `auth login --claudeai` como assinatura padrao e `--console` como uso com billing de API.

- `npm ci`: 211 pacotes; 0 vulnerabilidades reportadas.
- `npm run lint`: passou em 103 arquivos.
- `npm run typecheck`: passou em todos os workspaces e recompilou contracts/runtime antes dos consumidores.
- `npm test`: passou, 99 testes em 24 arquivos.
- `npm run build`: passou; bundle Vite produzido.
- `git diff --check`: passou.
- Diff funcional sanitizado: nenhum padrao de credencial encontrado.

As fixtures cobrem perfis read/write, status de assinatura explicito e ambiguo, API keys/modos cloud herdados, auth, cota, contexto, permissao, cancelamento, timeout, log limit, JSONL invalido, rota inelegivel, limite de troca, snapshot em base/tempo invalidos e ausencia de dados privados no handoff.

Nenhum login, prompt ou inferencia real foi executado. O unico probe real foi `claude auth status --json`, que permaneceu `loggedIn=false`; portanto Claude continua inelegivel operacionalmente. Nao houve API, credito, extra, migration, merge ou deploy.

## Limites e rollback

O coordenador e biblioteca injetavel e ainda nao substitui o probe do consumidor BullMQ: falta o perfil operacional allowlisted de repositorio/comandos e a persistencia do resultado do workflow. Antigravity continua sem adapter. O usuario ainda precisa configurar uma assinatura Claude App, confirmar extras desligados e executar preflight sintetico antes de qualquer uso real.

Rollback e a reversao do commit `3420e241644fa64d7c525939c3cfebf407518681`; nao ha migration de banco. Reverter remove adapter, contratos multi-provider e coordenador, restaurando Codex como unico runtime.
