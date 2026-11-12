# Contexto automático por worktree

Aplica-se a qualquer projeto com Contexto instalado. O usuário não executa init por worktree. Criar worktree continua exigindo a autorização do fluxo; preparar contexto não concede permissão para criar branches, alterar código, instalar dependências ou acessar banco.

## Depois de criar ou reutilizar uma worktree

1. Confirme raiz Git, repo comum, branch/base e ownership, usando o criador autorizado (ORCA ou Git CLI). Nunca derive identidade só pelo path.
2. Execute automaticamente o helper `scripts/preparar-contexto.mjs` desta pasta com `--project RAIZ_REAL_DA_WORKTREE`. Se o repo ainda não estiver registrado ou for necessário atualizar defaults, acrescente `--config-from RAIZ_REAL_DA_CONFIGURACAO` selecionada no planejamento. Resolva os caminhos antes da execução. O helper encontra a ferramenta instalada ao lado das skills; `CONTEXTO_CLI` permite outra instalação.
3. Exija exit 0 e `prepared: true`, raiz correta e banco isolado. Registre o recibo no estado do lote antes de delegar. Falha real de permissão/runtime/topologia é bloqueio da preparação; nunca peça ao usuário para executar init manualmente. Continue tarefas independentes.
4. O helper faz ensure: configura se necessário, preserva configuração existente, confere identidade e indexa. Retomada é idempotente. Não copie `.contexto` do checkout principal ou de outra worktree; não compartilhe bancos/gates.

Este passo pertence a todo criador/executor do fluxo: orquestrador, criação avulsa autorizada, integração e retomada de worktree criada por outra ferramenta. O worktree-planner apenas inclui o passo nos prompts; não cria nem prepara worktrees durante planejamento.

## Repositório e hook

`setup --project RAIZ_REPO [--config-from RAIZ_CONFIG]` registra defaults nos metadados Git comuns e exclui `.contexto/` em info/exclude. Não altera `.gitignore` versionado nem Git config. O fluxo pode fazer setup; não há checkpoint humano além das permissões reais do host.

O post-checkout local prepara worktrees criadas por Git/gerenciadores que executem hooks. Hook anterior é preservado e executado primeiro; core.hooksPath customizado é mantido, com fallback explícito ao helper. Criadores que desabilitam hooks ou usam --no-checkout também dependem do helper obrigatório. Checkout principal e main/master não são preparados pelo hook.

CLI em uma worktree registrada sem configuração local também chama ensure no primeiro uso. Um MCP continua fixado na raiz com que foi iniciado; cada executor usa CLI com --project real ou processo MCP próprio. Não use MCP do checkout principal para aprovar outra worktree.

Configuração é compartilhada por herança; índice, tarefas, eventos, logs e aprovações permanecem locais. Paths operacionais são ajustados em relação à origem da configuração; isso não amplia permissões de leitura/escrita. Nenhuma otimização de cache compartilhado de conteúdo é prometida nesta versão.

## Antes de limpar

Sem executores ativos e com ownership comprovado, o coordenador roda `archive --project RAIZ_REAL` e registra o recibo. SQLite e capturas reconhecidas ficam preservados nos metadados Git comuns como histórico. Só trate `.contexto/` como removível quando `archived: true` e `cleanup_safe: true`; arquivo desconhecido ou mudança concorrente exige manter a worktree. Arquivo de banco não é copiado para outra worktree nem usado para reaproveitar aprovação. As demais verificações de limpeza Git/ORCA continuam obrigatórias. Archive não remove nada nem aprova a entrega.

Se Contexto não estiver instalado em outro projeto, preserve modo manual já existente, registrando ausência. Nesta instalação ele está ativo: não dispense a preparação alegando modo manual. Em outra máquina, copie ferramenta/skills, ative o repositório uma vez e deixe o fluxo preparar suas worktrees; não copie bancos operacionais.
