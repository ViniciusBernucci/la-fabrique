# FAC-010 — Preflight do segundo provider

Data: 2026-10-01
Estado: WAITING_PROVIDER
Baseline: `485ae88a772a499f6fc8034f5ea3df0e80c32ceb`

## Evidencia local

- `claude --version`: Claude Code `2.1.285`.
- `claude --help`: confirma `-p/--print`, `json|stream-json`, JSON Schema, sessoes, modos de permissao, `--restricted`, allow/deny de ferramentas e limite de budget para uso de API.
- `claude auth status --json`: `loggedIn=false`, `authMethod=none`, `apiProvider=firstParty`.
- `agy --version`: `1.2.14`.
- `agy --help`: confirma print nao interativo, JSON/stream-json, JSON Schema, modo plan/accept-edits e sandbox.
- `agy models`: falhou antes de listar modelos com pedido de login.
- O ambiente nao expoe nomes de variaveis `ANTHROPIC_*`, `CLAUDE_*`, `GOOGLE_*`, `GEMINI_*`, `OPENAI_API_KEY` ou `CODEX_API_KEY`.

Nenhum prompt foi enviado. Nao houve consumo de assinatura, API, credito ou extra usage.

## Elegibilidade

Nenhum segundo provider esta elegivel hoje. A documentacao oficial da Anthropic diferencia Claude App Pro/Max, que inclui Claude Code, de Anthropic Console, que exige billing de API. Para a politica deste repositorio, somente login oficial pela conta Claude App com assinatura confirmada pode ser considerado; Console/API, gateway, Bedrock e Vertex permanecem proibidos.

Claude Code e o candidato preferencial para retomar porque a versao instalada expoe saida estruturada e controles de permissao adequados ao papel de Reviewer. Isso nao e ativacao nem prova de plano/modelo/cota.

Fontes oficiais consultadas:

- https://docs.anthropic.com/en/docs/claude-code/getting-started
- https://docs.anthropic.com/en/docs/claude-code/cli-usage
- `FONTES.md`, com referencias oficiais de Claude Code e Antigravity ja adotadas pelo projeto.

## Gate para retomada

Este gate manual foi substituido pela decisao posterior do responsavel de configurar contas somente pelo software:

1. Aceitar o Centro de Configuracoes FAC-011.
2. Implementar acao de onboarding que inicia o login oficial sob a identidade do worker, mantendo a interacao sensivel no fluxo do cliente e fora dos DTOs/logs.
3. O responsavel escolhe no painel a instalacao e confirma plano/extras; a fabrica nao preenche nem armazena credencial.
4. O preflight repete status, lista capacidades/modelos realmente expostos e roda fixture sintetica de leitura antes de qualquer escrita.
5. Claude ou Antigravity so se tornam elegiveis pela evidencia real; nenhuma selecao cadastrada equivale a autenticacao.

Nao copiar arquivos de credencial, cookies ou tokens entre usuarios. Nao definir chave API para desbloquear o ticket.

## Rollback

Este checkpoint original alterou somente documentacao de estado. FAC-011 adiciona configuracao administrativa em revisao separada, ainda sem login. Reverter o checkpoint nao cria elegibilidade; nenhum login, configuracao externa do fornecedor ou chamada foi executado.
