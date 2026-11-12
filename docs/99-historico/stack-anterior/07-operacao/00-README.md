> Leitura: [Índice didático](../02-INDEX.md) · [Próximo →](01-deploy.md)

# Operar e recuperar

Código operativo existe, ativação/eficácia no serviço são gates separados. [Infraestrutura local integral](infraestrutura/01-referencia-v2.3.md), [dimensionamento](infraestrutura/00-dimensionamento-vps.md), [execução/recuperação](02-execucao-e-recuperacao.md), [handoff](04-handoff.md), [backup](05-backup.md), [restore](06-restore.md), [deploy](01-deploy.md), [troubleshooting](07-troubleshooting.md), [observabilidade](03-observabilidade.md), [economia](09-economia.md), [incidentes](08-incidentes.md).

Um executor/um writer global. Pedido de pausa, heartbeat stale, lease vencida e fila vazia não provam parada. Não aplicar migrations/ativar worker enquanto quiescência desconhecida. Auth oficial fora de sandbox/controle; só fluxo oficial humano, sem OAuth como API nem auth mount para contornar cliente incompatível.

[AS-IS](../02-arquitetura/01-as-is.md) descreve Compose/worker/código real. Tenancy/broker/egress detalhados da proposta não equivalem a enforcement implementado. Snapshots/runbooks locais preservam parâmetros, procedimentos e revisões; metas de RPO/RTO/retention/recursos não são resultados medidos.

## Sequência de leitura deste capítulo

1. [deploy](01-deploy.md).
2. [execucao-e-recuperacao](02-execucao-e-recuperacao.md).
3. [observabilidade](03-observabilidade.md).
4. [handoff](04-handoff.md).
5. [backup](05-backup.md).
6. [restore](06-restore.md).
7. [troubleshooting](07-troubleshooting.md).
8. [incidentes](08-incidentes.md).
9. [economia](09-economia.md).
