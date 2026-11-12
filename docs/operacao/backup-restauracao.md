# Backup e restauração (mínimo para desenvolvimento)

Escopo atual ([T02-04](../NEW%20MVP/03-BACKLOG.md#t02-04), parcial): **somente o banco**. O backup primário na VPS continua sendo o snapshot do provedor; volumes de artefatos/checkpoints, chaves de criptografia e skills entram com T04-03/T12-01.

```sh
make db-backup                       # backups/db-<UTC>.dump + .sha256 (MODE=prod na VPS)
make db-restore F=backups/db-XXXX.dump   # restaura em banco NOVO restore_<UTC>; nunca sobrescreve o da aplicação
```
O restore confere o SHA-256 antes, e depois lista migrations aplicadas, projetos e sessões do banco restaurado. Descarte o banco de teste com o comando impresso ao final. `backups/` é ignorado pelo Git; guarde cópias fora da máquina.

**Verificado (dev, 2026-11-12):** backup de 23 kB, SHA conferido, restore em banco isolado com 2 migrations, 3 projetos e 2 sessões iguais ao original — PASS.
**NOT_RUN:** restore a partir de snapshot do provedor, em destino isolado da VPS; teste de desencriptar conexões e hashes de artefatos (não existem ainda).
