# FAC-042 — Escritório virtual isométrico (demonstração visual)

Status READY por pedido explícito do responsável em 2026-10-24 (America/Sao_Paulo). Responsável: implementação nesta sessão; aceite pelo proprietário. Dependências: nenhuma de backend. Risco R1: apresentação no painel, sem API, dados persistidos ou execução.

## Objetivo e comportamento atual/esperado

Atual (base `d923802b120dc1ca0912c6e4459c8e3a7f6628d7`): o item Escritório do menu abre a prévia genérica de áreas futuras. Esperado: abrir um escritório 2D isométrico com os agentes como funcionários. Trabalhando: sentado na própria mesa, digitando. Entrega para outro agente: levanta, vai até a mesa do destinatário e entrega. Livre: circula, toma café ou lê. Mesas vazias disponíveis para novos agentes. Somente visual: sem cliques, ações ou efeitos.

## Escopo permitido e proibido

Permitido: `apps/web/src/office/*`, novo destino `office` em `DashboardLayout.tsx`/`App.tsx`, specs web, documentação. Proibido: API, contratos, providers, credenciais, novas dependências, merge/push/deploy. Dados simulados até ticket posterior ligar eventos reais.

## Critérios de aceite verificáveis

1. Menu Escritório abre o painel com `aria-current="page"` no item.
2. Canvas com `role="img"` e descrição; lista textual dos agentes e da atividade.
3. Agente trabalhando termina sentado digitando na própria mesa; entrega vai ao ponto de visita da mesa do destinatário; ocioso nunca ocupa tile bloqueado.
4. Laço de desenho só roda com o painel montado, visível na viewport e com aba ativa; fonte simulada sem timers quando não há assinante; código carregado sob demanda.
5. `prefers-reduced-motion`: sem animação contínua, posições finais redesenhadas apenas quando a atividade muda.

## Baseline e comandos reais de checks

Baseline web: 26 arquivos / 73 testes PASS. Gates: `npx biome check .`, `npm run typecheck -w @le-fabrique/web`, `npm run test -w @le-fabrique/web`, `npm run build -w @le-fabrique/web`, `python3 scripts/validar-documentacao.py`.

## Limites de runtime/custo e providers elegíveis

N/A: nenhuma execução de IA, provider ou gasto. Uso/custo da sessão não informado pelo ambiente.

## Entregáveis e impacto documental

Módulo de controle atualizado, entrega, changelog, backlog, índice, histórico de entregas e matriz. Sem ADR (decisão local de apresentação; Canvas 2D sem dependência nova).

## Evidências, revisão e aceite

[Entrega e resultados](../../../09-entregas/2026/2026-10-24-FAC-042-escritorio-virtual.md). AWAITING_HUMAN; DONE depende de aceite da revisão exata e de revisão independente.
