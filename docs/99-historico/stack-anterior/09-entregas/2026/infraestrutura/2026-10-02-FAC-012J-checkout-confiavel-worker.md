# FAC-012J — Checkout confiável no worker

Status: implementado em branch isolada; aguardando aceite humano.

Revisão de código: `c587569c484387e9f821a6a4c0f4dcfe81127c36`.

## Mudança

O worker ganhou configuração opcional `WORKER_CHECKOUT_ROOT` (absoluta) e `WORKER_REPOSITORY_HOSTS` (lista de hostnames DNS exatos, sem wildcard/IP/porta). Sem root/host, a função falha fechada. `prepareRepositoryCheckout` aceita apenas UUIDs de projeto/workflow, URL HTTPS sem userinfo/query/fragment e SHA Git lowercase de 40 caracteres. O path final é derivado dos UUIDs; um lock exclusivo evita escritores concorrentes para o mesmo destino.

Antes de obter credencial, a função exige que `gh auth status --hostname <host> --json hosts` confirme conta ativa e `tokenSource=keyring`. Só então lê `gh auth token` em memória. O token não entra em URL, argv, log, retorno, job ou controle; Git o recebe via cabeçalho de autenticação host-scoped no ambiente do processo. Git roda com `shell:false`, configurações sistêmica/global desabilitadas, protocolos HTTPS-only, hooks nulificados, template vazio, LFS skip-smudge e sem recursão de submodules. Clone inicial não faz checkout; o worker busca o SHA, confirma o objeto commit, checkout detached, confirma `HEAD` e ausência de ref simbólica, e publica o diretório privado por rename.

Falha/timeout/cancelamento remove apenas o temporário desta operação; stdout/stderr brutos não são propagados. O processo de preparação permanece no worker confiável, fora do sandbox e não é invocado pelo consumer.

## Verificações

`npm test`: passou — 21 contracts, 40 runtime, 55 API, 80 worker, 5 web e 2 launcher (203 total). `npm run typecheck`, `npm run lint`, `npm run build` e `git diff --check`: passaram na rodada final. Um lint anterior apontou apenas ordenação de imports e regex de controle; ambas foram corrigidas e todos os checks foram repetidos.

Testes do módulo usam `CheckoutCommandRunner` sintético. Verificam allowlist de hosts, URLs com credencial/traversal, SHA exato e divergente, confirmação de keyring, ausência de token herdado, comandos Git e ambiente, caminho/permissões, lock/destino existente, detached HEAD, saída não propagada, limpeza, timeout e cancelamento. Não foi executado `git`, `gh`, conexão remota, keyring real ou clone remoto.

## Limitações operacionais

- `main.ts` continua consumindo `le-fabrique.execution` pelo probe. O módulo não está ligado ao queue consumer, `DeveloperWorkflow`, checkout profile vindo da Factory Settings nem leases/fencing/writer.
- Imagem Docker atual não contém Git/GH CLI, não monta o checkout root e não comprova keyring ou login sob identidade do worker. As duas variáveis não são encaminhadas pela composição. Assim, a capacidade está implementada como biblioteca/configuração, mas indisponível para uso operacional até provisionamento/versionamento dos CLIs, volume privado e preflight humano de identidade/keyring.
- Nenhum provider de IA, piloto, custo, banco/fila real, VPS ou deploy foi usado. Não valida isolamento do Developer CLI (FAC-012D) ou do sandbox sob identidade de serviço.

## Rollback e estado

Reverter a implementação FAC-012J remove a função e validação ambiental; nenhum dado persistente foi alterado. Checkouts concluídos seriam artefatos locais privados e não são apagados automaticamente pelo rollback. Não houve merge, push ou deploy. Ticket aguarda revisão/aceite humano da revisão exata.
