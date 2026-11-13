# FAC-012AE — Eventos autenticados e retomáveis de projeto

Status: IMPLEMENTADO / AWAITING_HUMAN. Data: 2026-10-03. Baseline `f12b48a` (461 testes + 4 PostgreSQL). Branch `feat/fac-012ae-project-events`, worktree exclusiva `/home/vinicius/le-fabrique-fac-012ae`, checkpoint SHA/patch/untracked limpos, sem outro writer.

Objetivo: cumprir transporte SSE sanitizado/autenticado com sequência persistida/retomada e fallback HTTP do ADR-003. Painel deve atualizar tickets e execuções após mudanças sem reabrir projeto. Nenhum piloto ou provider é dependência.

Escopo: contratos de invalidation (IDs/tipo/seq/data, sem conteúdo de agente), tabelas/cursor por projeto e triggers transacionais PostgreSQL, API consulta limitada e SSE autenticado Bearer sem token em URL, cliente streaming cancelável/reconnect com cursor/limites e fallback HTTP atual, App/RunPanel e proxy nginx sem buffering nessa rota, testes API/parser/PostgreSQL, docs arquitetura/controle/operação/planejamento/lessons. Critérios: rollback não publica evento; sequência por projeto em ordem de commit, concorrência sem lacuna perdida; limit/cursor inválidos rejeitados; unauthorized nenhum read; reconexão não perde eventos nem aceita regresso de seq/projeto; limites/falha/cleanup não atualizam view abandonada; dados estritos/minimizados. Um writer, até duas rodadas de correção. Banco efêmero isolado, nenhum serviço existente/migration/deploy/provider/gasto. Rollback mantém dados/eventos e clientes HTTP compatíveis; aceite humano exato pendente.

Resultado: 9173532, 485 testes + 7 PostgreSQL, checks e nginx syntax passaram. Duas rodadas de correção, revisões finais íntegras. [Evidências](../../../09-entregas/2026/controle/2026-10-03-FAC-012AE-eventos-projeto.md). Sem aceite automático.
