# ADR-0023 — Checkpoints íntegros do workspace e do estado da execução

- **Estado:** ACEITA · **Ticket:** [T12-01](../NEW%20MVP/03-BACKLOG.md#t12-01) · **Épico:** E12 · **Relacionadas:** [ADR-0008](0008-sandbox-por-feature-em-containers.md), [ADR-0010](0010-motor-de-workflows.md), [ADR-0016](0016-supervisao-de-processos.md)

## Decisão
1. **O que entra:** o conteúdo do workspace (arquivos com modo e hash SHA-256, links como links, diretórios), o `TreeHash` do conjunto, a informação de Git quando há (HEAD, ramo, `git diff --binary HEAD` como objeto, não rastreados), o estado serializado dos nós (`RunState`), as referências externas (ramo, PR, IDs remotos) e o manifesto. Fifos, sockets e dispositivos são **pulados e listados** no manifesto.
2. **Identidade da árvore:** caminho, tipo, tamanho, hash do conteúdo, alvo do link e, só para arquivos, as permissões (`& 0777`). Dono, mtime, bits setuid/setgid/sticky e permissões de diretório/link **não** fazem parte (a restauração normaliza diretórios para 0755 e remove bits especiais).
3. **Armazenamento:** objetos endereçados por hash **por projeto** (`checkpoint_objects`, sem compartilhar entre projetos, com deduplicação dentro do projeto) e `checkpoints` com manifesto, estado, referências e os hashes de manifesto, árvore e estado. Persistência em PostgreSQL (durável no commit; entra no backup do banco, T02-04).
4. **Gravação atômica:** objetos e checkpoint entram numa **única transação** (com lock consultivo por execução para a sequência). Falha, queda ou cancelamento em qualquer ponto → rollback: o checkpoint anterior segue intacto, sem objetos órfãos e sem buraco de sequência.
5. **Integridade na leitura:** `Get`/`Latest` reconferem os hashes de manifesto, árvore e estado e a coerência do manifesto (caminhos válidos, sem duplicados, totais, tipos); `Object` reconfere o hash do conteúdo; objeto ausente ou adulterado → `ErrCorrupt`. Caminho do manifesto com `..`, absoluto ou com bytes de controle é recusado.
6. **Restauração:** só em workspace **vazio** (`ErrNotEmpty`); monta o tar a partir dos objetos verificados (links restaurados como links, sem segui-los), extrai como usuário sem privilégios e **recaptura** o workspace comparando o `TreeHash` com o do checkpoint (`ErrMismatch`, com os primeiros caminhos divergentes).
7. **Parada segura:** `CaptureAndRelease` exige sandbox **sem processo vivo** (`VerifyQuiescent`; senão `ErrNotQuiescent`, sem gravar nem liberar), captura, grava e **só então** destrói o sandbox. Qualquer falha antes mantém o sandbox.
8. **Limites (a captura é recusada, nunca parcial):** 16 MiB por arquivo, 128 MiB no total, 20 000 entradas. `Prune(keep)` mantém os N mais recentes da execução e remove só os objetos que nenhum checkpoint do projeto referencia.

## Limites
- **Git real: NOT_RUN** (as imagens do sandbox não têm git). O registro de HEAD/ramo/patch/não rastreados foi provado por um executor de git de mentira; a árvore inteira (incluindo `.git`, se existir) é capturada de qualquer forma.
- Não há API/tela nem ligação ao worker: o motor de execução (T07/T10) é quem chamará `Capture`/`Save`/`Restore`.
- Objetos em `bytea` no PostgreSQL servem ao tamanho-alvo (≤128 MiB por checkpoint); repositórios maiores exigirão armazenamento de objetos externo (nova ADR).
- A restauração recria o conteúdo, não os **processos** nem o estado do container (rede, variáveis).
- Os segredos não são tratados de forma especial: arquivos do workspace são capturados como estão (o workspace não deve conter segredos; o cofre é a fonte).
