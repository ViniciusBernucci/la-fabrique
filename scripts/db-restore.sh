#!/usr/bin/env bash
# Restaura um backup em um banco NOVO e isolado (nunca sobrescreve o banco da
# aplicação) e confere integridade.
# Uso: scripts/db-restore.sh <arquivo.dump> [dev|prod] [nome_do_banco_destino]
set -euo pipefail
cd "$(dirname "$0")/.."
dump="${1:?uso: scripts/db-restore.sh <arquivo.dump> [dev|prod] [banco_destino]}"
mode="${2:-dev}"
target="${3:-restore_$(date -u +%Y%m%d%H%M%S)}"
[[ -f "$dump" ]] || { echo "arquivo não encontrado: $dump" >&2; exit 1; }
[[ "$target" =~ ^[a-z_][a-z0-9_]{0,40}$ ]] || { echo "nome de banco inválido: $target" >&2; exit 2; }

if [[ -f "$dump.sha256" ]]; then
  (cd "$(dirname "$dump")" && shasum -a 256 -c "$(basename "$dump").sha256") \
    || { echo "SHA-256 não confere; abortando" >&2; exit 1; }
else
  echo "aviso: $dump.sha256 ausente; integridade do arquivo NÃO verificada" >&2
fi

dc() { scripts/compose.sh "$mode" "$@"; }
dc exec -T postgres sh -c "psql -U \"\$POSTGRES_USER\" -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE $target'"
dc exec -T postgres sh -c "pg_restore --no-owner -U \"\$POSTGRES_USER\" -d $target" < "$dump"

echo "restaurado em: $target. Conferência:"
dc exec -T postgres sh -c "psql -U \"\$POSTGRES_USER\" -d $target -At -c \
  \"select 'migrations_aplicadas=' || count(*) from goose_db_version where is_applied and version_id > 0; \
    select 'projetos=' || count(*) from projects; \
    select 'sessoes=' || count(*) from admin_sessions\""
echo "descarte quando terminar: scripts/compose.sh $mode exec -T postgres sh -c 'psql -U \"\$POSTGRES_USER\" -d postgres -c \"DROP DATABASE $target\"'"
