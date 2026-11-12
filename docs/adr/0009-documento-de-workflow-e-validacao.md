# ADR-0009 — Documento de workflow, catálogo de nós e validação

- **Estado:** ACEITA · **Ticket:** [T07-01](../NEW%20MVP/03-BACKLOG.md#t07-01) · **Épico:** E07 · **Decisões:** D09, D13, D44–45, D58–60, D64, D94

## Decisão
1. **Documento JSON por workflow** (`workflows.graph`): nós (`id`, `type`, posição, `props`) e conexões (`from`/`from_port` → `to`/`to_port`), com controle otimista (`expected_version`) e limites (200 nós, 400 conexões, 1 MiB). Rascunho e versão publicada são coisas distintas: publicação imutável e congelamento são do T07-03.
2. **Catálogo de nós como fonte única** (`workflows.Catalog`, exposto em `GET /workflows/node-types`): tipos `start`, `end`, `agent`, `skill`, `script`, `checks`, `git`, `docs`, `condition`, `loop`, `wait`, `human_decision`, `subflow`, `converge`, cada um com portas de entrada/saída e campos tipados (texto, área de texto, inteiro com faixa, enumeração). O editor desenha portas e formulários a partir dele; o validador usa o mesmo catálogo.
3. **Validação no servidor, com localização** (`Issue{node, field, code, message}`): início único, ao menos um fim, tipo e IDs válidos, campos obrigatórios/faixas/opções/desconhecidos, portas existentes, conexões sem duplicidade nem auto-ligação, **uma conexão por saída** (ramificar exige `condition`) e **uma por entrada** (exceto `end`, `loop` e `converge`), saídas obrigatórias conectadas (`error` é opcional), nós inalcançáveis e **ciclos que não passam por um `loop`**. O `loop` exige `max_iterations` de 1 a 100.
4. **Rascunho inválido pode ser salvo**; os problemas voltam em `issues` (leitura, gravação e `POST /workflows/validate`, que valida sem persistir e o editor usa ao editar). Só a publicação exigirá zero problemas.
5. **Editor**: canvas SVG próprio (sem dependência nova): arrastar nós em grade de 10, conectar clicando saída e depois entrada, selecionar e remover, painel de propriedades gerado do catálogo, erros junto do campo e lista de validação com "Localizar".

## Limites
- O editor não tem zoom, pan nem desfazer; testado em jsdom (não em navegador real).
- Referências a agentes, skills e subfluxos são texto livre até os catálogos (E08/E09) e a publicação (T07-03); o motor de execução é o T07-02.
