# ADR-0032 — Verificação de restauração

- **Estado:** ACEITA · **Ticket:** [T02-04](../NEW%20MVP/03-BACKLOG.md#t02-04) · **Épico:** E02 · **Relacionadas:** [ADR-0005](0005-autenticacao-sessoes-e-cofre-de-conexoes.md), [ADR-0023](0023-checkpoints-integros.md)

## Decisão
1. **Backup só vale depois de verificado.** `internal/restorecheck` e o comando `cmd/verify-restore` conferem um banco restaurado, **somente leitura**, e saem com erro se qualquer conferência falhar.
2. **O que é conferido:** (a) migrations: a maior aplicada é igual à embutida no binário; (b) administrador: existe e o segredo TOTP confirmado abre com a `VAULT_KEY`; (c) sessões: contagem e coerência com o administrador (sessões vencidas após o restore são esperadas); (d) cofre: **toda conexão ativa é decifrada** (`Vault.Resolve`); (e) checkpoints: manifesto, estado e **cada objeto de arquivo** reconferidos por hash; (f) skills: SHA-256 de cada arquivo; (g) pacotes de contexto: hash do conteúdo.
3. **Sem vazamento:** o relatório traz contagens, identificadores, nomes de conexão e o motivo; nunca valores. Limita-se a 20 problemas por conferência.
4. **Chave fora do dump:** a verificação recebe a chave do ambiente, como o produto. Dump sem a chave certa falha em (b) e (d), o que prova que o backup do banco **sozinho** não basta.
5. **Ensaio automatizado:** `make restore-drill` faz backup → restore em banco novo `restore_drill_<UTC>` (o da aplicação nunca é tocado) → verificação → descarte do banco de teste, mesmo em falha.
6. **Escopo do backup:** como checkpoints, skills e pacotes ficam no PostgreSQL, o dump do banco cobre esses artefatos; o snapshot do provedor continua sendo o backup primário.

## Limites
- Ensaio feito só em dev (Docker local). Restore a partir do snapshot da VPS e a execução em produção continuam NOT_RUN; o caminho de recuperação da chave segue sendo a cópia guardada à parte (sem rotação implementada).
- Não confere o conteúdo de domínio além dos hashes e decifrações (ex.: coerência entre tabelas de negócio).
