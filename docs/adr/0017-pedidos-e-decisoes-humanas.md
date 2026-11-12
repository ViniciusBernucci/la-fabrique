# ADR-0017 — Pedidos e decisões humanas

- **Estado:** ACEITA · **Ticket:** [T13-02](../NEW%20MVP/03-BACKLOG.md#t13-02) · **Épico:** E13 · **Decisões:** D07, D26, D34–35, D80–87, D93

## Decisão
1. **Pedido = problema + alternativas + impacto + evidência**, preso a **uma execução** (chave composta com o projeto) e a um nó, com tipo (`conflict`, `missing_requirement`, `sensitive_operation`, `uncertain_result`, `pr_closed`, `other`), de 2 a 6 alternativas e estado (`pending`, `resolved`, `stale`). O mesmo problema enquanto pendente (execução + nó + tipo) **não vira dois pedidos**: devolve o existente. A execução precisa estar `waiting`.
2. **Evidência redigida na criação** com o `Redactor` do cofre (valores exatos e padrões de token): nada de segredo no banco nem na tela.
3. **A resolução é uma transação restrita à execução do pedido**: bloqueia o pedido e a execução, grava a decisão, emite os eventos `decision_resolved` e `resume_requested` e move **só essa execução** de `waiting` para `queued` (sinal de retomada que o escalonador consumirá). Outras execuções, pedidos e eventos não são tocados.
4. **Decisão repetida, definida:** a **mesma opção** é idempotente (`X-Decision-Replayed: true`, sem novo evento, sem nova auditoria); **outra opção** depois de resolvido → 409 `decision_conflict` e a decisão registrada permanece. Concorrência: vencem as chamadas da mesma opção (uma só efetiva); entre opções diferentes só uma vence e as outras recebem conflito.
5. **Versão obsoleta, definida:** o pedido guarda a versão da execução no momento da criação. Se a execução mudou (outra versão ou deixou de estar `waiting`), a resolução é recusada com 409 `decision_stale` e o pedido passa a `stale`; novas tentativas repetem o mesmo resultado e nada é sinalizado.
6. **Operação sensível exige reautenticação recente** (403 `reauth_required`): aprovar uma operação é ação sensível e distinta do parecer de revisão de código. Cada resolução efetiva entra na trilha de auditoria.
7. **Central no painel:** abas Pendentes/Resolvidas/Obsoletas; cada pedido mostra problema, impacto, evidência e alternativas; resolvidas mostram a opção e a observação; obsoletas explicam que a decisão não foi aplicada.

## Limites
- **Nenhum fluxo do produto cria pedidos ainda**: a API é só de leitura e resolução; a criação (`decisions.Create`) é chamada pelo motor/worker nos épicos E10 e E12. Sem escalonador, o estado `queued` ainda não é consumido.
- Não há expiração de pedidos nem notificação; a lista é a Central.
- Autorizações com escopo, validade e uso único são do T13-03.
