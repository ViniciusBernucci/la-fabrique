#!/usr/bin/env bash
# Wrapper único do Docker Compose para dev e VPS.
# Uso: scripts/compose.sh dev|prod <argumentos do docker compose>
# Variáveis: ENV_FILE (padrão .env). Roda sempre a partir da raiz do repositório.
set -euo pipefail
cd "$(dirname "$0")/.."

mode="${1:-}"
if [[ "$mode" != "dev" && "$mode" != "prod" ]]; then
  echo "uso: scripts/compose.sh dev|prod <args do docker compose>" >&2
  exit 2
fi
shift
env_file="${ENV_FILE:-.env}"
if [[ ! -f "$env_file" ]]; then
  echo "arquivo $env_file não encontrado. Rode 'make bootstrap' (dev) ou copie .env.production.example para $env_file na VPS." >&2
  exit 1
fi
exec docker compose --env-file "$env_file" --project-directory . \
  -f infra/compose/compose.yaml -f "infra/compose/compose.${mode}.yaml" "$@"
