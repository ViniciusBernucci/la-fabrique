#!/usr/bin/env bash
# Ensaio de restauração ponta a ponta: backup → restore em banco isolado →
# verificação (migrations, TOTP e conexões decifráveis com a VAULT_KEY, hashes de
# checkpoints, skills e pacotes de contexto) → descarte do banco de teste.
# Uso: scripts/restore-drill.sh [dev|prod] [arquivo.dump]
#   Sem arquivo.dump, faz um backup novo. Em dev usa a porta POSTGRES_PORT do .env.
# O banco restaurado é SEMPRE um banco novo (restore_drill_<UTC>); o da aplicação
# nunca é tocado. Falha (exit != 0) se qualquer conferência falhar.
set -euo pipefail
cd "$(dirname "$0")/.."
mode="${1:-dev}"
dump="${2:-}"
[[ -f .env ]] || { echo "falta .env (make bootstrap)" >&2; exit 2; }
set -a; . ./.env; set +a

if [[ -z "$dump" ]]; then
  scripts/db-backup.sh "$mode" | tee /tmp/restore-drill-backup.$$ >&2
  dump="$(sed -n 's/^backup gravado: \(backups\/db-[^ ]*\.dump\).*/\1/p' /tmp/restore-drill-backup.$$)"
  rm -f /tmp/restore-drill-backup.$$
fi
[[ -f "$dump" ]] || { echo "dump não encontrado" >&2; exit 1; }

target="restore_drill_$(date -u +%Y%m%d%H%M%S)"
cleanup() {
  scripts/compose.sh "$mode" exec -T postgres sh -c "psql -U \"\$POSTGRES_USER\" -d postgres -c 'DROP DATABASE IF EXISTS $target'" >/dev/null 2>&1 || true
}
trap cleanup EXIT

scripts/db-restore.sh "$dump" "$mode" "$target" >&2

url="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@127.0.0.1:${POSTGRES_PORT}/${target}?sslmode=disable"
echo "== verificação do banco restaurado ($target) ==" >&2
# A chave vem do ambiente (.env): ela NÃO está no dump. Em um restore real, use a cópia guardada à parte.
DATABASE_URL="$url" go run ./cmd/verify-restore
