# DOC-MV-004 — Skills adaptadas para a La fabrique

2026-10-07 Europe/Berlin. Fontes adaptadas IMPLEMENTADAS / AWAITING_HUMAN; instalação física em .agents NÃO REALIZADA por read-only; descoberta automática NÃO VERIFICADA. Base de código 9439cde e DOC-MV-003 preservado na árvore. [Ticket](../../08-desenvolvimento/tickets/DOC-MV-004.md).

## Objetivo e comportamento

As 11 skills importadas continham fluxo de outro projeto. Adaptadas em skills/ com [perfil comum](../../../skills/00-perfil-la-fabrique.md), referências locais e papéis distintos. Backlog/tickets Git da fábrica substituem seleção/publicação externa. Stack React/Nest/worker TypeScript/PG/Redis/VPS, Manual Vivo numerado, dois ciclos de correção, um writer, assinaturas e gates locais governam o fluxo. Tenancy/broker/RLS continuam propostas. Nenhum subagente é criado automaticamente.

AGENTS/CLAUDE/ANTIGRAVITY e README apontam às fontes correntes e exigem leitura explícita se o catálogo oferecer a cópia importada. .agents/skills original não foi alterado; o ambiente não permite sua edição. Não afirmar instalação ou autoload; a ativação neste ambiente é fallback documental explícito. Sem configuração global ou links para contornar read-only.

Foram removidos das versões adaptadas perfis Windows/stack estrangeira, marcador de tarefa externa, publicadores ausentes, pausa obrigatória em ferramenta de worktree e gates artificiais de PR/publicação. Aprovações seguem autorização aplicável da sessão, sem confirmações repetidas de rotina. Relato de revisão não autoriza corrigir se não houver pedido; autorização existente de correção continua válida no escopo.

## Origem, revisão e preservação

[Manifesto](../../00-governanca/ADAPTACAO-SKILLS.json) identifica os 25 arquivos importados, preservados byte a byte em docs/99-historico/skills-importadas-2026-10-07. Referências específicas foram reescritas para a fábrica; materiais originais continuam disponíveis no snapshot. Atribuição original do procedimento de security-audit preservada; YAML de interface mantém campos e modo de invocação existentes.

[Checks](evidencias/DOC-MV-004/checks.json), [validação documental](evidencias/DOC-MV-004/validacao.json) e [patch editorial](evidencias/DOC-MV-004/editorial.patch) identificam o diff desta adaptação separadamente de DOC-MV-003. Não atribuir esta nova edição ao manifesto de revisão anterior. O diff desta tarefa usa a árvore DOC-MV-003 como baseline, reconstruída do patch anterior e conferida pelos hashes dos nove arquivos compartilhados; esses bytes também foram preservados no histórico.

## Validações e limites

Validador oficial skill-creator executado para as 11 skills. Scanner mudou para inventário de paths, sem snippets: fixture sintética verifica inclusão válida, omissão de credenciais/dependências/symlink e root inválido. Bash syntax e documentos/links/hashes conferidos. [Patch para instalação em .agents](evidencias/DOC-MV-004/instalacao-em-agents.patch) passou em git apply --check; NÃO APLICADO ao destino read-only. O patch rebasa links para a localização solicitada; as fontes canônicas editáveis continuam em skills/. Não houve scan de credenciais reais ou execução de auditoria operacional.

Revisão semântica própria dos papéis/autorizações/fontes; não é forward-test independente por nova sessão. Sem delegação, inferência adicional, instalação/provider/login/produção/migration/publicação/merge/commit. .git é read-only. Software de aplicação não foi alterado neste ticket; checks de software DOC-MV-003 não são reutilizados como evidência de funcionamento das skills. Nenhum ticket de software promovido a DONE.

## Uso, rollback e aceite

Abrir skills/00-perfil-la-fabrique.md e skills/<nome>/SKILL.md antes de aplicar o papel. Regras comuns continuam nas políticas; as skills não dão permissão de tool nem acesso a credenciais. Descoberta nativa precisa de sessão autorizada e ambiente compatível; não presumir registro automático de skills/ por existir pasta.

Rollback: reverter apenas a seção de catálogo/fontes correntes/validação desta entrega e retirar os arquivos skills/ depois de preservar alterações posteriores; originais .agents permanecem intactos. DOC-MV-003 e documentos históricos anteriores não precisam de rollback. Aplicada a lesson existente de documentação/proveniência; nenhuma lesson duplicada. Aceite humano da revisão exata e instalação física na pasta inicialmente pedida permanecem pendentes.
