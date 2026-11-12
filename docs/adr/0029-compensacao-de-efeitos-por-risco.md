# ADR-0029 — Compensação de efeitos parciais por risco

- **Estado:** ACEITA · **Ticket:** [T13-05](../NEW%20MVP/03-BACKLOG.md#t13-05) · **Épico:** E13 · **Relacionadas:** [ADR-0018](0018-ledger-de-efeitos-externos.md), [ADR-0026](0026-autorizacoes-com-escopo-uso-e-revogacao.md)

## Decisão
1. **Três classes de risco**, por operação do efeito original (`DefaultSpecs`; configurável): **baixo** (reversível: `create_subtask`, `add_link`, `add_dependency`), **sensível** (`push`, `create_pr`) e **irreversível** (`merge`, `send_message`, `comment` e **qualquer operação desconhecida** — padrão seguro).
2. **Só se compensa efeito confirmado** no ledger (`ErrNotConfirmed` caso contrário).
3. **Baixo risco:** a compensação é um novo efeito no **ledger** (chave `<chave original>:compensate`), logo idempotente, com intenção antes do envio e **prova** (Verify: o efeito original deixou de existir). Resposta perdida é reconciliada pela prova, sem desfazer duas vezes; repetir um efeito já compensado não faz nada.
4. **Sensível:** exige uma **autorização** (T13-03) presa exatamente a essa compensação (`ActionFor`: ferramenta `compensation`, operação inversa, recurso = chave do efeito, parâmetro = id do efeito); o consumo é atômico. Sem autorização, autorização de outra compensação, vencida, revogada ou esgotada ⇒ **recusa registrada**, nada é desfeito, nenhum uso é gasto nas divergências.
5. **Irreversível:** nada é desfeito; o efeito é **preservado** e a recusa fica registrada (`ErrIrreversible`), repetível sem efeito.
6. **Falha conserva o trabalho e bloqueia o que agrava:** compensação que falha (destino recusa, sem prova) fica `failed`, **nada é apagado**, e abre um bloqueio na execução. `CheckAggravating` recusa operações que alteram estado enquanto houver bloqueio aberto; leitura (`read*`, `list*`, `get*`, `search`, `status`, `diff`) e compensação continuam permitidas. O bloqueio some quando a compensação é repetida com sucesso ou por `ResolveBlock` (decisão humana, com observação). Outras execuções não são afetadas.
7. **Auditoria:** `compensations` (estado, risco, motivo, tentativas, autorização) e `compensation_events` (`refused`, `applied`, `failed`, `blocked`, `block_resolved`).

## Limites
- Os adaptadores concretos de desfazer (Jira, ClickUp, GitHub) não existem: o pacote recebe o adaptador por função; o que foi provado usa um destino de mentira.
- `CheckAggravating` ainda não é chamado pelo executor/worker; sem API nem tela para resolver bloqueios.
- A classificação é por nome de operação; uma operação nova precisa ser declarada para ser compensável.
