# FAC-007 — Sandbox e snapshots recuperaveis

Data: 2026-09-30
Estado: DONE
Revisao aceita: `2bf14f35de64f118ec8224fee6151e60027dedb0`, em 2026-09-30

## Resultado

O `packages/runtime` agora cria worktrees detached em uma raiz autorizada, executa comandos em sandbox Linux sem root e captura/restaura snapshots verificaveis de alteracoes rastreadas e untracked. Requests, limites, resultados e manifestos usam contratos Zod compartilhados.

## Funcionamento

- `WorkspaceManager` resolve o commit, cria `git worktree --detach` sob a raiz configurada e confirma o SHA resultante.
- `SandboxRunner` cria unidade systemd de usuario com `MemoryMax`, `CPUQuota`, `TasksMax`, `KillMode=control-group` e tempo maximo.
- `unshare` cria namespaces de usuario, mount, rede e PID. O launcher monta o workspace em `/mnt`, substitui `/home`, `/root`, `/run`, `/tmp`, `/var/tmp` e `/dev/shm` e entrega ambiente minimo ao comando.
- Rede externa fica ausente. Docker socket, repositorio de controle e autenticacao guardada no home nao aparecem no namespace.
- Timeout e limite de log sinalizam toda a unidade, escalam para `SIGKILL` e verificam que ela nao permanece ativa.
- `SnapshotManager` guarda patch Git binario, untracked regulares, modos, tamanhos e SHA-256 fora do workspace. Restauracao valida manifesto e todos os artefatos antes de exigir um worktree limpo no mesmo SHA.

## Evidencias

A fixture real executada por `systemd-run --user` confirmou cwd `/mnt`, home/auth ausente, `/home/vinicius/le-fabrique` ausente, Docker socket ausente, chave `OPENAI_API_KEY` sintetica ausente e conexao externa bloqueada. Dentro do processo, os arquivos do cgroup confirmaram `memory.max=268435456` e `pids.max=64`.

Outra fixture iniciou pai e filho que escreviam um heartbeat. O timeout retornou `TIMED_OUT`, confirmou a unidade parada e o arquivo deixou de crescer depois do retorno. Nenhuma unidade `le-fabrique-*` permaneceu carregada apos os testes.

O ensaio Git criou worktree detached sem mover `main`, alterou texto e binario, criou untracked binario e executavel, capturou o snapshot e restaurou tudo em um segundo worktree no SHA-base. Modo `0750`, bytes e status Git coincidiram. Symlink para `/etc/passwd` e patch adulterado foram rejeitados.

## Checks

- `npm run lint`: 67 arquivos, passou.
- `npm run typecheck`: contratos, runtime, API, worker e web, passou.
- `npm test`: 46 testes em 13 arquivos, passou; 27 testes pertencem ao runtime.
- `npm run build`: contratos, runtime, API, worker e web, passou.
- `git diff --check`: passou.
- Diff sanitizado: apenas nomes e valores sinteticos declarados em fixtures; nenhuma credencial real.

Houve duas rodadas de correcao. A primeira ajustou formatacao, compilacao e o ambiente confiavel necessario para acessar o bus systemd. A segunda removeu um remount de raiz incompativel com o user namespace e tornou o launcher disponivel tanto nos testes quanto no build. A verificacao seguinte passou.

## Limites

O sandbox foi comprovado nesta VPS Linux com systemd de usuario e user namespaces. Outros hosts precisam de preflight; ausencia dessas capacidades deve bloquear execucao. O filesystem raiz continua visivel conforme permissoes Unix, enquanto locais gravaveis e sensiveis conhecidos sao mascarados. Isto nao equivale a VM ou container privilegiado.

O `.git` do worktree aponta para metadados sob o repositorio de controle, que fica oculto dentro do sandbox. Portanto comandos Git confiaveis e snapshot rodam no supervisor fora do namespace; a integracao futura precisa manter essa separacao. Nao ha allowlist de servicos sinteticos, persistencia de snapshots, limpeza operacional, claim/lease/fencing nem chamada do sandbox pelo worker. Esses pontos pertencem a FAC-008 e ao ensaio posterior.

Nenhum provider, API, credito ou extra usage foi utilizado.

## Rollback

Reverter `a99fb1f30a48fa3e7a991f9cd425d6757543ad12` remove contratos, managers, runner, launcher e testes. As unidades sao transientes/coletadas e os testes removem seus diretorios temporarios. Nao ha migration, configuracao systemd persistente, login ou deploy.
