# Entrega 2026-11-12 — E07: editor visual de workflows (T07-01)

**Ticket:** [T07-01](../NEW%20MVP/03-BACKLOG.md#t07-01). Decisões: [ADR-0009](../adr/0009-documento-de-workflow-e-validacao.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 7 (`workflows`).
- **Backend:** `internal/workflows` (documento, catálogo de 14 tipos de nó, validador, persistência) e 6 operações (`node-types`, `validate`, listar, criar, ler, salvar).
- **Front:** menu **Workflows**, lista e editor de canvas (paleta, arraste, conexões, propriedades, validação com localização), `biome.json` com override das regras de a11y semânticas só para o canvas SVG.

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| Validador: fluxo complexo com todos os controles e loop limitado válido; localização de nó/campo; 13 casos estruturais; ciclo sem loop; limites; catálogo | PASS |
| `make test-integration`: starter válido; fluxo complexo (13 nós, 14 conexões) salvo e reaberto **em outra sessão** idêntico; rascunho inválido salvo com problemas (nó, campo); versão obsoleta, limites e corpo enorme; 8 gravações concorrentes → 1 vencedor; sessão e CSRF; validar não persiste; rollback de 6 migrations | PASS |
| vitest (53 testes; 5 novos do editor), `tsc`, Biome | PASS (executado na cópia no disco interno, ver nota do E05) |
| Editor em navegador real | **NOT_RUN** |

## Aceite → evidência
- "Montar fluxo complexo, salvar e reabrir mantendo nós, conexões e propriedades": `TestComplexWorkflowSurvivesSaveAndReopen` (API) e o teste de tela que monta início → agente → fim, edita propriedades e confere o corpo salvo.
- "Exibir erro com nó e campo relevantes": `TestInvalidDraftIsSavedAndReportsNodeAndField` e o teste de tela com "Nó agent-1 · prompt", nó marcado "com pendências" e o erro junto do campo.

## Rollback
`git revert`. A migration 7 tem `Down` (descarta os rascunhos).
