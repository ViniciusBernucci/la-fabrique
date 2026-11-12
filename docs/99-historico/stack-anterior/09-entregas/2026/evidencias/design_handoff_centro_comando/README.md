# Handoff: Tema Centro de Comando (todo o La fabrique)

## Visão geral
Novo visual da tela inicial (`HomeDashboard` → `DashboardLayout` sem `children`). Estilo HUD / "Jarvis": núcleo de orquestração animado no centro (JARVIS), ligado aos agentes; painéis de status, diagnóstico, feed, agentes, pipeline e métricas ao redor; menu lateral à esquerda e trilho de Configurações à direita.

## Sobre os arquivos
`Fabrica Dashboard v3.dc.html` é uma **referência de design em HTML** — não é código para copiar. Recriar em `apps/web` (React + TS + Vite, CSS em `home-dashboard.css`) seguindo os padrões existentes. Fidelidade: **hi-fi** (cores, tipografia, espaçamento e animações finais).

## Regras de integração (obrigatórias)
- Branch: **`developer`**. Antes de começar: `git fetch origin && git checkout developer && git pull && git merge origin/main` (incorporar as mudanças recentes da main; resolver conflitos preservando a main).
- Manter o contrato de `DashboardLayout`: props `onNavigate`, `onHome`, `activeDestination`, `children`; `HomeDestination = "control" | "settings" | "tasks"`.
- Quando há `children`, o frame (sidebar + topbar + `section.workspace-content`) continua igual em comportamento — só o visual muda. Classes `home-sidebar`, `home-topbar`, `workspace-content` precisam continuar existindo (spec).
- Menu lateral = itens existentes + novos, nesta ordem: Painel, Projetos (`control`), Tarefas (`tasks`), Agentes de IA (`settings`), **Núcleo IA**, **Pipeline**, **Revisões** (badge), **Repositórios**, **Histórico**, Escritório, Usuários, Configurações (`settings`). Os novos ainda não têm tela: abrem a prévia (`setPreview(nome)`), como Escritório/Usuários hoje. Não duplicar itens.
- Trilho direito (Geral, Modelos, Integrações, Acessos, Alertas, Equipe): atalhos para a mesma tela de Configurações (`onNavigate("settings")`); não substituem o item Configurações do menu lateral.
- Manter: comportamento da navegação (`aria-label` por item, `is-active` + `aria-current="page"`), sidebar recolhível + colapso em ≤760px, busca com ⌘K/Ctrl+K e resultados, `dialog.home-preview` (`aria-labelledby="home-preview-title"`), skip link, `h1` sr-only "Central de controle La fabrique", aviso de "dados simulados", `meter` com `aria-label="Atividade simulada de {agente}"`.
- Usar os dados mock já existentes no arquivo (agentes, notificações, projetos, atalhos) — os textos do HTML de referência são ilustrativos.
- A imagem `/images/control-room-reference.png` é substituída pelo núcleo animado → atualizar `HomeDashboard.spec.tsx` (trocar a asserção da imagem por uma do núcleo, ex. `aria-label="Núcleo de orquestração JARVIS"`).
- Seguir `CLAUDE.md`/políticas: registrar entrega em `docs/09-entregas/<ano>/`, changelog e backlog. Sem merge/push sem aprovação.

## Layout
- Raiz: coluna; fundo `#02060c` + `radial-gradient(1000px 640px at 50% 34%, rgba(45,226,255,.12), transparent 65%)` + grade 32px de linhas `rgba(45,226,255,.035)`.
- **Topbar** (uma linha, sem quebra): logo (anel tracejado ciano) + "LA FABRIQUE / CENTRO DE COMANDO"; pill status "OPERACIONAL" (`#19f5c8`); relógio central ao vivo (data mono 11px `#6fa3b8`, hora Orbitron 700 24px `#2de2ff` com `text-shadow 0 0 14px rgba(45,226,255,.6)`); busca (flex `0 1 200px`, min-width 0); sino/ajuda; chip de perfil. Borda inferior `rgba(45,226,255,.16)`, bg `rgba(2,10,18,.7)`, padding 12px 20px.
- **Sidebar esquerda** 226px: itens 10px 12px, raio 8, Rajdhani 600 15.5px, cor `#8fbccd`; ativo: bg `rgba(45,226,255,.1)`, borda `rgba(45,226,255,.4)`, texto `#e8fbff`; hover bg `rgba(45,226,255,.08)`. Badges mono 11px com borda na cor. Rodapé: card "Projeto ativo" com barra 3px ciano com glow.
- **Trilho direito** 78px: rótulo "CONFIG", botões 62px (ícone 20px + rótulo 11.5px) → `onNavigate("settings")`.
- **Main** (padding 16, gap 14, overflow hidden):
  - Linha superior em `flex-wrap`: coluna esquerda `flex:1 1 240px` (Status do núcleo + Diagnóstico com 3 gauges circulares 66px, traço 4px), centro `flex:2.2 1 420px` (núcleo), direita `flex:1 1 250px` (Feed). Em telas estreitas as laterais descem.
  - Linha inferior: `grid repeat(auto-fit, minmax(300px,1fr))` — Agentes (grid 2 col), Pipeline (barras de progresso 3px), Métricas (2×2, valores Orbitron 22px).
- Painéis: bg `rgba(4,16,28,.72)`, borda `1px rgba(45,226,255,.2)`, raio 12, padding 14. Título mono 11px, letter-spacing .16em, `#2de2ff`, estilo `NOME_DO_PAINEL`.
- Item do feed com ação humana: borda `rgba(255,93,122,.45)`, bg `rgba(255,93,122,.08)`.

## Núcleo animado (componente `OrchestrationCore`)
SVG `viewBox 0 0 600 600`, centro (300,300), largura 100% / max 520px, `aspect-ratio:1`. Camadas (de fora pra dentro):
1. 120 ticks (r 258→252, a cada 10º r 246), ciano, girando 120s.
2. Anel tracejado âmbar r 276 (`#ffb547`, opacidade .35, dash `2 7`), girando ao contrário 90s.
3. Dois arcos ciano r 208 (−30→60º, 150→215º), traço 3, glow, 26s.
4. Anel r 186 + 2 arcos `#19f5c8`, sentido inverso 18s.
5. Linhas centro→agente (6 agentes a r 236, ângulos −90º + i·60º): base com opacidade .18 + linha tracejada `6 22` animando `stroke-dashoffset` (1.6s + i·0.2s) + partícula r 3.5 percorrendo agente→centro (`animateMotion`, 2.4s + i·0.35s).
6. Esfera r 120 (radial gradient) + 6 elipses de longitude (rx 120, ry 34, rotações 0–150º) girando 14s.
7. Rede neural: 34 pontos dentro de r 88, cada um ligado aos 2 vizinhos mais próximos; pontos piscam (`blink`, 1.4–4s, delays escalonados). Usar seed fixa (determinístico, sem mudar entre renders).
8. Três órbitas elípticas (rx 158, ry 52) inclinadas −25º/40º/95º com partícula (9s/13s/11s).
9. 3 ondas de pulso (r 60, scale .6→1.6, 3s, delay 0/1/2s) + núcleo r 46 com gradiente branco→ciano pulsando (2.6s) e texto "JARVIS" (Orbitron 800 ~10.5px, precisa caber no círculo).
10. Nós dos agentes: círculo r 27 (bg `#03121d`, borda na cor do agente, glow), anel tracejado r 33 girando, sigla mono 12px, nome Rajdhani 700 15px acima/abaixo.
- Animações via `@keyframes` em CSS (`spin`, `spinr`, `pulse`, `wave`, `flow`, `blink`), `transform-box: view-box; transform-origin: 300px 300px`. Respeitar `prefers-reduced-motion: reduce` (pausar).
- Abaixo do núcleo: caixa de saudação ("Bom dia/Boa tarde/Boa noite, {usuário}") + campo de comando "Descreva uma nova tarefa…" + botão ENVIAR → `onNavigate("tasks")`.
- Agentes do núcleo: usar o array `agents` existente (JARVIS = centro; demais = nós).

## Interações
- Relógio atualiza a cada 1s (`setInterval` em `useEffect`, limpar no unmount).
- Itens clicáveis mantêm os destinos atuais (`onNavigate`/`setPreview`). Hover: borda vai para a cor do item.
- Foco: manter `outline 2px #54c5ff`.

## Tokens
- Cores: bg `#02060c`; ciano `#2de2ff`; verde-água `#19f5c8`; azul `#5aa8ff`; violeta `#b18cff`; âmbar `#ffb547`; rosa (alerta) `#ff5d7a`; cinza ocioso `#6b8797`; texto `#d6f6ff` / forte `#e8fbff`; secundário `#8fbccd` / `#6fa3b8`.
- Fontes (Google Fonts): Orbitron 500/700/800 (títulos, números), Rajdhani 500/600/700 (UI), JetBrains Mono 400/500/600 (rótulos técnicos).
- Raios: 8 (itens), 10–12 (painéis), 14 (núcleo), 99 (pills). Glow: `box-shadow 0 0 8px <cor>` em dots/barras.

## Aplicar em todo o software
O padrão HUD vale para **todas as telas**, não só a home. Escopo:
- **Tokens**: importar `theme-hud.css` em `main.tsx` (antes de `styles.css`). Trocar as cores, fontes e raios fixos de `styles.css`, `tasks.css` e `home-dashboard.css` pelas variáveis `--hud-*` (ex. `--home-bg`→`--hud-bg`, `--home-panel`→`--hud-surface`, `--home-line`→`--hud-line`, `--home-muted`→`--hud-text-muted`, `--home-blue`→`--hud-cyan`; mapa de `.tone-*`: blue→`--hud-blue`, purple→`--hud-violet`, green→`--hud-teal`, orange/yellow→`--hud-amber`, red→`--hud-rose`, cyan/teal→`--hud-cyan`, silver→`--hud-idle`). Sem novas cores fora dos tokens.
- **Frame**: todas as áreas (`control`, `tasks`, `settings`) já renderizam dentro de `DashboardLayout` → herdam topbar, sidebar e trilho direito novos.
- **Login** (form de token em `App.tsx`): card centralizado no fundo HUD, com o anel do núcleo pequeno acima, título em Orbitron e campo mono.
- **Projetos / Operação**: `OperationPanel`, `ProjectDefinitionPanel`, `RunPanel`, `RunControlPanel`, `RunRecoveryPanel`, `RunResumePanel`, `DeliveryPanel`, `ArtifactPanel`, `HandoffAlternatives`.
- **Tarefas**: `TasksPanel` + `tasks.css` (lista/tabela de tickets com badges de status).
- **Configurações**: `SettingsPanel`, `SettingsModal`, `TeamsPanel`, `PresetAgents`.
- **Mensagens** (`message` em `App.tsx`): usar o padrão "Mensagens do sistema" (OK/INFO/ATENÇÃO/ERRO).

Referência de componentes: `HUD UI Kit.dc.html`:
- **Painel**: bg `--hud-surface`, borda `1px --hud-line`, raio 12, padding 14–16. Título mono 11px, `letter-spacing .16em`, `--hud-cyan`, em MAIÚSCULAS com `_` (ex. `TICKETS_DO_PROJETO`).
- **Botões**: primário (bg `--hud-cyan`, texto `#02101a`, 700, glow `0 0 18px rgba(45,226,255,.35)`); secundário (bg `rgba(45,226,255,.06)`, borda `--hud-line-strong`); terciário (transparente, `--hud-text-muted`); perigo (bg `rgba(255,93,122,.1)`, borda `rgba(255,93,122,.5)`, texto `#ff8fa3`); desabilitado (`--hud-idle`). Raio 8, padding 9px 16px, Rajdhani 15px.
- **Inputs**: bg `--hud-surface-solid`, borda `rgba(45,226,255,.25)`, raio 8, padding 10px 12px. Foco: borda `--hud-cyan` + `0 0 0 3px rgba(45,226,255,.15)`. Erro: borda `rgba(255,93,122,.6)` + mensagem mono 11px `#ff8fa3`. SHA, URL e IDs usam a fonte mono.
- **Status de ticket** (badge mono 11px, borda = cor +40% alpha, bg = cor +8%): DRAFT `--hud-idle`, READY `--hud-cyan`, RUNNING `--hud-blue`, IN_REVIEW `--hud-violet`, AWAITING_HUMAN `--hud-rose`, BLOCKED `--hud-amber`, DONE `--hud-teal`, FAILED `--hud-rose`. Mapear todos os status reais de `@le-fabrique/contracts`; nenhum pode ficar sem cor.
- **Tabela**: cabeçalho mono 10.5px `--hud-text-dim`, linha 1px `rgba(45,226,255,.08)`, hover `rgba(45,226,255,.04)`, rolagem horizontal em telas estreitas.
- **Abas / segmentado**: contêiner com borda `rgba(45,226,255,.15)`; aba ativa com bg `rgba(45,226,255,.12)`.
- **Log de execução**: bg `#020a12`, mono 12.5px, linha 1.8, prefixo colorido por agente.
- **Modal** (`dialog`): bg `--hud-surface-solid`, borda `--hud-line-strong`, glow `0 0 40px rgba(45,226,255,.15)`, backdrop `rgba(2,6,12,.75)`.
- **Estado vazio**: borda tracejada, anel do núcleo pequeno, título + texto + botão secundário.
- Não mudar regras de negócio, chamadas à API, textos de erro testados nem a estrutura/aria que os specs verificam. Rodar os specs de cada painel.

## Arquivos
- `theme-hud.css`: tokens globais prontos para importar.
- `HUD UI Kit.dc.html`: referência dos componentes base.
- `Fabrica Dashboard v3.dc.html` — referência visual (abrir no navegador).
- `PROMPT-CLAUDE-CODE.md` — prompt pronto para o Claude CLI.
