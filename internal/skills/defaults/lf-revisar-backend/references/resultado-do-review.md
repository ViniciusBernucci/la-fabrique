# Resultado do review Backend

O schema da etapa tem precedência. Use este formato inicial em uso manual ou avaliação. Se a saída configurada estiver ausente ou for incompatível, sinalize a pendência pelo canal do workflow.

| Campo | Conteúdo |
|---|---|
| `origem` | Contrato, subtarefa, execução, tentativa, repositório, base, revisão candidata e risco efetivo. Em correção ou revalidação, inclua referências ao relatório anterior, aos achados recebidos e à revisão da correção, conforme disponíveis. |
| `modo` | `AVALIACAO`, `CORRECAO` ou `REVALIDACAO`, conforme o despacho. |
| `criterios` | Referência do critério, resultado `ATENDIDO`, `NAO_ATENDIDO` ou `NAO_VALIDADO` e evidência ou motivo. |
| `checks` | Comando ou procedimento, ambiente, revisão testada, evidência sanitizada e resultado `PASSOU`, `FALHOU` ou `NAO_EXECUTADO`. Em falha, registre origem comprovada ou indeterminada. Identifique quais checks eram obrigatórios para o review. |
| `achados` | ID estável, severidade segundo a política, critérios afetados, localização, cenário, impacto, evidência e correção sugerida. Diferencie achados confirmados, falhas preexistentes comprovadas e pontos que ainda precisam de prova. Em revalidação, preserve os IDs e indique `RESOLVIDO`, `PERSISTE` ou `BLOQUEADO`. Registre regressão nova com referência à correção que a introduziu. |
| `veredito` | Em avaliação ou revalidação, use `REPROVADO` quando houver falha confirmada que bloqueia o aceite pela política. Na ausência de falha confirmada bloqueante, use `BLOQUEADO` quando faltar evidência obrigatória. Use `APROVADO` quando as evidências obrigatórias forem suficientes e os achados restantes forem aceitos pela política. |
| `correcao` | Apenas no modo de correção, informe IDs tratados, alterações ou commits confirmados conforme o despacho, checks executados, pendências e revisão que deverá ser revalidada. A etapa de correção entrega trabalho para o gate seguinte. |

Registre causas, impactos e próximos passos para pendências. Inclua somente os trechos necessários das evidências e exclua credenciais, dados reais e detalhes privados do host. Resultado técnico do agente é insumo para a decisão da Sentinela.
