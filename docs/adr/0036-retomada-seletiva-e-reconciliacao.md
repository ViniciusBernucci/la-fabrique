# ADR-0036 — Retomada seletiva e reconciliação de insumos

- **Estado:** ACEITA · **Ticket:** [T12-04](../NEW%20MVP/03-BACKLOG.md#t12-04) · **Épico:** E12 · **Relacionadas:** [ADR-0023](0023-checkpoints-integros.md), [ADR-0027](0027-publicacao-e-congelamento-de-workflows.md), [ADR-0017](0017-pedidos-e-decisoes-humanas.md), [ADR-0033](0033-sincronizacao-manual-e-preflight.md)

## Decisão
1. **Seleção pelo grafo congelado:** o nó precisa existir no `FlatGraph` da versão publicada à qual a execução está presa, ser um nó de trabalho (agente, skill, script, checks, git, docs) e já ter sido percorrido. O conjunto afetado são os **descendentes** do nó nesse grafo; o restante dos nós percorridos é "reaproveitável".
2. **Validação antes de refazer (nesta ordem):** último **checkpoint íntegro** da execução (hashes reconferidos; estado dos nós lido dele); **versão publicada ainda executável** (`releases.Revalidate`: agentes, contas, modelos); **efeitos externos** dos nós afetados (convenção de chave `exec:<execução>:<nó>:`): efeito **incerto/em envio** bloqueia e abre decisão `uncertain_result`; efeito **confirmado** é listado como reaproveitado (o ledger não o repete); efeito de nó fora do conjunto não bloqueia.
3. **Insumos × baseline:** `resume_baselines` guarda refs, requisito (título, descrição, estado, prioridade, rótulos, dependências) e hash da configuração no último ponto reconciliado. `Reconcile` classifica cada diferença: **compatível** — prioridade, rótulos, estado e ref que avançou comprovadamente (fast-forward, via função injetada); **conflito** — título/descrição/dependências do requisito, ref divergente ou sem prova (falha fechada) e configuração congelada diferente.
4. **Compatível é incorporado** (baseline atualizado, trilha `incorporated`). **Conflito não altera nada:** abre uma decisão `conflict` (evidência por diferença; opções manter o trabalho ou replanejar), a execução vai a `waiting`, e estado, checkpoint, baseline e edições humanas ficam como estavam. Repetir não duplica a decisão.
5. **Reexecução só do nó:** `Rerun` executa **apenas** o nó escolhido, sem tocar o estado de entrada. **Saída igual:** os dependentes continuam válidos. **Saída diferente:** só os descendentes são invalidados (visitado, saídas, chegadas, laços, esperas, pendências, fins) e a fila retoma logo após o nó, então o motor refaz **somente** o invalidado; os demais nós ficam idênticos. Falha na reexecução mantém o estado anterior.
6. **Workflow original fixado:** a retomada usa sempre `ExecutionVersion`; publicar outra versão depois não altera a execução.
7. **Auditoria:** `resume_events` (`resumed`, `blocked`, `incorporated`) com seleção, resultado da reexecução, efeitos reaproveitados, versão e checkpoint.

## Limites
- Sem rotas/tela de retomada e sem ligação ao worker: nada chama `Resume` automaticamente nem persiste o novo `RunState` (o chamador grava um novo checkpoint, T12-01). O baseline é gravado pelo chamador (`RecordBaseline`).
- A convenção de chave de efeito `exec:<execução>:<nó>:` ainda não é seguida pelos executores reais (E11/E13); hoje é a regra que eles devem adotar.
- A prova de fast-forward depende de função injetada (Git real NOT_RUN); sem ela toda mudança de ref é conflito.
- A decisão `replan` só é aberta; replanejar a partir dela é T06-05/T10-x.
