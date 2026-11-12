> Leitura: [← Anterior](04-CHECKLIST-ENTREGA.md) · [Índice didático](../02-INDEX.md) · [Próximo →](06-fontes-tecnicas.md)

# Carregamento das instruções dos agentes

## Codex — orientação histórica do pacote

AGENTS.md de raiz é ponto de entrada. A descoberta considera overrides e hierarquia até o diretório de trabalho; há limite combinado de conteúdo. Um link para política não é inclusão automática: o wrapper exige leitura explícita. Reiniciar sessão e confirmar fontes carregadas após integração.

O pacote registra consulta em 2026-10-06 (não reexecutada nesta migração): [Custom instructions with AGENTS.md](https://learn.chatgpt.com/docs/agent-configuration/agents-md). A documentação pública descreve descoberta; a eficácia na versão local deve ser testada. Nenhuma configuração global foi alterada.

## Claude e Antigravity

As referências históricas de setembro estão no [catálogo de fontes](06-fontes-tecnicas.md). Este trabalho não revalida todo modo/plano/CLI nem promete carregamento nativo. CLAUDE.md deve ser verificado no modo instalado; ANTIGRAVITY.md é guia explícito e .agents/rules/documentacao.md é ponte a verificar, não regra supostamente universal. Injeção explícita de conteúdo é fallback editorial, sem conceder mais permissões.

## Teste de adoção no repo real

Abrir nova sessão no root e pedir: “Liste os arquivos de instrução ativos, leia as duas políticas e diga o destino de uma entrega, de um contrato e de uma lesson.” Conferir com arquivos/logs de carregamento quando disponíveis, sem ler auth. Resultado esperado: docs/09-entregas, docs/05-contratos e docs/10-lessons condicional; nenhuma nova política duplicada. Autorrelato do agente é indício de leitura, não gate técnico da fábrica.

## Observação local 2026-10-07

Somente --version executado: codex-cli 0.159.2; Claude Code 2.1.285; agy 1.2.17. Instruções hierárquicas de raiz/.agents e fixtures inventariadas; sem AGENTS ancestral localizado em /home/vinicius, /home ou /. Nenhuma nova sessão de inferência/cliente executada: disponibilidade de binário não comprova provider elegível, cobrança de assinatura/extra desligado ou leitura das políticas. O escopo documental não altera autenticação/permissões para testar autoload. Resultado NÃO VERIFICADO por agente.

Fallback obrigatório: abrir explicitamente AGENTS/CLAUDE/ANTIGRAVITY e docs/00-governanca/01-POLITICA-IA.md + POLITICA-DOCUMENTACAO.md no job; registrar SHA-256/config e omissões antes de editar. A versão instalada pode carregar hierarquia/override específicos; confirmar numa sessão autorizada, sem ampliar permissões nem ler auth.

## Destinos após DOC-MV-002

AGENTS/CLAUDE/ANTIGRAVITY continuam na raiz. Todos exigem ler políticas e apontam explicitamente docs/08-desenvolvimento/09-backlog.md, docs/04-CHANGELOG.md e docs/00-governanca/03-GUIA-DE-INTEGRACAO.md, além de piloto e templates. Os atalhos de apoio da raiz e a pasta documentacoes foram retirados. Ponte .agents permanece válida; leitura automática continua NÃO VERIFICADA.

## Skills adaptadas para a fábrica

[Perfil comum](../../skills/00-perfil-la-fabrique.md) e skills/ contêm as versões correntes adaptadas. .agents/skills veio de outro projeto e está read-only neste ambiente; suas cópias não foram sobrescritas. Os guias de raiz exigem abrir a versão adaptada mesmo quando o catálogo da sessão expõe uma cópia importada. Não se afirma autoload ou registro automático; leitura explícita é o fallback.

| Papel | Fonte corrente | Resultado esperado |
|---|---|---|
| arquiteto-de-software | [arquiteto-de-software](../../skills/arquiteto-de-software/SKILL.md) | AS-IS/alvo, contratos e decisão proposta |
| tech-lead | [tech-lead](../../skills/tech-lead/SKILL.md) | Tickets locais e prioridades verificáveis |
| worktree-planner | [worktree-planner](../../skills/worktree-planner/SKILL.md) | Isolamento Git e sequência de integração |
| implementador-de-ticket | [implementador-de-ticket](../../skills/implementador-de-ticket/SKILL.md) | Incremento no escopo do ticket READY |
| orquestrador-fluxo-ia | [orquestrador-fluxo-ia](../../skills/orquestrador-fluxo-ia/SKILL.md) | Próxima fase e estado local com evidências |
| fechar-entrega | [fechar-entrega](../../skills/fechar-entrega/SKILL.md) | Relato, checks e commit quando autorizado |
| revisor-de-codigo | [revisor-de-codigo](../../skills/revisor-de-codigo/SKILL.md) | Achados da revisão exata, sem editar |
| review-loop-driver | [review-loop-driver](../../skills/review-loop-driver/SKILL.md) | Fila e correções no escopo autorizado |
| revisor-de-qa | [revisor-de-qa](../../skills/revisor-de-qa/SKILL.md) | Cenários observados e gate funcional |
| security-audit | [security-audit](../../skills/security-audit/SKILL.md) | Fronteiras/ameaças e achados fundamentados |
| simplify | [simplify](../../skills/simplify/SKILL.md) | Clareza preservando comportamento/contratos |

Adaptação e snapshots: [manifesto](ADAPTACAO-SKILLS.json). Task/branch/aceite usam backlog/Git; sem ClickUp/ORCA ou publicação externa. Dois ciclos de correção, um writer, papéis independentes e estado planejado/implementado/verificado seguem as políticas. As skills não implementam novas integrações nem alteram runtime.
