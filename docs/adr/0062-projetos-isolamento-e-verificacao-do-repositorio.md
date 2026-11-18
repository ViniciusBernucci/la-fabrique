# ADR-0062 — Projetos e repositórios: isolamento, unicidade e verificação (base: modelo do Orca)

- **Estado:** ACEITA · **Relacionadas:** [ADR-0061](0061-repositorio-do-projeto-tres-modos.md), [ADR-0060](0060-conexoes-por-oauth.md), [ADR-0005](0005-autenticacao-sessoes-e-cofre-de-conexoes.md)

## Contexto
O responsável forneceu a análise técnica do sistema de projetos do Orca 1.4.259 (projeto → cópia/checkout → workspace/worktree; clone com admissão de destino; unicidade por `(host, caminho real)`; hooks aprovados por hash; limpeza conservadora; PR com `operationId`). A fábrica roda como **uma instalação**, local ou numa VPS Linux, com **vários projetos** e agentes executando código de terceiros. Adotamos o modelo do Orca adaptado a essa realidade.

## Mapa Orca → fábrica
- Projeto → checkout principal (`WORKSPACES_DIR/<slug>/…`, nunca alterado pela fábrica) → worktrees por tentativa (`internal/worktrees`, fase 2).
- **Sem multi-host:** SSH, WSL, VM efêmera, terminal SSH, grupos, pastas sem git, sparse e importação de worktrees externas ficam fora. O host é a própria instalação; `host_id` fica reservado para o futuro.
- Clone com admissão de destino e limpeza só do que a tentativa criou: já entregue (ADR-0061).

## Decisão (fase 1, implementada)
1. **Unicidade da pasta.** Dois projetos nunca usam a mesma pasta de repositório: índice único em `lower(repo_path)` (migration 00064) e comparação por **caminho real** (links simbólicos e maiúsculas não escapam) antes de clonar, abrir ou criar; conflito = 409 `workspace_taken`.
2. **Repositório exclusivo entre projetos ativos.** O mesmo `dono/nome` (sem diferenciar maiúsculas) não pode estar em dois projetos ativos: dois escritores sobre as mesmas branches. Vale como check de validação `repository.unique`, que bloqueia ativar e retomar (decisão do responsável: bloquear).
3. **Verificação do repositório** (`POST /api/projects/{id}/repository/check`, o "Re-check" do Orca), sem escrever em lugar nenhum: pasta dentro da raiz e existente; é repositório git; `origin` confere com dono/nome; branch base existe na pasta; árvore limpa (aviso); conta GitHub acessa o repositório com a conexão escolhida; **permissão de push** e repositório não arquivado; branch base existe no GitHub. Cada passo tem estado (`ok`/`warn`/`failed`/`skipped`), motivo e ação.
4. **A ativação exige a verificação** (`repository.verified`), vinculada por impressão digital à configuração (modo, pasta, dono, nome, branch, conexão e versão do segredo): mudar qualquer item, ou rotacionar o segredo, invalida e exige verificar de novo. Mesmo padrão da verificação da origem; resultado na trilha do projeto (`repository_checked`).

## Decisões para as próximas fases (ainda não implementadas)
- **Fase 2 — worktrees por tentativa** (implementada, ver abaixo).
- **Fase 3 — setup do repositório e limpeza** (implementada, ver abaixo).

## Decisão (fase 2, implementada)
1. **Worktrees ligadas ao checkout do projeto.** `worktrees.Provider.For(projeto)` usa a pasta do projeto (`repo_path`, resolvida dentro da raiz, sem links para fora) como checkout principal, que a fábrica **nunca altera** (branch e árvore intactas; testado). As worktrees nascem em `<raiz>/.lf-worktrees/…` (pasta oculta, fora da navegação); caminhos de projeto com segmento oculto (`.x`) passam a ser recusados.
2. **Base da tentativa** (`Manager.ResolveBase`): referência explícita > branch base do projeto > `origin/HEAD`; cada uma lida como branch local e depois `origin/<ref>`; explícita inexistente falha (não cai no padrão). Nomes validados com `git check-ref-format --branch`; a worktree é criada com `--no-track`.
3. **Um escritor por projeto.** Toda mudança de refs/worktrees (provisionar, checkpoint, liberar) passa por uma trava: mutex no processo + `flock` em `<git-common-dir>/lf-writer.lock` (api e worker compartilham o volume), com espera que respeita o contexto.
4. **Limites** de tentativas ativas: 4 por projeto e 8 por servidor (padrões; `Manager.MaxPerProject/MaxPerHost`, negativo = sem limite). Só tentativas novas contam; repetir uma existente (retry) segue permitido. Excedido: `ErrLimit`.
5. **Registro de operações** (`worktree_operations`, migration 00065): `Spec.OperationID` repetido devolve a mesma tentativa (recria o diretório se sumiu, nunca duplica); o mesmo ID em outra tentativa é recusado.
6. **Links para fora** (`Manager.Escapes`): lista links simbólicos da worktree que apontam para fora dela, para o executor recusar a entrega.

**Ainda não ligado:** nenhum `cmd/` constrói o `Manager` hoje (a execução de tentativas ainda não roda no worker); o `Provider` fica pronto para quando o executor for ligado. Montar só a worktree no sandbox depende dessa ligação.

## Decisão (fase 3, implementada)
1. **Setup versionado e aprovado por hash.** O script é `.lafabrique/setup.sh` (arquivo de texto UTF-8, até 64 KB), lido **do commit** da ref (nunca da árvore de trabalho). A tela e `GET /api/projects/{id}/setup` mostram o conteúdo e o SHA-256; `POST …/setup/approve` recebe a ref e o hash que o usuário revisou e **recusa (409 `setup_changed`)** se o conteúdo atual for outro. A aprovação é por projeto, ref e hash; mudar o conteúdo muda o hash e exige nova aprovação; pode ser revogada. Aprovar, revogar e rodar entram na trilha do projeto.
2. **Só no sandbox.** `projectsetup.Service.Run` exige aprovação do hash **lido naquele instante** e envia exatamente esse conteúdo por stdin ao `sh -s` dentro do sandbox (sem janela entre checar e usar). O sandbox usa o perfil escolhido com **rede desligada**, sem segredos, sem montar a raiz de workspaces: recebe só o `git archive` da árvore pedida (a branch/SHA da tentativa). Nunca roda no host da API. O resultado (ok/failed/timeout, código e fim da saída) fica em `project_setup_runs`.
3. **Limpeza desligada por padrão**, por projeto (`worktree_cleanup_policy`: ligada, dias parada para arquivar, dias arquivada para remover). **Simulação antes de apagar:** `GET …/cleanup` devolve itens (`archive`/`remove`/`skip`, cada um com motivo) e um hash; `POST …/cleanup/apply` refaz a simulação e só executa se o hash for o que o usuário viu (409 `cleanup_plan_changed`).
4. **Arquivar e remover são ações distintas.** Arquivar = `Release` (checkpoint com arquivos não versionados + ref de arquivo; tira só a pasta) para tentativa ativa **sem escritor vivo** e parada há mais que o prazo. Remover (irreversível: apaga a ref de arquivo e os checkpoints) só para tentativa já arquivada há mais que o prazo, cuja ponta **já está na branch base** e cujo último checkpoint **não guarda arquivos só locais**. Tentativa com processo escritor vivo nunca é tocada; sem branch base conhecida, nada é removido.

**Limites:** o setup roda sem rede (um repositório que precisa baixar dependências no setup depende de um gateway de saída, ainda não existente); a execução do setup é uma função do serviço, **não há rota que a dispare** (a API do stack não tem Docker; ela será chamada pelo executor quando ele for ligado ao worker, assim como o `Manager`); a aprovação é uma ação autenticada (sessão + CSRF), sem reautenticação, por decisão do responsável de não pedir senha/2FA nessas ações.

## Isolamento entre projetos (princípios)
Cada agente roda no sandbox com **só a sua worktree** montada (nunca a raiz de workspaces); token por cabeçalho no ambiente do processo e conexão por projeto; hooks do repositório desligados, submódulos fora, links simbólicos que saiam da worktree recusados e nomes de branch validados com `git check-ref-format` (feito na fase 2); só dados recriáveis ficam no volume, e o aviso de backup recai sobre repositórios `new`/`local` sem remoto.

## Limites
A verificação usa a API do GitHub com a conexão do projeto; sem conexão válida a ativação fica bloqueada. Projetos já ativos não são reverificados automaticamente (só ao suspender/retomar). A verificação não faz `fetch`: compara a pasta com o que ela já tem.
