# Backup e restauração (mínimo para desenvolvimento)

Escopo atual ([T02-04](../NEW%20MVP/03-BACKLOG.md#t02-04), parcial): **somente o banco**. O backup primário na VPS continua sendo o snapshot do provedor; volumes de artefatos/checkpoints e skills entram com T12-01. A chave do cofre está descrita abaixo.

```sh
make db-backup                       # backups/db-<UTC>.dump + .sha256 (MODE=prod na VPS)
make db-restore F=backups/db-XXXX.dump   # restaura em banco NOVO restore_<UTC>; nunca sobrescreve o da aplicação
```
O restore confere o SHA-256 antes, e depois lista migrations aplicadas, projetos e sessões do banco restaurado. Descarte o banco de teste com o comando impresso ao final. `backups/` é ignorado pelo Git; guarde cópias fora da máquina.

**Verificado (dev, 2026-11-12):** backup de 23 kB, SHA conferido, restore em banco isolado com 2 migrations, 3 projetos e 2 sessões iguais ao original — PASS.
**NOT_RUN:** restore a partir de snapshot do provedor, em destino isolado da VPS; teste de desencriptar conexões e hashes de artefatos (não existem ainda).

## Ensaio de restauração e verificação (T02-04)

```sh
make restore-drill                 # backup novo → restore em restore_drill_<UTC> → verificação → descarte
make restore-drill F=backups/db-XXXX.dump   # ensaia um backup existente
```
A verificação (`cmd/verify-restore`, somente leitura) usa a `VAULT_KEY` do ambiente e confere: migrations completas, TOTP do administrador e **todas as conexões** decifráveis, hashes de checkpoints (e de cada objeto), de arquivos de skills e de pacotes de contexto. Falha com exit ≠ 0 e diz o motivo, sem imprimir valores. Checkpoints, skills e pacotes ficam no PostgreSQL: o dump os cobre. Um backup só deve ser dado como bom depois de passar nesta verificação. Decisão: [ADR-0032](../adr/0032-verificacao-de-restauracao.md).

**Verificado (dev, 2026-11-12):** mecânica ponta a ponta PASS; a conferência reprovou corretamente um dev sem migrations e sem administrador. **NOT_RUN:** snapshot da VPS e produção.

## Chave do cofre (`VAULT_KEY`)

O banco guarda conexões e o segredo TOTP **cifrados**; a chave (`VAULT_KEY`, 32 bytes em base64) mora só no `.env`, **fora do dump**. Consequências:

- Um dump sem a chave **não** abre o cofre nem o TOTP. Guarde uma cópia da chave em local separado do backup do banco (gerenciador de senhas, cofre offline). Não a coloque em Git, logs nem chat.
- Restaurar o banco com a chave **errada**: a API sobe e registra o aviso `conexões ativas não puderam ser decifradas: VAULT_KEY incorreta?`; o login por TOTP falha, mas o login por **código de recuperação** continua possível (não depende da chave) e as conexões precisam ser recriadas.
- Trocar a chave exige re-cifrar os segredos (**não implementado**; ADR-0005).

**Verificado (integração, PostgreSQL real):** com a chave correta `Vault.Resolve` devolve o segredo; com outra chave falha; o cifrado copiado para outra linha não abre. **NOT_RUN:** restore ponta a ponta de dump + chave em destino isolado da VPS.
