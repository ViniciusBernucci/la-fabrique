# ADR-0026 — Autorizações com escopo, validade, usos atômicos e revogação com parada segura

- **Estado:** ACEITA · **Ticket:** [T13-03](../NEW%20MVP/03-BACKLOG.md#t13-03) · **Épico:** E13 · **Relacionadas:** [ADR-0012](0012-politica-de-risco-no-executor.md), [ADR-0017](0017-pedidos-e-decisoes-humanas.md), [ADR-0018](0018-ledger-de-efeitos-externos.md), [ADR-0023](0023-checkpoints-integros.md)

## Decisão
1. **O que uma aprovação cobre:** uma autorização vale para **uma ação canônica** — ferramenta, operação, recurso, destino (minúsculo) e o **hash dos parâmetros** (`policy.Canonical`, independente da ordem das chaves) — a mesma estrutura que a política avalia. Tem prazo (até 24 h) e número de usos (1 a 100), pertence a uma execução (FK composta com o projeto) e, opcionalmente, a uma **decisão humana já resolvida da mesma execução** (`ErrDecision` caso contrário).
2. **Consumo atômico:** `Consume` é um único `UPDATE … WHERE state='active' AND uses < max_uses AND expires_at > agora AND tool/operation/resource/target/params_hash = …` com `RETURNING`: o bloqueio da linha serializa os concorrentes e **a corrida pelo último uso autoriza exatamente um** (provado com 30 e 50 goroutines, repetido 6× sob `-race`).
3. **Recusa sem gastar uso:** divergência de argumento, destino, recurso, operação ou ferramenta (`MismatchError{Field}`), prazo vencido, usos esgotados e revogação recusam a operação, **não consomem uso** e ficam na trilha (`authorization_events`). Precedência do diagnóstico: revogada → expirada → divergência → esgotada.
4. **Uso em andamento:** cada consumo abre um registro (`authorization_uses`, `started`) até `Finish` (ou `Do`, que consome, roda a operação e encerra). Uso que nunca terminou continua "em andamento".
5. **Revogação:** imediata (o commit já barra novos consumos; idempotente). Se havia uso em andamento, chama o **`Halter`** (parada segura: cancelar, confirmar quiescência e gravar checkpoint — T11-03/T12-01) fora da transação da revogação. Falha do `Halter`, ou ausência dele, devolve `ErrHaltFailed` **com a revogação já em vigor** e a falha na trilha (`halt_failed`). Sem uso em andamento, só revoga.
6. **Trilha:** `created`, `consumed`, `refused` (com o motivo), `revoked`, `halted`, `halt_failed`.

## Limites
- O pacote não está ligado à API nem ao worker: falta o ponto em que o executor pede a autorização antes de uma ação `approve` e o `Halter` concreto do runtime (T10/T11 integrados). Não há rotas nem tela de revogação (uma rota de revogação sem `Halter` real prometeria uma parada que não existe).
- O uso é contado ao **iniciar** (conservador): uma operação que falha depois do consumo gastou o uso.
- O `Halter` de verificação nos testes usa o sandbox Docker e o `checkpoints.Store` reais; o do runtime ainda será escrito.
