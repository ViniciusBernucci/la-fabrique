# Execução portátil — Claude, Antigravity e Codex

Leia perfil-projeto.md. As convenções de fonte/espelhos abaixo são do preset Intranet; em outro projeto use a instalação real das skills e regras locais.

- Fonte de manutenção local: {FONTE_CLAUDE}/skills (diretório real das regras centrais). Não há cópias nos repositórios: ~/.claude/skills (Claude) e ~/.agents/skills (Codex e Antigravity) contêm links para a fonte, conferidos por planejador-de-lote/scripts/sincronizar-fluxo.mjs. Edite só a fonte. Se encontrar cópia antiga de uma skill do fluxo num repositório, ela é resíduo: use a fonte pelo caminho absoluto.
- AGENTS.md, CLAUDE.md e GEMINI.md apontam para as mesmas regras centrais; .antigravity/agents é compatibilidade legada, não descoberta de skills.
- Para acionar uma etapa, leia o SKILL.md pelo caminho real, suas referências necessárias e execute o contrato. Se o cliente oferecer invocação nativa, pode usá-la. Não presuma Skill, TodoWrite, Agent, Task, FLUXO_EVENTO como ferramentas universais; FLUXO_EVENTO é texto de saída.
- Na execução do lote, delegação está autorizada. Use o mecanismo nativo disponível de agentes/sessões, fornecendo repositório, branch, base/HEAD, especificação, escopo e caminhos das skills. Com ORCA gerenciado, leia orca-cli e use sua CLI para criação, inventário e terminais; no Git puro use worktree add. Não invente comandos de um cliente instalado.
- Ao criar/reutilizar worktree por qualquer caminho autorizado, aplique worktrees-automaticas.md e prepare Contexto pelo helper antes de delegar. Nunca peça init manual nem copie bancos entre árvores.
- Implementação e review exigem sessões independentes; auditoria/QA conforme perfis-de-risco.md. Consulte contexto-e-automacao.md para pacotes mínimos e automação. A skill pode ser lida em qualquer cliente; não force Claude CLI. Sem executor independente disponível, registre gate BLOCKED e isole a tarefa. Nunca apresente troca de papel na mesma sessão como revisão independente aprovada.
- Nenhuma skill amplia permissões do host. Para acesso externo, instalação ou operação negada, use o mecanismo de aprovação do cliente quando necessário; não contorne sandbox. No lote, registre bloqueio da tarefa e continue o que for independente. “Sem checkpoints humanos” remove decisões intermediárias do processo, não permissões do sistema nem proteção de dados.
- Data operacional: timezone configurado; Intranet usa America/Sao_Paulo, incluindo retomadas após meia-noite; manifesto guarda data e identificador originais. Não derive novos nomes da data UTC do host.
- Em worktrees antigas, carregue as regras e skills centrais atuais por caminho absoluto; não reescreva planos históricos nem siga checkpoints legados. Registre a migração do estado antes de retomar.

Fontes verificadas em 21/10/2026: [OpenAI Docs](https://learn.chatgpt.com/docs/build-skills), [Claude Code](https://code.claude.com/docs/en/skills), [Antigravity](https://antigravity.google/docs/skills).
