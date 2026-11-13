# FAC-012T — Retomada explícita do snapshot

Status: AWAITING_HUMAN. Código `42e7037f81c2fe19cefee85bb64c67373b29795f`; 355 testes/checks passaram. Data: 2026-10-03. Baseline `93735b0`; 336 testes/checks FAC-012S. Branch/worktree: `feat/fac-012t-resume`, `/home/vinicius/le-fabrique-fac-012t`.

Objetivo: operador solicita retomada de tentativa terminal/parada com resultado/bundle/checkpoint íntegros. Outbox congela snapshot/origem e nova intenção; claim emite fence novo somente para essa intenção e preserva histórico. Worker mede baseline limpo, restaura snapshot verificado, reconstrói contexto e continua objetivo original; não reclassifica regressão preservada como preexistente.

Caminhos: contracts, API orchestration, worker workflow/consumer, runtime restauração segura, UI e testes; docs controle/operação/runtime/handoff/planejamento/lessons. Sem migration aplicada, serviços/banco/fila/provider real, piloto/login/push/deploy. Um writer isolado, fixtures; duas correções antes de checkpoint. Contas/modelos atuais da UI, nenhum hardcode/fallback pago.

Critérios: contrato estrito/expectedVersion/attempt/hash; stop e evidência exigidos, unknown/terminal humano recusados; intenção/outbox atômicas e replay sem outro writer; claim apenas intenção correspondente, fence novo, antigo rejeitado; baseline antes do restore/contexto depois, patch/untracked íntegros e symlinks recusados; botão com confirmação, histórico preservado; testes/checks/documentação exata e AWAITING_HUMAN.

Limites: snapshot até 64 KiB; nova tentativa humana tem orçamento próprio explícito. Crash sem prova e recuperação apenas de upload pertencem incremento separado, nunca presumir processo morto. Rollback de código com writer parado; preservar dados/evidências e outbox pendente.
