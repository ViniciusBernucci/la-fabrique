# Executar no checkout developer com o diff revisado; todos os comandos estão comentados.
# Este roteiro usa os patches para criar commits separados sem misturar o diff de outras etapas.
# Conferir HEAD = de5dbfac0b00f3e1699b999f987856f592f65899 e index limpo antes da primeira etapa.
# git status --short
# git apply --cached --check docs/09-entregas/2026/evidencias/FAC-034/01-tokens.patch
# git apply --cached docs/09-entregas/2026/evidencias/FAC-034/01-tokens.patch
# git commit -m "feat(web): adicionar tokens globais Centro de Comando"
# git apply --cached docs/09-entregas/2026/evidencias/FAC-034/02-frame-home.patch
# git commit -m "feat(web): redesenhar frame e home com núcleo SVG"
# git apply --cached docs/09-entregas/2026/evidencias/FAC-034/03-login-primitivos.patch
# git commit -m "feat(web): aplicar HUD ao login e componentes comuns"
# git apply --cached docs/09-entregas/2026/evidencias/FAC-034/04-projetos-operacao.patch
# git commit -m "feat(web): aplicar HUD a projetos e operação"
# git apply --cached docs/09-entregas/2026/evidencias/FAC-034/05-tarefas.patch
# git commit -m "feat(web): aplicar HUD ao quadro e lista de tarefas"
# git apply --cached docs/09-entregas/2026/evidencias/FAC-034/06-configuracoes-modais.patch
# git commit -m "feat(web): aplicar HUD a configurações e modais"
# git add docs/00-governanca/ORDEM-LEITURA.json docs/02-INDEX.md docs/04-CHANGELOG.md docs/03-modulos/controle/00-README.md docs/03-modulos/configuracao/00-README.md docs/04-features/01-control-settings.md docs/08-desenvolvimento/05-matriz-cobertura.md docs/08-desenvolvimento/09-backlog.md docs/08-desenvolvimento/tickets/controle/FAC-034-centro-comando.md docs/09-entregas/00-README.md docs/09-entregas/2026/2026-10-07-FAC-034-centro-comando.md docs/09-entregas/2026/evidencias/FAC-034 docs/09-entregas/2026/evidencias/design_handoff_centro_comando
# git diff --cached --stat
# git commit -m "docs(web): registrar entrega FAC-034 e evidências"
# git push origin developer  # Somente após revisão e autorização posterior; não executado nesta tarefa.
