> Leitura: [← Anterior](12-componentes-alvo.md) · [Índice didático](../02-INDEX.md) · [Próximo →](14-dados-alvo.md)

> Proposta posterior do pacote, não estado implementado. Conciliação: [AS-IS](01-as-is.md). Sem liberação de provider/runtime por este diagrama.

# Fluxos principais

Revisão documental: 2026-10-06 (America/Sao_Paulo). Arquitetura **PLANEJADA**; implementação e eficácia **NÃO VERIFICADAS** na origem do pacote.

## Ticket até aceite

```mermaid
sequenceDiagram
  participant U as Usuário
  participant A as API / Orchestrator
  participant W as Worker / Executor
  participant R as Provider Runtime
  participant B as Broker / Sandbox
  U->>A: Ticket READY autorizado
  A->>A: Run + outbox transacional
  W->>A: Claim autenticado e idempotente
  W->>W: Validar escopo, recursos, writer e perfil
  W->>R: Nova sessão e contexto sanitizado
  R->>B: Operação restrita da tentativa
  B-->>R: Resultado não confiável limitado
  R-->>W: Resultado/eventos
  W->>W: Confirmar término, coletar, checks
  W->>R: Sessão independente de reviewer
  W->>A: Revisão, documentação e evidências
  A->>A: Gate e AWAITING_HUMAN
  U->>A: Aceite da revisão exata
```

Até duas correções; alterações exigem novos checks pertinentes. Exit code ou mensagem finished não comprova aceite.

## Handoff seguro

```mermaid
sequenceDiagram
  participant E as Executor
  participant A as Provider A
  participant C as Checkpoint / artifacts
  participant B as Provider B
  E->>A: Bloquear tools e cancelar árvore
  A-->>E: Término observado externamente
  E->>E: Confirmar quiescência ou BLOCKED_RECOVERY
  E->>C: Base/code SHA + patch + untracked + hashes
  C-->>E: Persistência e recuperabilidade confirmadas
  E->>E: Novo attempt / sandbox / fencing
  E->>B: Mesmo tenant, grant revalidado e checkpoint
  B->>C: Conferir revisão/hashes sem reaplicar patch
```

Se término não é comprovado, não há segundo writer. Falta de quota não seleciona conta de outro cliente. [Handoff canônico](../07-operacao/04-handoff.md).

## Recuperação e revogação

Reboot reconcilia processes/cgroups/claims/artefatos órfãos antes de admission. Revogar membership/instalação bloqueia claims/capabilities e interrompe attempts afetadas. Restore valida ownership e hashes em ambiente privado. Detalhes em [execução e recuperação](../07-operacao/02-execucao-e-recuperacao.md).

Máquina de estados e transições pertencem ao [contrato de workflow](../05-contratos/schemas/01-workflow.md); ciclo de sandbox pertence ao [módulo sandbox](../03-modulos/sandbox/00-README.md).
