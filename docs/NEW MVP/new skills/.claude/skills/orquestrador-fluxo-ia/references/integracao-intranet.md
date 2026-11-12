# Criação, integração e limpeza segura

Leia execução-portavel e regras Git centrais. Git CLI local via Bash, um repositório por vez, caminhos com barra normal e entre aspas. Permissões do host continuam aplicáveis. Nunca main/master como destino, force, reset, exclusão de branches, push ou PR.

## Worktrees

| Repo | Raiz Git | Raiz padrão das worktrees |
| --- | --- | --- |
| Backend | {RAIZ_BACKEND_CONFIRMADA} | {RAIZ_WORKTREES_BACKEND_CONFIRMADA} |
| Frontend | {RAIZ_FRONTEND_CONFIRMADA} | {RAIZ_WORKTREES_FRONTEND_CONFIRMADA} |

Confirme git rev-parse --git-common-dir e git worktree list --porcelain. ORCA gerenciado exige orca-cli; use seus caminhos reais e mantenha registro das sessões. Em Git puro: git -C REPO worktree add -b BRANCH PATH SHA_BASE, com cada valor real entre aspas. O plano não cria a worktree; chamar o orquestrador autoriza criá-la. Não altere checkout principal do usuário.

Fetch pode atualizar refs remotas; registre resultado e mantenha o SHA-base do manifesto. Fetch indisponível não invalida um objeto local já confirmado; registre a limitação. Se o objeto requerido não existir, isole as tarefas afetadas. Nunca pull origin main em cada executor; não mude a base silenciosamente.

Worktree criada antes pelo usuário no ORCA: reaproveite quando repo comum, branch final (ou renomeio simples da branch gerada pelo ORCA, nunca -M), base e árvore limpa conferirem com o plano; registre path e proveniência no estado e siga sem perguntar. Divergência de base ou trabalho alheio na árvore isola a tarefa.

Retomada: reutilize path registrado somente quando repo comum, branch, base e commits pertencerem à tarefa e nenhum outro executor estiver ativo. Branch existente sem worktree pode ser reutilizada apenas com proveniência confirmada no estado. Path ocupado ou branch desconhecida gera isolamento; não renomeie nem apague trabalho. Não use --force. Branch de origem remota não deve ficar upstream da tarefa: confira tracking e remova apenas vínculo automático à base quando comprovado; nunca ignore qualquer erro com || true sem verificar a causa.

## Branches de revisão

Padrão: revisao/AAAA-MM-DD-{slug-do-lote}; grupos incompatíveis usam -2, -3. Um branch por repositório/grupo, mesmo nome entre camadas da feature. Data/fuso vêm do manifesto, não do relógio UTC do host. Não reutilize uma branch do mesmo dia de outro lote.

Crie candidata em worktree de integração própria a partir do SHA-base fixado do repositório. Retome uma candidata existente só quando seu histórico e merges coincidirem com o estado; não acrescente commits desconhecidos. Rodada de reconstrução usa nome novo com -r2, -r3, registrado como candidato, sem apagar/rebasear histórico anterior.

Para cada tarefa pronta, dependências primeiro, execute merge --no-ff da branch final, mensagem em português. Registre SHA do merge e dos pais. Nunca force resolução automática de conflitos com ours/theirs. Um merge por unidade lógica permite git log --first-parent e git diff MERGE^1 MERGE por tarefa.

### Conflitos e múltiplas camadas

Antes de integrar uma feature entre camadas, prepare candidatas para as duas camadas e confira ambos os resultados. Só marque INTEGRADA quando todas as camadas e dependências forem incluídas e validadas. Se uma falhar após merge na outra, a candidata parcial é rejeitada no estado; ela não vira branch aprovada do pacote. Reconstrua candidato novo a partir da base com apenas o conjunto compatível, sem reset ou reescrita.

Em conflito, registre arquivos e encerre somente o merge iniciado por esta execução com git merge --abort, preservando saída/estado. Não aborte operação pré-existente alheia. Se a árvore não estava limpa antes ou abort falhar, mantenha a worktree e isole o candidato. Nenhuma limpeza sobre merge em andamento.

Conflito semântico entre tarefas exige análise do contrato. Se as tarefas podem ser entregues separadamente sem violar dependências, monte grupos separados com sufixos; uma tarefa e suas dependentes acompanham o grupo necessário. Não coloque tarefa dependente em grupo sem sua origem. Tarefa que cruza camadas acompanha o mesmo sufixo nos dois repos. Teste cada agrupamento viável uma vez, no máximo tantos grupos quanto tarefas; não tente -2/-3 indefinidamente. Conflito com a própria base, dependência incompatível ou contrato ambíguo isola a tarefa e dependentes, preservando ramos.

### Verificações finais

- Frontend: proxy válido, dependências existentes/restauração do lockfile permitida, build e verificações do ticket, sem adicionar pacote nem alterar manifests para destravar ambiente.
- Backend: php -l nos PHP alterados e verificações pertinentes. Qualquer teste/script que toque banco segue CLAUDE.md §2; sem isolamento provado não roda.
- revisor-de-codigo sobre interações e novos merges/resoluções, auditor-de-review para achados novos e revisor-de-qa para fluxos combinados afetados. Preserve evidências individuais válidas e cubra o impacto da combinação.
- Falha retorna ao especialista na branch da tarefa, dentro dos limites de ciclo. Commit, revalidação e reintegração; não ajuste código diretamente na candidata para evitar a proveniência da tarefa.
- Falha sem responsável claro bloqueia o grupo. Reconstrua candidatas novas para testar subconjuntos sem reescrever ramos. Não remova uma tarefa mantendo suas dependentes por engano.

Grave roteiro/relatório final no docs/historico/reviews de cada camada e commite explicitamente os documentos na candidata aprovada (commit documental é permitido). Atualize o pacote central com os SHAs finais. Somente candidatas com verificações finais aprovadas entram em branches_aprovadas do manifesto; outras ficam identificadas como rejeitadas/experimentais.

## Entrega no checkout principal

Verificação de telas e testes manuais do usuário acontecem no **checkout principal** de cada repositório, nunca numa worktree: é lá que estão o ambiente Docker, o proxy, `vendor`/`node_modules` e o servidor de desenvolvimento que ele usa. A worktree de integração existe só enquanto o lote roda.

Depois que a candidata for aprovada e seus documentos commitados, em cada repositório (backend e frontend, um por vez):

1. Registre no pacote a branch atual do checkout principal, para o usuário poder voltar.
2. Confirme o checkout principal limpo: `git status --porcelain=v1 --untracked-files=all` vazio e nenhuma operação Git pendente (merge, rebase, cherry-pick). Nunca use stash, reset, `checkout -f` ou `switch --discard-changes` para forçar.
3. Desfaça os links da worktree (ver "Links para o checkout principal" abaixo) e remova a worktree de integração com `git worktree remove`, sem `--force` — a branch não pode ficar ativa em dois lugares. A branch e os commits permanecem.
4. `git switch revisao/AAAA-MM-DD-{slug}` no checkout principal e confira com `git branch --show-current`.
5. Compare `composer.lock` (backend) e `package-lock.json` (frontend) entre a branch anterior e a de revisão. Se diferirem, registre no pacote que é preciso reinstalar dependências antes do teste; não instale.

Checkout principal sujo ou ocupado: não troque. Registre no pacote o motivo e os comandos exatos dos passos 3–4 para o usuário rodar. Mais de um grupo (`-2`, `-3`): coloque o primeiro no checkout principal e liste no pacote os comandos para trocar para os demais. As worktrees das tarefas seguem as regras de limpeza abaixo; nenhuma delas é o lugar do teste manual.

## Limpeza

A limpeza cobre todas as worktrees dos dois repositórios, incluindo legadas, mas **órfã não significa apenas diretório antigo ou fora deste lote**. Antes de remover, inventarie com Git e com ORCA quando gerenciado, consulte estados de todos os lotes e verifique sessões/terminais em execução. Se não puder comprovar ausência de dono/execução ativa, mantenha e registre.

Nunca remova checkout principal, diretório com outro git-common-dir, worktree locked, branch main/master, árvore com operação Git pendente ou worktree de tarefa não terminal. ISOLADA por si só não autoriza excluir a worktree: preserve quando for necessária para diagnóstico/retomada ou contiver estado de ambiente que não esteja recuperável.

A worktree de integração da candidata aprovada sai pela "Entrega no checkout principal" acima. Candidatas reprovadas deste lote podem ser removidas após guardar todos os commits, relatórios e pacote; não é necessário mantê-las para existir a branch. Para uma worktree órfã de qualquer lote, exija todas as condições:

1. Path absoluto normalizado, registrado em git worktree list, pertencente ao repo esperado e diferente do checkout principal. Confira path real e raiz antes da remoção; não siga junction desconhecida.
2. Nenhuma sessão/terminal/executor ativo ou ownership pendente, nem tarefa não terminal em estados/manifestos. Ausência de arquivo de estado não prova abandono; histórico/registro do gerenciador deve confirmar ausência de uso. Dúvida resulta em mantida.
3. git status --porcelain=v1 --untracked-files=all vazio e sem alterações staged, unstaged ou não rastreadas. Não use clean/reset/checkout para forçar árvore limpa. CLAUDE.local.md com âncora também conta como alteração; não descarte automaticamente.
4. Inspecione git ls-files --others --ignored --exclude-standard: arquivos ignorados com dados, relatórios, uploads, configurações locais únicas ou procedência incerta exigem manter. Só artefatos regeneráveis identificados (node_modules, build/cache sem dados de usuário) podem ser considerados descartáveis. Não imprima conteúdo de segredos. Se o Git exigir force por arquivos ignorados/submódulos, mantenha.
5. HEAD e todo trabalho commitado estão preservados por branch nomeada. Para branch de tarefa do lote, registre inclusão na branch de revisão aprovada (merge-base --is-ancestor) ou, se isolada, branch e SHA de preservação no pacote. HEAD destacado só é removível se uma branch nomeada contiver o commit e o registro comprovar ausência de trabalho ativo. Nunca apague branches.

### Links para o checkout principal — obrigatório antes de remover

O ORCA cria `node_modules` (e às vezes `vendor` ou outras pastas) da worktree como **junction/symlink para o checkout principal**. No Windows, `git worktree remove` segue o link e **apaga o conteúdo do destino**: já zerou o `intranetNova\node_modules` e derrubou o `ng serve` do usuário. A checagem de arquivos ignorados não detecta isso.

Antes de qualquer remoção, liste os links do primeiro nível da worktree (`cmd /c dir /AL "PATH"` no Windows; `find PATH -maxdepth 2 -type l` no Bash) e desfaça cada um removendo **só o link**: `cmd /c rmdir "PATH\node_modules"` (rmdir sem `/S` em junction apaga só o link). Nunca use `rm -rf`, `Remove-Item -Recurse` ou `rmdir /S` num link. Confira que o destino continua com conteúdo e que o link sumiu; só então rode `git worktree remove`. Se não conseguir desfazer um link com segurança, mantenha a worktree e registre.

Execute apenas git -C REPO worktree remove PATH, sem --force, um path conferido por vez. Em ORCA, use a remoção via gerenciador conforme orca-cli e confirme que sua opção preserva branches/commits; se a ferramenta só oferece remoção destrutiva, mantenha. Não use Remove-Item -Recurse, rm -rf ou exclusão de pasta para contornar Git/ORCA.

Entradas cujo path já desapareceu: inspecione git worktree prune --dry-run antes de prune; respeite entradas locked e expiração padrão. Não use --expire now para apagar registros recentes de execução concorrente. Releia worktree list depois e registre resultado.

Finalize pacote com removidas/mantidas e motivos. Nenhum arquivo não commitado fica esquecido: todo material próprio autorizado e pronto é commitado; bloqueios de commit e material alheio são listados explicitamente. Limpeza recusada não pode ser reportada como feita. Permissões insuficientes isolam a ação e aparecem no resultado final, sem contorno de sandbox.
