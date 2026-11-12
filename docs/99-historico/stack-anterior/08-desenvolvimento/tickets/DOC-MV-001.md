# DOC-MV-001 — Migração inicial do Manual Vivo

Status inicial READY pelo pedido explícito de 2026-10-07; risco R0, edição documental reversível. Estado final de revisão na entrega. Baseline developer a2cc5e0; única mudança prévia reorganizacao-documental/ untracked. Branch/worktree exclusiva docs/manual-vivo-inicial.

Objetivo: mesclar pacote com fontes reais, preservar bytes/proveniência/seções, explicar AS-IS/alvo, migrar navegação/governança/relatos/lessons. Escopo docs, guias/regra, prompts e exemplos de paths do gate. Não implementar backlog, trocar stack, instalar providers, ler credenciais ou executar produção/migration/deploy.

Critérios: destinos de documentos/seções fechados, snapshots/hashes íntegros, políticas comuns explícitas, C4 atual/alvo/canais corretos, módulos/contratos vinculados ao código, backlog/aceites preservados, links/anchors/fences/gate/documentação verificáveis. Checks: python3 scripts/validar-documentacao.py; git diff --check; typecheck/testes web para exemplos de UI; lint pertinente; inspeção PDF/Mermaid com limite explícito.

Um writer na worktree documental; serviço ativo de origem não tocado. Provider desta sessão Codex; modelo/uso/billing não observados, nenhum processo de cliente/inferência adicional autorizado para autoload. Até duas correções; checkpoint/diagnóstico se falha persistir. [Entrega](../../09-entregas/2026/2026-10-07-DOC-MV-001-migracao-manual-vivo.md) registra revisão/checks/risco/rollback/aceite.
