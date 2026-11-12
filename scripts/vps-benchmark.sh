#!/usr/bin/env bash
# Medições simples e NÃO destrutivas na VPS (T02-01). Rode NA VPS, sem outras
# cargas relevantes. Usa um diretório temporário e o Docker (se houver); limpa tudo ao final.
# Saída em Markdown para docs/operacao/vps-inventario.md. Não é um benchmark de
# produção: é um envelope inicial para propor reserva e limite.
set -uo pipefail
WORK="$(mktemp -d)"; trap 'rm -rf "$WORK"; docker rm -f lf-bench >/dev/null 2>&1 || true' EXIT
now() { date +%s.%N; }
dt() { awk -v a="$1" -v b="$2" 'BEGIN{printf "%.2f", b-a}'; }
printf '# Medições da VPS — %s\n\n| Medida | Resultado |\n|---|---|\n' "$(date -u +%FT%TZ)"

t0=$(now); head -c 268435456 /dev/zero | sha256sum >/dev/null; t1=$(now)
echo "| CPU: sha256 de 256 MiB (1 thread) | $(dt "$t0" "$t1") s |"

if have_nproc=$(command -v nproc); then
  n=$(nproc); t0=$(now)
  for _ in $(seq 1 "$n"); do head -c 134217728 /dev/zero | sha256sum >/dev/null & done; wait; t1=$(now)
  echo "| CPU: $n threads em paralelo (128 MiB cada) | $(dt "$t0" "$t1") s |"
fi

t0=$(now); dd if=/dev/zero of="$WORK/f" bs=1M count=512 conv=fsync 2>/dev/null; t1=$(now)
echo "| Disco: escrita sequencial 512 MiB com fsync | $(dt "$t0" "$t1") s |"
t0=$(now); for i in $(seq 1 2000); do echo x > "$WORK/s$i"; done; sync; t1=$(now)
echo "| Disco: 2000 arquivos pequenos | $(dt "$t0" "$t1") s |"

if command -v docker >/dev/null 2>&1; then
  t0=$(now); docker run --rm --name lf-bench --pull=missing alpine:3.22 true >/dev/null 2>&1; t1=$(now)
  echo "| Docker: iniciar e encerrar um container (imagem já baixada após a 1ª vez) | $(dt "$t0" "$t1") s |"
  t0=$(now); docker run --rm alpine:3.22 true >/dev/null 2>&1; t1=$(now)
  echo "| Docker: segundo início (imagem em cache) | $(dt "$t0" "$t1") s |"
fi
echo "| RAM livre agora | $(free -m 2>/dev/null | awk '/Mem:/{print $7" MiB disponíveis de "$2" MiB"}') |"
echo
echo "> Não medidos por este script (registrar como NÃO MEDIDO até haver ensaio): build de um projeto real, checks/testes de projeto, navegador headless, estabilidade após reinício e concorrência de dois projetos."
