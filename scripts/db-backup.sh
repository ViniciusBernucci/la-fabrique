#!/usr/bin/env bash
# Backup lógico do PostgreSQL (pg_dump formato custom) + SHA-256.
# Uso: scripts/db-backup.sh [dev|prod]   (padrão: dev)
# Saída: backups/db-<UTC>.dump e .sha256 (backups/ é ignorado pelo Git).
# O dump é gravado em .partial e só renomeado se não estiver vazio; um .partial
# remanescente indica backup que falhou e pode ser descartado manualmente.
# Cobre SOMENTE o banco. Volumes de artefatos/checkpoints e chaves entram nos
# tickets T04-03/T12-01; o snapshot do provedor da VPS continua sendo o backup
# primário (docs/operacao/backup-restauracao.md).
set -euo pipefail
cd "$(dirname "$0")/.."
mode="${1:-dev}"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p backups
file="backups/db-${stamp}.dump"

scripts/compose.sh "$mode" exec -T postgres sh -c \
  'pg_dump --format=custom --no-owner -U "$POSTGRES_USER" "$POSTGRES_DB"' > "${file}.partial"
if [[ ! -s "${file}.partial" ]]; then
  echo "backup vazio; ${file}.partial mantido para inspeção" >&2
  exit 1
fi
mv "${file}.partial" "$file"
(cd backups && shasum -a 256 "$(basename "$file")" > "$(basename "$file").sha256")
echo "backup gravado: $file ($(wc -c < "$file" | tr -d ' ') bytes)"
echo "sha256: $(cut -d' ' -f1 "$file.sha256")"
