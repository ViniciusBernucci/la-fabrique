# Backup e restauração (mínimo para desenvolvimento)

Escopo atual ([T02-04](../NEW%20MVP/03-BACKLOG.md#t02-04), parcial): **somente o banco**. O backup primário na VPS continua sendo o snapshot do provedor; volumes de artefatos/checkpoints e skills entram com T12-01. A chave do cofre está descrita abaixo.

```sh
make db-backup                       # backups/db-<UTC>.dump + .sha256 (MODE=prod na VPS)
make db-restore F=backups/db-XXXX.dump   # restaura em banco NOVO restore_<UTC>; nunca sobrescreve o da aplicação
```
O restore confere o SHA-256 antes, e depois lista migrations aplicadas, projetos e sessões do banco restaurado. Descarte o banco de teste com o comando impresso ao final. `backups/` é ignorado pelo Git; guarde cópias fora da máquina.

**Verificado (dev, 2026-11-12):** backup de 23 kB, SHA conferido, restore em banco isolado com 2 migrations, 3 projetos e 2 sessões iguais ao original — PASS.
**NOT_RUN:** restore a partir de snapshot do provedor, em destino isolado da VPS; teste de desencriptar conexões e hashes de artefatos (não existem ainda).

## Chave do cofre (`VAULT_KEY`)

O banco guarda conexões e o segredo TOTP **cifrados**; a chave (`VAULT_KEY`, 32 bytes em base64) mora só no `.env`, **fora do dump**. Consequências:

- Um dump sem a chave **não** abre o cofre nem o TOTP. Guarde uma cópia da chave em local separado do backup do banco (gerenciador de senhas, cofre offline). Não a coloque em Git, logs nem chat.
- Restaurar o banco com a chave **errada**: a API sobe e registra o aviso `conexões ativas não puderam ser decifradas: VAULT_KEY incorreta?`; o login por TOTP falha, mas o login por **código de recuperação** continua possível (não depende da chave) e as conexões precisam ser recriadas.
- Trocar a chave exige re-cifrar os segredos (**não implementado**; ADR-0005).

**Verificado (integração, PostgreSQL real):** com a chave correta `Vault.Resolve` devolve o segredo; com outra chave falha; o cifrado copiado para outra linha não abre. **NOT_RUN:** restore ponta a ponta de dump + chave em destino isolado da VPS.
