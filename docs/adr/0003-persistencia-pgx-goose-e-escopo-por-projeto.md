# ADR-0003 — Persistência: pgx, goose e escopo por projeto no banco

- **Estado:** ACEITA (2026-11-12) · **Ticket:** [T03-02](../NEW%20MVP/03-BACKLOG.md#t03-02) · **Épico:** E03

## Decisão

- **Driver/pool:** `pgx/v5` (`pgxpool`). SQL escrito à mão em repositórios finos (`internal/persistence`, `internal/projects`, `internal/security`), sem ORM.
- **Migrations:** `goose` (provider API) com SQL embutido no binário (`db/migrations`), aplicadas por `cmd/migrate` com conexão própria. Cada arquivo tem `Up` e `Down`; `Down` é só para dev/rollback de ensaio.
- **IDs opacos:** `uuid` (`gen_random_uuid()`); datas em `timestamptz` (UTC).
- **Controle otimista:** toda entidade mutável tem `version`; `UPDATE … WHERE version = $n`. Versão obsoleta → `ErrVersionConflict` (HTTP 409).
- **Escopo por projeto no próprio schema:** `features`, `executions`, `execution_events`, `outbox` e `artifacts` carregam `project_id`; as ligações entre elas usam **chaves estrangeiras compostas** `(project_id, id)`, de modo que o banco recusa vínculo entre projetos diferentes mesmo que o código erre. As consultas também recebem o escopo explicitamente.
- **Eventos monotônicos:** `(execution_id, seq)` com `seq` calculado sob `SELECT … FOR UPDATE` da execução (sem lacunas com escritores concorrentes).
- **Outbox idempotente:** `UNIQUE (project_id, business_key)`; repetir a mesma chave devolve a entrada existente. Transições de estado validadas em código + versão.
- **Artefatos por referência:** só metadados no banco (`sha256`, tamanho, `storage_ref` relativa, `schema_version`); `CHECK` no banco e validação no código recusam caminho absoluto e `..`.
- **Um administrador:** índice único parcial em `admins ((true))`.
- **Erros:** `persistence.MapError` traduz SQLSTATE (`23505`, `23503`, `23514`…) para erros de domínio; a API nunca devolve texto do driver.

## Compatibilidade e upgrade

Migrations são aditivas por padrão. Mudança incompatível (renomear/remover coluna usada) exige duas etapas em releases distintos (expandir → migrar dados → contrair). O teste `TestMigrationPathPreservesData` aplica a migration 1, insere fixture, aplica a 2 e confere que os dados sobrevivem. Backup/restauração: [backup-restauracao.md](../operacao/backup-restauracao.md).

## Fora do escopo desta ADR

Tabelas de catálogos, workflows, uso/orçamento, decisões/autorizações, memória e fila com leases chegam com seus épicos (E07–E13), cada uma em migration própria.
