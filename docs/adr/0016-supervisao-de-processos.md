# ADR-0016 — Supervisão de processos, timeout e inatividade

- **Estado:** ACEITA · **Ticket:** [T11-03](../NEW%20MVP/03-BACKLOG.md#t11-03) · **Épico:** E11 · **Decisões:** D02, D30–31, D35, D54, D70, D77–78, D84

## Decisão
1. **Dois prazos independentes** (`SupervisePolicy`): `MaxDuration` (absoluto) e `IdleTimeout` (inatividade), mais o cancelamento do contexto. Cada um produz um resultado distinto (`timeout`, `idle`, `cancelled`) além de `completed`.
2. **Atividade = qualquer sinal**: bytes novos na saída, **aumento de CPU ou de I/O de toda a árvore** de processos, ou mudança no número de processos. Assim um comando legítimo que fica **em silêncio mas trabalhando** (compilação, teste longo) continua, e um que não dá sinal algum pelo prazo é encerrado. A árvore é amostrada dentro do container (a partir do PID do comando, via `ps` e `/proc`).
3. **Encerramento com escalonamento e confirmação**: SIGTERM em toda a árvore, SIGKILL após a espera, e **confirmação por amostra** de que não resta processo. Só então o resultado fica confirmado.
4. **Estado incerto bloqueia o workspace**: se a parada não puder ser confirmada (falha ao cancelar, amostra indisponível ou processo ainda vivo), o resultado é `uncertain` e o sandbox entra em **quarentena**: `Exec` e `Start` passam a falhar com `ErrQuarantined`; inspeção e destruição seguem permitidas. A quarentena só sai por `VerifyQuiescent` (nenhum processo além do init e da manutenção do sandbox).
5. **Relatório auditável** (`Report`): resultado, motivo, código de saída, horários e duração, inatividade na decisão, bytes de saída, maior CPU/I/O observados, parada confirmada e processos vivos.

## Limites
- Um comando que **espera sem consumir CPU, I/O nem produzir saída** (por exemplo, bloqueado numa chamada de rede) é indistinguível de um travado e será encerrado pelo `IdleTimeout`; prazos devem ser calibrados por perfil.
- A amostragem depende de `ps`, `awk` e `/proc` das imagens (Alpine/BusyBox hoje).
- Ainda não há ligação com o worker/scheduler nem checkpoint/liberação de recursos (E12).
