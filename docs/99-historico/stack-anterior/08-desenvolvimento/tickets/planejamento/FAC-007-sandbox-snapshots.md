# FAC-007 — Sandbox e snapshots recuperaveis

Status: DONE

Revisao aceita: `2bf14f35de64f118ec8224fee6151e60027dedb0`, em 2026-09-30. O commit documental posterior apenas registra o aceite da revisao apresentada.

## Objetivo

Implementar a preparacao de worktrees descartaveis, execucao de comandos em namespace/cgroup restritos e snapshots verificaveis capazes de restaurar arquivos rastreados e untracked sobre a revisao-base exata.

## Escopo

- Contratos Zod de sandbox, worktree, snapshot, artefatos e restauracao em `packages/contracts`.
- `WorkspaceManager`, `SandboxRunner` e `SnapshotManager` em `packages/runtime`.
- Launcher Linux com user/mount/network/PID namespaces, home e `/run` substituidos, ambiente minimo, cgroup systemd e limites de processo/arquivo.
- Fixtures Git sinteticas e testes locais de isolamento, timeout, captura, integridade e restauracao.
- Documentacao de infraestrutura, operacao, runtime, planejamento e lesson aplicada.

Ficam fora: dispatcher BullMQ, claim/lease/fencing, integracao do adapter ao sandbox, containers de servicos por projeto, persistencia em PostgreSQL, upload externo, backup da VPS, deploy e piloto real.

## Criterios de aceite

1. Worktree e criado em raiz autorizada, detached na revisao resolvida, sem alterar a branch-base.
2. Comando recebe somente ambiente permitido, workspace em `/mnt`, rede isolada e home/`/run` efemeros; Docker socket e autenticacao do host ficam invisiveis.
3. systemd aplica cgroup, `TasksMax`, `MemoryMax`, `CPUQuota` e `KillMode=control-group`; timeout encerra a unidade e so retorna depois de confirmar estado inativo.
4. Snapshot registra base/head, patch binario, untracked, modos, tamanhos e SHA-256; limites e symlinks untracked inseguros sao rejeitados.
5. Restauracao exige worktree limpo na mesma revisao-base, verifica hashes antes de escrever e recupera alteracoes rastreadas e untracked byte a byte.
6. Teste de isolamento prova que fixture nao le home/auth/socket, nao herda chave sintetica e nao alcanca rede; teste de timeout prova fim da arvore.
7. Nenhum cliente de IA, API, credito ou extra usage e utilizado.

## Baseline, ambiente e limites

- Base: `developer` em `2154bcfdd3edc620f6f4b92fdcbc3304ad29f356`.
- Branch/worktree: `feat/fac-007-sandbox-snapshots` em `/home/vinicius/le-fabrique-fac-000-accepted`.
- Provider: nenhum; somente processos e repositorios sinteticos locais.
- Host verificado: Git 2.34.1, `systemd-run --user`, user namespaces, `unshare` e `prlimit` disponiveis sem root.
- Limites de teste: uma unidade, 256 MiB, 100% de uma CPU, 64 processos, timeout curto e snapshots de ate 200 arquivos/10 MiB.
- Um writer comprovado pelo worktree limpo; nenhuma arvore de provider ativa.

## Rollback

Reverter os commits FAC-007 remove contratos, launcher e gerenciadores. Worktrees e artefatos sinteticos de teste ficam somente em diretorios temporarios e sao removidos pelos testes. Nao ha migration, login, deploy ou alteracao permanente do systemd.
