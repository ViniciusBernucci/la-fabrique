# Integração e limpeza por projeto

Leia ../../_shared/perfil-projeto.md e regras locais. No preset Intranet, leia [integração Intranet](integracao-intranet.md); seus limites permanecem.

1. Confira repo/base/branch/ownership e estado antes de criar/reutilizar worktree. Use gerenciador existente; não altere checkout principal durante execução.
2. Integre só tarefas e dependências com gates válidos do perfil em branch de revisão configurada. Preserve histórico/SHAs por repo. Nunca main/master, push ou PR.
3. Conflito não autoriza ours/theirs, reset ou descarte. Aborte só a operação iniciada por esta sessão; preserve trabalho e isole falhas.
4. Valide impacto combinado e resoluções. Reaproveite evidência individual apenas se válida. Crítico exige validação integrada independente. Correção retorna à tarefa.
5. Entrega para teste manual segue convenção local. Só troque checkout principal limpo e autorizado; senão, comandos concretos no pacote.
6. Remova worktree só com topologia/ownership confirmados, sem executor, alteração, operação pendente ou ignorado único. Commits preservados em branch. Inspecione/desfaça só links conhecidos, sem seguir junction para apagar destino. Nunca force ou exclusão recursiva.
7. Pacote final registra branches/SHAs, evidências, roteiro, bloqueios e inventário real de limpeza. Publicação externa manual.
