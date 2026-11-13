# FAC-016 — Diagnóstico sanitizado da identidade

inspectProvider reconhece somente a exceção fixa Private provider root is required para informar setup/restart. Outras exceções continuam retornando falha genérica sem conteúdo bruto. O cliente real Codex retornou AUTH_REQUIRED após voltar a ter acesso à identidade privada; login/inferência permanecem ações separadas.

[Relatório, checks, diff e rollback](../configuracao/2026-10-05-FAC-016-feedback-verificacao-codex.md). Código ad78c84, AWAITING_HUMAN.
