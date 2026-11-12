# FAC-015 — Assinaturas autenticadas e catálogo compartilhado

Estado: AWAITING_HUMAN; baseline `a9aea9e`, commit funcional `6b8fe32`, implementação diretamente em `developer` por autorização do responsável. Provider elegível para inferência: nenhum liberado neste incremento. Sem execução concorrente de IA, prompts, migrations, API keys ou cobrança extra habilitada.

## Comportamento implementado

A autenticação pertence à conta do provedor, não a cada modelo. La fabrique utiliza os fluxos dos clientes oficiais sob identidades privadas por ID de instalação. O fluxo Codex device auth existente continua no painel; após conclusão de um login iniciado nessa tela, uma verificação é solicitada automaticamente. Também existe `Verificar conta e modelos` para atualizar manualmente.

O worker autentica status ChatGPT e inicia um app-server stdio isolado, inicializa o protocolo, reconfirma `account/read` com tipo `chatgpt` e consulta `model/list` paginado. Não cria threads nem turns. Filtra entradas ocultas, valida IDs em runtime, limita catálogo a 50 modelos, saída a 1 MiB, duração a 15 segundos e paginação/cursor repetido. Erros são mensagens controladas, sem stdout/stderr ou credenciais no controle.

A API incorpora os modelos observados em contas `AVAILABLE`, deduplica e preserva nomes manuais e o modelo padrão existente. Se não existe padrão, escolhe o primeiro modelo da lista resultante. Configuração é novamente validada em runtime e a versão incrementada na transação. Falha de descoberta conserva autenticação observada e catálogo anterior, com mensagem explícita; não comprova acesso a um modelo manual.

Polling do painel incorpora estado/modelos/padrão sem perder rótulos e atribuições em edição. O modal aberto recebe o catálogo observado. Funcionários digitais e suas alternativas consultam a mesma lista da instalação; salvar uma conta não pode restaurar um estado de autenticação antigo do modal. Roteamento real continua sujeito à disponibilidade, adapter, permissões e gates existentes.

Claude usa `npm run providers:login -- ID_DA_CONTA`: lê somente metadados do controle local autenticado, exige conta Claude salva e habilitada, prepara a mesma identidade do worker e executa `claude auth login --claudeai`. O usuário confirma o fluxo oficial em seu terminal/navegador; o processo não herda variáveis de autenticação/API do supervisor. Após término solicita verificação ao worker. Status oficial não retorna catálogo garantido: os nomes de modelos Claude continuam informados explicitamente pelo usuário. Antigravity permanece sem adapter de execução; não há integração de login nesta entrega.

## Preparação e teste humano

`npm run providers:setup` provisiona raiz canônica 0700 fora do repositório e acrescenta somente variáveis ausentes no `.env` para raiz e caminhos absolutos dos clientes. Não substitui valores existentes nem copia sessões pessoais. Nesta VPS foi executado; segundo uso confirmou idempotência. Reiniciar o processo dev é necessário para o supervisor transmitir as novas variáveis ao worker.

1. Na VPS, interrompa `npm run dev` com Ctrl+C e execute novamente `npm run dev`.
2. Abra Configurações e configure a conta Codex: habilite e salve. Clique em **Verificar conta e modelos** para atualizar eventual estado ERROR antigo.
3. Reabra o modal se necessário; com AUTH_REQUIRED clique em **Conectar assinatura Codex**, abra a URL oficial e confirme o código usando sua conta ChatGPT. Se device auth estiver desabilitado, habilite-o nas configurações de segurança/workspace do provedor.
4. Aguarde verificação automática. Confira AVAILABLE e lista de modelos. Caso tenha recarregado a página durante login, use **Verificar conta e modelos** manualmente.
5. Abra um funcionário digital, selecione a conta e um modelo da lista, ajuste limites/permissão, salve e recarregue para confirmar persistência.
6. Para Claude, salve/habilite a conta e rode na VPS `npm run providers:login -- claude-default` (ou o ID mostrado no modal). Complete login oficial da assinatura. Reabra configurações, cadastre IDs de modelos permitidos na assinatura e confirme estado com a verificação. Não use login Console/API.

Credenciais ficam no armazenamento privado do cliente na VPS, separadas por instalação. Não envie tokens/códigos privados ao chat. Persistência do catálogo não remove os limites de plano e não prova que uma inferência ocorreu.

## Evidências e limites

- `npm test`: 537 testes passaram (7 scripts, 45 contratos, 65 runtime, 179 API, 207 worker, 34 web). Os novos casos cobrem protocolo paginado sem inferência, conta API recusada, cursores repetidos, descoberta somente após auth subscription, erro de catálogo, persistência transacional e preservação de edição do painel.
- `npm run typecheck`, `npm run build`, `npm run lint`: passaram; lint mantém um warning anterior em `run-delivery.service.ts:209`.
- Clientes reais nas identidades privadas: Codex `codex-cli 0.159.2` e Claude `2.1.285 (Claude Code)`, ambos AUTH_REQUIRED, listas vazias. Não houve login nem inferência com a conta do usuário.
- GET local autenticado `/api/settings`: HTTP 200, versão 7 no instante da leitura; três contas desabilitadas. Conta Codex tinha ERROR antigo, logo requer verificar após restart.
- Catálogo sem autenticação recusado pelo caminho de cliente real. Catálogo autenticado real e navegação visual ainda dependem do usuário; testes de protocolo usam fixture local, não assinatura real.
- Nove migrations posteriores e duas attempts sem parada comprovada descritas em OPS-009 permanecem pendentes. `WORKER_EXECUTION_ENABLED=false` preservado. Login/catálogo não são liberação do writer ou teste ponta a ponta de execução.

## Fontes oficiais consultadas

- [Codex authentication](https://learn.chatgpt.com/docs/auth): login de assinatura e device auth.
- [Codex app-server](https://learn.chatgpt.com/docs/app-server): initialize, account/read e model/list.
- [Claude authentication](https://code.claude.com/docs/en/authentication) e [CLI reference](https://code.claude.com/docs/en/cli-reference): login oficial e consulta de status.

## Rollback e aceite

Reverter o commit funcional restaura a verificação/painel anteriores. Os nomes descobertos já persistidos podem ser editados no painel; não apagar credenciais privadas automaticamente. Para desfazer somente provisionamento, com dev parado, remover apenas as variáveis acrescentadas nesta operação e restaurar configuração anterior; diretórios/sessões ficam preservados. Nenhuma migration adicionada.

Aceite pendente da revisão exata, login humano e confirmação visual/persistência. Documentação de runtime e operação ligada no índice; lesson atualizada com segregação entre conta, catálogo e inferência.

Diff sanitizado: [FAC-015-diff.patch](../evidencias/configuracao/evidencias/FAC-015-diff.patch), formato sem linhas de contexto (`git apply --unidiff-zero` para inspeção/aplicação em baseline compatível); inclui somente arquivos rastreados do commit funcional, sem .env real, tokens ou arquivos dos clientes.
