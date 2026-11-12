# FAC-043 — Visual sóbrio com seletor Novo/Antigo

Status READY por pedido explícito do responsável em 2026-10-25 (America/Sao_Paulo), escolhendo "novo design + seletor". Responsável: implementação nesta sessão; aceite pelo proprietário. Risco R1: apresentação, sem API ou dados.

## Objetivo e comportamento atual/esperado

Atual: tema HUD ciano (FAC-034) em todas as telas; orquestrador chamado JARVIS. Esperado: aplicar a proposta sóbria do canvas de design (grafite, um acento azul, Geist/Geist Mono, sem brilho nem grade) em todas as telas, mantendo o HUD disponível pelo seletor Novo/Antigo no topo até a decisão final. Painel novo: título, faixa "Requer sua decisão", indicadores em uma faixa e Orquestração em largura total; sem barra de comando, feed ou projetos na home. O cérebro do sistema passa a se chamar Núcleos (menu, núcleo, agentes).

## Escopo permitido e proibido

Permitido: `apps/web/src` (tema, frame, home, núcleo, specs) e documentação. Proibido: API, contratos, dependências, providers, merge/push/deploy. Página própria de lista de projetos (canvas) fica para ticket posterior por depender de dados reais.

## Critérios de aceite verificáveis

1. Seletor Novo/Antigo no topo de todas as telas, com `aria-pressed`, preferência salva no navegador e padrão "novo".
2. "Antigo" mantém o HUD e a home anteriores; "novo" troca tokens/tipografia e mostra a home sóbria.
3. Nenhuma ocorrência de JARVIS na interface; menu exibe Núcleos.
4. Contratos de navegação/aria existentes preservados nas duas opções.

## Baseline e comandos reais de checks

Baseline após FAC-042: 28 arquivos / 82 testes web PASS. Gates: `npx biome check .`, `npm run typecheck -w @le-fabrique/web`, `npm run test -w @le-fabrique/web`, `npm run build -w @le-fabrique/web`, `python3 scripts/validar-documentacao.py`.

## Limites de runtime/custo e providers elegíveis

N/A: sem execução de IA, provider ou gasto.

## Entregáveis e impacto documental

Módulo de controle, entrega, changelog, backlog, matriz, índices. Sem ADR: decisão visual ainda em avaliação pelo responsável.

## Evidências, revisão e aceite

[Entrega e resultados](../../../09-entregas/2026/2026-10-25-FAC-043-visual-sobrio.md). AWAITING_HUMAN.
