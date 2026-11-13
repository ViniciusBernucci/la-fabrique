# ADR-0045 — Recuperação de tentativas interrompidas

- **Estado:** ACEITA · **Ticket:** [T12-03](../NEW%20MVP/03-BACKLOG.md#t12-03) · **Épico:** E12 · **Relacionadas:** [ADR-0042](0042-scheduler-limite-global-leases-e-justica.md), [ADR-0023](0023-checkpoints-integros.md), [ADR-0018](0018-ledger-de-efeitos-externos.md), [ADR-0036](0036-retomada-seletiva-e-reconciliacao.md)

## Decisão
1. **Tentativa interrompida** = item `running` com a lease vencida (worker caiu, VPS reiniciou). Com `Queue.RequireRecovery` (produção), `Claim` **não** devolve esse item à fila às cegas: ele segue ocupando a vaga até ser diagnosticado. `ExpireLeases` continua sendo o caminho sem diagnóstico (T10-03). Um worker vivo (lease válida) nunca é tocado (`ErrNotStale`).
2. **Diagnóstico item a item** (`Recoverer.Recover`), provando antes de agir: (a) **quietude** do sandbox (nenhum escritor vivo); (b) **último checkpoint íntegro** da execução (hashes reconferidos; corrompido é incerteza); (c) **efeitos externos** da execução (chaves `exec:<execução>:…`): incerto ou em envio é incerteza; efeitos confirmados sem checkpoint para retomar também; (d) **estado verificável** — Git, artefatos e políticas — por um `Inspector`.
3. **Seguro → automático:** com tudo provado e dentro do limite, o item volta à fila sozinho, com a vaga e a lease liberadas, a tarefa em `pending` e o **checkpoint a usar** gravado no item; sem checkpoint e sem efeitos, recomeça do início. A geração sobe na próxima reclamação, então o dono antigo, se voltar, não grava.
4. **Incerteza → decisão, só do afetado:** abre uma decisão `uncertain_result` com o diagnóstico, põe a execução em espera e o item em `waiting` (motivo `decision`). A **vaga do projeto só é devolvida com quietude provada E checkpoint íntegro** (um único escritor por workspace); sem isso o item segura a vaga. Os demais itens elegíveis (do mesmo projeto, se a vaga foi devolvida, ou de outros) seguem. Cada item é processado de forma independente (`RecoverAll`).
5. **Limites separados por classe** — infraestrutura (3), IA (2), revisão (2), configuráveis: cada classe tem o seu contador por item. Esgotado o limite da classe, a recuperação deixa de ser automática e abre decisão (`limit`); uma classe esgotada não consome nem bloqueia as outras.
6. **Inspeção indisponível** (ex.: runtime de containers fora do ar, verificação que falha) **adia** o item sem alterá-lo e sem consumir tentativa; ele é recuperado quando a inspeção volta.
7. **Trilha** em `recovery_events` (ação, classe, checkpoint, vaga liberada, razões) e contadores em `recovery_attempts`.

## Limites
- O `Inspector` real (consulta ao Docker, `git status`/HEAD, verificação de artefatos e políticas) não está implementado: o contrato e o efeito sobre a fila estão provados com um inspetor de teste. Também não há laço de varredura no `cmd/worker` que chame `RecoverAll` após o reboot, nem rotas/tela.
- A classe da falha é informada por quem chama (padrão: infraestrutura); a distinção real entre IA e revisão depende dos executores (T14-03/T09-x).
- Mudanças já aplicadas na origem por uma tentativa interrompida dependem da convenção de chave `exec:<execução>:` ainda não seguida pelos executores.
