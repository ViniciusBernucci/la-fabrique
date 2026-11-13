# ADR-0030 — Contexto seletivo e renovação de sessão

- **Estado:** ACEITA · **Ticket:** [T12-02](../NEW%20MVP/03-BACKLOG.md#t12-02) · **Épico:** E12 · **Relacionadas:** [ADR-0023](0023-checkpoints-integros.md), [ADR-0025](0025-orcamento-reserva-atomica-e-uso.md)

## Decisão
1. **Contexto por contrato:** fontes tipadas (`requirement`, `criterion`, `decision`, `memory`, `file`, `evidence`, `output`), cada uma com referência estável, prioridade e marca de obrigatória. A montagem (`contextpack.Build`) é **determinística**: o texto sai em ordem de tipo e referência, independente da ordem de entrada.
2. **Manifesto de fontes:** para cada fonte, hash do conteúdo **efetivamente incluído**, bytes, tokens, se foi redigida, truncada ou omitida e o motivo (`limit`, `empty`, `duplicate`). O manifesto tem hash próprio e guarda o limite e o método de contagem (`estimate:bytes/3`).
3. **Segredos saem antes do hash:** o `Redactor` roda sobre cada fonte; o pacote ainda é recusado (`ErrSecretLeft`) se o texto final contiver algo reconhecível, e `Verify` repete essa checagem ao gravar e ao retomar.
4. **Obrigatórias nunca somem em silêncio:** requisito/critério marcado como obrigatório que não cabe no limite faz a montagem **falhar** (`ErrRequiredOmitted`). Só fontes opcionais são omitidas ou truncadas (sem cortar no meio de um caractere UTF-8).
5. **Renovar antes de saturar:** `Decide` usa a métrica **oficial** quando existe; sem ela usa a estimativa local **com margem** (×1,25), o que renova mais cedo, nunca mais tarde. Limiar padrão 80 % da janela; sem janela conhecida, falha. O método e os números ficam gravados na sessão.
6. **Sessão ligada a prova:** `Renew` só grava depois de conferir o pacote (hash do texto, total de tokens, segredos) e o checkpoint (hashes do manifesto e do estado; pertencer à mesma execução e ao projeto). Pacote e sessão entram na **mesma transação**, com sequência serializada por bloqueio consultivo.
7. **Marcos:** `planning`, `implementation`, `tests`, `review`, `suspension`, `session_change`, `saturation`. O checkpoint é criado pelo chamador (`CaptureAndRelease`, T12-01) e **referenciado** pela sessão.
8. **Troca de modelo exige autorização explícita** (`ErrModelChange`); sem ela nada é gravado.
9. **Retomada verificável:** `Resume` devolve a sessão mais recente com o pacote e o checkpoint **reconferidos**; qualquer hash divergente aborta com `ErrCorrupt`. Se há checkpoint mais novo que o da sessão, `NewerCheckpoint` avisa (o último íntegro governa a restauração).

## Limites
- Sem rotas, tela nem ligação ao worker: nada decide sozinho quando renovar nem cria o checkpoint do marco.
- A **memória** (T15-05) e as decisões/requisitos reais ainda não alimentam o `Build`; o pacote recebe fontes já resolvidas pelo chamador.
- A contagem é estimativa (bytes/3); a métrica oficial só entra se o chamador a fornecer.
