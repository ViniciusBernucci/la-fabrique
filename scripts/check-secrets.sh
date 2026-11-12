#!/usr/bin/env bash
# Varredura simples por segredos em arquivos rastreados (não substitui um scanner
# dedicado no CI — T02-03). Falha se achar padrões típicos.
set -euo pipefail
cd "$(dirname "$0")/.."
pattern='(-----BEGIN [A-Z ]*PRIVATE KEY-----|ghp_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{30,}|sk-[A-Za-z0-9]{32,}|AKIA[0-9A-Z]{16}|xox[baprs]-[A-Za-z0-9-]{10,})'
# Exclui o histórico arquivado e este próprio script.
hits="$(git ls-files -z | xargs -0 grep -InE "$pattern" -- 2>/dev/null | grep -v '^scripts/check-secrets.sh:' | grep -v '^docs/99-historico/' || true)"
if [[ -n "$hits" ]]; then
  echo "possíveis segredos encontrados:" >&2
  echo "$hits" | cut -c1-160 >&2
  exit 1
fi
if git ls-files --error-unmatch .env >/dev/null 2>&1; then
  echo ".env está rastreado pelo Git!" >&2
  exit 1
fi
echo "check-secrets: nenhum padrão de segredo em arquivos rastreados; .env não rastreado."
