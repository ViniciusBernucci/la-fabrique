// Package retention aplica a retenção com integridade.
//
// Logs detalhados (execution_events) com mais de 30 dias são removidos, mas ANTES o resumo
// permanente da execução é gravado (contagens por tipo, primeiro e último horário e o último
// seq): o histórico resumido fica para sempre e o seq nunca é reaproveitado, então os cursores
// do painel continuam válidos. Checkpoints de execuções CONCLUÍDAS (concluída com sucesso ou
// cancelada) são removidos só depois de uma carência (padrão 14 dias), e os objetos que ficam
// sem referência saem junto; os que outro checkpoint usa permanecem.
//
// Nada é removido de uma execução ligada a um incidente (decisão pendente, estado de espera ou
// incerto, efeito externo sem confirmação, item da fila em andamento ou suspenso) nem de um
// checkpoint BLOQUEADO (proteção explícita) ou referenciado por uma sessão do agente. Esses
// continuam íntegros e restauráveis; a execução confere isso ao final.
//
// O relógio é injetável (relógio de teste), há modo de simulação (dry-run) e toda decisão — o
// que saiu e o que ficou, com o motivo — é auditada.
package retention
