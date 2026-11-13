# ADR-0054 — Central e auditoria operacional

- **Estado:** ACEITA · **Ticket:** [T16-02](../NEW%20MVP/03-BACKLOG.md#t16-02) · **Épico:** E16 · **Relacionadas:** [ADR-0017](0017-pedidos-e-decisoes-humanas.md), [ADR-0026](0026-autorizacoes-com-escopo-uso-e-revogacao.md), [ADR-0052](0052-revisao-externa-e-pr-fechada.md)

## Decisão
1. **Aprovar, recusar e revogar pelo painel, com evento e estado no servidor:**
   - aprovar/escolher: `POST /decisions/{id}/resolve` (já existente);
   - **recusar:** `POST /decisions/{id}/refuse` resolve o pedido com a alternativa de recusa que ele declara (`refuse`, `deny` ou `reject`); sem ela, 422 `no_refusal_option` e nada muda. É a mesma resolução (mesmo evento `decision_resolved`, mesma retomada, mesma idempotência e auditoria);
   - **revogar:** `POST /authorizations/{id}/revoke`, ação sensível (reautenticação), imediata; repetir é idempotente (`already_revoked`); com uso em andamento dispara a parada segura e, se ela falhar, a resposta traz `halt_failed` e a revogação continua valendo.
   Cada ação grava o evento de domínio e a auditoria do administrador (`auth_events`). Nada sai para fora: nenhuma notificação externa é criada.
2. **Autorizações visíveis (`GET /authorizations`):** situação calculada pelo servidor (`active`, `expired`, `exhausted`, `revoked`), usos, o que resta, uso em andamento, validade (`expires_in_seconds`, negativo = vencida), decisão de origem e motivo da revogação.
3. **Sequência do incidente e da retomada (`GET /executions/{id}/sequence`):** linhas em ordem de tempo reunidas de eventos da execução, pedidos e resoluções, eventos de autorização (concessão, uso, revogação, parada), recuperação e retomada. Dados já redigidos na origem; até 500 linhas; execução de outro projeto não é misturada (a busca usa o projeto da própria execução).
4. **Painel:** a Central de Decisões ganha o botão Recusar (quando o pedido declara a recusa), a aba Autorizações (validade, usos, revogação com motivo) e a sequência sob demanda por pedido e por autorização.

## Limites
- A sequência une registros de tabelas distintas por horário; eventos no mesmo instante seguem uma ordem fixa por origem, não uma ordem causal provada.
- A parada segura na revogação com uso em andamento depende do `Halter` do executor, que a API não tem ligado (sem ele, `halt_failed` é verdadeiro e a revogação vale).
- A recusa depende de o criador do pedido declarar a alternativa; pedidos sem ela só podem ser resolvidos pelas alternativas listadas.
- Navegador real NOT_RUN (testes de componente com `fetch` simulado). Sem filtro por projeto nem paginação além do limite.
