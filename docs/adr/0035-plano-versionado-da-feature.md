# ADR-0035 — Plano versionado da feature

- **Estado:** ACEITA · **Ticket:** [T10-01](../NEW%20MVP/03-BACKLOG.md#t10-01) · **Épico:** E10 · **Relacionadas:** [ADR-0018](0018-ledger-de-efeitos-externos.md), [ADR-0019](0019-escrita-na-origem-sem-duplicacao.md), [ADR-0033](0033-sincronizacao-manual-e-preflight.md)

## Decisão
1. **Contrato do plano:** `Plan{summary, tasks[]}`; cada tarefa tem chave estável (`^[a-z0-9][a-z0-9_-]{0,39}$`), título, descrição, **critérios verificáveis**, risco inicial (`low|medium|high`), `depends_on` (DAG), estimativa de `order` (desempate) e caminhos. Limites: 50 tarefas, 12 critérios por tarefa.
2. **Critério verificável** tem um `check`: `command` (comando de uma linha), `test` (alvo), `file` (caminho relativo, sem `..` nem absoluto) ou `review` (evidência observável de ≥ 15 caracteres). Sem `check` ou com campo obrigatório vazio, o critério é recusado; tarefa sem critério também.
3. **Validador puro** recusa chave inválida/repetida, plano vazio ou grande, risco ausente, dependência ausente, repetida ou para si mesma e **ciclo** (o ciclo é mostrado, `a → b → c → a`). `Order` devolve a ordem topológica estável (menor `order`, depois a chave).
4. **Versões:** plano aceito vira a versão vigente (a anterior fica `superseded`); o **mesmo conteúdo** (hash do plano normalizado, independente da ordem de escrita) não cria versão. Plano **inválido ou conflitante** é gravado como `rejected`, com os problemas, e **a versão vigente não muda**. Propostas concorrentes serializam por bloqueio da feature.
5. **Identidade estável:** a tarefa é (feature, chave) e atravessa as versões, preservando estado (`pending|running|done|blocked`) e ID remoto. **Conflito** é alterar ou remover uma tarefa que já saiu de `pending`. Tarefa pendente removida do plano fica `removed` (a identidade não se perde).
6. **Origem sem duplicar:** `Store.Publish` cria **uma subtarefa por tarefa** e **uma relação por dependência** pelo ledger (T13-04), com chave `plan:<feature>@<identidade remota>:<tarefa>` (a identidade remota entra para que uma troca de origem crie os itens na nova origem), em ordem topológica. Resposta perdida é reconciliada pela prova (marcador), repetir não duplica e replanejar só cria o que é novo; subtarefas existentes mantêm o ID remoto. A descrição leva risco e critérios. Tarefa removida do plano não é apagada na origem.
7. **API:** `PUT/GET /api/projects/{id}/features/{featureId}/plan` (proposta com 422 e lista de problemas; leitura com vigente, tarefas e versões).

## Limites
- Não há o agente planejador nem a leitura do código/contexto (é o produtor do plano, via executor, E11); aqui está o contrato e o que o plano precisa cumprir.
- `Publish` é função Go (sem rota: o servidor ainda não monta o ledger/adaptador por projeto); não liga ao scheduler (T10-02) nem à tela. Jira/ClickUp reais NOT_RUN.
- A verificação dos critérios (rodar o comando/teste) pertence à execução e ao E14; aqui só se garante que são verificáveis.
