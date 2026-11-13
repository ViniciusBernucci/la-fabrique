# ADR-0056 — Tarefas periódicas do worker

- **Estado:** ACEITA · **Tickets:** T10-02, T12-03, T13-02, T16-04 (ligação ao worker) · **Épicos:** E10, E12, E13, E16 · **Relacionadas:** [ADR-0042](0042-scheduler-limite-global-leases-e-justica.md), [ADR-0045](0045-recuperacao-de-tentativas-interrompidas.md), [ADR-0050](0050-retencao-com-integridade.md)

## Decisão
1. **O worker passa a ter tarefas periódicas** (`worker.Job`, registradas com `AddJob`; cada uma roda no seu intervalo, com situação visível em `GET /jobs` do endpoint de saúde e nos logs). Falha de uma tarefa não derruba o worker nem as outras.
2. **Tarefas ligadas** (as que não dependem do executor de agentes):
   - `liberar-decisoes` (a cada `WORKER_JOBS_INTERVAL`, 30 s): devolve à fila os itens em espera por decisão cuja decisão já foi resolvida; a porta `DecisionGate` decide, então item com decisão pendente segue esperando;
   - `recuperar-tentativas` (idem): `Recoverer.RecoverAll` sobre itens com lease vencida; volta à fila quando é seguro e abre decisão quando há dúvida (fila com `RequireRecovery` ligado);
   - `retencao-simulada` ou `retencao` (a cada `RETENTION_INTERVAL`, 24 h): por padrão **só simula e registra**; a remoção real exige `RETENTION_APPLY=true`.
3. **Inspetor real (`recovery.DockerInspector`):** encontra os containers vivos da feature pelos rótulos `lf.*` e confere a quietude (nenhum processo do projeto) e, havendo checkpoint, a divergência do workspace. Sem container, a feature está quieta (a próxima execução restaura do checkpoint). **Falha fechada:** se o runtime de containers não responde, a inspeção dá erro e a recuperação **adia** sem consumir tentativa nem alterar nada.
4. **Fora do worker, de propósito:** a consulta do merge e dos comentários da PR segue sob demanda (D89, sem polling): é chamada pela API.

## Limites
- O worker ainda **não reclama nem executa tarefas com agentes**: isso depende do executor de sandbox e IA, que não existe. Sem ele não há item "em execução" gerado pelo worker; a recuperação age sobre o que outro processo deixou com lease vencida.
- A imagem do worker não tem o CLI do Docker nem o socket: no Compose atual a inspeção real não responde e a recuperação adia (comportamento seguro). Habilitá-la exige decidir como o worker acessa o runtime (isolamento do socket), o que fica para o responsável.
- Sem escolha de classe de falha: toda interrupção é tratada como infraestrutura.
