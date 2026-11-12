> Leitura: [← Anterior](00-README.md) · [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../observabilidade-economia/00-README.md)

> Instruções de criação em documentacoes/ ou lessons/ abaixo são histórico da fonte; novas entregas/lessons seguem exclusivamente docs/00-governanca/POLITICA-DOCUMENTACAO.md.

> Nota editorial DOC-MV-001, 2026-10-07: conteúdo integral local preservado como referência da evolução v2.3. Afirmações por incremento têm a revisão/data original; trechos sobre consumer ausente, recuperação pendente e caminhos antigos não descrevem necessariamente HEAD a2cc5e0. O [AS-IS conferido](../../02-arquitetura/01-as-is.md) e a [auditoria](../../00-governanca/08-AUDITORIA.md) delimitam o estado atual; números de testes abaixo são evidência histórica, não reexecução desta migração.

# Centro de configuracoes

FAC-022 apresenta Configurações dentro do template padrão com menus persistentes e fontes compactas, incluindo os modais existentes. Rascunhos/salvamento/integrações preservados. [Relatório](../../09-entregas/2026/controle/2026-10-05-FAC-022-template-tipografia.md).

FAC-018: Login Claude/Antigravity e abas; implementação pronta, consentimento real e keyring Antigravity pendentes. [Relatório](../../09-entregas/2026/configuracao/2026-10-05-FAC-018-login-claude-antigravity.md).

FAC-017 reserva aba de autorização no clique Codex e navega para URL oficial ao receber desafio; mantém link manual/código e distingue sucesso confirmado de pedido iniciado. [Guia](../../09-entregas/2026/configuracao/2026-10-05-FAC-017-login-device-codex.md).

FAC-016 torna envio/espera/resultado das verificações visíveis no modal e trata falha de consulta. [Guia](../../09-entregas/2026/configuracao/2026-10-05-FAC-016-feedback-verificacao-codex.md).

FAC-015 integra catálogo Codex observado às contas e opções dos funcionários, com login oficial por instalação e setup privado. [Guia e limites](../../09-entregas/2026/configuracao/2026-10-05-FAC-015-contas-autenticadas-modelos.md).

FAC-014: funcionários digitais e GitHub também mostram resumo clicável com formulário em modal. Cancelar/Escape descarta edição da configuração; salvar persiste pela API versionada. Verificações/login/PR permanecem ações explícitas sobre configuração salva; fechar modal não desfaz pedidos enviados. [Evidências](../../09-entregas/2026/configuracao/2026-10-05-FAC-014-modais-funcionarios-github.md).
FAC-013: IA e contas usa lista compacta com nome/provider/habilitação/estado. Clicar em conta ou Adicionar abre modal nativo; Cancelar/Escape descarta campos, Salvar conta persiste configuração corrente e fecha após sucesso. Verificação/login operam sobre conta salva; remoção exige Salvar alterações. [Relatório](../../09-entregas/2026/configuracao/2026-10-05-FAC-013-modal-contas-ia.md).
Os checks do perfil de execução pertencem à definição de cada projeto, não às contas/rotas de IA. O operador aprova individualmente os checks autônomos e enumera arquivos de contexto; alteração de nome/comando/argv invalida a aprovação correspondente. O Centro de Configurações continua sendo a única fonte para instalação de clientes, contas e modelos.

Para checkout autenticado FAC-012J, apenas o GitHub CLI oficial sob a identidade do worker pode fornecer credencial efêmera; a origem precisa ser verificada como `keyring`. A integração futura deve reconciliar host/estado conectados nesta interface e nunca persistir token em configuração, job ou controle.

Status: Centro e integracao GitHub ACEITOS ate FAC-011D, revisao `76df60bbad4eac78b5d87fad8c2e79282355bf8c`; provas reais continuam futuras.

## Funcionamento atual

O painel administrativo possui as areas `Operacao` e `Configuracoes`. O Centro de Configuracoes organiza tres secoes:

- `IA e contas`: instalacoes Codex, Claude Code e Antigravity, com multiplas contas, rotulo, executavel, estado, ativacao, catalogo permitido e modelo padrao;
- `Funcionarios digitais`: Planner, Developer, Reviewer, QA, Documentation e Security, cada um com conta/modelo, modo de permissao, timeout, tentativas e ativacao proprios;
- `GitHub`: modo `GH_CLI`, estado da conexao, host, owner, repositorio, branch base e intencao de criar PR depois de gate humano.

A API autenticada expoe `GET /api/settings` e `PUT /api/settings`. A configuracao e um singleton PostgreSQL versionado; a atualizacao exige `expectedVersion` e rejeita gravacao obsoleta. Os contratos Zod estritos validam relacoes: IDs, modelos e funcoes unicos, modelo pertencente a uma instalacao habilitada, conta/modelo selecionados juntos e owner/repository do GitHub preenchidos juntos. Estados observados de provider e GitHub nao podem ser alterados pelo update administrativo: pertencem a evidencia futura do worker.

FAC-012E adiciona `GET /api/internal/worker-settings`, protegido por `WORKER_API_TOKEN`, que retorna apenas `version`, `observedAt` e a configuração validada. Não transporta credenciais, e versão `0` com defaults inativos representa a ausência de linha persistida sem fazer `upsert`. O worker valida a mesma resposta via `workerConfigurationSnapshotSchema`; esse bridge ainda não conecta o consumer nem seleciona/executa provider.

FAC-012F adiciona `ConfiguredAgentRouter`: em cada resolução ele busca o snapshot atual, aplica a atribuição por função e exige instalação `AVAILABLE`, modelo do catálogo e adapter compatível. Não mantém cache ou fallback. Reviewer com permissão de escrita falha fechado; nenhuma resolução inicia o cliente. Consumer ainda não chama o router.

## Seguranca e limites

Credenciais nao fazem parte do schema, do DTO, da tabela ou da interface. Fluxos de login suportados sao executados pelo worker confiavel usando o cliente oficial. O browser recebe somente desafios efemeros permitidos, metadados e estados sanitizados. Estado `AVAILABLE` nao pode ser inferido pelo cadastro: os defaults ficam desabilitados, `AUTH_REQUIRED` e sem modelos ate preflight real.

API paga, extra usage, creditos pagos, autorecharge, fallback pago e merge permanecem `false` por contrato. O painel os exibe como protecoes nao editaveis. Salvar configuracao nao executa clientes, nao conecta GitHub, nao cria PR e nao altera cobranca externa.

## Referencia funcional

O Orca inspirou a separacao entre agentes, contas e integracoes e a exibicao distinta de configuracao e disponibilidade. A implementacao e identidade visual sao proprias. A La fabrique nao adotou o padrao de permissao irrestrita do Orca: escrita continua explicita por funcao e o runtime mantem sandbox e gates conservadores.

## Proximos incrementos

1. Worker detecta instalacoes e executa login oficial sem transportar segredo pelo painel.
2. Preflight grava versao, modelos realmente acessiveis, estado e evidencia.
3. FAC-010 consome atribuicoes validadas para roteamento e handoff.
4. Instalar `gh`/keyring, repetir as provas reais e validar um PR draft controlado; merge continua manual.

## Verificacao de instalacoes

FAC-010A implementa `Verificar instalacao`. Pedido/outbox sao persistidos e o worker executa apenas comandos allowlisted, nunca o caminho editavel no painel. Historico mostra estado, versao e modelos sanitizados; output bruto nao e armazenado. Essa verificacao nao faz login nem torna uma conta elegivel quando o cliente informa `AUTH_REQUIRED`.

## Login efemero do Codex

FAC-010B implementa `Conectar assinatura Codex` para instalacao habilitada em `AUTH_REQUIRED`. A sessao e a intencao ficam no PostgreSQL/outbox, mas URL e codigo temporarios existem somente no Redis privado por ate dez minutos e no estado em memoria da tela. O worker executa `codex login --device-auth`, nunca o executavel editavel, e confirma com `codex login status` antes de atualizar o estado observado.

O painel nao recebe senha, token, cookie ou cache de autenticacao. FAC-018 acrescenta login gerenciado Claude/Antigravity; ver limitações operacionais no relatório atual. Detalhes e evidencias: [2026-10-01-FAC-010B-login-efemero-codex.md](../../09-entregas/2026/configuracao/2026-10-01-FAC-010B-login-efemero-codex.md).

## Rota por funcionario

FAC-010C consome a escolha de conta, modelo e permissao feita em `Funcionarios digitais`. A rota so existe quando a atribuicao esta habilitada, a instalacao esta habilitada e `AVAILABLE`, o modelo pertence ao catalogo e o provider possui adapter. Claude Console/API, autenticacao ambigua e Antigravity sem adapter falham fechado; nenhuma troca silenciosa substitui a escolha configurada.

FAC-012G conecta essa configuração ao workflow: Developer e Reviewer consultam suas funções separadamente em cada chamada, usando a conta/modelo selecionados na interface. O router não troca para outra conta se a opção estiver indisponível; a execução real ainda não está ligada à fila.

## Verificacao GitHub CLI

FAC-011A adiciona `Verificar GitHub CLI`. Pedido e outbox sao atomicos; a API nao executa cliente. O worker usa o binario fixo `gh`, remove variaveis de token e consulta somente `--version` e `auth status --hostname <host> --json hosts`. O painel recebe estado, versao e mensagem controlada, nunca JSON bruto, login, scopes ou origem/token.

`CONNECTED` exige uma conta ativa com estado `success` e `tokenSource=keyring`; host sem conta ativa vira `AUTH_REQUIRED`; CLI ausente, armazenamento inseguro, timeout ou resposta invalida viram `ERROR`. Alterar o host durante um job invalida somente aquela verificacao. O ambiente atual nao possui `gh`, portanto a integracao real permanece nao verificada e nenhum login foi iniciado.

## Login efemero GitHub

FAC-011B adiciona `Conectar GitHub CLI` quando a verificacao informa `AUTH_REQUIRED`. Sessao/outbox guardam apenas metadados; URL e codigo do device flow ficam no Redis privado por ate dez minutos e na memoria da tela. O worker usa somente o fluxo `--web` oficial, HTTPS, sem PAT, clipboard, chave SSH ou armazenamento inseguro solicitado.

A conclusao exige nova leitura de status: conta ativa `success` e `tokenSource=keyring`. Fallback do cliente para `hosts.yml` aparece como `PLAINTEXT_FILE/ERROR` e tambem nao passa pela verificacao posterior. O controle nunca le ou remove token. Detalhes: [2026-10-01-FAC-011B-login-efemero-github.md](../../09-entregas/2026/configuracao/2026-10-01-FAC-011B-login-efemero-github.md).

## Verificacao do repositorio GitHub

FAC-011C adiciona `Verificar repositorio salvo (somente leitura)` para configuracao persistida `CONNECTED`. A API cria snapshot/outbox e o worker usa duas requisicoes autenticadas `GET` pelo `gh api`: metadados/permissao do repositorio e existencia da branch base. Somente identidade, `permissions.pull=true` e branch exatas produzem `READABLE`.

O painel recebe apenas owner/repositorio canonicos, branch padrao/base, visibilidade, arquivamento e mensagem controlada. Falhas descartam observacao parcial. Nao ha clone, leitura de arquivo, escrita ou teste de PR. Detalhes: [2026-10-02-FAC-011C-verificacao-repositorio-github.md](../../09-entregas/2026/configuracao/2026-10-02-FAC-011C-verificacao-repositorio-github.md).

## Pull request sob gate humano

FAC-011D separa `Preparar` de `Aprovar e enviar ao worker`. Preparacao grava payload/version/digest sem outbox; aprovacao exata reconfere opt-in e leitura antes de publicar. O worker exige `permissions.push`, reconcilia PR aberto e so depois usa `gh pr create` com argv fixo e body por stdin. Draft e configuravel por pedido; merge permanece impossivel neste fluxo. Detalhes: [2026-10-02-FAC-011D-criacao-pull-request-gate-humano.md](../../09-entregas/2026/configuracao/2026-10-02-FAC-011D-criacao-pull-request-gate-humano.md).

FAC-019: abas Contas, Integrações IA, Equipes com modais preservados. [Relatório](../../09-entregas/2026/configuracao/2026-10-05-FAC-019-abas-configuracoes.md).


FAC-023: Equipes agora contém Agentes/Skills. Registro de múltiplos agentes e skills por projeto persiste no JSON versionado existente; vínculos validados e limpos na exclusão. Funções operacionais existentes preservadas; registro não instala nem executa skills automaticamente. [Relatório atual](../../09-entregas/2026/configuracao/2026-10-05-FAC-023-agentes-skills-projeto.md).


FAC-026: seis agentes pré-configurados visíveis junto dos personalizados, sem agrupamento Funções de execução. Switch por agente persiste enabled; clique explica papel/fluxo e botão abre ajustes existentes. Modal distingue funções integradas e não automáticas; ativar não executa jobs. [Relatório atual](../../09-entregas/2026/configuracao/2026-10-06-FAC-026-agentes-preconfigurados.md). FAC-027 adiciona menu lateral recolhível persistente nas telas.
