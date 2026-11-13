# DOC-MV-005 — Checkpoint de commit e limpeza

2026-10-07 Europe/Berlin. BLOCKED_READ_ONLY. Pedido explícito do usuário para commit/merge em developer, descarte de redundâncias e remoção de worktrees/branches órfãs. HEAD 9439cde4ce52a09225c9476c19baa89319802523; alterações DOC-MV-003/004 presentes e preservadas na worktree principal developer.

## Resultado observado

Validador documental PASS antes do checkpoint: 377 Markdown, 2.280 links, 601 snapshots, 2.444 seções. git add dos paths explícitos autorizados falhou: Unable to create .git/index.lock: Read-only file system. Nenhum commit/merge/remoção de branch foi realizado; não contornar as restrições do ambiente. Esta é uma tentativa posterior às entregas originais, não uma revisão aceita do software.

Só a worktree principal está registrada; git worktree prune --dry-run --verbose não encontrou registros órfãos. Busca nos locais de worktrees conhecidos não encontrou diretórios adicionais. Há 69 branches locais integradas e sem worktree, incluindo docs/manual-vivo-inicial; main e developer foram preservadas no plano. Ser elegível não significa que tenha sido apagada. [Inventário/checkpoint](evidencias/DOC-MV-005/checkpoint.json) registra nomes/SHAs, escopo, erro e próxima ação.

## Preservação e retomada

Não descartar capítulos numerados, skills correntes, snapshots, evidências ou alterações locais para forçar estado limpo. A fonte importada .agents/skills permanece read-only e o patch de instalação de DOC-MV-004 continua preparado, não aplicado. Um ambiente com Git gravável é necessário para prosseguir; a autorização de commit/merge/limpeza já está registrada, sem nova confirmação de rotina.

Na retomada: conferir branch/HEAD/diff/staged/untracked e hashes das entregas, preservar trabalho posterior, validar e adicionar somente paths próprios. Criar commit local; as alterações já estão em developer, sem branch externa a mergear. Limpeza exige reconferir a ancestralidade e worktrees antes de git branch -d; nunca force para descartar commits. Sem push/deploy autorizado por esta operação.

Rollback deste checkpoint: retirar apenas o registro e suas entradas de índice/changelog/backlog se necessário, preservando DOC-MV-003/004. Não há alteração Git/banco/serviço a reverter nesta tentativa. Software, autoload e revisão independente não receberam novos PASS ou DONE.
