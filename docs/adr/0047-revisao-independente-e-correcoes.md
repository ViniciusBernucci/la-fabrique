# ADR-0047 — Revisão independente e correções

- **Estado:** ACEITA · **Ticket:** [T14-03](../NEW%20MVP/03-BACKLOG.md#t14-03) · **Épico:** E14 · **Relacionadas:** [ADR-0040](0040-checks-baseline-e-classificacao-de-falhas.md), [ADR-0043](0043-classificacao-de-risco-por-pisos-deterministicos.md), [ADR-0044](0044-controles-operacionais-pausa-e-suspensao.md), [ADR-0031](0031-fallback-e-disponibilidade-de-contas.md)

## Decisão
1. **Revisor por projeto, com fallback:** `review_configs` guarda o revisor, a cadeia de alternativas e o **limite de rodadas (padrão 2)**. `Pick` escolhe o primeiro da cadeia que **não seja quem implementou** e esteja disponível; sem nenhum, `ErrNoReviewer`.
2. **Revisão inicial independente:** a primeira rodada exige revisor diferente do implementador (`ErrNotIndependent`) — e uma **restrição do banco** repete a garantia. Uma única revisão ativa por tarefa. Nas rodadas seguintes o revisor pode corrigir (`BeginFix`).
3. **Achados com severidade:** `critical` e `relevant` **bloqueiam**; `low` fica **registrado** (estado `noted`) e não bloqueia. Corpo dos achados passa pelo `Redactor`.
4. **Correção roda checks:** `CompleteFix` só aceita a correção se a nova revisão do código for **diferente** e os **checks dela estiverem aprovados** (gate de T14-01: evidência avaliada, sem regressão nem origem incerta); marca os bloqueantes como `fixed` (com a revisão da correção) e abre a **rodada seguinte** sobre a nova revisão, com o mesmo revisor (revalidação).
5. **Esgotamento suspende com checkpoint:** se a rodada do limite ainda tem achados bloqueantes, a revisão fica `exhausted`. Com a fila ligada, o checkpoint seguro é produzido (`Checkpointer`), o item é suspenso (`Queue.SuspendOne`: vaga devolvida, dono antigo sem direito de gravar) e abre-se uma **decisão humana** com os achados. Sem checkpoint seguro a suspensão **não acontece às cegas**: o erro é devolvido e o item segue em execução. Nada retoma sozinho.
6. **Aprovação por revisão do código:** `Gate` só vale para a revisão exata (outra revisão exige nova revisão); `required` vem do nível de risco (médio e alto exigem); "em andamento" e "esgotada" bloqueiam.

## Limites
- O revisor em si (o agente que lê o diff e produz os achados) não está aqui: o pacote conduz o processo e as regras; quem chama entrega os achados. A escolha de disponibilidade usa um predicado injetado (não consulta T09-04 sozinho).
- O `Checkpointer` real (parar o processo, esperar a quietude, `CaptureAndRelease`) e a ligação ao worker e à tela não existem; foram provados com um checkpointer de teste.
- A ordem dos achados dentro da rodada não é garantida; sem rotas nem tela de revisão.
