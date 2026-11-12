# DOC-MV-002 — Organização da raiz e retirada de documentacoes

2026-10-07 Europe/Berlin. IMPLEMENTADO / AWAITING_HUMAN. Base developer 5845754. Pedido explícito do usuário para organizar Markdown da raiz e eliminar documentacoes após preservar seu conteúdo. [Ticket](../../08-desenvolvimento/tickets/DOC-MV-002.md).

## Objetivo e funcionamento

A raiz conserva README.md, AGENTS.md, CLAUDE.md e ANTIGRAVITY.md como pontos de entrada. Backlog real completo em [desenvolvimento](../../08-desenvolvimento/09-backlog.md); changelog completo em [docs](../../04-CHANGELOG.md); [guia de integração](../../00-governanca/03-GUIA-DE-INTEGRACAO.md) em governança. Piloto/plano/controle/matriz, especificação/fontes e prompt já tinham fontes canônicas; seus atalhos redundantes foram removidos.

Guias dos três agentes apontam explicitamente políticas, piloto, backlog, changelog, templates, prompt e guia atuais. Proíbem recriar os arquivos de apoio na raiz e a pasta legada. A ponte .agents existente continua válida, sem edição em área read-only. Autoload continua NÃO VERIFICADO; ler guias/políticas explicitamente é fallback obrigatório.

## Preservação, origem e diff

[Mapa/hash dos dez arquivos da raiz](../../00-governanca/REALOCACAO-RAIZ.json) preserva versão anterior íntegra em docs/99-historico/raiz-5845754. Somente links foram rebaseados ao mover backlog/changelog; IDs, aceites e relatos prévios preservados. Guia recebe atualização factual sobre integração e pacote removido, sem alterar original histórico.

[Mapa/hash da retirada de documentacoes](../../00-governanca/REMOCAO-DOCUMENTACOES.json): 11 atalhos preservados em compatibilidade-5845754; 138 evidências comparadas byte a byte com destinos existentes e snapshots originais. Nenhuma evidência exclusiva foi descartada. Links editoriais atuais, inclusive os dos relatos, apontam aos destinos novos; snapshots/patches/logs históricos permanecem íntegros. Mapas originais mantêm source/hash e atualizam apenas destination. Inventário novo suplementa a migração anterior.

O validador agora confere também os snapshots adicionais, destinos, ausência de arquivos redundantes da raiz e ausência de documentacoes. O painel altera exclusivamente o exemplo de paths do campo de documentação; não modifica configuração de projeto salva. [Diff](evidencias/DOC-MV-002/editorial.patch) e [checks](evidencias/DOC-MV-002/checks.json) descrevem esta revisão de trabalho. O manifesto REVISAO.json anterior continua referente à entrega DOC-MV-001, não a este novo diff.

## Verificação, riscos e limites

[Resultado estrutural](evidencias/DOC-MV-002/validacao.json) registra links/anchors/fences, índice, fontes e hashes. [Logs/checks](evidencias/DOC-MV-002/checks.json) registram somente checks efetivamente executados. Validador não comprova eficácia operacional ou veracidade semântica por si só. Nenhuma sessão adicional de provider, banco, produção ou render Mermaid foi executada.

Permissões atuais deixam .git read-only: alteração no diretório principal autorizado, sem branch/worktree/commit/merge nesta etapa. Diff segue revisável; revisão independente e aceite humano pendentes. Nenhum ticket de software promovido a DONE. Não há nova lesson: aplicada a lesson existente de [documentação e proveniência](../../10-lessons/12-documentacao-proveniencia.md).

## Rollback

Restaurar os dez arquivos da raiz a partir de raiz-5845754 e os onze atalhos de compatibilidade-5845754; as 138 evidências originais estão no snapshot repo-a2cc5e0. Reverter também os links/mapas/políticas/validador e o exemplo do painel usando o diff e baseline, preservando edições posteriores. Nenhum rollback de banco/serviço/auth é necessário. Sem reset destrutivo ou overwrite de trabalho posterior.
