#!/usr/bin/env bash
# Inventário SOMENTE LEITURA da VPS (T02-01). Rode NA VPS e cole a saída em
# docs/operacao/vps-inventario.md. Não altera nada, não lê segredos nem
# variáveis de ambiente de processos; nomes de usuários e comandos de processos
# vizinhos são listados, argumentos de linha de comando NÃO.
set -uo pipefail
have() { command -v "$1" >/dev/null 2>&1; }
sec() { printf '\n## %s\n\n' "$1"; }
blk() { printf '```\n'; cat; printf '```\n'; }

printf '# Inventário da VPS — %s\n' "$(date -u +%FT%TZ)"
sec "Sistema"
{ uname -srm; [ -r /etc/os-release ] && . /etc/os-release && echo "$PRETTY_NAME"; uptime; } 2>&1 | blk
sec "CPU e memória"
{ have nproc && echo "vCPUs: $(nproc)"; have lscpu && lscpu | grep -E 'Model name|Thread|Core|Socket|MHz' ; have free && free -m; swapon --show 2>/dev/null; } 2>&1 | blk
sec "Disco"
{ df -hT -x tmpfs -x devtmpfs 2>&1; have lsblk && lsblk -o NAME,SIZE,TYPE,ROTA,MOUNTPOINT 2>&1; } | blk
sec "Rede (interfaces e portas em escuta; sem tráfego)"
{ have ip && ip -br addr; if have ss; then ss -tlnH | awk '{print $4}' | sort -u; fi; } 2>&1 | blk
sec "Processos vizinhos (10 maiores em memória; só nome do comando)"
ps -eo user:12,pcpu,pmem,comm --sort=-pmem 2>/dev/null | head -11 | blk
sec "Serviços ativos (systemd)"
{ have systemctl && systemctl list-units --type=service --state=running --no-pager --no-legend | awk '{print $1}'; } 2>&1 | blk
sec "Docker"
{ if have docker; then docker version --format 'cliente {{.Client.Version}} servidor {{.Server.Version}}' 2>&1; docker info --format 'containers={{.Containers}} imagens={{.Images}} driver={{.Driver}} cgroup={{.CgroupVersion}} rootless={{.SecurityOptions}}' 2>&1; docker ps --format '{{.Names}}\t{{.Image}}\t{{.Status}}' 2>&1; else echo "docker não instalado"; fi; } | blk
sec "Agentes do provedor, backup e acesso (conferir manualmente no painel do provedor)"
{ for p in qemu-guest-agent cloud-init amazon-ssm-agent google-guest-agent do-agent; do systemctl is-active "$p" 2>/dev/null | sed "s/^/$p: /"; done
  echo "sshd PermitRootLogin/PasswordAuthentication:"; grep -hE '^\s*(PermitRootLogin|PasswordAuthentication)' /etc/ssh/sshd_config /etc/ssh/sshd_config.d/*.conf 2>/dev/null
  have ufw && ufw status 2>/dev/null | head -5; } 2>&1 | blk
printf '\n> Backup/snapshot do provedor, plano contratado e janela de manutenção não são visíveis daqui: registre-os à mão.\n'
