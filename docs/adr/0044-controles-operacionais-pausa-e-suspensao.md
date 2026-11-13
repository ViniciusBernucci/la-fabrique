# ADR-0044 — Controles operacionais: pausa após a feature atual e suspensão imediata

- **Estado:** ACEITA · **Ticket:** [T10-04](../NEW%20MVP/03-BACKLOG.md#t10-04) · **Épico:** E10 · **Relacionadas:** [ADR-0039](0039-fila-local-por-projeto.md), [ADR-0042](0042-scheduler-limite-global-leases-e-justica.md), [ADR-0023](0023-checkpoints-integros.md)

## Decisão
1. **Três modos por escopo** (global ou um projeto), persistidos em `controls` e auditados em `control_events` (de/para, administrador, justificativa): `active`, `draining` (**pausar após a feature atual**) e `suspended` (**suspender agora**). Sobrevivem ao reboot (estado só no banco). A mudança de modo **não** altera o que já roda.
2. **Pausa após a feature atual (`draining`):** nenhuma feature **nova** inicia (feature nova = nenhuma tarefa sua saiu de `pending`); o que está rodando termina e as demais tarefas da feature em curso (e correções de PR dela) seguem normalmente.
3. **Suspensão imediata (`suspended`):** nada novo inicia e o que roda é parado (`SuspendNow`) **só com checkpoint seguro** — íntegro, da execução do item e sandbox quieto, entregue por um `Suspender` (em produção: cancelar, esperar quietude, `CaptureAndRelease`). Cada item suspenso vai a `waiting` com motivo `suspended`, **devolve a vaga**, perde a lease e tem a **geração incrementada** (o dono antigo não conclui nem grava). Item sem checkpoint seguro, ou cuja parada falhou, **segue em execução** e é listado em `Failed`.
4. **Efeitos diferentes:** a pausa deixa terminar; a suspensão para e guarda o estado. Ambas valem em `Claim`, `Start` e `Candidates` (um único ponto de elegibilidade).
5. **Escopos independentes:** controle de um projeto não afeta os demais, que continuam executando. **Retomar** (`ResumeScope`) volta o escopo a `active` e devolve à fila os itens suspensos que não continuam sob **outra** suspensão (retomar o global não reativa o que o projeto mantém suspenso, e vice-versa). Item suspenso só volta por `ResumeScope` (não há `Block`/`Resume` manual para `suspended`).

## Limites
- Sem rotas nem tela, e o `Suspender` real (cancelar processo + capturar checkpoint no sandbox) não está ligado ao worker: o contrato e o efeito sobre a fila estão provados com um suspensor de teste; o encadeamento com `execution`/`checkpoints` é do executor.
- "Feature em curso" é inferida pelas tarefas do plano (qualquer uma fora de `pending`); não há noção de término da feature além disso (T15-x).
- A recuperação de itens interrompidos por crash/reboot é o T12-03.
