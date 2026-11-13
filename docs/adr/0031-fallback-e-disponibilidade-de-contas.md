# ADR-0031 — Fallback e disponibilidade de contas

- **Estado:** ACEITA · **Ticket:** [T09-04](../NEW%20MVP/03-BACKLOG.md#t09-04) · **Épico:** E09 · **Relacionadas:** [ADR-0025](0025-orcamento-reserva-atomica-e-uso.md), [ADR-0023](0023-checkpoints-integros.md), [ADR-0030](0030-contexto-seletivo-e-renovacao-de-sessao.md)

## Decisão
1. **Quatro causas distintas:** `auth`, `quota`, `unavailable`, `billing`. Regras de retorno próprias: `auth` e `billing` não somem com o tempo (só `Clear` após revalidação ou um sucesso); `quota` volta no prazo informado (ou 1 h) ou quando uma observação oficial posterior mostra folga; `unavailable` volta pelo backoff.
2. **Sinais oficiais + histórico:** o erro marcado como oficial pelo provedor e a **cota observada** pelo adaptador (T09-03) valem como sinal oficial. Indisponibilidade só vira estado depois de **2 falhas seguidas** (uma isolada é ruído) e cresce com backoff `30 s · 2^(n−1)`, teto de 15 min. Toda falha, sucesso e limpeza ficam em `provider_events`.
3. **Cadeia explícita, sem rodízio:** `Resolver.Choose` percorre só as contas da cadeia do agente, em ordem, e escolhe a primeira **saudável, compatível** (`models.Check`: provedor, conta, modelo, ferramentas) e **dentro do orçamento** (reserva de teste, liberada em seguida; sem orçamento global a alternativa é recusada). Conta saudável fora da cadeia nunca é usada. Com todos saudáveis a escolha é sempre o principal; quando ele volta, novas escolhas o usam, mas nada move uma tarefa em andamento.
4. **Preservar antes de trocar:** trocar de conta exige **checkpoint íntegro da própria execução** (hashes reconferidos) e sandbox em **quiescência** (prova do chamador); sem isso, `ErrNotSafe` e nenhuma troca é registrada.
5. **Sem alternativa:** a tarefa **espera** (`waiting_tasks`) com horário de nova tentativa (o menor prazo das contas descartadas, ou backoff por tentativa; sem horário quando tudo depende de intervenção). Com checkpoint seguro a **vaga é liberada** (`wait`); sem ele a tarefa **mantém a vaga** (`hold`). Outras execuções seguem; `Due` lista as vencidas.
6. **Auditoria:** `fallback_events` registra troca/espera/manutenção com origem, destino, motivo e checkpoint; `Decision.Skipped` explica cada passo descartado.

## Limites
- Não há worker, executor, rotas nem tela: nada classifica o erro real de um provedor nem chama `Choose`; `Failure` é fornecida pelo chamador.
- Provedores reais (Codex, Claude, OpenRouter, Vercel) NOT_RUN: a classificação 401/402/429/5xx de cada um ainda precisa ser mapeada nos adaptadores.
- A quiescência é declarada por quem chama; o pacote não consulta o sandbox.
