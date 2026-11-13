# ADR-0018 — Ledger de efeitos externos

- **Estado:** ACEITA · **Ticket:** [T13-04](../NEW%20MVP/03-BACKLOG.md#t13-04) · **Épico:** E13 · **Decisões:** D07, D26, D34–35, D80–87, D93

## Decisão
1. **Intenção antes do envio.** `Ledger.Run` grava a intenção no `outbox` (chave de negócio única por projeto, pedido já redigido com o `Redactor` do cofre) e só depois reivindica o envio (`intent`/`failed` → `sending`, por versão: só um processo vence). Chave de negócio estável entre tentativas, ex.: `projeto:feature:tentativa:operação:alvo`.
2. **Já confirmado, não reenvia** (`already_confirmed`). **Inconclusivo, bloqueado** (`ErrBlocked`) até decisão humana. **Envio em andamento dentro da concessão** (2 min por padrão) → `ErrInFlight`; **concessão vencida** (queda do processo) → reconcilia antes de qualquer repetição.
3. **Resposta perdida não presume nada.** Depois de um erro ambíguo (timeout, queda), o adaptador `Verify` procura **prova verificável** no destino, mesmo sem chave de idempotência (ex.: marcador com a chave de negócio, busca por referência, SHA remoto), e devolve:
   - **confirmado** → fecha como confirmado com a prova; **não repete**;
   - **ausente** → repetir é seguro: registra a ausência (`failed`), reivindica nova tentativa e envia de novo, até `MaxAttempts` (3); esgotadas, termina `failed`, ainda repetível numa nova execução;
   - **inconclusivo** (ou falha ao verificar) → `uncertain`, **bloqueia a repetição automática** e abre o bloqueio com evidência.
4. **Recusa definitiva** (`ErrNotApplied`, ex.: 403) falha sem verificar e sem repetir.
5. **Bloqueio com evidência = pedido na Central** (`DecisionEscalator`, T13-02): tipo "resultado incerto", com operação, chave, tentativas e o resultado da verificação (redigido). A decisão humana volta ao ledger por `ResolveFromDecision`, chamada ao resolver o pedido: **"o efeito existe"** fecha como confirmado (prova manual, sem reenviar); **"não existe"** libera a repetição. É idempotente.
6. As gravações do ledger usam um contexto desacoplado do cancelamento do chamador: cancelar a execução não perde o estado nem a prova.

## Limites
- Os adaptadores concretos de prova (subtarefa no Jira/ClickUp, push/ref e PR no GitHub, comentário) são de T06-03 e E15; aqui há o framework e um destino de teste. Em APIs sem busca confiável a prova será `inconclusive` por desenho.
- O ledger não decide o estado da execução: quem chama coloca a execução em `waiting` antes de escalar (a Central exige isso).
- Compensação de efeitos (desfazer) não está implementada: reversíveis de baixo risco, sensíveis e irreversíveis seguem como política futura.
