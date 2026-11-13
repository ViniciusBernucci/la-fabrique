# FAC-012H — Isolamento da raiz de checks

## Resultado

Corrigida uma falha de isolamento em `SandboxRunner`: antes, um processo em namespace de mount ainda conseguia ler `/etc/hostname` do filesystem do host. O launcher agora cria rootfs efêmero, monta somente `/usr` read-only e workspace em `/mnt`, move o `/proc` do PID namespace, pivota a raiz e desanexa `/.oldroot` antes de iniciar o comando. Em seguida remove capabilities/permissões de elevação com `setpriv` e aplica `prlimit`.

## Evidências e verificações

- Antes: `/usr/bin/cat /etc/hostname` dentro do sandbox retornou `COMPLETED`, `stoppedConfirmed=true` e saída com 11 bytes. O conteúdo do hostname não foi registrado.
- Depois: teste Linux sintético confirma `/etc/hostname`, home do host, `/var/lib`, cgroup, Docker socket e alvo de symlink host invisíveis; workspace aparece em `/mnt`, `/tmp` interno funciona, rede externa fica bloqueada, CapEff/CapPrm são zero, `NoNewPrivs=1`, remount RW de `/mnt` falha e término é confirmado.
- Regressões cobertas: escrita explícita por path, escrita negada fora da lista, recusa de symlink e timeout com cgroup/filhos encerrados.
- Checks: `npm run lint` (145 arquivos), `npm run typecheck`, `npm test` (187 testes), `npm run build` e `git diff --check` passaram.
- Diff sanitizado: somente launcher/runtime tests e documentação; sem credentials, workers, fila, banco, migrations, cliente/provider, checkout ou piloto.

## Limites, rollback e aceite

`/usr` é a única árvore persistente do host exposta read-only; `/opt` e configurações completas de `/etc` não são expostas. Projetos que exijam toolchain fora de `/usr` precisam de whitelist operacional confiável antes de execução. A prova foi executada no usuário interativo atual (que pertence ao grupo `docker`), não sob identidade systemd dedicada; Docker socket e caminhos de host ficaram invisíveis mesmo nesse ensaio, mas a validação operacional do serviço ainda é necessária. FAC-012D segue necessária para confinar o Developer CLI; o consumer BullMQ permanece no probe. Rollback: reverter `2392a04`, sem mudança de dados/schema. Status: aguarda aceite humano desta revisão; não mesclado nem implantado.
