# FAC-043 — Visual sóbrio com seletor Novo/Antigo

Data: 2026-10-25, America/Sao_Paulo. Implementação: IMPLEMENTADO localmente, diff sem commit sobre `main` em `d923802b120dc1ca0912c6e4459c8e3a7f6628d7`, acumulado com FAC-042; status AWAITING_HUMAN. [Ticket](../../08-desenvolvimento/tickets/controle/FAC-043-visual-sobrio.md).

## Objetivo, problema e solução

O responsável achou o HUD carregado e pouco confiável e aprovou no canvas uma proposta mais sóbria, ainda com cara de central de comando. Pediu para levá-la ao código mantendo um seletor entre o novo e o antigo até decidir. Solução: tema por atributo `data-layout` no `<html>`, sem duplicar telas. "Novo" redefine os tokens `--hud-*` e neutraliza brilhos/grade; "Antigo" deixa o HUD intacto.

## Funcionamento

- **Preferência** (`layout-preference.ts`): `novo | antigo`, padrão `novo`, salva em `localStorage` (`le-fabrique.layout.v1`) com fallback silencioso; `main.tsx` aplica antes do primeiro render, evitando piscar.
- **Seletor**: grupo Novo/Antigo na barra superior do `DashboardLayout`, presente em todas as telas, com `aria-pressed`.
- **Tema** (`theme-sobrio.css`): fundo grafite `#0B0D12`, superfícies `#12151C`, linhas `#232834`, acento `#6E9BFF`, status verde/âmbar/vermelho/violeta, Geist e Geist Mono, raio 10px, sem glow, grade ou caixa alta nos títulos; trilho de configurações à direita oculto (Configurações segue no menu). Projetos, Tarefas, Configurações, Conta e Escritório herdam pelos tokens.
- **Painel novo** (`SobrioHome.tsx`): data e saudação, título, Novo projeto/Nova tarefa, faixa "Requer sua decisão" (notificação mais urgente), indicadores numa faixa, Orquestração em largura total com o núcleo animado existente e a lista de agentes com `meter`. Dados continuam simulados.
- **Núcleos**: JARVIS renomeado para Núcleos no menu (antes "Núcleo IA"), no núcleo SVG (`aria-label` "Núcleos, centro de orquestração da La fabrique") e nos agentes, nas duas opções.
- `DashboardLayout` aceita `layout` opcional para forçar um visual (usado nos testes).

## Arquivos, dados e decisões

Novos: `layout-preference.ts`, `layout-preference.spec.ts`, `SobrioHome.tsx`, `theme-sobrio.css`. Alterados: `DashboardLayout.tsx`, `OrchestrationCore.tsx`, `HomeDashboard.spec.tsx`, `main.tsx`. Sem API, dados, dependências (fontes via Google Fonts, como o HUD já fazia) ou autenticação. Não implementado nesta entrega: página separada de lista de projetos do canvas, por depender de dados reais. Esta entrega altera `DashboardLayout.tsx` depois do hash registrado na FAC-042; a evidência daquele arquivo passa a ser a desta tabela.

| Arquivo | SHA-256 |
|---|---|
| DashboardLayout.tsx | `fee6162937659ec6faf256d4b15107782e2d5d39df5a3a797dc5fd115a271734` |
| OrchestrationCore.tsx | `abb9397bf244f671d1ff2b8938015a5294b48f7cb3613995d7e8c434582effa3` |
| HomeDashboard.spec.tsx | `d9663511f50a5d042ddf7d924a3bc5e5e17735e72b99edd6160d396f8c6b1a16` |
| main.tsx | `1d66044c12af30a248d3ade3364f237f8c58084b0f4407aa2e6ea31f9a2da0fb` |
| layout-preference.ts | `1249325111404fc84d6c3ecc83a1bcfed86b3b0665d2e08c3ea412e3ea24f3a9` |
| layout-preference.spec.ts | `7152b6b6bd829aa5a2640d4b6172abff0e38d1b5d9c3d093852efabd3a740fb7` |
| SobrioHome.tsx | `f302a4e6fafdcd2d8752ed63f41ecc68a2000f77af42d42bc30e9f80523ca6b3` |
| theme-sobrio.css | `6a4d6e986e761001e978758e92785f28b02a99ed3c3189df8384d3c4d2fd1b52` |

## Verificações

Checks em cópia descartável da mesma base com os arquivos finais acima.

| Comando/procedimento | Resultado | Evidência |
|---|---|---|
| `npm run test -w @le-fabrique/web` | PASS | 29 arquivos, 86 testes (82 antes) |
| `npm run typecheck -w @le-fabrique/web` | PASS | sem erros |
| `npx biome check .` | PASS com 1 aviso pré-existente | `apps/api/.../run-delivery.service.ts`, fora do diff |
| `npm run build -w @le-fabrique/web` | PASS | bundle principal 143,4 kB gzip |
| Home "novo" renderizada no servidor e aberta em Chromium com o CSS final | PASS | topo com seletor, faixa de decisão, indicadores e Orquestração conferidos visualmente |
| `python3 scripts/validar-documentacao.py` | FAIL pré-existente | mesmos erros de `skills/` da base; nenhum nos arquivos desta entrega |
| App completo no navegador (telas internas, interação do seletor) | NOT_RUN | sem servidor do projeto em execução nesta sessão |
| Revisão independente | NOT_RUN | pendente |

## Riscos, limites e rollback

Telas internas mudam só por tokens/regras globais; algum detalhe pode precisar de ajuste fino ao ver no app. Rollback: selecionar "Antigo" (sem deploy) ou remover `theme-sobrio.css`, `SobrioHome.tsx`, `layout-preference*` e reverter os quatro arquivos alterados. Quando o responsável decidir, remover o tema descartado e o seletor em ticket próprio.

## Documentação e aceite

Atualizados: [módulo de controle](../../03-modulos/controle/00-README.md), [changelog](../../04-CHANGELOG.md), [backlog](../../08-desenvolvimento/09-backlog.md), [matriz](../../08-desenvolvimento/05-matriz-cobertura.md), [índice](../../02-INDEX.md) e [histórico de entregas](../00-README.md). Aceite humano da revisão exata pendente.
