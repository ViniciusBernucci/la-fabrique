# FAC-002 — Preflight do Codex oficial

Status: DONE

Implementacao tecnica: `ebc89bc72a01a41e46a2aabccb47f6c3e0ce814d`. Revisao aceita: `1666ee108343563e35edb6971bea11234a62d47e`. Os cenarios de leitura, isolamento e escrita descartavel passaram. O responsavel confirmou o gate financeiro e aceitou a revisao em 2026-09-30. Evidencias: `documentacoes/runtime/2026-09-30-FAC-002-preflight-codex.md`.

## Objetivo

Comprovar, na VPS e com fixture sintetica, que o Codex CLI oficial pode executar em modo nao interativo usando a autenticacao ChatGPT ja salva, sem chaves de API herdadas e com sandbox explicito.

## Escopo

- Cliente inicial: OpenAI Codex CLI instalado em `/usr/bin/codex` pelo pacote oficial `@openai/codex`.
- Fixtures permitidas: `fixtures/codex-preflight`, `fixtures/codex-preflight-isolation` e uma copia descartavel de `fixtures/codex-preflight-write` sob `.artifacts/fac-002`.
- Executor permitido: `scripts/fac-002-codex-preflight.sh`.
- Artefatos locais: `.artifacts/fac-002`, ignorados pelo Git e sanitizados antes de qualquer registro documental.
- Dados: somente marcadores sinteticos.

Ficam fora deste ticket: adapter integrado ao worker, execucao de projeto externo, mudanca de login/plano, API key, creditos, extra usage, autorecharge, merge e deploy.

## Criterios de aceite

1. Versao e origem do binario sao registradas.
2. `codex login status` confirma autenticacao ChatGPT com `OPENAI_API_KEY` e `CODEX_API_KEY` removidas do processo.
3. O script recusa a execucao se alguma dessas chaves estiver herdada.
4. `codex exec` conclui a fixture em JSONL, modo efemero, perfil de permissao que nega o filesystem por padrao, libera somente runtime minimo e leitura da fixture, desliga rede dos comandos e usa aprovacao `never`.
5. Os marcadores do `AGENTS.md` e do `README.md` aparecem na resposta, sem mudanca no workspace.
6. Um canario confirma que comandos no perfil nao conseguem ler `~/.codex/auth.json`, sem abrir ou exibir o arquivo.
7. Uma copia descartavel permite somente escrita no workspace, aplica uma alteracao minima e passa em `node check.cjs` sem rede.
8. Duracao, recursos e uso exposto pelo cliente sao medidos; modelo ou cota ausentes permanecem `UNKNOWN`/`null`.
9. O responsavel confirma separadamente que extras, creditos e autorecharge estao desligados na conta, porque o CLI nao comprova configuracoes de cobranca do portal.

## Baseline e limites

- Base: `developer` em `8214cec2790f4e51f7cad4aae490c8937f633932`.
- Branch: `feat/fac-002-codex-preflight` em worktree isolada.
- Provider elegivel para a sondagem: Codex autenticado por ChatGPT; elegibilidade final depende dos criterios acima.
- Uma execucao por vez, ate 30 minutos, sem escrita, rede solicitada ou comandos do projeto.
- Nao ler, copiar, registrar ou mover `~/.codex/auth.json`.

## Rollback

Remover a fixture, o script e este documento. Apagar `.artifacts/fac-002` remove somente evidencias locais do preflight; login e configuracao da conta nao sao alterados.
