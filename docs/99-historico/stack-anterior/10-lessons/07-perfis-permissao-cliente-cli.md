> Leitura: [← Anterior](06-sandbox-worktree-snapshot.md) · [Índice didático](../02-INDEX.md) · [Próximo →](08-lifecycle-processo-cli.md)

# Perfis de permissao para clientes CLI

## Conceito aplicado

Um sandbox chamado `read-only` pode impedir escrita sem esconder arquivos sensiveis. Isolamento de credenciais exige negar leitura fora do workspace e reabrir somente os caminhos indispensaveis ao runtime.

No FAC-002, o modo legado do Codex permitiu `test -r ~/.codex/auth.json`. A correcao usou um perfil de permissao com `:root = deny`, `:minimal = read`, raiz ativa em leitura ou escrita conforme o cenario e rede de comandos desativada. O canario passou de `AUTH_FILE_READABLE` para `AUTH_FILE_BLOCKED` sem abrir o arquivo.

## Exemplo do repositorio

`scripts/fac-002-codex-preflight.sh` passa o perfil por argumentos separados ao cliente. A fixture de leitura comprova carregamento de regras; a fixture de isolamento testa somente a permissao; a fixture de escrita roda numa copia descartavel sob `.artifacts`.

O cliente confiavel ainda precisa acessar sua autenticacao para falar com o fornecedor. A politica restringe os comandos gerados, que sao a superficie exposta ao codigo do projeto. Cada identidade real de worker deve repetir o teste, pois permissao observada em outro usuario ou container nao prova o ambiente final.

## Limite

FAC-012V separa seleção de conta e confinamento: `ProviderIdentityManager` deriva stores privados do ID escolhido na UI e fornece ambiente allowlisted comum ao login/status/runtime. `ConfiguredAgentRouter` chama factory por rota em vez de um adapter global por marca; testes com duas contas do mesmo provider provam essa distinção. Diretórios privados não isolam código do mesmo UID; escrita Claude permanece bloqueada até prova granular, sem supor que configurar conta significa sandbox comprovado.

Esse perfil nao substitui cgroups, timeout, kill da arvore de processos, worktree isolada ou bloqueio de sockets. Esses controles pertencem aos tickets de runtime e sandbox.

No FAC-010C, o perfil Claude combina `safe-mode`, `restricted`, MCP estrito, browser desligado, sessao nao persistida e prompts de permissao `none`. Read-only expoe apenas leitura/busca; workspace-write acrescenta somente Edit/Write. O ambiente remove tambem token/base URL e seletores AWS/Bedrock/Vertex/Foundry. Como configuracao administrada ainda pode existir fora do processo, elegibilidade exige `authMethod` de assinatura explicitamente reconhecido e `apiProvider=firstParty`; qualquer ambiguidade falha fechado.

FAC-012AA separa perfil e prova: `claudePermissionSettings` usa dontAsk/allow literal por caminho e nega metadados, mas `ProviderIdentityManager` exige prova privada ligada ao binário/UID/ID antes de escrita. `fac-012aa-claude-preflight.ts` correlaciona tentativas/results e verifica canários; ausência de tentativa não é negação provada, nem exit zero valida isolamento. Fixture positiva demonstra somente transporte e algoritmo de evidência; mudança de CLI invalida fingerprint, enquanto prova oficial permanece pendente.

## FAC-015: conta, catálogo e execução são evidências distintas

`codex-models.ts` exige account/read tipo chatgpt antes de model/list e nunca cria turn/thread. `provider-verification.service.ts` publica apenas modelos observados com estado AVAILABLE, preservando configuração manual. Um login válido sem catálogo deixa mensagem explícita, e um catálogo não libera gates do writer. No Claude, status confirma assinatura mas não lista modelos: não inferir entitlement de um ID digitado. `scripts/providers.ts` e ProviderIdentityManager usam o mesmo armazenamento privado por instalação, sem herdar API keys do shell. Fixture de protocolo comprova troca de metadados, não login real; observação real AUTH_REQUIRED demanda ação humana.
