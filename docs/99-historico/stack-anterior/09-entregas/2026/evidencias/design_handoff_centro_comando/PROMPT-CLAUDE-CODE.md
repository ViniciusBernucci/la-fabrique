# Prompt para o Claude CLI

Antes de rodar, coloque esta pasta (`design_handoff_centro_comando/`) dentro do repositório, por exemplo em `docs/09-entregas/2026/evidencias/design_handoff_centro_comando/`.

```bash
git fetch origin
git checkout developer
git pull origin developer
git merge origin/main   # traz as mudanças recentes da main; resolva conflitos preservando a main
```

Cole no Claude CLI:

```
Leia CLAUDE.md, as políticas indicadas nele e docs/09-entregas/2026/evidencias/design_handoff_centro_comando/README.md por inteiro.

Tarefa: aplicar na branch developer o tema "Centro de Comando" em TODO o apps/web: (1) home nova em DashboardLayout.tsx + home-dashboard.css, com referência hi-fi em "Fabrica Dashboard v3.dc.html"; (2) tokens globais de theme-hud.css em todas as telas (login, Projetos/Operação, Tarefas, Configurações, modais), seguindo "HUD UI Kit.dc.html" e a seção "Aplicar em todo o software" do README. Faça em commits separados: tokens → frame/home → cada área.

Restrições:
- Antes de editar, compare developer com origin/main (git log/diff) e preserve todas as mudanças recentes da main em DashboardLayout.tsx, home-dashboard.css, App.tsx e specs.
- Manter o contrato de DashboardLayout (props, HomeDestination, navegação, aria-labels, sidebar recolhível, busca ⌘K, dialog de prévia, aviso de dados simulados).
- Criar o componente OrchestrationCore (SVG animado em CSS, respeitando prefers-reduced-motion) no lugar da imagem control-room-reference.png.
- Não alterar regras de negócio, chamadas de API nem textos/aria usados nos specs.
- Atualizar HomeDashboard.spec.tsx e DashboardLayout.spec.tsx só onde o visual mudou; rodar lint (biome), typecheck e testes até passar.
- Registrar a entrega em docs/09-entregas/2026/, changelog e backlog conforme a política.
- Não fazer push nem merge; ao final mostre o diff resumido e os comandos de commit/push comentados.
```

Commit e push (rodar depois de revisar):

```bash
# git add apps/web docs
# git commit -m "feat(web): tema Centro de Comando em todo o app e núcleo de orquestração animado"
# git push origin developer
```
