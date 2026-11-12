# FAC-011 — Centro de configuracoes e atribuicao de agentes

Status: DONE

## Objetivo

Criar a fundacao administravel do Centro de Configuracoes para cadastrar instalacoes de clientes oficiais, declarar modelos permitidos, escolher provider e modelo por funcao da fabrica e manter metadados da integracao GitHub, sem expor ou persistir segredos no painel.

## Contexto e referencia

O responsavel determinou que contas, agentes, modelos e GitHub sejam configuraveis no software antes do segundo provider. O Orca foi usado como referencia funcional para organizacao por secoes, deteccao/ativacao de agentes, contas e integracoes; codigo, marca e identidade visual nao serao copiados. A La fabrique preserva seus limites mais conservadores: permissao ampla nao e padrao, credenciais ficam no runtime confiavel e APIs/extras pagos continuam bloqueados.

## Escopo permitido

- Contratos Zod compartilhados para instalacoes, modelos, atribuicoes por funcao e GitHub.
- Persistencia PostgreSQL e API administrativa autenticada para leitura e atualizacao da configuracao.
- Painel React responsivo com secoes `IA e contas`, `Funcionarios` e `GitHub`.
- Catalogo inicial de Codex, Claude e Antigravity em modo `SUBSCRIPTION_CLI`.
- Selecao de instalacao e modelo permitido para Developer, Reviewer, Planner, QA, Documentation e Security.
- Estado explicito de autenticacao/conexao e avisos de politica financeira.
- Testes de contrato, servico e view model; documentacao e evidencias.

## Fora do escopo

- Executar login, copiar OAuth/token, receber segredo no browser ou persistir credenciais no PostgreSQL.
- Descobrir modelos por chamada real, executar providers, trocar conta ativa ou implementar handoff.
- Conectar GitHub OAuth/PAT, criar PR, merge, deploy ou alterar repositorios remotos.
- Habilitar API, extra usage, creditos, autorecharge, fallback pago ou permissao irrestrita.
- Replicar integralmente a interface, codigo ou assets do Orca.

## Criterios de aceite verificaveis

1. A API autenticada retorna e atualiza uma configuracao validada em runtime, com controle otimista por `version`.
2. Cada instalacao registra provider, rotulo, comando, estado, modo de autenticacao, modelos permitidos e modelo padrao; nenhum campo aceita segredo.
3. Cada funcao seleciona uma instalacao habilitada e um modelo pertencente a ela, ou permanece sem atribuicao; referencia invalida e rejeitada atomicamente.
4. GitHub registra somente modo `GH_CLI`, host, owner/repository/base branch e estado; tokens, cookies e chaves nao fazem parte dos contratos nem das respostas.
5. O painel permite editar as tres secoes, mostra estados desconhecidos/desconectados e deixa API, extras pagos, autorecharge e fallback visivelmente desligados e nao editaveis.
6. A configuracao inicial nao afirma login nem modelos observados: todos os providers comecam `AUTH_REQUIRED`, sem modelos, ate preflight posterior.
7. Testes cobrem payload invalido, concorrencia obsoleta, atribuicao inconsistente, ausencia de segredos nos DTOs e renderizacao do resumo.
8. Lint, typecheck, testes, build e `git diff --check` passam na revisao entregue.

## Baseline, caminhos e checks

- Base: `0186bcf5420d941b50f38e4a77f119aaa8faa678`.
- Branch/worktree: `feat/fac-011-settings-center` em `/home/vinicius/le-fabrique-fac-011`.
- Caminhos principais: `packages/contracts`, `apps/api`, `apps/web`, schema/migration Prisma e documentacao afetada.
- Checks: `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`, `git diff --check`.

## Risco, provider e limites

Risco R3 por modelar contas e integracoes, mitigado pela exclusao estrutural de segredos e ausencia de login neste incremento. Codex CLI `0.159.2` e o unico provider previamente aceito, mas nenhuma chamada de IA sera feita. API e orcamento adicional permanecem zero. Um writer; ate duas rodadas de correcao; sem merge/deploy.

## Dependencias e sequenciamento

FAC-003 e FAC-009 aceitos. FAC-010 permanece `WAITING_PROVIDER` e passa a depender desta fundacao para receber configuracao administrativa antes do preflight e do adapter secundario.

## Aceite

A entrega foi aceita explicitamente pelo responsavel em 2026-10-01 na revisao documental exata `e41ec1e45273aba3205f266e8753dfca305c5952`.
