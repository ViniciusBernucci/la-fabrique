# ADR-0028 — Seleção determinística de fluxo e subfluxo por regras

- **Estado:** ACEITA · **Ticket:** [T07-04](../NEW%20MVP/03-BACKLOG.md#t07-04) · **Épico:** E07 · **Relacionadas:** [ADR-0027](0027-publicacao-e-congelamento-de-workflows.md)

## Decisão
1. **Regras ordenadas por projeto** (`selection_rules`, posição única): cada regra tem de 1 a 10 condições (todas precisam casar), um fluxo `feature` e, opcionalmente, um subfluxo `subflow`. Há um **padrão explícito** (`selection_defaults`). A configuração é substituída de forma **atômica** (uma configuração inválida não altera a anterior).
2. **Atributos normalizados** antes de comparar: `platform`, `type`, `priority`, `title` (texto em minúsculas, espaços colapsados) e `labels`, `components` (listas sem repetição e ordenadas). Operadores: `eq`, `in`, `prefix`, `contains` (texto) e `has_any`, `has_all` (listas); operador incompatível com o tipo do campo é recusado.
3. **Escolha determinística:** a primeira regra ativa que casa vence; as outras que também casariam aparecem no motivo (`also_matched`). Sem regra, vale o padrão; sem padrão, a seleção falha de forma explícita (`ErrNoSelection`, 409 `no_selection`). Reordenar muda o vencedor: a ordem é o critério.
4. **Rastro persistido:** toda seleção grava `selection_traces` com a entrada normalizada, a regra, o uso do padrão e o **motivo** (inclusive nas falhas). Destino não publicado: o rastro explica e a seleção recusa (`not_published`).
5. **Simulação** explica regra por regra (casou ou por que não casou, regra desativada) e **não grava** rastro.
6. API: `GET/PUT /api/projects/{id}/selection`, `POST …/selection/simulate`, `GET …/selection/traces`.

## Limites
- Os atributos da feature vêm de quem chama (T05/T06 alimentam a partir de Jira/ClickUp); a leitura real da origem é NOT_RUN.
- Sem tela; sem expressões livres nem regex (por desenho: valores explícitos e auditáveis).
