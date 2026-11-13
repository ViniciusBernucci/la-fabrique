# ADR-0053 — Memória técnica pendente e oficial

- **Estado:** ACEITA · **Ticket:** [T15-05](../NEW%20MVP/03-BACKLOG.md#t15-05) · **Épico:** E15 · **Relacionadas:** [ADR-0051](0051-observacao-do-merge-e-propagacao.md), [ADR-0030](0030-contexto-seletivo-e-renovacao-de-sessao.md), [ADR-0047](0047-revisao-independente-e-correcoes.md)

## Decisão
1. **Registro (`memory_records`):** chave semântica normalizada (minúsculas, espaços colapsados), tipo (`fact`, `decision`, `constraint`, `lesson`), conteúdo redigido, fonte, **evidências vinculadas** (1 a 20, rótulo + referência), revisão a que a evidência se refere, confiança, vínculo com projeto, feature, branch e PR, e estado `pending`, `official` ou `invalidated` (com motivo). Sem evidência, revisão ou branch o registro é recusado.
2. **Deduplicação:** uma chave pendente por branch. Repetir a chave soma as evidências e, se o conteúdo mudou, a versão mais recente vale (versão incrementada). Entre pendente e oficial, a promoção deduplica (abaixo).
3. **Escopo por projeto:** toda leitura e escrita filtra por `project_id`; ler, invalidar ou promover registro/entrega de outro projeto é "não encontrado". Nada de memória de A chega a B.
4. **Consumo (`Relevant`):** os oficiais do projeto e os pendentes **só da própria branch**, filtrados por termos; invalidados nunca.
5. **Promoção (`Promote`):** só para entrega `merged` com SHA (observada pelo T15-03). PR aberta, fechada, cancelada ou integrada sem SHA devolve `ErrNotMerged` e não muda nada. Para cada pendente da branch: vale se a revisão da evidência é a entregue, um commit da feature ou o SHA do merge, e o validador opcional (`Validate`) aceita; senão é invalidado com o motivo. Igual a um oficial → duplicata (evidências somadas ao oficial); diferente → o oficial anterior é invalidado ("substituída pelo merge") e o novo vira oficial com a revisão integrada. Repetir não muda nada. Pendentes de outras branches ficam como estão.
6. **Extração (`ExtractFromDelivery`):** achados de revisão críticos ou relevantes já corrigidos viram lições pendentes da branch, com a revisão e o achado como evidência (chave pelo título: repetir não duplica).

## Limites
- Sem rotas, tela nem ligação ao montador de contexto (T12-02) ou ao worker; `Relevant` é a porta de consulta. A promoção precisa ser chamada após `Observe` confirmar o merge.
- A extração cobre só achados de revisão; fatos de checks, decisões e E2E ainda dependem de quem chama `Record`.
- "Válido para a revisão integrada" é a checagem determinística por revisão (e o validador opcional); não há avaliação semântica do conteúdo nem leitura do código integrado.
- Nada disso foi exercitado contra o GitHub real.
