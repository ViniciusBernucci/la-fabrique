# FAC-006 — Context Builder e RuntimeGuard

Data: 2026-09-30
Estado: DONE
Revisao aceita: `ea8cf7afb0e55840722f306262b5334bb408b51a`, em 2026-09-30

## Resultado

O `packages/runtime` agora monta contexto deterministico a partir de fontes explicitamente selecionadas e aplica uma politica subscription-only antes de tentativas do runtime. Os contratos compartilhados validam requests, manifestos, omissoes, politicas, estado e decisoes com Zod.

## Funcionamento

- Fontes relativas sao ordenadas lexicalmente; cada entrada aceita registra papel, bytes e SHA-256.
- O hash do manifesto inclui revisao-base, fontes, omissoes, total e indicador de truncamento.
- Traversal, caminho absoluto, symlink, dependencia/gerado, segredo por nome ou conteudo, binario e limites produzem omissoes explicitas.
- `RuntimeGuard.authorizeAttempt` aplica tempo total, tentativas e trocas de provider.
- `recordFailure` pausa quando a mesma impressao de falha ocorre duas vezes consecutivas; sucesso limpa a sequencia.
- A politica aceita somente assinatura, budget de API zero, fallback desligado e extras desligados.
- O ambiente entregue pelo adapter Codex usa o sanitizador comum e remove chaves conhecidas de OpenAI, Codex, Anthropic, Google, Gemini e Azure OpenAI.

## Evidencias

Uma execucao local sobre tres arquivos reais desta revisao selecionou 7.511 bytes sem omissoes. O manifesto produzido foi `7aa5d3348626429d91819e9654c93e72ac89415a4b31beba474fc799fd97c229`; os tres arquivos receberam SHA-256 individual.

Na mesma prova, duas ocorrencias consecutivas de `TOOL_DENIED:profile` resultaram em `PAUSE/REPEATED_FAILURE`. Um ambiente sintetico contendo `OPENAI_API_KEY` e `ANTHROPIC_API_KEY` conservou apenas `PATH` e relatou somente os nomes removidos, sem valores.

## Checks

- `npm run lint`: 60 arquivos, passou.
- `npm run typecheck`: contratos, runtime, API, worker e web, passou.
- `npm test`: 39 testes em 11 arquivos, passou; 21 testes pertencem ao runtime.
- `npm run build`: contratos, runtime, API, worker e web, passou.
- `git diff --check`: passou.
- Diff sanitizado: nenhum valor de chave ou token foi incluido.

A primeira rodada encontrou somente formatacao e uma expectativa de ordem incorreta no teste; a segunda rodada passou.

## Limites

O chamador ainda seleciona as fontes; nao ha analise de imports, busca semantica ou persistencia de manifesto. A deteccao de segredo usa nomes e padroes fortes, mas nao substitui um secret scanner completo. RuntimeGuard e uma biblioteca pura: o worker ainda nao despacha jobs nem persiste seu estado. Timeout de processo continua no adapter, e claim, lease, fencing, sandbox e orquestracao pertencem aos tickets seguintes.

Nenhum provider foi chamado e nenhuma assinatura, API, credito ou extra usage foi consumido pelo FAC-006.

## Rollback

Reverter `61722dd9e4fc012ce155be5c85a36a4547967a2c` remove os contratos, o Context Builder, o RuntimeGuard e a integracao do sanitizador no adapter. Nao ha migration, dado persistente, login alterado ou deploy.
