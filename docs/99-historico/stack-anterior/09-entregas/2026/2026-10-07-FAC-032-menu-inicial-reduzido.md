# FAC-032 — Menu inicial reduzido

Data: 2026-10-07, Europe/Berlin. Implementação: IMPLEMENTADO localmente; status AWAITING_HUMAN. Base `9439cde4ce52a09225c9476c19baa89319802523`, branch `developer`, diff sem commit. Pedido e continuação no workspace autorizados pelo responsável; alterações documentais anteriores preservadas.

## Objetivo e comportamento

Reduzir o menu principal a Painel, Projetos, Agentes de IA, Escritório, Usuários e Configurações, nessa ordem. Retirados Conteúdos, Vendas, CRM / Leads, Eventos, Oportunidades e Financeiro; renomeado Agentes IA e adicionado Escritório. Destinos existentes, recolhimento e atributos acessíveis preservados. Escritório e Usuários abrem a prévia já existente para áreas futuras. Atalhos da home fora do escopo.

## Arquivos, dados e decisões

Código: `apps/web/src/DashboardLayout.tsx`, somente lista `navigation`. Sem mudança de API, dados, eventos, dependências, autenticação ou arquitetura. ADR/lesson novos não se aplicam a este ajuste de navegação.

## Diff e fontes

Diff recuperável com `git diff -- apps/web/src/DashboardLayout.tsx`; SHA-256 do arquivo final `30b865b623531adbf78a639cc0afc225a8424b8b0379c4de0eeb22f82070ab8b`.

Leitura explícita: `AGENTS.md` (SHA-256 `7ca4664cba5de95bb0d19d0b8131c078f3d917486b143afda60edc841a0665ca`), `docs/00-governanca/01-POLITICA-IA.md` (`cf459b7d679a9796427d40b734dc0f2cabf5405de94c7ab05912574e1c3e6971`) e `docs/00-governanca/02-POLITICA-DOCUMENTACAO.md` (`10d419ece5476593c37a092525d18f8c358f95f582eb5faf6f42540f1b1c022e`). Manual, README, piloto, prompt inicial, guia, ADR-003 e capítulos afetados lidos. Autoload continua NÃO VERIFICADO; nenhuma credencial lida.

## Verificações

Revisão: base acima + diff local, arquivo final identificado pelo hash acima.

| Comando/procedimento | Resultado | Evidência |
|---|---|---|
| `npm run test -w @le-fabrique/web` | PASS | 18 arquivos, 46 testes |
| `npm run typecheck -w @le-fabrique/web` | PASS | saída 0 |
| `npm run build -w @le-fabrique/web` | PASS | 135 módulos, bundle gerado |
| `npx biome check apps/web/src/DashboardLayout.tsx` | PASS | 1 arquivo, sem correções |
| Conferência do diff | PASS | seis entradas, ordem e destinos conferidos |
| Browser / revisão independente | NOT_RUN | não executados nesta sessão |

Baseline documental antes das atualizações FAC-032: `python3 scripts/validar-documentacao.py` PASS, zero erros. Validação final após atualização: PASS, 379 Markdown correntes, 2298 links locais, zero erros. Sem novo teste unitário para mudança de baixo impacto; suite existente executada.

## Riscos, limites e rollback

Escritório e Usuários ainda são prévias. Não há alegação de tela funcional nova, deploy ou aceite. Rollback: restaurar somente as entradas alteradas de `navigation` e ajustar documentação relacionada, preservando trabalho prévio; não usar reset global.

## Documentação

Atualizados [controle](../../03-modulos/controle/00-README.md), [feature](../../04-features/01-control-settings.md), [backlog](../../08-desenvolvimento/09-backlog.md), [matriz](../../08-desenvolvimento/05-matriz-cobertura.md), [changelog](../../04-CHANGELOG.md), [índice](../../02-INDEX.md) e [histórico](../00-README.md). Relato é apêndice, sem novo capítulo didático.

## Uso de IA e aceite

Cliente Codex; modelo efetivo, tokens, custos fixos/extra/API e modo de cobrança não disponíveis nesta sessão. Sem instalação, login, handoff ou agente adicional. Revisão independente e aceite humano da revisão exata pendentes; sem commit, push ou deploy.
