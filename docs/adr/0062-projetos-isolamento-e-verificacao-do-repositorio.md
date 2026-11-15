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
- **Fase 3 — setup do repositório e limpeza.** Scripts de setup (equivalente ao `orca.yaml`) **só rodam dentro do sandbox**, nunca no host da API, e exigem **aprovação do usuário vinculada ao hash** do arquivo, por projeto e ref; conteúdo alterado pede nova aprovação (decisão do responsável). Limpeza: regras desligadas por padrão, simulação (dry-run) antes de apagar, nunca remove árvore suja, arquivos só locais ou processo vivo; arquivar e remover são ações distintas.

## Decisão (fase 2, implementada)
1. **Worktrees ligadas ao checkout do projeto.** `worktrees.Provider.For(projeto)` usa a pasta do projeto (`repo_path`, resolvida dentro da raiz, sem links para fora) como checkout principal, que a fábrica **nunca altera** (branch e árvore intactas; testado). As worktrees nascem em `<raiz>/.lf-worktrees/…` (pasta oculta, fora da navegação); caminhos de projeto com segmento oculto (`.x`) passam a ser recusados.
2. **Base da tentativa** (`Manager.ResolveBase`): referência explícita > branch base do projeto > `origin/HEAD`; cada uma lida como branch local e depois `origin/<ref>`; explícita inexistente falha (não cai no padrão). Nomes validados com `git check-ref-format --branch`; a worktree é criada com `--no-track`.
3. **Um escritor por projeto.** Toda mudança de refs/worktrees (provisionar, checkpoint, liberar) passa por uma trava: mutex no processo + `flock` em `<git-common-dir>/lf-writer.lock` (api e worker compartilham o volume), com espera que respeita o contexto.
4. **Limites** de tentativas ativas: 4 por projeto e 8 por servidor (padrões; `Manager.MaxPerProject/MaxPerHost`, negativo = sem limite). Só tentativas novas contam; repetir uma existente (retry) segue permitido. Excedido: `ErrLimit`.
5. **Registro de operações** (`worktree_operations`, migration 00065): `Spec.OperationID` repetido devolve a mesma tentativa (recria o diretório se sumiu, nunca duplica); o mesmo ID em outra tentativa é recusado.
6. **Links para fora** (`Manager.Escapes`): lista links simbólicos da worktree que apontam para fora dela, para o executor recusar a entrega.

**Ainda não ligado:** nenhum `cmd/` constrói o `Manager` hoje (a execução de tentativas ainda não roda no worker); o `Provider` fica pronto para quando o executor for ligado. Montar só a worktree no sandbox depende dessa ligação.

## Isolamento entre projetos (princípios)
Cada agente roda no sandbox com **só a sua worktree** montada (nunca a raiz de workspaces); token por cabeçalho no ambiente do processo e conexão por projeto; hooks do repositório desligados, submódulos fora, links simbólicos que saiam da worktree recusados e nomes de branch validados com `git check-ref-format` (feito na fase 2); só dados recriáveis ficam no volume, e o aviso de backup recai sobre repositórios `new`/`local` sem remoto.

## Limites
A verificação usa a API do GitHub com a conexão do projeto; sem conexão válida a ativação fica bloqueada. Projetos já ativos não são reverificados automaticamente (só ao suspender/retomar). A verificação não faz `fetch`: compara a pasta com o que ela já tem.
