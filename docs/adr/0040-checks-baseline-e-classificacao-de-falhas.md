# ADR-0040 — Checks no baseline e classificação de falhas

- **Estado:** ACEITA · **Ticket:** [T14-01](../NEW%20MVP/03-BACKLOG.md#t14-01) · **Épico:** E14 · **Relacionadas:** [ADR-0008](0008-sandbox-por-feature-em-containers.md), [ADR-0034](0034-perfil-tecnico-e-recursos-do-projeto.md), [ADR-0022](0022-execucao-de-ferramentas-confinada-e-com-identidade.md)

## Decisão
1. **Execução gravada por revisão:** `RunSet` roda os checks (nome + comando de uma linha, sem segredo) em sequência, na revisão informada (`baseline` ou `candidate`), via um `Execer` injetado (`SandboxExecer` roda `sh -c` no sandbox, em `/workspace`). Cada execução guarda resultado (`pass`, `fail`, `error`, `timeout`, `not_viable`), código de saída, **assinatura**, trecho da saída, hash, duração e horários. Saída vai pelo `Redactor` antes de gravar e é truncada em 16 KiB (fica o **fim**). Falha de infraestrutura é `error`, nunca "falha do código".
2. **Assinatura da falha:** hash do código de saída, das linhas de erro normalizadas (até 20) e das 3 últimas linhas; cores, tempos, números, endereços e caminhos temporários não entram. Falhas iguais com ruído diferente têm a mesma assinatura; falhas diferentes não.
3. **Classificação da candidata contra o baseline** (`Evaluate`, por check):
   - passou → `pass`;
   - baseline passava → **`regression`**; baseline falha com **outra** assinatura → `regression` (falha nova);
   - baseline falha com a **mesma** assinatura → **`preexisting`** (falha anterior comprovada), registrada em `known_failures` com a execução de baseline que a comprova;
   - sem baseline utilizável (não rodou, `not_viable`, erro, tempo limite) → **`unknown`**: em investigação, **não** vira baseline por suposição. Exceção: falha com assinatura já comprovada antes (`known_failures`) é `preexisting`;
   - check que não terminou (erro/tempo) → `inconclusive`.
4. **Bloqueia** `regression`, `unknown` e `inconclusive`; `preexisting` fica documentada e não bloqueia. `Gate` devolve o veredito persistido sem reexecutar.
5. **Evidência por revisão:** o veredito só vale para a revisão exata; outra revisão (uma alteração depois dos checks) não herda nada e fica bloqueada com "sem evidência". Antes de avaliar, o gate também bloqueia ("ainda não foi avaliado").
6. **Investigação com prova:** `Resolve` só atua em `unknown`/`inconclusive`, exige justificativa e: `preexisting` **exige a execução de baseline** do mesmo check, com falha e a mesma assinatura (senão `ErrNoProof`); `regression` confirma o defeito (continua bloqueando). A resolução humana prevalece em reavaliações.
7. **Baseline inviável** é registrado com o motivo (`RecordNotViable`), e o motivo aparece na investigação.

## Limites
- Não há preparação do baseline (checkout da revisão anterior no sandbox), nem leitura dos comandos do perfil do projeto (T05-03) para montar os checks: quem chama passa os `Spec` e a revisão. Também não há baseline automático nem rotas/tela.
- A heurística de assinatura é textual: uma falha com mesma saída por causas diferentes seria tomada como a mesma. A prova humana (`Resolve`) cobre os casos duvidosos.
- Sem E2E de navegador, risco, revisão de código e níveis de severidade (T14-02 a T14-04).
