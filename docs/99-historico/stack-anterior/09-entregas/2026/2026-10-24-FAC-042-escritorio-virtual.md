# FAC-042 — Escritório virtual isométrico

Data: 2026-10-24, America/Sao_Paulo. Implementação: IMPLEMENTADO localmente, diff sem commit sobre `main` em `d923802b120dc1ca0912c6e4459c8e3a7f6628d7`; status AWAITING_HUMAN. [Ticket](../../08-desenvolvimento/tickets/controle/FAC-042-escritorio-virtual.md).

## Objetivo, problema e solução

O responsável pediu um escritório 2D isométrico onde os agentes aparecem como funcionários trabalhando, primeiro com dados simulados, mas preparado para receber o trabalho real, renderizando só quando aberto. Solução: painel `OfficePanel` carregado sob demanda pelo item Escritório, desenhado em Canvas 2D sem bibliotecas novas. Referências de abordagem: escritórios pixel-art de agentes (personagens com máquina de estados e busca de caminho em grade) e cidades de agentes em canvas; nenhum código ou arte de terceiros foi copiado.

## Funcionamento

- **Modelo puro** (`office-model.ts`): grade 15×11, 10 mesas em duas fileiras (cadeira e ponto de visita por mesa), copa, estante, sofás e plantas. BFS em 4 direções, gerador determinístico, simulação por agente com poses `walk/type/deliver/drink/read/stand`.
- **Comportamento**: `working` → caminha até a própria cadeira e digita; `handoff` → caminha até o ponto de visita da mesa do destinatário, entrega por 2,6 s e volta; `idle` → alterna entre circular, café e leitura, evitando pontos já reservados.
- **Fonte de dados** (`office-feed.ts`): contrato `OfficeFeed { snapshot, subscribe }` com `OfficeAgentInput { id, name, role, activity }`. A simulada muda um agente a cada 4,5 s e só mantém timer com assinante. A fonte real (ticket posterior) deve traduzir eventos de execução para `idle | working | handoff`.
- **Desenho** (`office-render.ts`): piso e paredes em camada estática cacheada; móveis e cadeiras como sprites pré-renderizados; por quadro, só ordenação por profundidade, sprites e ~10 agentes procedurais. Telas acendem quando o dono digita. Reconstrução apenas em resize.
- **Ciclo de vida** (`VirtualOffice.tsx`): `requestAnimationFrame` limitado a 30 fps, pausado fora da viewport (IntersectionObserver), com aba oculta (`visibilitychange`) e no unmount; DPR limitado a 2. Movimento reduzido: sem laço, posições finais. Canvas `role="img"` + lista textual de agentes; nenhuma interação.

## Arquivos, dados e decisões

Novos: `apps/web/src/office/{office-model.ts, office-feed.ts, office-render.ts, VirtualOffice.tsx, OfficePanel.tsx, office.css, office-model.spec.ts, OfficePanel.spec.tsx}`. Alterados: `App.tsx` (destino `office`, `lazy`), `DashboardLayout.tsx` (`HomeDestination` + item Escritório), `DashboardLayout.spec.tsx`. Sem API, contrato, schema, dependência, autenticação ou dados persistidos. ADR não aplicável (apresentação local); lesson não aplicada.

## Diff e fontes

Diff recuperável com `git diff -- apps/web/src/App.tsx apps/web/src/DashboardLayout.tsx apps/web/src/DashboardLayout.spec.tsx` e `git status -- apps/web/src/office`. SHA-256 finais:

| Arquivo | SHA-256 |
|---|---|
| office/office-model.ts | `86213ddf69efc71f1035257b57ae511500be2182975d1e5ba7b9ad6cd48742e5` |
| office/office-feed.ts | `6c082df413137c427ad1fd7e681b8a35c4ae91ed43565a35492f5f9e7d438dc0` |
| office/office-render.ts | `eba0733532c266f6f85b3b8d94ae0b21a477c341f8b7b0894d89cab170adf328` |
| office/VirtualOffice.tsx | `3ef648081b69e29ab6e9804c164ad8e0df6876b733daee3fae40e60fc5efee18` |
| office/OfficePanel.tsx | `e645b34dc5f34012490c428bb4b5f2b025bd5f4bf09bce594536248cd8851df2` |
| office/office.css | `6d5b553c8ddbaaaae24d1b6b0644ff696415582a193982d0614943666105c1a3` |
| office/office-model.spec.ts | `e43a762860d9dc25566f73f93af4efefbc909385ccf05f539af93b2942860d6e` |
| office/OfficePanel.spec.tsx | `f191f762208e21f9b7b6c79192fc26211def6808ff4f958e3a882fd423411947` |
| App.tsx | `3ae268b7f9ed971af76a33ca0e841afb3a18362ff0ab63ca7dc85b58d431e222` |
| DashboardLayout.tsx | `e480f61bdf1c9f1b411e068750c4feb0595006f7b2b3b101f4d6a42184275b65` |
| DashboardLayout.spec.tsx | `ba23968eb9d2da8b86f9a6ce6949c2692ddd32e7fe59123d55bfcf7cf47f802e` |

Leitura explícita: `CLAUDE.md` (`48f6bb58ec5df459e1e0200fe07c541f341cab37cce135279123a70598f09c25`), `docs/00-governanca/01-POLITICA-IA.md` (`971ef65f12d021b0cef4f898b69114949383d5af392ee18f4a878632cf3b2507`), `docs/00-governanca/02-POLITICA-DOCUMENTACAO.md` (`10d419ece5476593c37a092525d18f8c358f95f582eb5faf6f42540f1b1c022e`), módulo de controle e entrega FAC-032. Autoload continua NÃO VERIFICADO; nenhuma credencial lida.

## Verificações

Checks executados em cópia descartável do repositório (mesma base, `npm ci --ignore-scripts`), com os arquivos finais acima copiados depois para o workspace.

| Comando/procedimento | Resultado | Evidência |
|---|---|---|
| `npm run test -w @le-fabrique/web` (baseline) | PASS | 26 arquivos, 73 testes |
| `npm run test -w @le-fabrique/web` (final) | PASS | 28 arquivos, 82 testes |
| `npm run typecheck -w @le-fabrique/web` | PASS | sem erros |
| `npx biome check .` | PASS com 1 aviso pré-existente | aviso em `apps/api/src/orchestration/run-delivery.service.ts`, fora do diff |
| `npm run build -w @le-fabrique/web` | PASS | chunk `OfficePanel` separado: 18,8 kB (7,7 kB gzip) |
| Render isolado em Chromium (Playwright) do módulo `office` | PASS | quadros em 1,2 s e 20 s: agentes digitando, entrega com pasta, café e leitura; zero exceções |
| `python3 scripts/validar-documentacao.py` | FAIL pré-existente | erros apenas de links/índice para `skills/` reorganizado na base; nenhum erro nos arquivos desta entrega |
| Browser no app completo | NOT_RUN | ambiente sem servidor web do projeto em execução |
| Revisão independente | NOT_RUN | pendente |

## Riscos, limites e rollback

Dados simulados; não representam execuções. Etiquetas podem se sobrepor quando agentes ficam lado a lado. Fonte real, entrada/saída de agentes por cadastro e mais de 10 agentes ficam para ticket posterior. Rollback: remover `apps/web/src/office/` e reverter os três arquivos alterados.

## Documentação e aceite

Atualizados: [módulo de controle](../../03-modulos/controle/00-README.md), [changelog](../../04-CHANGELOG.md), [backlog](../../08-desenvolvimento/09-backlog.md), [matriz](../../08-desenvolvimento/05-matriz-cobertura.md), [índice](../../02-INDEX.md) e [histórico de entregas](../00-README.md). Aceite humano da revisão exata pendente.
