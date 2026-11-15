# ADR-0061 — Repositório do projeto: clonar do GitHub, pasta existente ou criar do zero

- **Estado:** ACEITA · **Relacionadas:** [ADR-0060](0060-conexoes-por-oauth.md) (conta GitHub), [ADR-0005](0005-autenticacao-sessoes-e-cofre-de-conexoes.md) (cofre)

## Contexto
O cadastro só aceitava dono/nome de um repositório já existente no GitHub; o worker clonava sozinho, de forma temporária. O responsável quer partir de qualquer um destes pontos: **clonar** um repositório do GitHub, **usar uma pasta** que já tem o projeto, ou **criar do zero** (com ou sem repositório no GitHub).

## Decisão
1. **Três modos** na etapa Repositório (`projects.repo_mode`: `github`, `local`, `new`) e o caminho da pasta (`projects.repo_path`, relativo à raiz). Migration 00063.
2. **Raiz de workspaces** (`WORKSPACES_DIR`; no compose `/workspaces`, volume Docker na VPS e pasta do host em dev via `WORKSPACES_HOST_DIR`). A API e o worker a montam. **Nenhum caminho fora dela é aceito**: segmentos simples, sem `..`, e links simbólicos que saiam da raiz são recusados (`internal/workspace`). Sem a variável, `github` só salva (o worker clona como antes) e `local`/`new` ficam indisponíveis, com a mensagem de como habilitar.
3. **Operações** (`POST /api/projects/{id}/repository/prepare`, sessão + CSRF, só rascunho/suspenso, checa `expected_version` antes de tocar o disco):
   - `github`: lista os repositórios da conta (`GET /api/github/repos`) e clona (`git clone --single-branch --branch`) na pasta do projeto; pasta ocupada por outro repositório = 409; o mesmo repositório é reaproveitado.
   - `local`: valida que a pasta é um repositório git e deduz branch atual, dono e nome a partir do `origin`; avisa se não há commits ou remoto.
   - `new`: `git init -b`, README e commit inicial na pasta (que não pode existir com conteúdo). Com `create_remote`, cria o repositório no GitHub (conta ou organização) com as permissões da conexão e envia o primeiro push. Falha ao criar no GitHub desfaz a pasta; falha no push mantém a pasta e o repositório e diz como continuar (modo `local`).
4. **Git em ambiente limpo:** sem config global, sem prompt, sem hooks do repositório (`core.hooksPath=/dev/null`, `fsmonitor` desligado), `safe.directory` só para a pasta em uso; o token vai por cabeçalho no ambiente do processo (não em argv, URL nem `.git/config`) e é removido das mensagens. As imagens da API e do worker passam a ter `git`.
5. Criar repositório no GitHub e enviar o commit inicial é ação do próprio usuário (ele clica e escolhe a conexão); não pede reautenticação, como o restante da conexão GitHub (ADR-0060).
6. **Apagar projeto** deixa de exigir reautenticação (decisão do responsável): `DELETE /api/projects/{id}` passa a `protected` (sessão + CSRF). Continua recusando projeto ativo (409) e com histórico (409).

## Limites assumidos
- Dono, nome e branch continuam obrigatórios para ativar (a fábrica abre PRs no GitHub); pasta sem `origin` precisa de dono/nome informados ou de um repositório criado no GitHub.
- Só `github.com`. A fábrica ainda não usa a pasta como clone principal das worktrees (o caminho fica registrado para essa integração).
- Em dev a pasta do host é criada pelo container (root); na VPS o volume pertence ao usuário 10001 da imagem.
