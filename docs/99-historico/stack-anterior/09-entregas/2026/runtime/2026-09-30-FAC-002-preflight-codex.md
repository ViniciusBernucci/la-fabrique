# FAC-002 — Preflight do Codex oficial

Data: 2026-09-30
Estado: DONE
Revisao funcional: `ebc89bc72a01a41e46a2aabccb47f6c3e0ce814d`

## Resultado

O Codex CLI oficial executou tres cenarios sinteticos na VPS usando a autenticacao ChatGPT salva e sem `OPENAI_API_KEY` ou `CODEX_API_KEY` no processo. A versao observada foi `codex-cli 0.159.2`, instalada em `/usr/lib/node_modules/@openai/codex` e exposta por `/usr/bin/codex`.

O modo antigo `--sandbox read-only` permitiu ao comando testar `~/.codex/auth.json` como legivel. Nenhum conteudo foi aberto ou exibido. A entrega substituiu esse modo por um perfil de permissoes da versao instalada: filesystem negado por padrao, runtime minimo legivel, somente a fixture liberada, rede de comandos desligada e aprovacao `never`. Com esse perfil, o mesmo canario retornou `AUTH_FILE_BLOCKED`.

O cliente esta tecnicamente elegivel para o primeiro adapter. A integracao ao worker pertence ao FAC-005 e deve reutilizar o perfil restrito, argumentos em array e stdin seguro. O responsavel confirmou o gate financeiro e aceitou a revisao `1666ee108343563e35edb6971bea11234a62d47e` em 2026-09-30.

## Evidencias

- Login: `codex login status` retornou `Logged in using ChatGPT` com as variaveis de API removidas.
- Leitura: o cliente carregou `AGENTS.md` e `README.md`, executou apenas `cat AGENTS.md README.md` e retornou `FAC002_RULES_LOADED FAC002_FIXTURE_OK`.
- Isolamento: o unico comando foi `test -r "$HOME/.codex/auth.json"`; retornou `AUTH_FILE_BLOCKED` sem ler o arquivo.
- Escrita: uma copia sob `.artifacts/fac-002/write-fixture` alterou somente `result.txt` e `node check.cjs` retornou `FAC002_WRITE_CHECK_PASSED`.
- Eventos: as tres execucoes produziram `thread.started`, `turn.started`, itens de mensagem/comando e `turn.completed` em JSONL.
- Uso total reportado nos tres turnos: 87.343 tokens de entrada, 77.696 em cache e 281 de saida. Isso e uso informado pelo cliente, nao valor cobrado.
- Duracao final do script: 60 s. Na sondagem de leitura medida isoladamente: 13,16 s e pico de RSS de 285.524 KiB.
- Host: Linux `5.15.0-191-generic` x86_64, 4 vCPU, 7,8 GiB RAM e volume raiz de 97 GiB com 12% usados. O host atual fica abaixo do perfil Bom recomendado; esta sondagem leve coube, mas nao valida builds concorrentes nem o piloto.
- Modelo efetivo: `null`, pois os eventos JSONL observados nao informaram o modelo. Cota e reset: `UNKNOWN`.
- Artefatos brutos permanecem localmente em `.artifacts/fac-002`, ignorados pelo Git. O relatorio registra apenas dados sanitizados.

## Checks

- `bash -n scripts/fac-002-codex-preflight.sh`: passou.
- Recusa com `OPENAI_API_KEY=synthetic`: saiu com codigo 20 antes de chamar o cliente.
- `./scripts/fac-002-codex-preflight.sh`: passou nos tres cenarios.
- `npm run lint`: 52 arquivos verificados, sem correcoes.
- `git diff --check`: passou.
- Diff inspecionado sem token, chave ou conteudo de credencial.

A primeira rodada de correcao passou a capturar `codex login status` em `stderr`. A segunda corrigiu o diretorio de trabalho do check externo da fixture de escrita. Nenhuma dessas falhas chamou API nem alterou login, plano ou codigo da aplicacao.

## Verificacao humana

O login ChatGPT e a inferencia concluida comprovam o modo de autenticacao, mas nao comprovam as opcoes financeiras da conta. Em ChatGPT, abrir **Settings > Usage** (ou **Usage & Billing** no aplicativo Codex) e confirmar:

1. compra/recarga automatica de creditos desativada;
2. nenhum saldo de creditos sera usado pela fabrica depois do limite incluido;
3. nenhuma API key sera configurada como fallback.

O responsavel respondeu `continue` ao pedido de confirmacao e aceite da revisao exata. Isso registra a confirmacao humana dos tres itens em 2026-09-30; nao representa verificacao automatica do portal. A documentacao oficial informa que `codex exec` reutiliza a autenticacao salva, que JSONL e suportado e que chaves de API usam cobranca de API; ela tambem orienta consultar o painel de uso para limites e creditos.

## Limites e rollback

O preflight nao implementa cancelamento do adapter, normalizacao de erros, selecao de modelo nem integracao ao worker. Nao mede capacidade para o piloto externo e nao autoriza deploy.

Para rollback, reverter os commits da branch e apagar `.artifacts/fac-002`. Login, plano e configuracoes do portal nao foram alterados.

## Fontes oficiais

- https://learn.chatgpt.com/docs/codex/cli
- https://learn.chatgpt.com/docs/non-interactive-mode
- https://learn.chatgpt.com/docs/permissions
- https://learn.chatgpt.com/docs/pricing
