# Resultado do review Frontend

Use o schema da etapa quando fornecido. Os campos abaixo são um formato inicial para uso manual ou avaliação. Se a saída automatizada estiver ausente ou incompatível, sinalize a pendência pelo canal do workflow.

| Campo | Conteúdo |
|---|---|
| `origem` | Referências de contrato, subtarefa, execução, tentativa, revisão candidata, revisão servida, versão da configuração visual e sessão MCP. Em revalidação, inclua a referência do review anterior e da correção. |
| `modo` | `AVALIACAO`, `CORRECAO` ou `REVALIDACAO`, conforme o despacho. |
| `cenarios` | Referência do critério, obrigatoriedade, rota, papel de teste, viewport quando relevante, pré-condições, ações, resultado esperado, resultado observado, evidência e estado `PASSOU`, `FALHOU` ou `BLOQUEADO`. Registre por que um cenário ficou bloqueado e qual insumo falta. |
| `achados` | ID estável, critérios afetados, severidade segundo a política do workflow, impacto, passos de reprodução e evidência. Identifique falhas preexistentes comprovadas e sugestões de baixa severidade. Ao revalidar, preserve o ID e indique se foi resolvido, persiste ou teve verificação bloqueada. |
| `veredito` | Em avaliação ou revalidação, use `REPROVADO` se houver falha confirmada que bloqueia o aceite. Na ausência de falha confirmada bloqueante, use `BLOQUEADO` quando algum cenário obrigatório não puder ser comprovado. Use `APROVADO` quando todos os cenários obrigatórios tiverem evidência válida e não restarem achados bloqueantes segundo a política. Registre também bloqueios parciais e sugestões. |
| `correcao` | Apenas na etapa de correção, devolva os IDs tratados, alterações ou commits confirmados conforme o despacho, testes realmente executados, pendências e revisão a revalidar. O resultado da correção alimenta a etapa de revalidação. |

Um cenário essencial sem navegador, sem evidência da revisão servida ou interrompido por ambiente indisponível fica `BLOQUEADO`. Se a aplicação estiver acessível e o fluxo real comprovar comportamento divergente do esperado, registre `FALHOU`. Capturas e snapshots devem demonstrar o estado observado. Em jornadas com interação, vincule-os aos passos que produziram esse estado.
