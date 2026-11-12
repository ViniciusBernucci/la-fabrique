"""Mantém as skills e regras do fluxo Intranet apontando para a fonte central.

- Cada skill de .claude/skills vira link (junction no Windows) em ~/.claude/skills e ~/.agents/skills.
- ~/.claude/agents aponta para .claude/agents.
- O CLAUDE.md central é espelhado em intranetNova/CLAUDE.md (comparação ignora CRLF/LF).

Diretório real no lugar de um link é movido para _backup-fluxo-ia/{data}/ antes de virar link:
o script nunca apaga conteúdo. Use --check para só listar divergências (código de saída 1 se houver).
"""

from __future__ import annotations

import argparse
import datetime
import os
import shutil
import subprocess
import sys
from pathlib import Path

RAIZ_DEV = Path("D:/Documentos/00 - Dev")
FONTE_CLAUDE = RAIZ_DEV / "docker-intranet" / ".claude"
FONTE_SKILLS = FONTE_CLAUDE / "skills"
FONTE_AGENTES = FONTE_CLAUDE / "agents"
CLAUDE_MD_CENTRAL = FONTE_CLAUDE / "CLAUDE.md"
ESPELHOS_CLAUDE_MD = [RAIZ_DEV / "intranetNova" / "CLAUDE.md"]
DIRETORIOS_GLOBAIS_SKILLS = [Path.home() / ".claude" / "skills", Path.home() / ".agents" / "skills"]
LINK_GLOBAL_AGENTES = Path.home() / ".claude" / "agents"
PASTA_BACKUP = RAIZ_DEV / "_backup-fluxo-ia" / datetime.date.today().isoformat()

# Cópias de repositório que sombreiam a fonte central na descoberta do cliente.
RESIDUOS_CONHECIDOS = [
    RAIZ_DEV / "intranetNova" / ".claude" / "CLAUDE.md",
    RAIZ_DEV / "intranetNova" / ".claude" / "agents",
    RAIZ_DEV / "intranetNova" / ".claude" / "commands",
    RAIZ_DEV / "docker-intranet" / "app" / ".agents" / "skills" / "worktree-planner",
]


def e_link(caminho: Path) -> bool:
    return caminho.is_symlink() or (hasattr(os.path, "isjunction") and os.path.isjunction(caminho))


def aponta_para(link: Path, alvo: Path) -> bool:
    return e_link(link) and os.path.normcase(os.path.realpath(link)) == os.path.normcase(os.path.realpath(alvo))


def criar_link(link: Path, alvo: Path) -> None:
    link.parent.mkdir(parents=True, exist_ok=True)
    if os.name == "nt":
        # Junction não exige privilégio de administrador nem modo desenvolvedor.
        subprocess.run(["cmd", "/c", "mklink", "/J", str(link), str(alvo)], check=True, capture_output=True)
    else:
        link.symlink_to(alvo, target_is_directory=True)


def remover_link(link: Path) -> None:
    # os.rmdir em junction/symlink de diretório remove só o link, nunca o conteúdo do alvo.
    if os.name == "nt":
        os.rmdir(link)
    else:
        link.unlink()


def garantir_link(link: Path, alvo: Path, somente_verificar: bool, divergencias: list[str]) -> None:
    if aponta_para(link, alvo):
        return
    if not link.exists() and not e_link(link):
        situacao = "ausente"
    elif e_link(link):
        situacao = f"aponta para {os.path.realpath(link)}"
    else:
        situacao = "diretório real (cópia antiga)"
    divergencias.append(f"{link}: {situacao}")
    if somente_verificar:
        return
    if e_link(link):
        remover_link(link)
    elif link.exists():
        destino_backup = PASTA_BACKUP / link.parent.parent.name / link.parent.name / link.name
        destino_backup.parent.mkdir(parents=True, exist_ok=True)
        shutil.move(str(link), str(destino_backup))
        print(f"  movido para backup: {destino_backup}")
    criar_link(link, alvo)
    print(f"  link criado: {link} -> {alvo}")


def sincronizar(somente_verificar: bool) -> int:
    divergencias: list[str] = []
    skills = sorted(pasta for pasta in FONTE_SKILLS.iterdir() if (pasta / "SKILL.md").is_file())

    for diretorio_global in DIRETORIOS_GLOBAIS_SKILLS:
        for skill in skills:
            garantir_link(diretorio_global / skill.name, skill, somente_verificar, divergencias)

    garantir_link(LINK_GLOBAL_AGENTES, FONTE_AGENTES, somente_verificar, divergencias)

    conteudo_central = CLAUDE_MD_CENTRAL.read_bytes()
    for espelho in ESPELHOS_CLAUDE_MD:
        # Com core.autocrlf=true o checkout grava CRLF; a quebra de linha não é divergência.
        if espelho.is_file() and espelho.read_bytes().replace(b"\r\n", b"\n") == conteudo_central.replace(b"\r\n", b"\n"):
            continue
        divergencias.append(f"{espelho}: diferente do CLAUDE.md central")
        if not somente_verificar:
            espelho.write_bytes(conteudo_central)
            print(f"  espelhado: {espelho}")

    residuos = [caminho for caminho in RESIDUOS_CONHECIDOS if caminho.exists()]
    for residuo in residuos:
        print(f"AVISO resíduo que sombreia a fonte central (remova manualmente): {residuo}")

    if divergencias:
        titulo = "Divergências encontradas" if somente_verificar else "Divergências corrigidas"
        print(f"{titulo} ({len(divergencias)}):")
        for divergencia in divergencias:
            print(f"  - {divergencia}")
    else:
        print(f"Tudo sincronizado: {len(skills)} skills, agentes e espelhos do CLAUDE.md.")

    return 1 if somente_verificar and (divergencias or residuos) else 0


def main() -> int:
    # O console do Windows usa cp1252 por padrão e corromperia os acentos das mensagens.
    sys.stdout.reconfigure(encoding="utf-8")
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--check", action="store_true", help="só verifica, sem alterar nada")
    argumentos = parser.parse_args()
    return sincronizar(somente_verificar=argumentos.check)


if __name__ == "__main__":
    sys.exit(main())
